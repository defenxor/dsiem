// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package rule

import (
	"net"
	"strconv"
	"strings"
	"sync"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"
)

// DirectiveRule defines the struct for directive rules, this is read-only
// struct.
type DirectiveRule struct {
	Name         string   `json:"name"`
	Stage        int      `json:"stage"`
	PluginID     int      `json:"plugin_id"`
	PluginSID    []int    `json:"plugin_sid"`
	Product      []string `json:"product,omitempty"`
	Category     string   `json:"category,omitempty"`
	SubCategory  []string `json:"subcategory,omitempty"`
	Occurrence   int      `json:"occurrence"`
	From         string   `json:"from"`
	To           string   `json:"to"`
	Type         string   `json:"type"`
	PortFrom     string   `json:"port_from"`
	PortTo       string   `json:"port_to"`
	Protocol     string   `json:"protocol,omitempty"`
	Reliability  int      `json:"reliability"`
	Timeout      int64    `json:"timeout"`
	StartTime    int64    `json:"start_time,omitempty"`
	EndTime      int64    `json:"end_time,omitempty"`
	RcvdTime     int64    `json:"rcvd_time,omitempty"`
	Status       string   `json:"status,omitempty"`
	Events       []string `json:"events,omitempty"`
	StickyDiff   string   `json:"sticky_different,omitempty"`
	CustomData1  string   `json:"custom_data1,omitempty"`
	CustomLabel1 string   `json:"custom_label1,omitempty"`
	CustomData2  string   `json:"custom_data2,omitempty"`
	CustomLabel2 string   `json:"custom_label2,omitempty"`
	CustomData3  string   `json:"custom_data3,omitempty"`
	CustomLabel3 string   `json:"custom_label3,omitempty"`
}

// StickyDiffData hold the previous data for stickydiff rule
// This is mutable, so its separated from DirectiveRule
type StickyDiffData struct {
	sync.RWMutex
	SDiffString []string
	SDiffInt    []int
}

// CustomData combine all custom fields into a struct for easier use
// by backlog and alarm
type CustomData struct {
	Label   string `json:"label"`
	Content string `json:"content"`
}

// SIDPair defines the fields to include during PluginRule quick check
type SIDPair struct {
	PluginID  int
	PluginSID []int
}

// TaxoPair defines the fields to include during TaxonomyRule quick check
type TaxoPair struct {
	Product  []string
	Category string
}

// QuickCheckPluginRule checks event against the key fields in a directive plugin rules
func QuickCheckPluginRule(pairs []SIDPair, e *event.NormalizedEvent) bool {
	for i := range pairs {
		if pairs[i].PluginID != e.PluginID {
			continue
		}
		for _, v := range pairs[i].PluginSID {
			if v == e.PluginSID {
				return true
			}
		}
	}
	return false
}

// QuickCheckTaxoRule checks event against the key fields in a directive taxonomy rules
func QuickCheckTaxoRule(pairs []TaxoPair, e *event.NormalizedEvent) bool {
	for i := range pairs {
		for _, v := range pairs[i].Product {
			if v == e.Product && pairs[i].Category == e.Category {
				return true
			}
		}
	}
	return false
}

// GetQuickCheckPairs returns SIDPairs and TaxoPairs for a given set of directive rules
func GetQuickCheckPairs(r []DirectiveRule) (sidPairs []SIDPair, taxoPairs []TaxoPair) {
	for i := range r {
		if r[i].PluginID != 0 && len(r[i].PluginSID) > 0 {
			sidPairs = append(sidPairs, SIDPair{
				PluginID: r[i].PluginID, PluginSID: r[i].PluginSID})
		}
		if len(r[i].Product) > 0 && r[i].Category != "" {
			taxoPairs = append(taxoPairs, TaxoPair{
				Product: r[i].Product, Category: r[i].Category})
		}
	}
	return
}

// DoesEventMatch check event against rule
// for rule with stickyDiff set, s will be appended as needed
func DoesEventMatch(e event.NormalizedEvent, r DirectiveRule, s *StickyDiffData, connID uint64) bool {

	if r.Type == "PluginRule" {
		return pluginRuleCheck(e, r, s, connID)
	}
	if r.Type == "TaxonomyRule" {
		return taxonomyRuleCheck(e, r, s, connID)
	}
	return false
}

func taxonomyRuleCheck(e event.NormalizedEvent, r DirectiveRule, s *StickyDiffData, connID uint64) (ret bool) {
	// product is required and category is required
	if r.Category != e.Category {
		return
	}
	prodMatch := false
	for i := range r.Product {
		if r.Product[i] == e.Product {
			prodMatch = true
			break
		}
	}
	if !prodMatch {
		return
	}

	// subcategory is optional and can use "ANY"
	l := len(r.SubCategory)
	if l != 0 {
		scMatch := false
		for i := range r.SubCategory {
			if r.SubCategory[i] == e.SubCategory || r.SubCategory[i] == "ANY" {
				scMatch = true
				break
			}
		}
		if !scMatch {
			return
		}
	}
	ret = ipPortCheck(e, r, s, connID)
	return
}

