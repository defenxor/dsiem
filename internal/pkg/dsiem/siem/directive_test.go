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
	"path"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
)

func TestInitDirective(t *testing.T) {

	allBacklogsMu.Lock()
	allBacklogs = []backlogs{}
	allBacklogsMu.Unlock()

	fmt.Println("Starting TestInitDirective.")

	setTestDir(t)

	t.Logf("Using base dir %s", testDir)
	fDir := path.Join(testDir, "internal", "pkg", "dsiem", "siem", "fixtures")
	evtChan := make(chan event.NormalizedEvent)
	err := InitDirectives(path.Join(fDir, "directive2"), evtChan, 0, 1000, 0)

	if err == nil {
		t.Fatal("expected error")
	}

	if err != ErrNoDirectiveLoaded {
		t.Fatalf("expected error to be '%s' but got '%s'", ErrNoDirectiveLoaded.Error(), err.Error())
	}

	err = InitDirectives(path.Join(fDir, "directive1"), evtChan, 0, 1000, 0)
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
	e.PluginSID = 2100384
	e.PluginID = 1001
	e.RcvdTime = time.Now().UnixNano()

	err = asset.Init(path.Join(testDir, "internal", "pkg", "dsiem", "asset", "fixtures", "asset1"))
	if err != nil {
		t.Fatal(err)
	}
	evtChan <- e
	if !isWhitelisted("192.168.0.2") {
		t.Fatal("expected 192.168.0.2 to be whitelisted")
	}
	if isWhitelisted("foo") {
		t.Fatal("expected foo not to be whitelisted")
	}
}
