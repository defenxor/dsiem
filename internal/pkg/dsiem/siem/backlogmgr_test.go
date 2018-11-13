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

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
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

func TestInitDirective(t *testing.T) {

	allBacklogs = []backlogs{}

	fmt.Println("Starting TestInitDirective.")

	setTestDir(t)

	t.Logf("Using base dir %s", testDir)
	fDir := path.Join(testDir, "internal", "pkg", "dsiem", "siem", "fixtures")
	var evtChan chan event.NormalizedEvent
	err := InitDirectives(path.Join(fDir, "directive2"), evtChan)
	if err == nil || !strings.Contains(err.Error(), "Cannot load any directive from") {
		t.Fatal(err)
	}
	err = InitDirectives(path.Join(fDir, "directive1"), evtChan)
	if err != nil {
		t.Fatal(err)
	}
}

func TestBacklogMgr(t *testing.T) {

	allBacklogs = []backlogs{}

	setTestDir(t)
	t.Logf("Using base dir %s", testDir)

	if !log.TestMode {
		t.Logf("Enabling log test mode")
		log.EnableTestingMode()
	}

	fDir := path.Join(testDir, "internal", "pkg", "dsiem", "siem", "fixtures")
	apm.Enable(true)

	err := asset.Init(path.Join(testDir, "configs"))
	if err != nil {
		t.Fatal(err)
	}

	tmpLog := "siem_alarm_events.log"
	cleanUp := func() {
		_ = os.Remove(tmpLog)
	}
	defer cleanUp()

	dirs, _, err := LoadDirectivesFromFile(path.Join(fDir, "directive1"), directiveFileGlob)

	var evtChan chan event.NormalizedEvent

	err = InitDirectives(path.Join(fDir, "directive1"), evtChan)
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

	go allBacklogs[0].manager(dctives, ch)
	if err = InitBackLogManager(tmpLog, bpChOutput, 1); err != nil {
		t.Fatal(err)
	}

	// start of event-based testing
	// will be rejected, missing rcvdTime
	fmt.Print("rejected event ..")
	e.Timestamp = "2018-10-08T07:16:50Z"
	verifyEventOutput(t, e, ch, "Cannot parse event received time, skipping event")

	fmt.Print("first event ..")
	e.RcvdTime = time.Now().Unix()
	e.Timestamp = time.Now().Add(time.Second * -300).UTC().Format(time.RFC3339)
	e.ConnID = 1
	verifyEventOutput(t, e, ch, "stage increased")

	fmt.Print("second event ..")
	e.ConnID = 2
	verifyEventOutput(t, e, ch, "backlog updating")

	fmt.Print("3rd event ..")
	e.ConnID = 3
	verifyEventOutput(t, e, ch, "backlog updating")

	fmt.Print("4th event ..")
	e.ConnID = 4
	verifyEventOutput(t, e, ch, "stage increased")

	// this should create new backlog
	fmt.Print("5th event ..")
	e.ConnID = 5
	verifyEventOutput(t, e, ch, "Incoming event with idx: 0")

	if len(allBacklogs[0].bl) != 2 {
		t.Fatal("allBacklogs.bl is expected to have a length of 2")
	}

	// will not match rule
	fmt.Print("6th event ..")
	e.PluginSID = 31337
	e.ConnID = 6
	verifyEventOutput(t, e, ch, "backlog doeseventmatch false")

	var blID string
	for k := range allBacklogs[0].bl {
		blID = k
		break
	}
	fmt.Print("Deleting the 1st backlog", blID, " ..")
	verifyFuncOutput(t, func() {
		blogs.delete(allBacklogs[0].bl[blID])
		time.Sleep(time.Second * 2)
	}, "backlog manager setting status to deleted")

	fmt.Print("Deleting it again ..")
	verifyFuncOutput(t, func() {
		blogs.delete(allBacklogs[0].bl[blID])
		time.Sleep(time.Second * 1)
	}, "backlog is already in the process of being deleted")

	fmt.Print("Sending overload signal=true to blogs bpCh ..")
	verifyFuncOutput(t, func() {
		allBacklogs[0].bpCh <- true
		time.Sleep(time.Second)
	}, "simulated server received backpressure data: true")
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

func verifyFuncOutput(t *testing.T, f func(), expected string) {
	out := log.CaptureZapOutput(f)
	t.Log("out: ", out)
	if !strings.Contains(out, expected) {
		t.Fatalf("Cannot find '%s' in output: %s", expected, out)
	} else {
		fmt.Println("OK")
	}
}
