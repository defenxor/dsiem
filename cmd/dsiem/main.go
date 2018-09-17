package main

import (
	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	log "dsiem/internal/dsiem/pkg/logger"
	"dsiem/internal/dsiem/pkg/server"
	"dsiem/internal/dsiem/pkg/siem"
	xc "dsiem/internal/dsiem/pkg/xcorrelator"
	"dsiem/internal/shared/pkg/fs"
	"flag"
	"fmt"
	"path"
)

var confDir string
var logDir string
var debugFlag bool

const (
	aEventsLogs = "siem_alarm_events.json"
	alarmLogs   = "siem_alarms.json"
)

func init() {
	dev := flag.Bool("dev", false, "enable/disable dev env specific directory.")
	dbg := flag.Bool("debug", false, "enable/disable debug level logging.")
	flag.Parse()
	d, err := fs.GetDir(*dev)
	if err != nil {
		exit("Cannot get current directory??", err)
	}
	confDir = path.Join(d, "configs")
	logDir = path.Join(d, "logs")

	if *dbg {
		debugFlag = true
	}
}

var eventChannel chan event.NormalizedEvent

func exit(msg string, err error) {
	fmt.Println("Exiting:", msg)
	panic(err)
}

func main() {
	log.Setup(debugFlag)

	eventChannel = make(chan event.NormalizedEvent)

	err := asset.Init(confDir)
	if err != nil {
		exit("Cannot initialize assets", err)
	}
	err = xc.InitIntel(confDir)
	if err != nil {
		exit("Cannot initialize threat intel", err)
	}
	err = xc.InitVuln(confDir)
	if err != nil {
		exit("Cannot initialize Vulnerability scan result", err)
	}
	err = siem.InitDirectives(confDir)
	if err != nil {
		exit("Cannot initialize directives", err)
	}
	siem.InitBackLog(path.Join(logDir, aEventsLogs))
	if err != nil {
		exit("Cannot initialize backlog", err)
	}

	siem.InitAlarm(path.Join(logDir, alarmLogs))
	server.Start(eventChannel)
}
