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
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	xc "github.com/defenxor/dsiem/internal/pkg/dsiem/xcorrelator"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"

	"github.com/spf13/viper"
)

func initAsset(d string, t *testing.T) {
	// needed by rule checkers
	err := asset.Init(path.Join(d, "internal", "pkg", "dsiem", "asset", "fixtures", "asset1"))
	if err != nil {
		t.Fatal(err)
	}
}

var testRootDir string

func initDirAndLog(t *testing.T) {
	if testRootDir != "" {
		return
	}
	d, err := test.DirEnv(true)
	if err != nil {
		t.Fatal(err)
	}
	testRootDir = d
}
func TestAlarm(t *testing.T) {
	initDirAndLog(t)

	t.Logf("Enabling log test mode")
	log.EnableTestingMode()

	initAsset(testRootDir, t)
	registerTI(testRootDir, t)
	registerVuln(testRootDir, t)
	xc.IntelEnabled = true
	xc.VulnEnabled = true

	t.Logf("Enabling log test mode")
	log.EnableTestingMode()

	apm.Enable(true)

	tmpLog := path.Join(os.TempDir(), "siem_alarm_events.log")
	defer os.Remove(tmpLog)

	viper.Set("medRiskMin", 0)
	viper.Set("medRiskMax", 6)
	viper.Set("tags", []string{"Identified Threat", "Valid Threat"})
	viper.Set("status", []string{"Open", "Closed"})

	if err := Init(`/\/\/\/`, false); err == nil {
		t.Fatal("expected to fail due to wrong path for log file")
	}
	if err := Init(tmpLog, false); err == nil {
		t.Fatal("expected to fail due to wrong medRiskMin")
	}

	viper.Set("medRiskMin", 3)
	if err := Init(tmpLog, false); err != nil {
		t.Fatal(err)
	}

	id := "12313"
	name := "Attack to Someone"
	kingdom := "Kingdom"
	category := "Category"
	srcIPs := []string{"10.0.0.1"}
	dstIPs := []string{"10.0.0.2"}
	lastSrcPort := 31337
	lastDstPort := 80
	risk := 1
	statusTime := time.Now().Unix()

	r := rule.DirectiveRule{}
	r.Name = "Rule1"
	r.Type = "PluginRule"
	r.PluginID = 1001
	r.PluginSID = []int{1, 2, 3}

	rules := []rule.DirectiveRule{r}

	connID := uint64(1)
	checkIntelVuln := true
	tx := apm.StartTransaction("test", "test", nil, nil)

	cd := []rule.CustomData{}
	fmt.Print("upserting low risk alarm ..")
	verifyFuncOutput(t, func() {
		Upsert(id, name, kingdom, category, srcIPs, dstIPs, cd, lastSrcPort, lastDstPort, risk, statusTime, rules, connID, checkIntelVuln, tx)
	}, "alarm updating Elasticsearch", true)

	fmt.Print("upserting medium risk alarm ..")
	risk = 3
	verifyFuncOutput(t, func() {
		Upsert(id, name, kingdom, category, srcIPs, dstIPs, cd, lastSrcPort, lastDstPort, risk, statusTime, rules, connID, checkIntelVuln, tx)
	}, "alarm updating Elasticsearch", true)

	fmt.Print("upserting high risk alarm ..")
	risk = 7
	verifyFuncOutput(t, func() {
		Upsert(id, name, kingdom, category, srcIPs, dstIPs, cd, lastSrcPort, lastDstPort, risk, statusTime, rules, connID, checkIntelVuln, tx)
	}, "alarm updating Elasticsearch", true)

	expected := 1
	if n := Count(); n != expected {
		t.Errorf("alarm count expected %v, returned %v", expected, n)
	}

	fmt.Print("Removing alarm ..")
	verifyFuncOutput(t, func() {
		ch := RemovalChannel()
		ch <- id
		time.Sleep(time.Second)
	}, "Removing alarm", true)

}
