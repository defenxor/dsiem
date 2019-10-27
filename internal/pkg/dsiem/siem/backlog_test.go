// Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.
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

package siem

import (
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/alarm"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/jonhoo/drwmutex"
)

var (
	assetInitialized bool
	alarmInitialized bool
)

func initAlarm(t *testing.T) {
	if alarmInitialized {
		return
	}
	viper.Set("medRiskMin", 3)
	viper.Set("medRiskMax", 6)
	viper.Set("tags", []string{"Identified Threat", "Valid Threat"})
	viper.Set("status", []string{"Open", "Closed"})
	tmpLog := path.Join(os.TempDir(), "siem_alarms.log")
	err := alarm.Init(tmpLog, false)
	if err != nil {
		t.Fatal(err)
	}
	alarmInitialized = true
}

func initAsset(t *testing.T) {
	// needed by rule checkers
	if assetInitialized {
		return
	}
	err := asset.Init(path.Join(testDir, "internal", "pkg", "dsiem", "asset", "fixtures", "asset1"))
	if err != nil {
		t.Fatal(err)
	}
	assetInitialized = true
}
func TestBackLog(t *testing.T) {

	fmt.Println("Starting TestBackLog.")

	setTestDir(t)
	t.Logf("Using base dir %s", testDir)

	if !log.TestMode {
		t.Logf("Enabling log test mode")
		log.EnableTestingMode()
	}

	initAlarm(t)
	initAsset(t)
	tmpLog := path.Join(os.TempDir(), "siem_alarm_events.log")
	fWriter.Init(tmpLog, 10)

	fDir := path.Join(testDir, "internal", "pkg", "dsiem", "siem", "fixtures")

	// use directive that expires fast and has only 3 stages
	dirs, _, err := LoadDirectivesFromFile(path.Join(fDir, "directive4"), directiveFileGlob, false)
	if err != nil {
		t.Fatal(err)
	}

	e := event.NormalizedEvent{}
	e.EventID = "1"
	e.Sensor = "sensor1"
	e.SrcIP = "10.0.0.1"
	e.DstIP = "192.168.0.1"
	e.Title = "ICMP Ping"
	e.Protocol = "ICMP"
	e.ConnID = 1
	dctives := dirs.Dirs[0]
	e.PluginID = dctives.Rules[0].PluginID
	e.PluginSID = dctives.Rules[0].PluginSID[0]

	e.Timestamp = time.Now().UTC().Format(time.RFC3339)

	apm.Enable(true)

	viper.Set("medRiskMin", 3)
	viper.Set("medRiskMax", 6)
	viper.Set("tags", []string{"Identified Threat", "Valid Threat"})
	viper.Set("status", []string{"Open", "Closed"})
	viper.Set("maxDelay", 100)

	b, err := createNewBackLog(dirs.Dirs[0], e)
	if err != nil {
		t.Fatal(err)
	}
	bLogs := backlogs{}
	bLogs.bpCh = make(chan bool)
	bLogs.DRWMutex = drwmutex.New()
	bLogs.bl = make(map[string]*backLog)
	bLogs.bl[b.ID] = b
	// bLogs.bl[b.ID].DRWMutex = drwmutex.New()
	b.bLogs = &bLogs

	go func() {
		for {
			<-bLogs.bpCh
		}
	}()
	go func() {
		for {
			<-b.chFound
		}
	}()
	go func() {
		for {
			<-b.chDone
		}
	}()

	// will raise stage to 2nd
	fmt.Println("first event (by start)")
	fmt.Println("using backlog: ", b.Directive.Name)
	fmt.Println("all_rules_always_active flag: ", b.Directive.AllRulesAlwaysActive)

	go b.start(e, 0)

	// will also raise stage to 3rd
	fmt.Print("under pressure ..")
	e.ConnID = 2
	e.RcvdTime = time.Now().Add(-700 * time.Second).Unix()
	e.PluginSID = b.Directive.Rules[1].PluginSID[0]
	verifyEventOutput(t, e, b.chData, "backlog is under pressure")

	fmt.Print("previous rule consuming event ..")
	e.ConnID = 3
	e.PluginSID = b.Directive.Rules[0].PluginSID[0]
	verifyEventOutput(t, e, b.chData, "consumes matching event")

	fmt.Print("out of order event ..")
	e.ConnID = 4
	e.PluginSID = b.Directive.Rules[1].PluginSID[0]
	e.Timestamp = time.Now().Add(time.Second * -300).UTC().Format(time.RFC3339)
	verifyEventOutput(t, e, b.chData, "event timestamp out of order")

	fmt.Print("invalid timestamp ..")
	e.ConnID = 5
	e.Timestamp = "#"
	verifyEventOutput(t, e, b.chData, "cannot parse event timestamp")

	_ = fWriter.Init("", 0)
	fmt.Print("err in updating ES ..")
	e.ConnID = 6
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	verifyEventOutput(t, e, b.chData, "failed to update Elasticsearch!")

	e.RcvdTime = time.Now().Add(-time.Second).Unix()
	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	e.ConnID = 7
	fmt.Print("reached max stage ..")
	verifyEventOutput(t, e, b.chData, "reached max stage and occurrence")

	fmt.Print("Check expiration ..")
	// maxTime < limit
	if !b.isExpired() {
		t.Fatal("expected to not yet expire")
	}
	fmt.Println("OK")

	fmt.Print("Check deletion ..")
	verifyFuncOutput(t, func() {
		b.delete()
	}, "", true)
	fmt.Println("OK")

	verifyFuncOutput(t, func() {
		fmt.Print("waiting for backlog to be deleted..")
		time.Sleep(time.Second * 8)
	}, "backlog manager deleting backlog from map", true)
}
