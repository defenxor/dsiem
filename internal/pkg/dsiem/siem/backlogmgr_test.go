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
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"

	"github.com/jonhoo/drwmutex"
)

var testDir string

func setTestDir(t *testing.T) {
	if testDir == "" {
		d, err := test.DirEnv()
		if err != nil {
			t.Fatal(err)
		}
		testDir = d
	}
}

func TestBacklogMgr(t *testing.T) {

	setTestDir(t)

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
			fmt.Println("simulated server receive bp info: ", bpFlag)
		}
	}()
	err = InitBackLogManager(tmpLog, bpChOutput, 6)
	if err != nil {
		t.Fatal(err)
	}

	go allBacklogs[0].manager(dctives, ch)

	// will be rejected, missing rcvdTime
	e.Timestamp = "2018-10-08T07:16:50Z"
	ch <- e

	// should expire right away
	fmt.Println("expired event")
	e.RcvdTime = time.Now().Unix()
	ch <- e

	fmt.Println("first event")
	e.Timestamp = time.Now().Add(time.Second * -300).UTC().Format(time.RFC3339)
	ch <- e
	fmt.Println("2nd event")
	e.ConnID = 2
	ch <- e
	fmt.Println("3rd event")
	e.ConnID = 3
	ch <- e
	fmt.Println("4th event")
	e.ConnID = 4
	ch <- e
	// will not match rule
	fmt.Println("5th event")
	e.PluginSID = 31337
	e.ConnID = 5
	ch <- e

	var blID string
	for k := range blogs.bl {
		blID = k
		break
	}
	fmt.Println("sending bp true")
	blogs.bpCh <- true
	//fmt.Println("Sending second")
	//bpCh <- true
	blogs.delete(blogs.bl[blID])
	blogs.delete(blogs.bl[blID])
	time.Sleep(1 * time.Second)
	fmt.Println("sending bp true")
	blogs.bpCh <- true
	time.Sleep(1 * time.Second)
	fmt.Println("sending bp true")
	blogs.bpCh <- true
	time.Sleep(1 * time.Second)
	fmt.Println("sending bp true")
	blogs.bpCh <- true
	time.Sleep(1 * time.Second)
	fmt.Println("sending bp true")
	blogs.bpCh <- true
	time.Sleep(3 * time.Second)
	fmt.Println("sending bp true")
	blogs.bpCh <- true
	e.PluginSID = 2100384
	fmt.Println("6th event")
	ch <- e
	time.Sleep(time.Second * 15)
}
