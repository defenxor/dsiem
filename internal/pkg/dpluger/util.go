package dpluger

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
)

const (
	FieldTypeText    = "text"
	FieldTypeKeyword = "keyword"
)

var ErrFieldMappingNotExist = errors.New("field mapping does not exist")

func directivesEqual(dir1, dir2 siem.Directives) error {
	for _, directive := range dir2.Dirs {
		var found bool
		for _, existing := range dir1.Dirs {
			if existing.ID == directive.ID && directive.Name == existing.Name {
				found = true

				if errors := directiveEqual(existing, directive); len(errors) > 0 {
					return fmt.Errorf(joinError(errors, ","))
				}

				break
			}
		}

		if !found {
			return fmt.Errorf("directive '%s' is not found in existing directives", directive.Name)
		}
	}

	return nil
}

func directiveEqual(dir1, dir2 siem.Directive) []error {
	errors := make([]error, 0)
	if dir1.ID != dir2.ID {
		errors = append(errors, fmt.Errorf("directive ID is not equal, %d != %d", dir1.ID, dir2.ID))
	}

	if dir1.Name != dir2.Name {
		errors = append(errors, fmt.Errorf("directive name is not equal, %s != %s", dir1.Name, dir2.Name))
	}

	if dir1.Priority != dir2.Priority {
		errors = append(errors, fmt.Errorf("directive priority is not equal, %d != %d", dir1.Priority, dir2.Priority))
	}

	if dir1.Disabled != dir2.Disabled {
		errors = append(errors, fmt.Errorf("directive disabled flag is not equal, %t != %t", dir1.Disabled, dir2.Disabled))
	}

	if dir1.AllRulesAlwaysActive != dir2.AllRulesAlwaysActive {
		errors = append(errors, fmt.Errorf("directive always active flag is not equal, %t != %t", dir1.AllRulesAlwaysActive, dir2.AllRulesAlwaysActive))
	}

	if dir1.Kingdom != dir2.Kingdom {
		errors = append(errors, fmt.Errorf("directive kingdom is not equal, %s != %s", dir1.Kingdom, dir2.Kingdom))
	}

	if dir1.Category != dir2.Category {
		errors = append(errors, fmt.Errorf("directive category is not equal, %s != %s", dir1.Category, dir2.Category))
	}

	if len(dir1.Rules) != len(dir2.Rules) {
		errors = append(errors, fmt.Errorf("rule count is not the same, %d != %d", len(dir1.Rules), len(dir2.Rules)))
	}

	for _, rule1 := range dir1.Rules {
		var found bool

		for _, rule2 := range dir2.Rules {
			if errs := ruleEqual(rule1, rule2); len(errs) > 0 {
				errors = append(errors, errs...)
			} else {
				found = true
				break
			}
		}

		if !found {
			errors = append(errors, fmt.Errorf("rule '%s' is not found", rule1.Name))
		}
	}

	return errors
}

func joinError(errors []error, sep string) string {
	str := make([]string, 0, len(errors))
	for _, err := range errors {
		str = append(str, err.Error())
	}

	return strings.Join(str, sep)
}

