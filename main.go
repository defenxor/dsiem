package main

import (
	"flag"
)

var progDir string
var devEnv bool

func init() {
	b := flag.Bool("dev", false, "enable/disable dev env specific settings.")
	flag.Parse()
	devEnv = *b
	d, _ := getDir()
	progDir = d
}

func main() {
	setupLogger()
	err := initAssets()
	if err != nil {
		logger.Info("Cannot initialize assets: ", err)
		return
	}
	err = initDirectives()
	if err != nil {
		logger.Info("Cannot initialize directives: ", err)
		return
	}
	initBackLog()
	initAlarm()
	startServer()
}
