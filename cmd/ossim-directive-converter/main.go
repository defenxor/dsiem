package main

import (
	conv "dsiem/internal/converter/pkg/converter"
	"flag"
	"fmt"
	"os"
)

const (
	ossimRefDir = "./ossimref"
)

var (
	srcFile string
	dstFile string
)

func init() {
	s := flag.String("in", "", "source OSSIM directive XML file to convert, e.g. point to user.xml path")
	d := flag.String("out", "./directives_ossim.json", "destination directive .json to produce")
	flag.Parse()
	srcFile = *s
	dstFile = *d
	if srcFile == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func exit(msg string, err error) {
	fmt.Println("Exiting:", msg)
	panic(err)
}

func main() {
	filename, err := conv.CreateTempOSSIMFile(srcFile)
	if err != nil {
		exit("Cannot create temporary XML file", err)
		return
	}
	err = conv.ParseOSSIMTSVs(ossimRefDir)
	if err != nil {
		exit("Cannot parse ossim reference TSV from "+ossimRefDir, err)
		return
	}
	err = conv.CreateSIEMDirective(filename, dstFile)
	if err != nil {
		exit("Cannot create SIEM json directive", err)
		return
	}
	fmt.Println("Done. Results in", dstFile)
}
