package main

import (
	"dsiem/internal/pkg/ossimcnv"
	"flag"
	"fmt"
	"os"
)

var (
	srcFile     string
	dstFile     string
	ossimRefDir string
)

func init() {
	s := flag.String("in", "", "source OSSIM directive XML file to convert, e.g. point to user.xml path")
	d := flag.String("out", "./directives_ossim.json", "destination directive .json to produce")
	o := flag.String("refdir", "./ossimref", "location of TSV files produced by running dumptable.sh in OSSIM server")
	flag.Parse()
	srcFile = *s
	dstFile = *d
	ossimRefDir = *o
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
	filename, err := ossimcnv.CreateTempOSSIMFile(srcFile)
	if err != nil {
		exit("Cannot create temporary XML file", err)
		return
	}
	err = ossimcnv.ParseOSSIMTSVs(ossimRefDir)
	if err != nil {
		exit("Cannot parse ossim reference TSV from "+ossimRefDir, err)
		return
	}
	err = ossimcnv.CreateSIEMDirective(filename, dstFile)
	if err != nil {
		exit("Cannot create SIEM json directive", err)
		return
	}
	fmt.Println("Done. Results in", dstFile)
}
