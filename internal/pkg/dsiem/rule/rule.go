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

	l := len(r.SubCategory)
	if l == 0 {
		return
	}
	scMatch := false
	for i := range r.SubCategory {
		if r.SubCategory[i] == e.SubCategory {
			scMatch = true
			break
		}
	}
	if !scMatch {
		return
	}
	ret = ipPortCheck(e, r, s, connID)
	return
}

// this by definition only applies to pluginRule not taxonomyRule
func customDataCheck(e event.NormalizedEvent, r DirectiveRule, s *StickyDiffData, connID uint64) (ret bool) {

	var r1, r2, r3 = true, true, true
	if r.CustomData1 != "" {
		r1 = r.CustomData1 == e.CustomData1
	}
	if r.CustomData2 != "" {
		r2 = r.CustomData2 == e.CustomData2
	}
	if r.CustomData3 != "" {
		r3 = r.CustomData3 == e.CustomData3
	}
	ret = r1 && r2 && r3
	return
}

func pluginRuleCheck(e event.NormalizedEvent, r DirectiveRule, s *StickyDiffData, connID uint64) (ret bool) {
	if e.PluginID != r.PluginID {
		return
	}
	sidMatch := false
	for i := range r.PluginSID {
		if r.PluginSID[i] == e.PluginSID {
			sidMatch = true
			break
		}
	}
	if !sidMatch {
		return
	}
	if r.StickyDiff == "PLUGIN_SID" {
		_ = isIntStickyDiff(e.PluginSID, s)
	}
	ret = ipPortCheck(e, r, s, connID)
	if ret {
		ret = customDataCheck(e, r, s, connID)
	}
	return
}

func ipPortCheck(e event.NormalizedEvent, r DirectiveRule, s *StickyDiffData, connID uint64) (ret bool) {
	eSrcInHomeNet := e.SrcIPInHomeNet()
	if r.From == "HOME_NET" && !eSrcInHomeNet {
		return
	}
	if r.From == "!HOME_NET" && eSrcInHomeNet {
		return
	}
	// covers  r.From == "IP", r.From == "IP1, IP2", r.From == CIDR-netaddr, r.From == "CIDR1, CIDR2"
	if r.From != "HOME_NET" && r.From != "!HOME_NET" && r.From != "ANY" &&
		!str.IsInCSVList(r.From, e.SrcIP) && !isIPinCIDR(e.SrcIP, r.From) {
		return
	}
	eDstInHomeNet := e.DstIPInHomeNet()
	if r.To == "HOME_NET" && !eDstInHomeNet {
		return
	}
	if r.To == "!HOME_NET" && eDstInHomeNet {
		return
	}
	// covers  r.To == "IP", r.To == "IP1, IP2", r.To == CIDR-netaddr, r.To == "CIDR1, CIDR2"
	if r.To != "HOME_NET" && r.To != "!HOME_NET" && r.To != "ANY" &&
		!str.IsInCSVList(r.To, e.DstIP) && !isIPinCIDR(e.DstIP, r.To) {
		return
	}
	if r.PortFrom != "ANY" && !str.IsInCSVList(r.PortFrom, strconv.Itoa(e.SrcPort)) {
		return
	}
	if r.PortTo != "ANY" && !str.IsInCSVList(r.PortTo, strconv.Itoa(e.DstPort)) {
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

// IsStringStickyDiff check if v fulfill stickydiff condition
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

// IsIntStickyDiff check if v fulfill stickydiff condition
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

func isIPinCIDR(ip string, netcidr string) (found bool) {
	// first convert to slice, because netcidr maybe in a form of "cidr1,cidr2..."
	cleaned := strings.Replace(netcidr, ",", " ", -1)
	cidrSlice := strings.Fields(cleaned)

	found = false
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}
	ipB, _, err := net.ParseCIDR(ip)
	if err != nil {
		log.Warn(log.M{Msg: "Unable to parse IP address: " + ip + ". Make sure the plugin is configured correctly!"})
		return
	}

	for _, v := range cidrSlice {
		if !strings.Contains(v, "/") {
			v = v + "/32"
		}
		_, ipnetA, err := net.ParseCIDR(v)
		if err != nil {
			log.Warn(log.M{Msg: "Unable to parse CIDR address: " + v + ". Make sure the directive is configured correctly!"})
			return
		}
		if ipnetA.Contains(ipB) {
			found = true
			break
		}
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