// this by definition only applies to pluginRule not taxonomyRule
func customDataCheck(e event.NormalizedEvent, r DirectiveRule, s *StickyDiffData, connID uint64) (ret bool) {

	var r1, r2, r3 = true, true, true
	if r.CustomData1 != "" && r.CustomData1 != "ANY" {
		r1 = matchText(r.CustomData1, e.CustomData1)
	}
	if r.CustomData2 != "" && r.CustomData2 != "ANY" {
		r2 = matchText(r.CustomData2, e.CustomData2)
	}
	if r.CustomData3 != "" && r.CustomData3 != "ANY" {
		r3 = matchText(r.CustomData3, e.CustomData3)
	}

	switch {
	case r.StickyDiff == "CUSTOM_DATA1":
		_ = isStringStickyDiff(e.CustomData1, s)
	case r.StickyDiff == "CUSTOM_DATA2":
		_ = isStringStickyDiff(e.CustomData2, s)
	case r.StickyDiff == "CUSTOM_DATA3":
		_ = isStringStickyDiff(e.CustomData3, s)
	default:
	}
	ret = r1 && r2 && r3
	return
}

func pluginRuleCheck(e event.NormalizedEvent, r DirectiveRule, s *StickyDiffData, connID uint64) bool {
	if e.PluginID != r.PluginID {
		return false
	}

	var sidMatch bool
	for i := range r.PluginSID {
		if r.PluginSID[i] == e.PluginSID {
			sidMatch = true
			break
		}
	}

	if !sidMatch {
		return false
	}

	if r.StickyDiff == "PLUGIN_SID" {
		_ = isIntStickyDiff(e.PluginSID, s)
	}

	isMatch := ipPortCheck(e, r, s, connID)
	if isMatch {
		isMatch = customDataCheck(e, r, s, connID)
	}

	return isMatch
}

func ipPortCheck(e event.NormalizedEvent, r DirectiveRule, s *StickyDiffData, connID uint64) (ret bool) {
	eSrcInHomeNet := e.SrcIPInHomeNet()
	if r.From == "HOME_NET" && !eSrcInHomeNet {
		return
	}
	if r.From == "!HOME_NET" && eSrcInHomeNet {
		return
	}
	// covers  r.From == "IP", r.From == "IP1, IP2, !IP3", r.From == CIDR-netaddr, r.From == "CIDR1, CIDR2, !CIDR3"
	if r.From != "HOME_NET" && r.From != "!HOME_NET" && r.From != "ANY" &&
		!isNetAddrMatchCSVRule(r.From, e.SrcIP) {
		return
	}
	eDstInHomeNet := e.DstIPInHomeNet()
	if r.To == "HOME_NET" && !eDstInHomeNet {
		return
	}
	if r.To == "!HOME_NET" && eDstInHomeNet {
		return
	}
	// covers  r.To == "IP", r.To == "IP1, IP2, !IP3", r.To == CIDR-netaddr, r.To == "CIDR1, CIDR2, !CIDR3"
	if r.To != "HOME_NET" && r.To != "!HOME_NET" && r.To != "ANY" &&
		!isNetAddrMatchCSVRule(r.To, e.DstIP) {
		return
	}
	if r.PortFrom != "ANY" && !isStringMatchCSVRule(r.PortFrom, strconv.Itoa(e.SrcPort)) {
		return
	}
	if r.PortTo != "ANY" && !isStringMatchCSVRule(r.PortTo, strconv.Itoa(e.DstPort)) {
		return
	}

	switch {
	case r.StickyDiff == "SRC_IP":
		_ = isStringStickyDiff(e.SrcIP, s)
	case r.StickyDiff == "DST_IP":
		_ = isStringStickyDiff(e.DstIP, s)
	case r.StickyDiff == "SRC_PORT":
		_ = isIntStickyDiff(e.SrcPort, s)
	case r.StickyDiff == "DST_PORT":
		_ = isIntStickyDiff(e.DstPort, s)
	default:
	}
	// SrcIP, DstIP, SrcPort, DstPort all match
	return true
}

// isStringStickyDiff check if v fulfill stickydiff condition
// ret code isn't used right now because the check is done in backlog
func isStringStickyDiff(v string, r *StickyDiffData) bool {
	// r could be nil on first check
	if r == nil {
		return true
	}
	r.RLock()
	for i := range r.SDiffString {
		if r.SDiffString[i] == v {
			r.RUnlock()
			return false
		}
	}
	r.RUnlock()
	// add it to the coll
	r.Lock()
	r.SDiffString = append(r.SDiffString, v)
	r.Unlock()
	return true
}

