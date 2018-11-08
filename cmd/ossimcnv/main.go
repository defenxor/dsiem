// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
//
// This file is part of Dsiem.
//
// Dsiem is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation version 3 of the License.
//
// Dsiem is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Dsiem. If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/defenxor/dsiem/internal/pkg/ossimcnv"
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
