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

	allBacklogsMu.Lock()
	allBacklogs = []backlogs{}
	allBacklogsMu.Unlock()

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
	e.PluginID = dirs.Dirs[0].Rules[0].PluginID
	e.PluginSID = 2100384
	e.CustomData1 = "test-1"
	e.CustomData2 = "test-2"
	e.CustomData3 = "test-3"

	var blogs backlogs
	ch := make(chan event.NormalizedEvent)
	ch2 := make(chan event.NormalizedEvent)

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

	go allBacklogs[0].manager(dirs.Dirs[0], ch, 0)
	go allBacklogs[0].manager(dirs.Dirs[1], ch2, 0)

	holdSecDuration := 4
	if err = InitBackLogManager(tmpLog, bpChOutput, holdSecDuration); err != nil {
		t.Fatal(err)
	}

	// start of event-based testing

	// will fail to create new backlog due to wrong date
	fmt.Print("failed event ..")
	e.RcvdTime = time.Now().UnixNano()
	e.Timestamp = ""
	e.ConnID = 1
	verifyEventOutput(t, e, ch, "Fail to create new backlog")

	// will fail to create new backlog due to wrong SRC_IP
	fmt.Print("event doesn't match rule SRC_IP (HOME_NET) ..")
	e.Timestamp = time.Now().Add(time.Second * -300).UTC().Format(time.RFC3339)
	e.SrcIP = "8.8.8.8"
	verifyFuncOutput(t, func() {
		verifyEventOutput(t, e, ch, "")
	}, "Creating new backlog", false)

	fmt.Print("first event ..")
	e.SrcIP = "10.0.0.1"
	verifyEventOutput(t, e, ch, "stage increased")

	fmt.Print("second event ..")
	e.ConnID = 2
	e.EventID = "2"
	verifyEventOutput(t, e, ch, "backlog updating")

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
	verifyFuncOutput(t, func() {
		verifyEventOutput(t, e, ch, "")
	}, "Creating new backlog", false)

	// will not match rule nor existing backlogs
	/// no text will be shown as it is rejected by QuickCheckTaxonomyRule
	fmt.Print("8th event (for the second manager)..")
	e.PluginSID = 0
	e.Product = "Non existant product"
	e.Category = "Random category"
	e.SubCategory = "Random subcat"
	e.ConnID = 8
	e.EventID = "8"
	verifyFuncOutput(t, func() {
		verifyEventOutput(t, e, ch2, "")
	}, "Creating new backlog", false)

	// should create a new backlog on the 2nd directive
	fmt.Print("9th event ..")
	e.ConnID = 9
	e.Product = "Firewall"
	e.Category = "Packet processing"
	e.SubCategory = "Drop"
	e.EventID = "9"
	verifyEventOutput(t, e, ch2, "stage increased")

	// first event for the 2nd stage
	fmt.Print("10th event ..")
	e.ConnID = 10
	e.EventID = "10"
	verifyEventOutput(t, e, ch2, "backlog updating Elasticsearch")

	// shouldn't pass sticky diff test
	fmt.Print("11th event...")
	e.ConnID = 11
	e.EventID = "11"
	verifyEventOutput(t, e, ch2, "stickydiff field")

	sum, act, ttl := CountBackLogs()
	if sum != 3 || act != 1 || ttl != 1 {
		t.Fatalf("sum|act|ttl is incorrect. Found: %d %d %d", sum, act, ttl)
	}

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

