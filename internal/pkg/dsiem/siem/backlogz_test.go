package siem

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/alarm"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/jonhoo/drwmutex"
)

var (
	assetInitialized bool
	alarmInitialized bool
)

func initAlarm(t *testing.T) {
	if alarmInitialized {
		return
	}
	viper.Set("medRiskMin", 3)
	viper.Set("medRiskMax", 6)
	viper.Set("tags", []string{"Identified Threat", "Valid Threat"})
	viper.Set("status", []string{"Open", "Closed"})
	err := alarm.Init("")
	if err != nil {
		t.Fatal(err)
	}
	alarmInitialized = true
}

func initAsset(t *testing.T) {
	// needed by rule checkers
	if assetInitialized {
		return
	}
	err := asset.Init(path.Join(testDir, "internal", "pkg", "dsiem", "asset", "fixtures", "asset1"))
	if err != nil {
		t.Fatal(err)
	}
	assetInitialized = true
}

func TestBackLog(t *testing.T) {

	fmt.Println("Starting TestBackLog.")

	setTestDir(t)
	t.Logf("Using base dir %s", testDir)

	if !log.TestMode {
		t.Logf("Enabling log test mode")
		log.EnableTestingMode()
	}

	initAlarm(t)
	initAsset(t)

	fDir := path.Join(testDir, "internal", "pkg", "dsiem", "siem", "fixtures")

	// use directive that expires fast and has only 2 stages
	dirs, _, err := LoadDirectivesFromFile(path.Join(fDir, "directive4"), directiveFileGlob)
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
	dctives := dirs.Dirs[0]
	e.PluginID = dctives.Rules[0].PluginID
	e.PluginSID = 2100384

	e.Timestamp = time.Now().UTC().Format(time.RFC3339)

	apm.Enable(true)

	b, err := createNewBackLog(dirs.Dirs[0], e)
	if err != nil {
		t.Fatal(err)
	}
	b.DRWMutex = drwmutex.New()
	viper.Set("medRiskMin", 3)
	viper.Set("medRiskMax", 6)
	viper.Set("tags", []string{"Identified Threat", "Valid Threat"})
	viper.Set("status", []string{"Open", "Closed"})
	viper.Set("maxDelay", 100)

	bLogs := backlogs{}
	bLogs.bpCh = make(chan bool)
	bLogs.DRWMutex = drwmutex.New()
	bLogs.bl = make(map[string]*backLog)
	bLogs.bl["test"] = b
	b.bLogs = &bLogs

	go func() {
		for {
			<-b.bLogs.bpCh
		}
	}()
	go func() {
		for {
			<-b.chFound
		}
	}()
	go func() {
		for {
			<-b.chDone
		}
	}()

	fmt.Println("first event (by start)")
	go b.start(e)

	fmt.Print("second event ..")
	e.ConnID = 2
	verifyEventOutput(t, e, b.chData, "reached max stage and occurrence")

	fmt.Print("under pressure ..")
	e.RcvdTime = time.Now().Add(-700 * time.Second).Unix()
	verifyEventOutput(t, e, b.chData, "backlog is under pressure")

	fmt.Print("out of order event ..")
	e.Timestamp = time.Now().Add(time.Second * -300).UTC().Format(time.RFC3339)
	verifyEventOutput(t, e, b.chData, "event timestamp out of order")

	fmt.Print("invalid timestamp ..")
	e.Timestamp = "#"
	verifyEventOutput(t, e, b.chData, "cannot parse event timestamp")

	fmt.Print("3rd event ..")
	e.ConnID = 3
	verifyEventOutput(t, e, b.chData, "backlog incoming event")

	/*

			}
			// discard out of order event
			if !b.isTimeInOrder(idx, ts) {
				b.warn("event timestamp out of order, discarding it", evt.ConnID)
				l.Unlock()
				continue
			}

			if b.isUnderPressure(evt.RcvdTime, int64(maxDelay)) {
				b.warn("backlog is under pressure", evt.ConnID)
				select {
				case b.bLogs.bpCh <- true:
				default:
				}

		/*
			for k := range allBacklogs[0].bl {
				fmt.Print("Deleting " + k + " through backlog member function ..")
				verifyFuncOutput(t, func() {
					allBacklogs[0].bl[k].delete()
					time.Sleep(time.Second * 1)
				}, "", true)
			}
	*/
}
