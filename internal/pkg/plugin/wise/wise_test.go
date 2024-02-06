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

package wise

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"testing"

	"github.com/buaazp/fasthttprouter"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"
	"github.com/defenxor/dsiem/pkg/intel"
	"github.com/valyala/fasthttp"
)

type intelTests struct {
	ip            string
	expectedFound bool
	expectedRes   []intel.Result
}

var tblIntel = []intelTests{
	{"10.0.0.1", false, nil},
	{"not-an-ip", false, nil},
	{"10.0.0.2", true, []intel.Result{{Provider: "Dummy", Term: "10.0.0.2", Result: "Detected in DB"}}},
}

type intelSource struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	Plugin  string `json:"plugin"`
	Config  string `json:"config"`
}

type intelSources struct {
	IntelSources []intelSource `json:"intel_sources"`
}

type wisePayload struct {
	WiseLines []WiseLine
	IP        string
}
type WiseLine struct {
	Field string `json:"field"`
	Len   int    `json:"len"`
	Value string `json:"value"`
}

func mockWise(t *testing.T) {
	wp1 := wisePayload{
		IP: "8.8.8.8",
		WiseLines: []WiseLine{
			{Field: "alienvault.id", Len: 3, Value: "12"},
			{Field: "alienvault.reliability", Len: 3, Value: "9"},
			{Field: "alienvault.threat-level", Len: 2, Value: "2"},
			{Field: "alienvault.activity", Len: 9, Value: "Spamming"},
		},
	}
	wp2 := wisePayload{
		IP: "8.8.4.4",
		WiseLines: []WiseLine{
			{Field: "criticalstack.type", Len: 12, Value: "Intel::ADDR"},
			{Field: "criticalstack.source", Len: 64, Value: "from https://www.dan.me.uk/torlist/ via intel.criticalstack.com"},
		},
	}
	p1, err := json.MarshalIndent(wp1.WiseLines, " ", " ")
	if err != nil {
		t.Error(err)
	}
	p2, err := json.MarshalIndent(wp2.WiseLines, " ", " ")
	if err != nil {
		t.Error(err)
	}
	router := fasthttprouter.New()

	router.GET("/ip/:ipAddr", func(ctx *fasthttp.RequestCtx) {
		ip := ctx.UserValue("ipAddr").(string)
		var resp string
		switch ip {
		case wp1.IP:
			resp = string(p1)
		case wp2.IP:
			resp = string(p2)
		default:
			resp = "[]"
		}
		fmt.Fprint(ctx, resp+"\n")
		ctx.SetStatusCode(fasthttp.StatusOK)
	})
	_ = fasthttp.ListenAndServe("127.0.0.1:8082", router.Handler)
}

func TestWise(t *testing.T) {
	_, err := test.DirEnv(false)
	if err != nil {
		t.Fatal(err)
	}

	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	go mockWise(t)

	cfg := path.Join(d, "fixtures", "intel_wise.json")

	var it intelSources
	file, err := os.Open(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	byteValue, _ := io.ReadAll(file)
	err = json.Unmarshal(byteValue, &it)
	if err != nil {
		t.Fatal(err)
	}

	w := Wise{}
	if err = w.Initialize([]byte(it.IntelSources[0].Config)); err != nil {
		t.Fatal(err)
	}

	found, _, err := w.CheckIP(context.Background(), "10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("Expected to not find a match")
	}

	found, res, err := w.CheckIP(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
	if !found || res[0].Result != "alienvault.activity: Spamming" {
		t.Fatal("Expected to find a match")
	}
	found, res, err = w.CheckIP(context.Background(), "8.8.4.4")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(res)
	if !found || res[0].Result != "criticalstack.type: Intel::ADDR" {
		t.Fatal("Expected to find a match")
	}

	// for error in config

	w = Wise{}
	if err = w.Initialize([]byte(it.IntelSources[1].Config)); err != nil {
		t.Fatal(err)
	}
	found, _, err = w.CheckIP(context.Background(), "10.0.0.1")
	if err == nil {
		t.Fatal("expected to error due to mistake in config")
	}

	w = Wise{}
	if err = w.Initialize([]byte(it.IntelSources[2].Config)); err != nil {
		t.Fatal(err)
	}
	found, _, err = w.CheckIP(context.Background(), "10.0.0.1")
	if err == nil {
		t.Fatal("expected to error due to mistake in config")
	}

}