func TestBacklogManagerCustomData(t *testing.T) {
	waitTime := 100 * time.Millisecond
	fmt.Println("Starting TestBackLogMgr.")
	allBacklogsMu.Lock()
	allBacklogs = make([]backlogs, 0)
	allBacklogsMu.Unlock()
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

	dirs, _, err := LoadDirectivesFromFile(path.Join(fDir, "directive5"), directiveFileGlob, false)
	if err != nil {
		t.Fatal(err)
	}

	if len(dirs.Dirs) != 1 {
		t.Fatalf("expected only 1 directive to be loaded, but got %d", len(dirs.Dirs))
	}

	blogs := &backlogs{
		DRWMutex: drwmutex.New(),
		id:       1,
		bpCh:     make(chan bool),
		bl:       make(map[string]*backLog),
	}

	if err = InitBackLogManager(tmpLog, nil, 4); err != nil {
		t.Fatal(err)
	}

	testDirective := dirs.Dirs[0]
	testEvent := event.NormalizedEvent{
		EventID:      "1",
		ConnID:       1,
		Sensor:       "test-sensor",
		SrcIP:        "1.1.1.1",
		DstIP:        "2.2.2.2",
		Title:        "Test Event",
		Protocol:     "TEST",
		PluginID:     1337,
		PluginSID:    1,
		CustomLabel1: "fsoo",
		CustomData1:  "bar",
		Timestamp:    time.Now().Add(time.Second * -300).UTC().Format(time.RFC3339),
	}

	input := make(chan event.NormalizedEvent)
	go blogs.manager(testDirective, input, 0)

	// first event
	input <- testEvent
	time.Sleep(waitTime)

	var testBl *backLog
	blogs.Lock()
	if len(blogs.bl) != 1 {
		t.Fatalf("expected 1 backlog, but got %d", len(blogs.bl))
		blogs.Unlock()
	}

	for _, v := range blogs.bl {
		testBl = v
		break
	}
	blogs.Unlock()

	testBl.Lock()
	if testBl.CurrentStage != 2 {
		t.Errorf("expected current stage to be 2 but got %d", testBl.CurrentStage)
	}
	testBl.Unlock()
	// 2nd event
	testEvent.ConnID = 2
	testEvent.EventID = "2"
	testEvent.CustomLabel2 = "foo2"
	testEvent.CustomData2 = "bar2"

	input <- testEvent
	time.Sleep(waitTime)

	testBl.Lock()
	if testBl.CurrentStage != 2 {
		t.Errorf("expected current stage to be 2 but got %d", testBl.CurrentStage)
	}
	testBl.Unlock()

	// 3rd event
	testEvent.ConnID = 3
	testEvent.EventID = "3"

	input <- testEvent
	time.Sleep(waitTime)

	testBl.Lock()
	if testBl.CurrentStage != 3 {
		t.Errorf("expected current stage to be 3 but got %d", testBl.CurrentStage)
	}
	testBl.Unlock()

	// 4th event
	testEvent.ConnID = 4
	testEvent.EventID = "4"

	testEvent.CustomData3 = "bar3"

	input <- testEvent
	time.Sleep(waitTime)

	testBl.Lock()
	if testBl.CurrentStage != 3 {
		t.Errorf("expected current stage to be 3 but got %d", testBl.CurrentStage)
	}
	testBl.Unlock()

	// 5th event -> different custom data, creates a new backlog
	testEvent2 := testEvent
	testEvent2.ConnID = 5
	testEvent2.EventID = "5"
	testEvent2.CustomLabel1 = "aaa"
	testEvent2.CustomData1 = "bbb"

	input <- testEvent2
	time.Sleep(waitTime)

	// expected 2 backlogs now
	var testBl2 *backLog
	blogs.Lock()
	if len(blogs.bl) != 2 {
		blogs.Unlock()
		t.Fatalf("expected 2 backlog, but got %d", len(blogs.bl))

	}

	for _, v := range blogs.bl {
		if testBl == v {
			continue
		}

		testBl2 = v
		break
	}
	blogs.Unlock()

	testBl2.Lock()
	if testBl2.CurrentStage != 2 {
		t.Errorf("expected current stage to be 2 but got %d", testBl2.CurrentStage)
	}
	testBl2.Unlock()

	// 6th event
	testEvent2.ConnID = 6
	testEvent2.EventID = "6"

	input <- testEvent2
	time.Sleep(waitTime)

	testBl2.Lock()
	if testBl2.CurrentStage != 2 {
		t.Errorf("expected current stage to be 2 but got %d", testBl2.CurrentStage)
	}
	testBl2.Unlock()

	// 7th event -> stage increased for second backlog
	testEvent2.ConnID = 7
	testEvent2.EventID = "7"

	input <- testEvent2
	time.Sleep(waitTime)

	testBl2.Lock()
	if testBl2.CurrentStage != 3 {
		t.Errorf("expected current stage to be 3 but got %d", testBl2.CurrentStage)
	}
	testBl2.Unlock()

	// 8th event -> this event has no custom data, therefore new backlog should created
	testEvent3 := event.NormalizedEvent{
		EventID:      "8",
		ConnID:       8,
		Sensor:       "test-sensor",
		SrcIP:        "1.1.1.1",
		DstIP:        "2.2.2.2",
		Title:        "Test Event",
		Protocol:     "TEST",
		PluginID:     1337,
		PluginSID:    1,
		CustomLabel1: "",
		CustomData1:  "",
		Timestamp:    time.Now().Add(time.Second * -300).UTC().Format(time.RFC3339),
	}

	input <- testEvent3
	time.Sleep(waitTime)

	var testBl3 *backLog
	blogs.Lock()
	if len(blogs.bl) != 3 {
		blogs.Unlock()
		t.Fatalf("expected 3 backlog, but got %d", len(blogs.bl))
	}

	for _, blog := range blogs.bl {
		if testBl == blog || testBl2 == blog {
			continue
		}

		testBl3 = blog
		break
	}
	blogs.Unlock()

	if testBl3 == nil {
		t.Fatal("expected third backlog to exist")
	}

	testBl3.Lock()
	if testBl3.CurrentStage != 2 {
		t.Errorf("expected current stage of third backlog to be 2 but got %d", testBl3.CurrentStage)
	}

	if testBl3.LastEvent.EventID != testEvent3.EventID {
		t.Errorf("expected last event id for third backlog to be %s but got %s", testEvent3.EventID, testBl3.LastEvent.EventID)
	}
	testBl3.Unlock()

	testEvent3.EventID = "9"
	testEvent3.ConnID = 9

	input <- testEvent3
	time.Sleep(waitTime)

	testBl3.Lock()
	if testBl3.CurrentStage != 2 {
		t.Errorf("expected current stage of third backlog to be 2 but got %d", testBl3.CurrentStage)
	}

	if testBl3.LastEvent.EventID != testEvent3.EventID {
		t.Errorf("expected last event id for third backlog to be %s but got %s", testEvent3.EventID, testBl3.LastEvent.EventID)
	}
	testBl3.Unlock()

	// 10th event, identical to 5th event, should increase second backlog stage instead of the third one
	testEvent5 := testEvent2
	testEvent5.EventID = "10"
	testEvent5.ConnID = 10

	input <- testEvent5
	time.Sleep(waitTime)

	testBl2.Lock()
	if testBl2.CurrentStage != 3 {
		t.Errorf("expected current stage of third backlog to be 3 but got %d", testBl2.CurrentStage)
	}

	if testBl2.LastEvent.EventID != testEvent5.EventID {
		t.Errorf("expected last event id for third backlog to be %s but got %s", testEvent5.EventID, testBl2.LastEvent.EventID)
	}
	testBl2.Unlock()

}
