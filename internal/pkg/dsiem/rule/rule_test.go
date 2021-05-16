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
	"path"
	"reflect"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

func TestTermInCSV(t *testing.T) {
	type termTest struct {
		term      string
		csvRules  string
		isNetAddr bool
		expected  bool
	}

	log.Setup(false)

	var tbl = []termTest{
		{"192.168.0", "192.168.0.0/16", true, false},
		{"192.168.0.1", "192.168.0/16", true, false},
		{"192.168.0.1", "192.168.0.0/16", true, true},
		{"192.168.0.1", "!10.0.0.1/16", true, true},
		{"192.168.0.1", "!10.0.0.1/16, 192.168.0.0/24", true, true},
		{"192.168.0.1", "!192.168.0.1/16", true, false},
		{"192.168.0.1", "10.0.0.0/16, !192.168.0.1/16", true, false},
		{"192.168.0.1", "10.0.0.0/16, !192.168.0.1/16, 192.168.0.0/16", true, false},
		{"1231", "1000, 1001", false, false},
		{"1231", "!1231, 1001", false, false},
		{"1231", "1000, !1231", false, false},
		{"1231", "1231, !1231", false, true},
		{"1231", "!1231, 1231", false, false},
		{"1231", "!1000, !1001", false, true},
		{"1231", "!1000, 1001", false, true},
		{"1231", "1001, !1000", false, true},
		{"1231", "!1000, 1231", false, true},
		{"foo", "!bar, foobar, foo", false, true},
	}

	for _, tt := range tbl {
		actual := isStrMatchCSVRule(tt.csvRules, tt.term, tt.isNetAddr)
		if actual != tt.expected {
			t.Errorf("IP %s in %s result is %v. Expected %v.", tt.term, tt.csvRules, actual, tt.expected)
		}
	}

}

func TestGetQuickCheckPairs(t *testing.T) {
	spRef := []SIDPair{
		{PluginID: 1, PluginSID: []int{1, 2}},
		{PluginID: 1, PluginSID: []int{2, 3}},
	}
	tpRef := []TaxoPair{
		{Product: []string{"P1"}, Category: "C1"},
		{Product: []string{"P1, P2"}, Category: "C1"},
	}
	dr := []DirectiveRule{}
	for i := range spRef {
		dr = append(dr, DirectiveRule{
			PluginID:  spRef[i].PluginID,
			PluginSID: spRef[i].PluginSID,
			Product:   tpRef[i].Product,
			Category:  tpRef[i].Category,
		})
	}
	sp, tp := GetQuickCheckPairs(dr)
	if !reflect.DeepEqual(sp, spRef) {
		t.Fatalf("sp expected to be equal to spRef. sp: %v spRef: %v", sp, spRef)
	}
	if !reflect.DeepEqual(tp, tpRef) {
		t.Fatalf("tp expected to be equal to tpRef. tp: %v tpRef: %v", tp, tpRef)
	}
}

func TestQuickCheck(t *testing.T) {
	type tpTest struct {
		pair     []TaxoPair
		evt      event.NormalizedEvent
		expected bool
	}

	var tpTbl = []tpTest{
		{
			[]TaxoPair{TaxoPair{Product: []string{"P1", "P2"}, Category: "C1"}},
			event.NormalizedEvent{Product: "P1", Category: "C1"},
			true,
		},
		{
			[]TaxoPair{TaxoPair{Product: []string{"P1", "P2"}, Category: "C1"}},
			event.NormalizedEvent{Product: "P1", Category: "C2"},
			false,
		},
		{
			[]TaxoPair{TaxoPair{Product: []string{"P1", "P2"}, Category: "C1"}},
			event.NormalizedEvent{Product: "P3", Category: "C1"},
			false,
		},
	}

	for _, tt := range tpTbl {
		actual := QuickCheckTaxoRule(tt.pair, &tt.evt)
		if actual != tt.expected {
			t.Fatalf("QuickCheck taxo actual != expected. TaxoPair: %v, Event: %v",
				tt.pair, tt.evt)
		}
	}

	type spTest struct {
		pair     []SIDPair
		evt      event.NormalizedEvent
		expected bool
	}

	var spTbl = []spTest{
		{
			[]SIDPair{SIDPair{PluginID: 10, PluginSID: []int{1, 2}}},
			event.NormalizedEvent{PluginID: 10, PluginSID: 1},
			true,
		},
		{
			[]SIDPair{SIDPair{PluginID: 10, PluginSID: []int{1, 2}}},
			event.NormalizedEvent{PluginID: 10, PluginSID: 3},
			false,
		},
		{
			[]SIDPair{SIDPair{PluginID: 10, PluginSID: []int{1, 2}}},
			event.NormalizedEvent{PluginID: 9, PluginSID: 1},
			false,
		},
	}

	for _, tt := range spTbl {
		actual := QuickCheckPluginRule(tt.pair, &tt.evt)
		if actual != tt.expected {
			t.Fatalf("QuickCheck SID actual != expected. SIDPair: %v, Event: %v",
				tt.pair, tt.evt)
		}
	}

}

