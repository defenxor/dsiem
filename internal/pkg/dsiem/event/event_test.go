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

package event

import (
	"path"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/sebdah/goldie"
)

func TestValid(t *testing.T) {
	var e NormalizedEvent
	if e.Valid() {
		t.Errorf("Event is valid %v", e)
	}
	e.EventID = "1001"
	e.Timestamp = "2018-10-08T07:16:50Z"
	e.Sensor = "sensor1"
	e.SrcIP = "10.0.0.1"
	e.DstIP = "8.8.8.8"
	ts := struct{ Timestamp string }{Timestamp: e.Timestamp}
	b, _ := e.ToBytes()
	if e.Valid() {
		t.Errorf("Event is valid %v", e)
	}

	e.Product = "IDS"
	e.Category = "Malware"
	e.PluginID = 1001
	e.PluginSID = 50001
	if !e.Valid() {
		t.Errorf("Event is valid %v", e)
	}

	goldie.FixtureDir = "testdata"
	goldie.AssertWithTemplate(t, "event", ts, b)
	if !e.Valid() {
		t.Errorf("Event is not valid %v", e)
	}
}
func TestFromToBytes(t *testing.T) {
	e := NormalizedEvent{
		EventID:   "1001",
		Timestamp: "2018-10-08T07:16:50Z",
		Sensor:    "sensor1",
		SrcIP:     "10.0.0.1",
		DstIP:     "8.8.8.8",
		Product:   "IDS",
		Category:  "Malware",
		PluginID:  1001,
		PluginSID: 50001,
	}

	b, err := e.ToBytes()
	if err != nil {
		t.Error(err)
	}
	if err := e.FromBytes(b); err != nil {
		t.Error(err)
	}
}

func TestInHomeNet(t *testing.T) {
	log.Setup(true)
	e := NormalizedEvent{
		EventID:   "1001",
		Timestamp: "2018-10-08T07:16:50Z",
		Sensor:    "sensor1",
		SrcIP:     "10.0.0.1",
		DstIP:     "8.8.8.8",
	}

	err := asset.Init(path.Join("testdata", "configs"))
	if err != nil {
		t.Fatal(err)
	}
	if !e.SrcIPInHomeNet() {
		t.Errorf("SrcIP not in Homenet")
	}
	if e.DstIPInHomeNet() {
		t.Errorf("DstIP in Homenet")
	}
}
