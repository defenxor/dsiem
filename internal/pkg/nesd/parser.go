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

package nesd

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/gocarina/gocsv/v2"
)

const (
	nessusGlob = "nessus_*.csv"
)

type nessusScans struct {
	entries []nScan
}

// Plugin ID,CVE,CVSS,Risk,Host,Protocol,Port,Name,Synopsis,Description,Solution,See Also,Plugin Output
type nScan struct {
	PluginID     int64   `csv:"Plugin ID"`
	CVE          string  `csv:"CVE"`
	CVSS         float32 `csv:"CVSS"`
	Risk         string  `csv:"Risk"`
	Host         string  `csv:"Host"`
	Protocol     string  `csv:"Protocol"`
	Port         int     `csv:"Port"`
	Name         string  `csv:"Name"`
	Synopsis     string  `csv:"Synopsis"`
	Description  string  `csv:"Description"`
	Solution     string  `csv:"Solution"`
	SeeAlso      string  `csv:"See Also"`
	PluginOutput string  `csv:"Plugin Output"`
}

var vulns nessusScans

// InitCSV read nessus scan results from CSV
func InitCSV(dir string) error {
	csvDir = dir
	p := path.Join(csvDir, nessusGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}

	for i := range files {
		var n nessusScans
		file, err := os.Open(files[i])
		if err == nil {
			defer file.Close()

			byteValue, _ := io.ReadAll(file)
			err = gocsv.UnmarshalBytes(byteValue, &n.entries)
			if err != nil {
				return err
			}
			for j := range n.entries {
				vulns.entries = append(vulns.entries, n.entries[j])
			}
		}
	}

	total := len(vulns.entries)
	if total == 0 {
		return errors.New("cannot find valid nessus scan results to load from " + dir)
	}
	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(total) + " scan entries."})

	return nil
}
