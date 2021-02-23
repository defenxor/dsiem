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

package alarm

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	xc "github.com/defenxor/dsiem/internal/pkg/dsiem/xcorrelator"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/ip"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/pkg/intel"
	"github.com/defenxor/dsiem/pkg/vuln"
)

var registeredTI bool
var registeredVuln bool

type intelTests struct {
	ip            string
	expectedFound bool
	expectedRes   []intel.Result
}

var tblIntel = []intelTests{
	{"8.8.8.8", true, []intel.Result{{Provider: "Dummy", Term: "8.8.8.8", Result: "Detected in DB"}}},
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
func registerTI(d string, t *testing.T) {
	if registeredTI {
		return
	}
	intel.RegisterExtension(new(Dummy), "Dummy")

	confDir := path.Join(d, "internal", "pkg", "dsiem", "xcorrelator", "fixtures", "plugin3")
	t.Log("using confDir:", confDir)
	fmt.Print("initializing intel xcorrelator ..")
	verifyFuncOutput(t, func() {
		if err := xc.InitIntel(confDir, 20); err != nil {
			t.Fatal(err)
		}
	}, "Loaded 1 threat intelligence sources", true)
	registeredTI = true
}

func TestAsyncIntelCheck(t *testing.T) {

	initDirAndLog(t)

	t.Logf("Enabling log test mode")
	log.EnableTestingMode()

	registerTI(testRootDir, t)

	apm.Enable(true)
	tx := apm.StartTransaction("test", "test", nil, nil)
	if tx == nil {
		t.Fatal("cannot create a new APM transaction")
	}

	a := alarm{}
	a.SrcIPs = []string{"10.0.0.2"}
	a.DstIPs = []string{"8.8.4.4"}

	fmt.Print("checking alarm with no intel match ..")
	verifyFuncOutput(t, func() {
		asyncIntelCheck(&a, 0, false, tx)
		time.Sleep(time.Second)
	}, "Found intel result for "+a.DstIPs[0], false)

	a.Lock()
	a.DstIPs = []string{"8.8.8.8"}
	a.Unlock()

	fmt.Print("checking alarm with an intel match ..")
	verifyFuncOutput(t, func() {
		asyncIntelCheck(&a, 0, false, tx)
		time.Sleep(time.Second)
	}, "Found intel result for "+a.DstIPs[0], true)

	fmt.Print("checking the same alarm (already exist) ..")
	verifyFuncOutput(t, func() {
		asyncIntelCheck(&a, 0, false, tx)
		time.Sleep(time.Second)
	}, "", true)

}

type vulnTests struct {
	ip            string
	port          int
	expectedFound bool
	expectedRes   []vuln.Result
}

var tblVuln = []vulnTests{
	{"10.0.0.2", 80, true, []vuln.Result{{Provider: "Dummy", Term: "10.0.0.2:80", Result: "Detected in DB"}}},
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

func registerVuln(d string, t *testing.T) {
	if registeredVuln {
		return
	}
	vuln.RegisterExtension(new(DummyV), "DummyV")

	confDir := path.Join(d, "internal", "pkg", "dsiem", "xcorrelator", "fixtures", "plugin3")
	t.Log("using confDir:", confDir)
	fmt.Print("initializing vuln xcorrelator ..")
	verifyFuncOutput(t, func() {
		if err := xc.InitVuln(confDir, 20); err != nil {
			t.Fatal(err)
		}
	}, "Loaded 1 vulnerability scan result sources", true)
	registeredVuln = true
}
func TestAsyncVulnCheck(t *testing.T) {
	initDirAndLog(t)

	t.Logf("Enabling log test mode")
	log.EnableTestingMode()

	registerVuln(testRootDir, t)

	apm.Enable(true)
	tx := apm.StartTransaction("test", "test", nil, nil)
	if tx == nil {
		t.Fatal("cannot create a new APM transaction")
	}

	a := alarm{}
	srcPort := 31337
	dstPort := 80
	r := rule.DirectiveRule{}
	r.To = "10.0.0.4,10.0.0.4,ANY"
	r.From = "8.8.8.8,ANY"
	r.PortFrom = "ANY,40004,400TYPO"
	r.PortTo = "ANY,80,80TYPO"
	a.Rules = []rule.DirectiveRule{r}

	fmt.Print("checking alarm with no vuln match ..")
	verifyFuncOutput(t, func() {
		asyncVulnCheck(&a, srcPort, dstPort, 0, tx)
		time.Sleep(time.Second)
	}, "Found vulnerability for "+r.To, false)

	r.To = "10.0.0.2"
	a.Rules = []rule.DirectiveRule{r}

	fmt.Print("checking alarm with a vuln match ..")
	verifyFuncOutput(t, func() {
		asyncVulnCheck(&a, srcPort, dstPort, 0, tx)
		time.Sleep(time.Second)
	}, "Found vulnerability for "+r.To, false)

	fmt.Print("checking the same alarm (already exist) ..")
	verifyFuncOutput(t, func() {
		asyncVulnCheck(&a, srcPort, dstPort, 0, tx)
		time.Sleep(time.Second)
	}, "", true)

}

func verifyFuncOutput(t *testing.T, f func(), expected string, expectMatch bool) {
	out := log.CaptureZapOutput(f)
	t.Log("out: ", out)
	if !strings.Contains(out, expected) == expectMatch {
		t.Fatalf("Expected match %v: '%s' in output: %s", expectMatch, expected, out)
	} else {
		fmt.Println("OK")
	}
}
