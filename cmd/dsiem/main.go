package main

import (
	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/dsiem/pkg/server"
	"dsiem/internal/dsiem/pkg/siem"
	xc "dsiem/internal/dsiem/pkg/xcorrelator"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"flag"
	"fmt"
	"os"
	"path"
)

var (
	confDir   string
	logDir    string
	debugFlag bool
	port      int
	buildTime string
	version   string
	addr      string
)

const (
	progName    = "dsiem"
	aEventsLogs = "siem_alarm_events.json"
	alarmLogs   = "siem_alarms.json"
)

func init() {
	dev := flag.Bool("dev", false, "enable/disable dev env specific directory.")
	dbg := flag.Bool("debug", false, "enable/disable debug level logging.")
	ver := flag.Bool("version", false, "display version and build time.")
	usage := flag.Bool("usage", false, "display acceptable CLI argument.")
	a := flag.String("address", "0.0.0.0", "IP address to listen on.")
	p := flag.Int("port", 8080, "TCP port to listen to.")
	flag.Parse()
	if *ver {
		fmt.Println(progName, version, "("+buildTime+")")
		os.Exit(0)
	}
	if *usage {
		flag.Usage()
		os.Exit(0)
	}
	addr = *a
	port = *p
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
	if debugFlag {
		fmt.Println(msg)
		panic(err)
	} else {
		fmt.Println("Exiting: " + msg + ": " + err.Error())
		os.Exit(1)
	}
}

func main() {
	log.Setup(debugFlag)

	eventChannel = make(chan event.NormalizedEvent)

	log.Info("Starting "+progName+" "+version, 0)

	err := asset.Init(confDir)
	if err != nil {
		exit("Cannot initialize assets from "+confDir, err)
	}
	err = xc.InitIntel(confDir)
	if err != nil {
		exit("Cannot initialize threat intel", err)
	}
	err = xc.InitVuln(confDir)
	if err != nil {
		exit("Cannot initialize Vulnerability scan result", err)
	}
	err = siem.InitDirectives(confDir, eventChannel)
	if err != nil {
		exit("Cannot initialize directives", err)
	}
	err = siem.InitBackLog(path.Join(logDir, aEventsLogs))
	if err != nil {
		exit("Cannot initialize backlog", err)
	}
	err = siem.InitAlarm(path.Join(logDir, alarmLogs))
	if err != nil {
		exit("Cannot initialize alarm", err)
	}
	err = server.Start(eventChannel, confDir, addr, port)
	if err != nil {
		exit("Cannot start server", err)
	}
}
