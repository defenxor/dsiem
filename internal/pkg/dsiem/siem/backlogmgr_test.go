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

package siem

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"

	"github.com/jonhoo/drwmutex"
)

var testDir string

func setTestDir(t *testing.T) {
	if testDir == "" {
		d, err := test.DirEnv(true)
		if err != nil {
			t.Fatal(err)
		}
		testDir = d
	}
}

func TestBacklogMgr(t *testing.T) {

	fmt.Println("Starting TestBackLogMgr.")

	allBacklogs = []backlogs{}

	setTestDir(t)
	t.Logf("Using base dir %s", testDir)

	if !log.TestMode {
		t.Logf("Enabling log test mode")
		log.EnableTestingMode()
	}

	fDir := path.Join(testDir, "internal", "pkg", "dsiem", "siem", "fixtures")
	apm.Enable(true)

	tmpLog := path.Join(os.TempDir(), "siem_alarm_events.log")
	fWriter.Init(tmpLog, 10)

	cleanUp := func() {
		_ = os.Remove(tmpLog)
	}
	defer cleanUp()

	initAlarm(t)
	initAsset(t)

	dirs, _, err := LoadDirectivesFromFile(path.Join(fDir, "directive3"), directiveFileGlob, false)
	if err == nil {
		t.Error("Badly formatted file, expected err to be non nil")
	}
	dirs, _, err = LoadDirectivesFromFile(path.Join(fDir, "directive3"), `\\?\C:\*`, false)
	if err == nil {
		t.Error("Bad glob supplied, expected err to be non nil")
	}

	dirs, _, err = LoadDirectivesFromFile(path.Join(fDir, "directive1"), directiveFileGlob, false)
	if err != nil {
		t.Fatal(err)
	}

	e := event.NormalizedEvent{}
	e.EventID = "1"
	e.Sensor = "sensor1"
	e.SrcIP = "10.0.0.1"
	e.DstIP = "8.8.8.8"
	e.Title = "ICMP Ping"
	e.Protocol = "ICMP"
	e.ConnID = 1
	dctives := dirs.Dirs[0]
	e.PluginID = dctives.Rules[0].PluginID
	e.PluginSID = 2100384

	var blogs backlogs
	ch := make(chan event.NormalizedEvent)
	blogs.DRWMutex = drwmutex.New()
	blogs.id = 1
	blogs.bpCh = make(chan bool)
	blogs.bl = make(map[string]*backLog) // have to do it here before the append

	allBacklogs = append(allBacklogs, blogs)

	bpChOutput := make(chan bool)
	go func() {
		for {
			bpFlag := <-bpChOutput
			log.Info(log.M{Msg: "simulated server received backpressure data: " + strconv.FormatBool(bpFlag)})
		}
	}()

	go allBacklogs[0].manager(dctives, ch, 0)

	holdSecDuration := 4
	if err = InitBackLogManager(tmpLog, bpChOutput, holdSecDuration); err != nil {
		t.Fatal(err)
	}

	// start of event-based testing
	// will be rejected, missing rcvdTime
	fmt.Print("rejected event ..")
	e.Timestamp = "2018-10-08T07:16:50Z"
	verifyEventOutput(t, e, ch, "Cannot parse event received time, skipping event")

	// will fail to create new backlog due to wrong date
	fmt.Print("failed event ..")
	e.RcvdTime = time.Now().Unix()
	e.Timestamp = ""
	e.ConnID = 1
	verifyEventOutput(t, e, ch, "Fail to create new backlog")

	fmt.Print("first event ..")
	e.Timestamp = time.Now().Add(time.Second * -300).UTC().Format(time.RFC3339)
	verifyEventOutput(t, e, ch, "stage increased")

	fmt.Print("second event ..")
	e.ConnID = 2
	e.EventID = "2"
	verifyEventOutput(t, e, ch, "backlog updating")

	/*
		fmt.Print("3rd event, will also fail updating ES ..")
		e.ConnID = 3
		e.EventID = "3"
		// bLogFile = ""
		verifyEventOutput(t, e, ch, "failed to update Elasticsearch")
		// bLogFile = tmpLog
	*/
	fmt.Print("third event ..")
	e.ConnID = 3
	e.EventID = "3"
	verifyEventOutput(t, e, ch, "backlog updating")

	fmt.Print("4th event ..")
	e.ConnID = 4
	e.EventID = "4"
	verifyEventOutput(t, e, ch, "stage increased")

	// this should create new backlog
	fmt.Print("5th event ..")
	e.ConnID = 5
	e.EventID = "5"
	e.SrcIP = "192.168.0.1"
	e.DstIP = "192.168.0.3"
	verifyEventOutput(t, e, ch, "Creating new backlog")

	if len(allBacklogs[0].bl) != 2 {
		t.Fatal("allBacklogs.bl is expected to have a length of 2")
	} else {
		t.Log("backlogs total event = 2 as expected.")
	}

	// should not trigger new backlog due to duplicate event ID
	fmt.Print("6th event ..")
	e.ConnID = 6
	e.SrcIP = "192.168.0.100"
	e.DstIP = "192.168.0.1"
	verifyEventOutput(t, e, ch, "skipping backlog creation for event")

	// will not match rule nor existing backlogs
	/// no text will be shown as it is rejected by QuickCheckPluginRule
	fmt.Print("7th event ..")
	e.PluginSID = 31337
	e.ConnID = 7
	e.EventID = "7"
	verifyEventOutput(t, e, ch, "")

	var blID string
	for k := range allBacklogs[0].bl {
		blID = k
		break
	}

	fmt.Print("Deleting the 1st backlog: ", blID, " ..")
	verifyFuncOutput(t, func() {
		blogs.delete(allBacklogs[0].bl[blID])
		time.Sleep(time.Second * 2)
	}, "backlog manager setting status to deleted", true)

	fmt.Print("Deleting it again ..")
	verifyFuncOutput(t, func() {
		blogs.delete(allBacklogs[0].bl[blID])
		time.Sleep(time.Millisecond * 1)
	}, "backlog is already in the process of being deleted", true)

	fmt.Print("Sending overload signal=true to blogs bpCh ..")
	verifyFuncOutput(t, func() {
		allBacklogs[0].bpCh <- true
		time.Sleep(time.Millisecond)
	}, "simulated server received backpressure data: true", true)

	fmt.Print("Sending another signal=true to blogs bpCh ..")
	verifyFuncOutput(t, func() {
		allBacklogs[0].bpCh <- true
		time.Sleep(time.Millisecond)
	}, "simulated server received backpressure data: true", false)

	// this one expect the timer from holdSecDuration already reset the signal to false
	fmt.Print("Sending another signal=true to blogs bpCh, expecting timer to set prevstate to false ..")
	verifyFuncOutput(t, func() {
		time.Sleep(time.Second * 4)
		allBacklogs[0].bpCh <- true
		time.Sleep(time.Millisecond)
	}, "simulated server received backpressure data: true", true)
}

func verifyEventOutput(t *testing.T, e event.NormalizedEvent, ch chan event.NormalizedEvent, expected string) {
	out := log.CaptureZapOutput(func() {
		ch <- e
		time.Sleep(time.Second * 1)
	})
	t.Log("out: ", out)
	if !strings.Contains(out, expected) {
		t.Fatalf("Cannot find '%s' in output: %s", expected, out)
	} else {
		fmt.Println("OK")
	}
}

func verifyFuncOutput(t *testing.T, f func(), expected string, expectMatch bool) {
	out := log.CaptureZapOutput(f)
	t.Log("out: ", out)
	if !strings.Contains(out, expected) == expectMatch {
		t.Fatalf("Cannot find '%s' in output: %s", expected, out)
	} else {
		fmt.Println("OK")
	}
}
