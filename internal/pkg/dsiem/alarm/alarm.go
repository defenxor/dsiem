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

package alarm

import (
	"errors"
	"sync"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	xc "github.com/defenxor/dsiem/internal/pkg/dsiem/xcorrelator"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"

	"github.com/defenxor/dsiem/pkg/intel"
	"github.com/defenxor/dsiem/pkg/vuln"

	"github.com/spf13/viper"
)

var aLogFile string
var mediumRiskLowerBound int
var mediumRiskUpperBound int
var defaultTag string
var defaultStatus string
var alarmRemovalChannel chan string
var intelCheckPrivateIP bool
var fWriter fs.FileWriter

const maxFileQueueLength = 1000

type alarm struct {
	sync.RWMutex `json:"-"`
	IntelMu      sync.Mutex `json:"-"`
	VulnMu       sync.Mutex `json:"-"`
	//drwmutex.DRWMutex `json:"-"`
	ID              string               `json:"alarm_id"`
	Title           string               `json:"title"`
	Status          string               `json:"status"`
	Kingdom         string               `json:"kingdom"`
	Category        string               `json:"category"`
	CreatedTime     int64                `json:"created_time"`
	UpdateTime      int64                `json:"update_time"`
	Risk            int                  `json:"risk"`
	RiskClass       string               `json:"risk_class"`
	Tag             string               `json:"tag"`
	SrcIPs          []string             `json:"src_ips"`
	DstIPs          []string             `json:"dst_ips"`
	ThreatIntels    []intel.Result       `json:"intel_hits,omitempty"`
	Vulnerabilities []vuln.Result        `json:"vulnerabilities,omitempty"`
	CustomData      []rule.CustomData    `json:"custom_data,omitempty"`
	Networks        []string             `json:"networks"`
	Rules           []rule.DirectiveRule `json:"rules"`
}

// alarms group all the alarm in a single collection
var alarms struct {
	sync.RWMutex
	al map[string]*alarm
}

// Init initialize alarm, storing result into logFile
func Init(logFile string, intelPrivIPFlag bool) error {
	if err := fWriter.Init(logFile, maxFileQueueLength); err != nil {
		return err
	}

	intelCheckPrivateIP = intelPrivIPFlag

	alarms.Lock()
	alarms.al = make(map[string]*alarm)
	alarmRemovalChannel = make(chan string)
	alarms.Unlock()

	mediumRiskLowerBound = viper.GetInt("medRiskMin")
	mediumRiskUpperBound = viper.GetInt("medRiskMax")
	defaultTag = viper.GetStringSlice("tags")[0]
	defaultStatus = viper.GetStringSlice("status")[0]

	if mediumRiskLowerBound < 2 || mediumRiskUpperBound > 9 ||
		mediumRiskLowerBound == mediumRiskUpperBound {
		return errors.New("Wrong value for medRiskMin or medRiskMax: " +
			"medRiskMax should be between 3-10, medRiskMin should be between 2-9, and medRiskMin should be < mdRiskMax")
	}

	aLogFile = logFile
	removalListener()

	return nil
}

// Upsert creates or update alarms
// backlog struct is decomposed here to avoid circular dependency
func Upsert(id, name, kingdom, category string,
	srcIPs, dstIPs []string, customData []rule.CustomData, lastSrcPort, lastDstPort, risk int,
	statusTime int64, rules []rule.DirectiveRule, connID uint64, checkIntelVuln bool,
	tx *apm.Transaction) {

	if apm.Enabled() && tx != nil {
		defer tx.Recover()
	}

	a := findOrCreateAlarm(id)
	a.Lock()

	a.Title = name
	if a.Status == "" {
		a.Status = defaultStatus
	}
	if a.Tag == "" {
		a.Tag = defaultTag
	}

	a.Kingdom = kingdom
	a.Category = category
	if a.CreatedTime == 0 {
		a.CreatedTime = statusTime
	}
	a.UpdateTime = statusTime
	a.Risk = risk
	switch {
	case a.Risk < mediumRiskLowerBound:
		a.RiskClass = "Low"
	case a.Risk >= mediumRiskLowerBound && a.Risk <= mediumRiskUpperBound:
		a.RiskClass = "Medium"
	case a.Risk > mediumRiskUpperBound:
		a.RiskClass = "High"
	}
	a.SrcIPs = srcIPs
	a.DstIPs = dstIPs
	a.CustomData = customData

	a.Unlock()
	if xc.IntelEnabled && checkIntelVuln {
		// do intel check in the background
		asyncIntelCheck(a, connID, intelCheckPrivateIP, tx)
	}

	if xc.VulnEnabled && checkIntelVuln {
		// do vuln check in the background
		asyncVulnCheck(a, lastSrcPort, lastDstPort, connID, tx)
	}

	a.Lock()
	for i := range a.SrcIPs {
		a.Networks = append(a.Networks, asset.GetAssetNetworks(a.SrcIPs[i])...)
	}
	for i := range a.DstIPs {
		a.Networks = append(a.Networks, asset.GetAssetNetworks(a.DstIPs[i])...)
	}
	a.Networks = str.RemoveDuplicatesUnordered(a.Networks)
	a.Rules = []rule.DirectiveRule{}
	for _, v := range rules {
		r := v
		r.Events = []string{} // so it will be omitted during json marshaling
		// r.StickyDiff = ""
		a.Rules = append(a.Rules, r)
	}
	a.Unlock()
	updateElasticsearch(a, "Upsert", connID, tx)
}

// Count set and return the count of alarms
func Count() (count int) {
	alarms.RLock()
	count = len(alarms.al)
	alarms.RUnlock()
	return
}

// RemovalChannel returns the channel used to send alarm ID to delete
func RemovalChannel() chan string {
	return alarmRemovalChannel
}
