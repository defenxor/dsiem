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
	"strings"

	"github.com/defenxor/dsiem/internal/pkg/shared/tsv"
)

type PluginSID struct {
	Name     string `tsv:"plugin"`
	ID       int    `tsv:"id"`
	SID      int    `tsv:"sid"`
	SIDTitle string `tsv:"title"`
	Category string `tsv:"category"`
	Kingdom  string `tsv:"kingdom"`

	lastIndex int
}

// Defaults is implementation of tsv.Castable
func (p *PluginSID) Defaults(in interface{}) {
	v, ok := in.(PluginSID)
	if in == nil || !ok {
		v = PluginSID{}
	}

	if p.Name == "" {
		p.Name = v.Name
	}

	if p.ID == 0 {
		p.ID = v.ID
	}

	if p.SID == 0 {
		p.SID = v.SID
	}

	if p.SIDTitle == "" {
		p.SIDTitle = v.SIDTitle
	}

	if p.Category == "" {
		p.Category = v.Category
	}

	if p.Kingdom == "" {
		p.Kingdom = v.Kingdom
	}
}

// Next is implementation of tsv.Castable
func (p *PluginSID) Next(b tsv.Castable) bool {
	switch p.lastIndex {
	case 0:
		p.Name = b.String()
	case 1:
		p.ID = b.Int()
	case 2:
		p.SID = b.Int()
	case 3:
		p.SIDTitle = b.String()
	case 4:
		p.Category = b.String()
	case 5:
		p.Kingdom = b.String()
	default:
		return false
	}

	p.lastIndex++
	return true
}

type tsvRef struct {
	Sids  map[int]PluginSID
	fname string
}

func (c *tsvRef) setFilename(pluginName string, confFile string) {
	dir := path.Dir(confFile)
	c.fname = path.Join(dir, pluginName+"_plugin-sids.tsv")
}

func (c *tsvRef) init(pluginName string, confFile string) {
	c.Sids = make(map[int]PluginSID)
	c.setFilename(pluginName, confFile)
	f, err := os.OpenFile(c.fname, os.O_RDONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()

	parser := tsv.NewParser(f)
	for {
		var ref PluginSID
		ok := parser.Read(&ref, nil)
		if !ok {
			break
		}
		c.Sids[ref.SID] = ref
	}
}

func (c *tsvRef) upsert(pluginName string, pluginID int,
	pluginSID *int, category, sidTitle string) (shouldIncreaseID bool) {

	// replace " character in title and category, if any
	sidTitle = strings.ReplaceAll(sidTitle, "\"", "'")
	category = strings.ReplaceAll(category, "\"", "'")

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
		return tKey == *pluginSID
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
	r := PluginSID{
		Name:     pluginName,
		SID:      *pluginSID,
		ID:       pluginID,
		SIDTitle: sidTitle,
		Category: category,
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
	if _, err := f.WriteString("plugin\tid\tsid\ttitle\tcategory\tkingdom\n"); err != nil {
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
		if _, err := f.WriteString(
			v.Name + "\t" +
				strconv.Itoa(v.ID) + "\t" +
				strconv.Itoa(v.SID) + "\t" +
				v.SIDTitle + "\t" +
				v.Category + "\t" +
				v.Kingdom + "\n"); err != nil {
			return err
		}
	}
	return nil
}
