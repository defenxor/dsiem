package event

import (
	"path"
	"testing"

	"dsiem/internal/pkg/dsiem/asset"
	"dsiem/internal/pkg/shared/fs"
	log "dsiem/internal/pkg/shared/logger"

	"github.com/sebdah/goldie"
)

var e = NormalizedEvent{}

func TestValid(t *testing.T) {
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
	goldie.AssertWithTemplate(t, "event", ts, b)
	if !e.Valid() {
		t.Errorf("Event is not valid %v", e)
	}
}
func TestFromToBytes(t *testing.T) {
	b, err := e.ToBytes()
	if err != nil {
		t.Error(err)
	}
	if err := e.FromBytes(b); err != nil {
		t.Error(err)
	}
}

func helperDirEnv(t *testing.T) (dir string) {
	dir, err := fs.GetDir(true)
	if err != nil {
		t.Fatal(err)
	}
	err = log.Setup(false)
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestInHomeNet(t *testing.T) {
	d := helperDirEnv(t)
	t.Logf("Using base dir %s", d)
	err := asset.Init(path.Join(d, "configs"))
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