func TestRule(t *testing.T) {

	d, err := test.DirEnv(false)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Using base dir %s", d)
	err = asset.Init(path.Join(d, "configs"))
	if err != nil {
		t.Fatal(err)
	}

	type ruleTests struct {
		n        int
		e        event.NormalizedEvent
		r        DirectiveRule
		s        *StickyDiffData
		expected bool
	}

	e1 := event.NormalizedEvent{
		PluginID:    1001,
		PluginSID:   50001,
		Product:     "IDS",
		Category:    "Malware",
		SubCategory: "C&C Communication",
		SrcIP:       "192.168.0.1",
		DstIP:       "8.8.8.200",
		SrcPort:     31337,
		DstPort:     80,
	}
	r1 := DirectiveRule{
		Type:        "PluginRule",
		PluginID:    1001,
		PluginSID:   []int{50001},
		Product:     []string{"IDS"},
		Category:    "Malware",
		SubCategory: []string{"C&C Communication"},
		From:        "HOME_NET",
		To:          "ANY",
		PortFrom:    "ANY",
		PortTo:      "ANY",
		Protocol:    "ANY",
	}
	s1 := &StickyDiffData{}

	r2 := r1
	r2.Type = "TaxonomyRule"

	r3 := r1
	r3.PluginSID = []int{50002}

	r4 := r2
	r4.Category = "Scanning"

	r5 := r1
	r5.PluginID = 1002

	r6 := r2
	r6.Product = []string{"Firewall"}

	r7 := r2
	r7.SubCategory = []string{}

	r8 := r2
	r8.SubCategory = []string{"Firewall Allow"}

	r9 := r1
	r9.Type = "Unknown"

	e2 := e1
	e2.SrcIP = e1.DstIP
	e2.DstIP = e1.SrcIP
	r10 := r1

	r11 := r1
	r11.From = "!HOME_NET"

	r12 := r1
	r12.From = "192.168.0.10"

	r13 := r1
	r13.To = "HOME_NET"

	e3 := e1
	e3.DstIP = e1.SrcIP
	r14 := r1
	r14.To = "!HOME_NET"

	r15 := r1
	r15.To = "192.168.0.10"

	r16 := r1
	r16.PortFrom = "1337"

	r17 := r1
	r17.PortTo = "1337"

	// rules with custom data

	rc1 := r1
	rc1.CustomData1 = "deny"
	ec1 := e1

	rc2 := rc1
	ec2 := ec1
	ec2.CustomData1 = "deny"

	rc3 := rc1
	ec3 := ec2
	rc3.CustomData2 = "malware"

	rc4 := rc3
	ec4 := ec3
	ec4.CustomData2 = "malware"

	rc5 := rc4
	ec5 := ec4
	ec5.CustomData2 = "exploit"

	rc6 := rc5
	ec6 := ec5
	rc6.CustomData3 = "7000"

	rc7 := rc6
	ec7 := ec6
	ec7.CustomData3 = "7000"

	rc8 := rc7
	ec8 := ec7
	ec8.CustomData2 = "malware"

	rc9 := rc8
	rc9.CustomData2 = "!malware"
	ec9 := ec8

	// StickyDiff rules
	// TODO: add the appropriate test that test the length of stickyDiffData
	// before and after

	rs1 := r1
	rs1.StickyDiff = "PLUGIN_SID"

	s2 := &StickyDiffData{}
	s2.SDiffInt = []int{50001}
	rs2 := rs1

	s2.SDiffString = []string{"192.168.0.1", "8.8.8.200"}
	rs3 := rs1
	rs3.StickyDiff = "SRC_IP"

	rs4 := rs1
	rs4.StickyDiff = "DST_IP"

	rs5 := rs3

	rs6 := rs1
	rs7 := rs3

	s3 := &StickyDiffData{}
	s3.SDiffInt = []int{31337, 80}
	rs8 := rs1
	rs8.StickyDiff = "SRC_PORT"

	rs9 := rs1
	rs9.StickyDiff = "DST_PORT"

	s4 := &StickyDiffData{}
	s4.SDiffString = []string{"foo", "bar"}
	rs10 := rs1
	rs10.CustomData1 = "foo"
	e1.CustomData1 = "foo"
	rs10.StickyDiff = "CUSTOM_DATA1"

	rs11 := rs1
	rs11.CustomData2 = "bar"
	e1.CustomData2 = "bar"
	rs11.StickyDiff = "CUSTOM_DATA2"

	rs12 := rs1
	rs12.CustomData3 = "foo"
	e1.CustomData3 = "custom"
	rs12.StickyDiff = "CUSTOM_DATA3"

	var tbl = []ruleTests{
		{1, e1, r1, s1, true}, {2, e1, r2, s1, true}, {3, e1, r3, s1, false}, {4, e1, r4, s1, false},
		{5, e1, r5, s1, false}, {6, e1, r6, s1, false}, {7, e1, r7, s1, true}, {8, e1, r8, s1, false},
		{9, e1, r9, s1, false}, {10, e2, r10, s1, false}, {11, e1, r11, s1, false},
		{12, e1, r12, s1, false}, {13, e1, r13, s1, false}, {14, e3, r14, s1, false},
		{15, e1, r15, s1, false}, {16, e1, r16, s1, false}, {17, e1, r17, s1, false},

		{51, ec1, rc1, s1, false},
		{52, ec2, rc2, s1, true},
		{53, ec3, rc3, s1, false},
		{54, ec4, rc4, s1, true},
		{55, ec5, rc5, s1, false},
		{56, ec6, rc6, s1, false},
		{57, ec7, rc7, s1, false},
		{58, ec8, rc8, s1, true},
		{58, ec9, rc9, s1, false},

		{101, e1, rs1, s1, true}, {102, e1, rs2, s2, true}, {103, e1, rs3, s2, true},
		{104, e1, rs4, s2, true}, {105, e1, rs5, s1, true},
		{106, e1, rs6, nil, true}, {107, e1, rs7, nil, true},
		{108, e1, rs8, s3, true}, {109, e1, rs9, s3, true},
		{110, e1, rs10, s4, true}, {111, e1, rs11, s4, true}, {112, e1, rs12, s4, false},
	}

	for _, tt := range tbl {
		actual := DoesEventMatch(tt.e, tt.r, tt.s, 0)
		if actual != tt.expected {
			t.Fatalf("Rule %d actual %t != expected %t. Event: %v, Rule: %v, Sticky: %v",
				tt.n, actual, tt.expected, tt.e, tt.r, tt.s)
		}
	}
}

func TestAppendUniqCustomData(t *testing.T) {
	cd := []CustomData{}
	cd = AppendUniqCustomData(cd, "", "data1")
	cd = AppendUniqCustomData(cd, "label1", "data1")
	cd = AppendUniqCustomData(cd, "label1", "data1")
	cd = AppendUniqCustomData(cd, "label2", "data2")
	if len(cd) != 2 {
		t.Fatal("customData length expected to be 2")
	}
	if cd[0].Label != "label1" || cd[0].Content != "data1" {
		t.Fatal("customData expected to contain label1 = data1")
	}
	if cd[1].Label != "label2" || cd[1].Content != "data2" {
		t.Fatal("customData expected to contain label2 = data2")
	}
}
