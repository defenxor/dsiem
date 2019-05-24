package siem

import (
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
)

func TestInitDirective(t *testing.T) {

	allBacklogs = []backlogs{}

	fmt.Println("Starting TestInitDirective.")

	setTestDir(t)

	t.Logf("Using base dir %s", testDir)
	fDir := path.Join(testDir, "internal", "pkg", "dsiem", "siem", "fixtures")
	evtChan := make(chan event.NormalizedEvent)
	err := InitDirectives(path.Join(fDir, "directive2"), evtChan, 0)
	if err == nil || !strings.Contains(err.Error(), "Cannot load any directive from") {
		t.Fatal(err)
	}
	err = InitDirectives(path.Join(fDir, "directive1"), evtChan, 0)
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
