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

package dpluger

import (
	"os"
	"path"
	"sort"
	"strconv"

	"github.com/dogenzaka/tsv"
)

type pluginSIDRef struct {
	Name     string `tsv:"plugin"`
	ID       int    `tsv:"id"`
	SID      int    `tsv:"sid"`
	SIDTitle string `tsv:"title"`
}

type tsvRef struct {
	Sids  map[int]pluginSIDRef
	fname string
}

func (c *tsvRef) setFilename(pluginName string, confFile string) {
	dir := path.Dir(confFile)
	c.fname = path.Join(dir, pluginName+"_plugin-sids.tsv")
	return
}

func (c *tsvRef) init(pluginName string, confFile string) {
	c.Sids = make(map[int]pluginSIDRef)
	c.setFilename(pluginName, confFile)
	f, err := os.OpenFile(c.fname, os.O_RDONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	ref := pluginSIDRef{}
	parser, _ := tsv.NewParser(f, &ref)
	for {
		eof, err := parser.Next()
		if err != nil {
			continue
		}
		c.Sids[ref.SID] = ref
		if eof {
			break
		}
	}
}

func (c *tsvRef) upsert(pluginName string, pluginID int,
	pluginSID *int, sidTitle string) (shouldIncreaseID bool) {

	// First check the title, exit if already exist
	tKey := 0
	for k, v := range c.Sids {
		if v.SIDTitle == sidTitle {
			tKey = k
			break
		}
	}
	if tKey != 0 {
		// should increase new SID number if tKey == pluginSID by coincidence
		if tKey == *pluginSID {
			return true
		}
		return false
	}
	// here title doesnt yet exist so we add it

	// first find available SID
	for {
		_, used := c.Sids[*pluginSID]
		if !used {
			break
		}
		*pluginSID++
	}
	r := pluginSIDRef{
		Name:     pluginName,
		SID:      *pluginSID,
		ID:       pluginID,
		SIDTitle: sidTitle,
	}
	c.Sids[*pluginSID] = r
	return true
}

func (c tsvRef) save() error {
	f, err := os.OpenFile(c.fname, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString("plugin\tid\tsid\ttitle\n"); err != nil {
		return err
	}
	// use slice to get a sorted keys, ikr
	var keys []int
	for k := range c.Sids {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		v := c.Sids[k]
		if _, err := f.WriteString(v.Name + "\t" +
			strconv.Itoa(v.ID) + "\t" + strconv.Itoa(v.SID) + "\t" +
			v.SIDTitle + "\n"); err != nil {
			return err
		}
	}
	return nil
}
