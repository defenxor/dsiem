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
	"strconv"

	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/ip"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"

	"os"
	"path"
	"reflect"
	"testing"

	"github.com/defenxor/dsiem/pkg/vuln"
)

type vulnTests struct {
	ip            string
	port          int
	expectedFound bool
	expectedRes   []vuln.Result
}

var tblVuln = []vulnTests{
	{"10.0.0.1", 80, false, nil},
	{"not-an-ip", 80, false, nil},
	{"10.0.0.2", 80, true, []vuln.Result{{Provider: "Dummy", Term: "10.0.0.2", Result: "Detected in DB"}}},
	{"10.0.0.2", 80, true, []vuln.Result{{Provider: "Dummy", Term: "10.0.0.2", Result: "Detected in DB"}}},
}

type DummyV struct{}

func (d DummyV) Initialize(b []byte) (err error) {
	cfg := config{}
	return json.Unmarshal(b, &cfg)
}

func (d DummyV) CheckIPPort(ctx context.Context, ipstr string, port int) (found bool, results []vuln.Result, err error) {
	_, err = ip.IsPrivateIP(ipstr)
	if err != nil {
		return
	}
	for _, v := range tblVuln {
		if ipstr == v.ip && port == v.port {
			return v.expectedFound, v.expectedRes, nil
		}
	}
	return
}

func TestVuln(t *testing.T) {
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
	th := tx.GetTraceContext()
	vuln.RegisterExtension(new(DummyV), "DummyV")

	vulnFileGlob = "vuln_dummy.json"
	confDir := path.Join(d, "fixtures", "plugin1")
	if err = InitVuln(confDir, 0); err == nil {
		t.Fatal("Expected to fail initializing vuln")
	}
	confDir = path.Join(d, "fixtures", "plugin2")
	if err = InitVuln(confDir, 0); err != nil {
		t.Fatal("Expected to only give warning on failure to load config: ", err)
	}
	confDir = path.Join(d, "fixtures", "plugin3")
	if err = InitVuln(confDir, 0); err != nil {
		t.Fatal(err)
	}

	for _, tt := range tblVuln {
		_, _ = CheckVulnIPPort(tt.ip, tt.port, th)
		found, res := CheckVulnIPPort(tt.ip, tt.port, th)
		if found != tt.expectedFound {
			t.Errorf("Vuln: %v %v, expected found %v, actual %v", tt.ip, tt.port, tt.expectedFound, found)
		}
		if !reflect.DeepEqual(res, tt.expectedRes) {
			t.Errorf("Vuln: %v %v, expected result %v, actual %v", tt.ip, tt.port, tt.expectedRes, res)
		}
	}

	// for corrupted cache
	ip := tblVuln[0].ip
	port := tblVuln[0].port
	vulnCache.Set(ip+":"+strconv.Itoa(port), []byte("foo"))
	if found, _ := CheckVulnIPPort(ip, port, nil); found {
		t.Errorf("Vuln: expected to fail on corrupted cache")
	}
}
