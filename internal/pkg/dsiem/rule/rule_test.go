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
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

func TestIPinCIDR(t *testing.T) {
	type ipTest struct {
		ip       string
		cidr     string
		expected bool
	}

	log.Setup(false)

	var tbl = []ipTest{
		{"192.168.0", "192.168.0.0/16", false},
		{"192.168.0.1", "192.168.0/16", false},
		{"192.168.0.1", "192.168.0.0/16", true},
	}

	for _, tt := range tbl {
		actual := isIPinCIDR(tt.ip, tt.cidr)
		if actual != tt.expected {
			t.Errorf("IP %s in %s result is %v. Expected %v.", tt.ip, tt.cidr, actual, tt.expected)
		}
	}

}

func TestRule(t *testing.T) {

	d, err := test.DirEnv()
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

	// StickyDiff rules

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

	var tbl = []ruleTests{
		{1, e1, r1, s1, true}, {2, e1, r2, s1, true}, {3, e1, r3, s1, false}, {4, e1, r4, s1, false},
		{5, e1, r5, s1, false}, {6, e1, r6, s1, false}, {7, e1, r7, s1, false}, {8, e1, r8, s1, false},
		{9, e1, r9, s1, false}, {10, e2, r10, s1, false}, {11, e1, r11, s1, false},
		{12, e1, r12, s1, false}, {13, e1, r13, s1, false}, {14, e3, r14, s1, false},
		{15, e1, r15, s1, false}, {16, e1, r16, s1, false}, {17, e1, r17, s1, false},

		{101, e1, rs1, s1, true}, {102, e1, rs2, s2, false}, {103, e1, rs3, s2, false},
		{104, e1, rs4, s2, false}, {105, e1, rs5, s1, true},
		{106, e1, rs6, nil, true}, {107, e1, rs7, nil, true},
		{108, e1, rs8, s3, false}, {109, e1, rs9, s3, false},
	}

	for _, tt := range tbl {
		actual := DoesEventMatch(tt.e, tt.r, tt.s, 0)
		if actual != tt.expected {
			t.Fatalf("Rule %d actual != expected", tt.n)
		}
	}
}