func ruleEqual(rule1, rule2 rule.DirectiveRule) []error {
	errors := make([]error, 0)

	if rule1.Name != rule2.Name {
		errors = append(errors, fmt.Errorf("rule name is not equal, %s != %s", rule1.Name, rule2.Name))
	}

	if rule1.Stage != rule2.Stage {
		errors = append(errors, fmt.Errorf("rule stage is not the same, %d != %d", rule1.Stage, rule2.Stage))
	}

	if rule1.PluginID != rule2.PluginID {
		errors = append(errors, fmt.Errorf("rule PluginID is not the same, %d != %d", rule1.PluginID, rule2.PluginID))
	}

	if !reflect.DeepEqual(rule1.PluginSID, rule2.PluginSID) {
		errors = append(errors, fmt.Errorf("rule pluginSID is not the same,%#v != %#v", rule1.PluginSID, rule2.PluginSID))
	}

	if !reflect.DeepEqual(rule1.Product, rule2.Product) {
		errors = append(errors, fmt.Errorf("rule Product is not the same,%#v != %#v", rule1.Product, rule2.Product))
	}

	if rule1.Category != rule2.Category {
		errors = append(errors, fmt.Errorf("rule Category is not the same, %s != %s", rule1.Category, rule2.Category))
	}

	if !reflect.DeepEqual(rule1.SubCategory, rule2.SubCategory) {
		errors = append(errors, fmt.Errorf("rule SubCategory is not the same,%#v != %#v", rule1.SubCategory, rule2.SubCategory))
	}

	if rule1.Occurrence != rule2.Occurrence {
		errors = append(errors, fmt.Errorf("rule Occurrence is not the same, %d != %d", rule1.Occurrence, rule2.Occurrence))
	}

	if rule1.From != rule2.From {
		errors = append(errors, fmt.Errorf("rule From is not the same, %s != %s", rule1.From, rule2.From))
	}

	if rule1.To != rule2.To {
		errors = append(errors, fmt.Errorf("rule To is not the same, %s != %s", rule1.To, rule2.To))
	}

	if rule1.Type != rule2.Type {
		errors = append(errors, fmt.Errorf("rule Type is not the same, %s != %s", rule1.Type, rule2.Type))
	}

	if rule1.PortFrom != rule2.PortFrom {
		errors = append(errors, fmt.Errorf("rule PortFrom is not the same, %s != %s", rule1.PortFrom, rule2.PortFrom))
	}

	if rule1.PortTo != rule2.PortTo {
		errors = append(errors, fmt.Errorf("rule PortTo is not the same, %s != %s", rule1.PortTo, rule2.PortTo))
	}

	if rule1.Protocol != rule2.Protocol {
		errors = append(errors, fmt.Errorf("rule Protocol is not the same, %s != %s", rule1.Protocol, rule2.Protocol))
	}

	if rule1.Reliability != rule2.Reliability {
		errors = append(errors, fmt.Errorf("rule Reliability is not the same, %d != %d", rule1.Reliability, rule2.Reliability))
	}

	if rule1.Timeout != rule2.Timeout {
		errors = append(errors, fmt.Errorf("rule Timeout is not the same, %d != %d", rule1.Timeout, rule2.Timeout))
	}

	if rule1.Occurrence != rule2.Occurrence {
		errors = append(errors, fmt.Errorf("rule Occurrence is not the same, %d != %d", rule1.Occurrence, rule2.Occurrence))
	}

	if rule1.StartTime != rule2.StartTime {
		errors = append(errors, fmt.Errorf("rule StartTime is not the same, %d != %d", rule1.StartTime, rule2.StartTime))
	}

	if rule1.EndTime != rule2.EndTime {
		errors = append(errors, fmt.Errorf("rule EndTime is not the same, %d != %d", rule1.EndTime, rule2.EndTime))
	}

	if rule1.RcvdTime != rule2.RcvdTime {
		errors = append(errors, fmt.Errorf("rule RcvdTime is not the same, %d != %d", rule1.RcvdTime, rule2.RcvdTime))
	}

	if rule1.Status != rule2.Status {
		errors = append(errors, fmt.Errorf("rule Status is not the same, %s != %s", rule1.Status, rule2.Status))
	}

	if !reflect.DeepEqual(rule1.Events, rule2.Events) {
		errors = append(errors, fmt.Errorf("rule Events is not the same,%#v != %#v", rule1.Events, rule2.Events))
	}

	if rule1.StickyDiff != rule2.StickyDiff {
		errors = append(errors, fmt.Errorf("rule StickyDiff is not the same, %s != %s", rule1.StickyDiff, rule2.StickyDiff))
	}

	if rule1.CustomData1 != rule2.CustomData1 {
		errors = append(errors, fmt.Errorf("rule CustomData1 is not the same, %s != %s", rule1.CustomData1, rule2.CustomData1))
	}

	if rule1.CustomLabel1 != rule2.CustomLabel1 {
		errors = append(errors, fmt.Errorf("rule CustomLabel1 is not the same, %s != %s", rule1.CustomLabel1, rule2.CustomLabel1))
	}

	if rule1.CustomData2 != rule2.CustomData2 {
		errors = append(errors, fmt.Errorf("rule CustomData2 is not the same, %s != %s", rule1.CustomData2, rule2.CustomData2))
	}

	if rule1.CustomLabel2 != rule2.CustomLabel2 {
		errors = append(errors, fmt.Errorf("rule CustomLabel2 is not the same, %s != %s", rule1.CustomLabel2, rule2.CustomLabel2))
	}

	if rule1.CustomData3 != rule2.CustomData3 {
		errors = append(errors, fmt.Errorf("rule CustomData3 is not the same, %s != %s", rule1.CustomData3, rule2.CustomData3))
	}

	if rule1.CustomLabel3 != rule2.CustomLabel3 {
		errors = append(errors, fmt.Errorf("rule CustomLabel3 is not the same, %s != %s", rule1.CustomLabel3, rule2.CustomLabel3))
	}

	return errors
}

var (
	ErrIntValueExceedBoundary = errors.New("integer value exceeds maximum value boundary")
)

// toInt safely convert interface into int.
func toInt(v interface{}) (int, error) {
	if v == nil {
		return 0, nil
	}

	switch t := v.(type) {
	case int:
		return t, nil
	case float64:
		if t >= 0 && t < math.MaxInt32 {
			return int(t), nil
		}

		return 0, ErrIntValueExceedBoundary
	case int64:
		if t >= 0 && t < math.MaxInt32 {
			return int(t), nil
		}

		return 0, ErrIntValueExceedBoundary
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("expecting numeric value, got '%s'", t)
		}

		if n >= 0 && n < math.MaxInt32 {
			return int(n), nil
		}

		return 0, ErrIntValueExceedBoundary
	}

	return 0, fmt.Errorf("invalid numeric value type, %T", v)
}
