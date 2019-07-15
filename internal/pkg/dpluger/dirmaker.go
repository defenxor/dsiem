// Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.
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

package dpluger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	"github.com/dogenzaka/tsv"
)

type tsvEntry struct {
	Plugin string `tsv:"plugin"`
	ID     int    `tsv:"id"`
	SID    int    `tsv:"sid"`
	Title  string `tsv:"title"`
}

type tsvEntries struct {
	records []tsvEntry
}

// CreateDirective starts directive creation
func CreateDirective(tsvFile, outFile, kingdom, category string, priority, reliability, dirNumber int) (err error) {

	f1, err := os.Open(tsvFile)
	if err != nil {
		return err
	}
	defer f1.Close()

	t := tsvEntries{}
	rec := tsvEntry{}
	parser, _ := tsv.NewParser(f1, &rec)
	for {
		eof, err := parser.Next()
		if err != nil {
			return err
		}
		if eof {
			break
		}
		t.records = append(t.records, rec)
	}

	// load existing directives first if any
	locDir := filepath.Dir(outFile)
	locFile := filepath.Base(outFile)
	dirs, _, _ := siem.LoadDirectivesFromFile(locDir, locFile)

	addedCount := 0

	for _, v := range t.records {

		// first check if the directive title already exist
		d := siem.Directive{}
		d.Name = v.Title + " (SRC_IP to DST_IP)"
		if isDirectiveNameExist(dirs, d) {
			fmt.Println("Skipping an existing directive " + d.Name)
			continue
		}

		// fmt.Println("DEBUG:", v.Plugin, v.Title, v.ID, v.SID)
		r1 := rule.DirectiveRule{}
		r1.Name = v.Title
		r1.Type = "PluginRule"
		r1.Stage = 1
		r1.PluginID = v.ID
		r1.PluginSID = append(r1.PluginSID, v.SID)
		r1.Occurrence = 1
		r1.From = "ANY"
		r1.To = "ANY"
		r1.PortFrom = "ANY"
		r1.PortTo = "ANY"
		r1.Protocol = "TCP/IP"
		r1.Reliability = 1
		r1.Timeout = 0

		r2 := rule.DirectiveRule{}
		r2.Name = v.Title
		r2.Type = "PluginRule"
		r2.Stage = 2
		r2.PluginID = v.ID
		r2.PluginSID = append(r2.PluginSID, v.SID)
		r2.Occurrence = 10
		r2.From = ":1"
		r2.To = ":1"
		r2.PortFrom = "ANY"
		r2.PortTo = "ANY"
		r2.Protocol = "TCP/IP"
		r2.Reliability = 5
		r2.Timeout = 3600

		r3 := rule.DirectiveRule{}
		r3.Name = v.Title
		r3.Type = "PluginRule"
		r3.Stage = 3
		r3.PluginID = v.ID
		r3.PluginSID = append(r3.PluginSID, v.SID)
		r3.Occurrence = 10000
		r3.From = ":1"
		r3.To = ":1"
		r3.PortFrom = "ANY"
		r3.PortTo = "ANY"
		r3.Protocol = "TCP/IP"
		r3.Reliability = 10
		r3.Timeout = 21600

		d.Priority = priority
		d.Kingdom = kingdom
		d.Category = category
		d.Rules = append(d.Rules, r1, r2, r3)

		d.ID = dirNumber
		for isDirectiveNumberExist(dirs, d) {
			d.ID++
			dirNumber = d.ID
		}

		dirs.Dirs = append(dirs.Dirs, d)
		addedCount++
		dirNumber = dirNumber + 1
	}

	b, err := json.MarshalIndent(dirs, "", "  ")
	if err != nil {
		return
	}

	fmt.Printf("Found %v new directives\n", addedCount)

	err = fs.OverwriteFile(string(b), outFile)
	return
}

func isDirectiveNameExist(ref siem.Directives, dir siem.Directive) bool {
	for _, v := range ref.Dirs {
		if v.Name == dir.Name {
			return true
		}
	}
	return false
}

func isDirectiveNumberExist(ref siem.Directives, dir siem.Directive) bool {
	for _, v := range ref.Dirs {
		if v.ID == dir.ID {
			return true
		}
	}
	return false
}
