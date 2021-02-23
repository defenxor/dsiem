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

package xcorrelator

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/ip"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"
	"github.com/defenxor/dsiem/pkg/intel"
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
	{"10.0.0.2", true, []intel.Result{{Provider: "Dummy", Term: "10.0.0.2", Result: "Detected in DB"}}},
}

type Dummy struct{}

type config struct {
	API string `json:"api_key"`
}

func (d Dummy) Initialize(b []byte) (err error) {
	cfg := config{}
	return json.Unmarshal(b, &cfg)
}

func (d Dummy) CheckIP(ctx context.Context, ipstr string) (found bool, results []intel.Result, err error) {
	_, err = ip.IsPrivateIP(ipstr)
	if err != nil {
		return
	}
	for _, v := range tblIntel {
		if ipstr == v.ip {
			return v.expectedFound, v.expectedRes, nil
		}
	}
	return
}

func TestIntel(t *testing.T) {
	_, err := test.DirEnv(false)
	if err != nil {
		t.Fatal(err)
	}

	d, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	apm.Enable(true)
	apmString := "dummy"
	tx := apm.StartTransaction(apmString, apmString, nil, nil)
	ts := tx.GetTraceContext()
	intel.RegisterExtension(new(Dummy), "Dummy")

	intelFileGlob = "intel_dummy.json"
	confDir := path.Join(d, "fixtures", "plugin1")
	if err = InitIntel(confDir, 0); err == nil {
		t.Fatal("Expected to fail initializing intel")
	}
	confDir = path.Join(d, "fixtures", "plugin2")
	if err = InitIntel(confDir, 0); err != nil {
		t.Fatal("Expected to only give warning on failure to load config: ", err)
	}
	confDir = path.Join(d, "fixtures", "plugin3")
	if err = InitIntel(confDir, 0); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tblIntel {
		_, _ = CheckIntelIP(tt.ip, 0, ts)
		found, res := CheckIntelIP(tt.ip, 0, ts)
		if found != tt.expectedFound {
			t.Errorf("Intel: %v, expected found %v, actual %v", tt.ip, tt.expectedFound, found)
		}
		if !reflect.DeepEqual(res, tt.expectedRes) {
			t.Errorf("Intel: %v, expected result %v, actual %v", tt.ip, tt.expectedRes, res)
		}
	}

	// for corrupted cache
	ip := tblIntel[0].ip
	intelCache.Set(ip, []byte("foo"))
	if found, _ := CheckIntelIP(ip, 0, nil); found {
		t.Errorf("Intel: expected to fail on corrupted cache")
	}

}
