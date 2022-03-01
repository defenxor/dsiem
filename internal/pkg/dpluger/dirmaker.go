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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	"github.com/defenxor/dsiem/internal/pkg/shared/tsv"
)

type tsvEntries struct {
	records []PluginSID
}

// CreateDirective starts directive creation
func CreateDirective(tsvFile, outFile, kingdom, titleTemplate string, priority, reliability, dirNumber int) error {
	f1, err := os.Open(tsvFile)
	if err != nil {
		return err
	}
	defer f1.Close()

	// load existing directives first if any
	dirs, _, err := siem.LoadDirectivesFromFile(filepath.Dir(outFile), filepath.Base(outFile), true)
	if err != nil && err != siem.ErrNoDirectiveLoaded {
		return err
	}

	dirs, err = createDirective(f1, dirs, kingdom, titleTemplate, priority, reliability, dirNumber)
	if err != nil {
		return err
	}

	return fs.OverwriteFileValueIndent(dirs, outFile)
}

const (
	maxReliability = 10
	minReliability = 1
)

func createDirective(in io.Reader, dirs siem.Directives, kingdom, titleTemplate string, priority,
	reliability, dirNumber int) (siem.Directives, error) {

	parser := tsv.NewParser(in)

	defaultRef := PluginSID{
		Kingdom: kingdom,
	}

	entries := tsvEntries{}
	for {
		var ref PluginSID
		ok := parser.Read(&ref, defaultRef)
		if !ok {
			break
		}

		entries.records = append(entries.records, ref)
	}

	addedCount := 0
	for _, v := range entries.records {
		if v.SIDTitle == "" || v.SID == 0 {
			fmt.Println("Skipping an empty title or SID in TSV file")
			continue
		}

		d := siem.Directive{}
		d.Name = strings.ReplaceAll(titleTemplate, "EVENT_TITLE", v.SIDTitle)

		if index, exist := isDirectiveNameExistIndex(dirs, d); exist {
			fmt.Printf("merging plugin-sid list of an existing directive: %s\n", d.Name)
			for i := range dirs.Dirs[index].Rules {
				dirs.Dirs[index].Rules[i].PluginSID = mergeUniqueSort(dirs.Dirs[index].Rules[i].PluginSID, []int{v.SID})
			}
			continue
		}

		SIDList := []int{v.SID}
		// fmt.Println("DEBUG:", v.Plugin, v.Title, v.ID, v.SID)

		if reliability < minReliability {
			reliability = minReliability
		} else if reliability > maxReliability {
			reliability = maxReliability
		}

		r1 := rule.DirectiveRule{
			Name:        v.SIDTitle,
			Type:        "PluginRule",
			Stage:       1,
			PluginID:    v.ID,
			PluginSID:   SIDList,
			Occurrence:  1,
			From:        "ANY",
			To:          "ANY",
			PortFrom:    "ANY",
			PortTo:      "ANY",
			Protocol:    "ANY",
			Reliability: reliability,
			Timeout:     0,
		}

		nextReliability := reliability + 4
		if nextReliability > maxReliability {
			nextReliability = maxReliability
		}

		r2 := rule.DirectiveRule{
			Name:        v.SIDTitle,
			Type:        "PluginRule",
			Stage:       2,
			PluginID:    v.ID,
			PluginSID:   SIDList,
			Occurrence:  10,
			From:        ":1",
			To:          ":1",
			PortFrom:    "ANY",
			PortTo:      "ANY",
			Protocol:    "ANY",
			Reliability: nextReliability,
			Timeout:     3600,
		}

		r3 := rule.DirectiveRule{
			Name:       v.SIDTitle,
			Type:       "PluginRule",
			Stage:      3,
			PluginID:   v.ID,
			PluginSID:  SIDList,
			Occurrence: 10000,
			From:       ":1",
			To:         ":1",
			PortFrom:   "ANY",
			PortTo:     "ANY",
			Protocol:   "ANY",

			// Stage #3 rule always have maximum reliability
			Reliability: maxReliability,
			Timeout:     21600,
		}

		d.Priority = priority
		d.Kingdom = v.Kingdom
		d.Category = v.Category
		d.Rules = []rule.DirectiveRule{r1, r2, r3}

		d.ID = dirNumber
		for isDirectiveNumberExist(dirs, d) {
			d.ID++
			dirNumber = d.ID
		}

		dirs.Dirs = append(dirs.Dirs, d)
		addedCount++
		dirNumber = dirNumber + 1
	}

	fmt.Printf("Found %v new directives\n", addedCount)
	return dirs, nil
}

func isDirectiveNameExistIndex(ref siem.Directives, dir siem.Directive) (int, bool) {
	for idx, v := range ref.Dirs {
		if v.Name == dir.Name {
			return idx, true
		}
	}
	return 0, false
}

func isDirectiveNumberExist(ref siem.Directives, dir siem.Directive) bool {
	for _, v := range ref.Dirs {
		if v.ID == dir.ID {
			return true
		}
	}
	return false
}

// TODO: (rkspx) move to util file for reuse
// mergeUniqueSort merge the two slice of int, returning sorted unique slice of int
func mergeUniqueSort(s1, s2 []int) []int {
	m := map[int]bool{}
	for _, z := range s2 {
		if _, ok := m[z]; ok {
			continue
		} else {
			m[z] = true
		}
	}

	for _, s := range s1 {
		if _, ok := m[s]; ok {
			continue
		} else {
			m[s] = true
		}
	}

	res := make([]int, 0, len(m))
	for k := range m {
		res = append(res, k)
	}

	sort.Sort(intlist(res))

	return res
}

type intlist []int

func (s intlist) Len() int           { return len(s) }
func (s intlist) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s intlist) Less(i, j int) bool { return s[i] < s[j] }
