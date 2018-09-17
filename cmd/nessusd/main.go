package main

import (
	"dsiem/internal/nessusd/pkg/server"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"

	"flag"
	"fmt"
	"os"
)

var (
	csvDir    string
	debugFlag bool
	port      int
	buildTime string
	version   string
	addr      string
)

const (
	progName = "nessusd"
)

func init() {
	dev := flag.Bool("dev", false, "enable/disable dev env specific directory.")
	dbg := flag.Bool("debug", false, "enable/disable debug level logging.")
	ver := flag.Bool("version", false, "display version and build time.")
	usage := flag.Bool("usage", false, "display acceptable CLI argument.")
	dir := flag.String("dir", "", "path of the CSV scan results.")
	a := flag.String("address", "127.0.0.1", "IP address to listen on.")
	p := flag.Int("port", 8081, "TCP port to listen to.")
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
	if *dir == "" {
		csvDir = d
	} else {
		csvDir = *dir
	}

	if *dbg {
		debugFlag = true
	}
}

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

	log.Info("Starting "+progName+" "+version, 0)

	err := server.InitCSV(csvDir)
	if err != nil {
		exit("Cannot read Nessus CSV from "+csvDir, err)
	}

	err = server.Start(addr, port)
	if err != nil {
		exit("Cannot start server", err)
	}
}
