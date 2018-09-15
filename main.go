package main

import (
	"flag"
	"net"

	"github.com/sirupsen/logrus"
)

type logger struct {
	logger *logrus.Logger
}

var progDir string
var devEnv bool

func init() {
	b := flag.Bool("dev", false, "enable/disable dev env specific settings.")
	dbg := flag.Bool("debug", false, "enable/disable debug level logging.")
	flag.Parse()
	devEnv = *b
	d, _ := getDir()
	progDir = d
	level := logrus.InfoLevel
	if *dbg {
		level = logrus.DebugLevel
	}
	setupLogger(level)

	for _, cidr := range []string{
		"127.0.0.0/8",    // IPv4 loopback
		"10.0.0.0/8",     // RFC1918
		"172.16.0.0/12",  // RFC1918
		"192.168.0.0/16", // RFC1918
		"::1/128",        // IPv6 loopback
		"fe80::/10",      // IPv6 link-local
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateIPBlocks = append(privateIPBlocks, block)
	}
}

func main() {
	err := initAssets()
	if err != nil {
		logInfo("Cannot initialize assets: "+err.Error(), 0)
		return
	}
	err = initIntel()
	if err != nil {
		logInfo("Cannot initialize threat intel: "+err.Error(), 0)
		return
	}
	err = initVuln()
	if err != nil {
		logInfo("Cannot initialize Vulnerability scan result: "+err.Error(), 0)
		return
	}
	err = initDirectives()
	if err != nil {
		logInfo("Cannot initialize directives: "+err.Error(), 0)
		return
	}
	initBackLog()
	initAlarm()
	startServer()
}
