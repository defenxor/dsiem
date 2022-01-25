package siem

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"
)

var (
	validRuleType = []string{
		"PluginRule",
		"TaxonomyRule",
	}
)

var (
	refRegexp               = regexp.MustCompile("^:[1-9][0-9]?$")
	ErrZeroStage            = errors.New("can not use 0 as rule stage")
	ErrInvalidRuleType      = fmt.Errorf("invalid rule type, valid types: %v", validRuleType)
	ErrInvalidPluginID      = errors.New("PluginRule requires PluginID to be 1 or higher")
	ErrNoPluginSID          = errors.New("PluginRule requires PluginSID to be defined")
	ErrInvalidPluginSID     = errors.New("PluginRule requires PluginSID to be 1 or higher")
	ErrNoProduct            = errors.New("TaxonomyRule requires Product to be defined")
	ErrNoCategory           = errors.New("TaxonomyRule requires Category to be defined")
	ErrNoDirectiveName      = errors.New("Directive name cannot be empty")
	ErrNoDirectiveKingdom   = errors.New("Directive kingdom cannot be empty")
	ErrNoDirectiveCategory  = errors.New("Directive category cannot be empty")
	ErrReferenceOnFirstRule = errors.New("first rule cannot contain reference")
	ErrInvalidReference     = errors.New("invalid reference number, must be larger than 0 and less than the rule count")
	ErrEmptyFromTo          = errors.New("rule From/To cannot be empty")
)

func ValidateDirective(d *Directive, res *Directives) (err error) {
	for _, v := range res.Dirs {
		if v.ID == d.ID {
			return fmt.Errorf("id '%d' is already used as an ID by other directive", d.ID)
		}
	}

	if d.Name == "" {
		return ErrNoDirectiveName
	}

	if d.Kingdom == "" {
		return ErrNoDirectiveKingdom
	}

	if d.Category == "" {
		return ErrNoDirectiveCategory
	}

	if d.Priority < 1 || d.Priority > 5 {
		log.Warn(log.M{Msg: fmt.Sprintf("directive %d has wrong priority set (%d), configuring it to 1", d.ID, d.Priority)})
		d.Priority = 1
	}

	if len(d.Rules) <= 1 {
		return fmt.Errorf("directive %d has no rule therefore has no effect, or only 1 rule and therefore will never expire", d.ID)
	}

	if err := ValidateRules(d); err != nil {
		return fmt.Errorf("directive %d contains invalid rule, %s", d.ID, err.Error())
	}

	return nil
}

func ValidateRules(d *Directive) error {
	stages := make([]int, 0)
	for idx := range d.Rules {
		if d.Rules[idx].Stage == 0 {
			return ErrZeroStage
		}

		for i := range stages {
			if stages[i] == d.Rules[idx].Stage {
				return fmt.Errorf("duplicate rule stage found (%d)", d.Rules[idx].Stage)
			}
		}

		if d.Rules[idx].Stage == 1 && d.Rules[idx].Occurrence != 1 {
			log.Warn(log.M{Msg: fmt.Sprintf("Directive '%d' rule %d has wrong occurence set (%d), configuring it to 1", d.ID, d.Rules[idx].Stage, d.Rules[idx].Occurrence)})
			d.Rules[idx].Occurrence = 1
		}

		if !isValidRuleType(d.Rules[idx].Type) {
			return ErrInvalidRuleType
		}

		if d.Rules[idx].Type == "PluginRule" {
			if d.Rules[idx].PluginID < 1 {
				return ErrInvalidPluginID
			}

			if len(d.Rules[idx].PluginSID) == 0 {
				return ErrNoPluginSID
			}

			for i := range d.Rules[idx].PluginSID {
				if d.Rules[idx].PluginSID[i] < 1 {
					return ErrInvalidPluginSID
				}
			}
		}

		if d.Rules[idx].Type == "TaxonomyRule" {
			if len(d.Rules[idx].Product) == 0 {
				return ErrNoProduct
			}
			if d.Rules[idx].Category == "" {
				return ErrNoCategory
			}
		}

		// reliability maybe 0 for the first rule!
		if d.Rules[idx].Reliability < 0 {
			log.Warn(log.M{Msg: fmt.Sprintf("Directive %d rule %d has wrong reliability set (%d), configuring it to 0", d.ID, d.Rules[idx].Stage, d.Rules[idx].Reliability)})
			d.Rules[idx].Reliability = 0
		}

		if d.Rules[idx].Reliability > 10 {
			log.Warn(log.M{Msg: fmt.Sprintf("Directive %d rule %d has wrong reliability set (%d), configuring it to 10", d.ID, d.Rules[idx].Stage, d.Rules[idx].Reliability)})
			d.Rules[idx].Reliability = 10
		}

		isFirstStage := d.Rules[idx].Stage == 1
		ruleCount := len(d.Rules)
		if err := validateFromTo(d.Rules[idx].From, isFirstStage, ruleCount); err != nil {
			return err
		}

		if err := validateFromTo(d.Rules[idx].To, isFirstStage, ruleCount); err != nil {
			return err
		}

		if err := validatePort(d.Rules[idx].PortFrom, isFirstStage, ruleCount); err != nil {
			return err
		}

		if err := validatePort(d.Rules[idx].PortTo, isFirstStage, ruleCount); err != nil {
			return err
		}

		stages = append(stages, d.Rules[idx].Stage)
	}

	return nil
}

func validatePort(s string, isFirstRule bool, ruleCount int) error {
	if s == "ANY" {
		return nil
	}

	if isReference(s) {
		if isFirstRule {
			return ErrReferenceOnFirstRule
		}

		return validateReference(s, int64(ruleCount))
	}

	sSlice := str.CsvToSlice(s)
	for _, v := range sSlice {
		isInverse := strings.HasPrefix(v, "!")
		if isInverse {
			v = str.TrimLeftChar(v)
		}
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		if n <= 1 || n >= 65535 {
			return fmt.Errorf("%s is not a valid TCP/IP port number", v)
		}
	}
	return nil
}

func validateFromTo(s string, isFirstRule bool, ruleCount int) (err error) {
	if s == "" {
		return ErrEmptyFromTo
	}

	if s == "ANY" || s == "HOME_NET" || s == "!HOME_NET" {
		return nil
	}

	if isReference(s) {
		if isFirstRule {
			return ErrReferenceOnFirstRule
		}

		return validateReference(s, int64(ruleCount))
	}

	// covers  r.To == "IP", r.To == "IP1, IP2, !IP3", r.To == CIDR-netaddr, r.To == "CIDR1, CIDR2, !CIDR3"
	sSlice := str.CsvToSlice(s)
	for _, v := range sSlice {
		if !strings.Contains(v, "/") {
			v = v + "/32"
		}
		isInverse := strings.HasPrefix(v, "!")
		if isInverse {
			v = str.TrimLeftChar(v)
		}

		if _, _, err := net.ParseCIDR(v); err != nil {
			return fmt.Errorf("%s is not a valid IPv4 address", v)
		}
	}

	return nil
}

func isValidRuleType(t string) bool {
	for _, rt := range validRuleType {
		if rt == t {
			return true
		}
	}

	return false
}

func isReference(str string) bool {
	return strings.HasPrefix(str, ":")
}

func validateReference(ref string, ruleCount int64) error {
	if !refRegexp.MatchString(ref) {
		return ErrInvalidReference
	}

	if n, _ := str.RefToDigit(ref); n >= ruleCount {
		return ErrInvalidReference
	}

	return nil
}