// isIntStickyDiff check if v fulfill stickydiff condition
// ret code isn't used right now because the check is done in backlog
func isIntStickyDiff(v int, r *StickyDiffData) (match bool) {
	// r could be nil on first check
	if r == nil {
		return true
	}
	r.RLock()
	for i := range r.SDiffInt {
		if r.SDiffInt[i] == v {
			r.RUnlock()
			return false
		}
	}
	r.RUnlock()
	// add it to the coll
	r.Lock()
	r.SDiffInt = append(r.SDiffInt, v)
	r.Unlock()
	return true
}

func isNetAddrMatchCSVRule(rulesInCSV, term string) bool {
	// rulesInCSV is something like stringA, stringB, !stringC, !stringD
	sSlice := str.CsvToSlice(rulesInCSV)

	var ipB net.IP
	if !strings.Contains(term, "/") {
		term = term + "/32"
	}
	var err error
	ipB, _, err = net.ParseCIDR(term)
	if err != nil {
		log.Warn(log.M{Msg: "Unable to parse IP address: " + term + ". Make sure the plugin is configured correctly!"})
		return false
	}

	var match bool
	for _, v := range sSlice {

		isInverse := strings.HasPrefix(v, "!")
		if isInverse {
			v = str.TrimLeftChar(v)
		}

		termIsEqual := false
		if !strings.Contains(v, "/") {
			v = v + "/32"
		}
		_, ipnetA, err := net.ParseCIDR(v)
		if err != nil {
			log.Warn(log.M{Msg: "Unable to parse CIDR address: " + v + ". Make sure the directive is configured correctly!"})
			return false
		}
		termIsEqual = ipnetA.Contains(ipB)

		/*
			The correct logic here is to AND all inverse rules,
			and then OR the result with all the non-inverse rules.
			The following code implement that with shortcuts.
		*/

		// break early if !condition is violated
		if isInverse && termIsEqual {
			match = false
			break
		}
		// break early if condition is fulfilled
		if !isInverse && termIsEqual {
			match = true
			break
		}
		// if !condition is fulfilled, continue evaluation of next in item
		if isInverse && !termIsEqual {
			match = true
		}
		// !isInverse && !termIsEqual should result in match = false (default)
		// so there's no need to handle it
	}

	return match
}

// matchText match the given term against the subject, if the subject is a comma-separated-values,
// split it into slice of strings, match its value one by one, and returns if one of the value matches.
// otherwise, matchText will do non case-sensitve match for the subject and term.
func matchText(subject, term string) bool {

	if isCSV(subject) {
		return isStringMatchCSVRule(subject, term)
	}

	return matchTextNonSensitive(subject, term)
}

// isCSV determines wether the given term is a comma separated list of strings or not.
// FIXME: this is currently implemented by checking if the term contains comma character ",", which
// can cause misbehave if the term is actually a non-csv long string that contains comma character.
func isCSV(term string) bool {
	return strings.Contains(term, ",")
}

func matchTextNonSensitive(term1, term2 string) bool {
	var inverse bool
	if strings.HasPrefix(term1, "!") {
		term1 = str.TrimLeftChar(term1)
		inverse = true
	}

	match := strings.TrimSpace(strings.ToLower(term1)) == strings.TrimSpace(strings.ToLower(term2))

	if inverse {
		return !match
	}

	return match
}

func isStringMatchCSVRule(rulesInCSV string, term string) (match bool) {
	// s is something like stringA, stringB, !stringC, !stringD
	sSlice := str.CsvToSlice(rulesInCSV)
	for _, v := range sSlice {

		isInverse := strings.HasPrefix(v, "!")
		if isInverse {
			v = str.TrimLeftChar(v)
		}

		termIsEqual := false
		termIsEqual = v == term

		/*
			The correct logic here is to AND all inverse rules,
			and then OR the result with all the non-inverse rules.
			The following code implement that with shortcuts.
		*/

		// break early if !condition is violated
		if isInverse && termIsEqual {
			match = false
			break
		}
		// break early if condition is fulfilled
		if !isInverse && termIsEqual {
			match = true
			break
		}
		// if !condition is fulfilled, continue evaluation of next in item
		if isInverse && !termIsEqual {
			match = true
		}
		// !isInverse && !termIsEqual should result in match = false (default)
		// so there's no need to handle it
	}

	return
}

// AppendUniqCustomData returns unique custom data slice
func AppendUniqCustomData(prev []CustomData, label string, content string) []CustomData {
	if label == "" || content == "" {
		return prev
	}
	for _, v := range prev {
		if v.Label == label && v.Content == content {
			return prev
		}
	}
	cd := CustomData{
		Label:   label,
		Content: content,
	}
	return append(prev, cd)
}
