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
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/defenxor/dsiem/internal/pkg/shared/tsv"
)

const (
	TSVFileSuffix = "_plugin-sids.tsv"
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

func (p PluginSID) IsEmpty() bool {
	if p.Name != "" {
		return false
	}

	if p.ID != 0 {
		return false
	}

	if p.SID != 0 {
		return false
	}

	if p.SIDTitle != "" {
		return false
	}

	if p.Category != "" {
		return false
	}

	if p.Kingdom != "" {
		return false
	}

	return true
}

// Defaults is implementation of tsv.Castable
func (p *PluginSID) Defaults(in interface{}) {
	if in == nil {
		return
	}

	v, ok := in.(PluginSID)
	if !ok {
		return
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

// tsvRef is reference to TSV file containing list of Signature IDs.
type tsvRef struct {
	SIDs     map[int]PluginSID
	filename string
}

func (c *tsvRef) setFilename(pluginName string, base string) {
	c.filename = path.Join(base, fmt.Sprintf("%s%s", pluginName, TSVFileSuffix))
}

func (c *tsvRef) addPlugin(ref PluginSID) {
	c.SIDs[ref.SID] = ref
}

func (c *tsvRef) initWithReader(pluginName, base string, r io.Reader) {
	c.SIDs = make(map[int]PluginSID)
	c.setFilename(pluginName, base)

	c.initSIDList(r)
}

func (c *tsvRef) initWithConfig(pluginName string, configFile string) {
	c.SIDs = make(map[int]PluginSID)
	c.setFilename(pluginName, path.Dir(configFile))
	f, err := os.OpenFile(c.filename, os.O_RDONLY, 0600)
	if err != nil {
		return
	}

	defer f.Close()
	c.initSIDList(f)
}

func (c *tsvRef) initSIDList(ref io.Reader) {
	parser := tsv.NewParser(ref)
	for {
		var ref PluginSID
		ok := parser.Read(&ref, nil)
		if !ok {
			break
		}

		if ref.IsEmpty() {
			continue
		}

		c.addPlugin(ref)
	}
}

func (c tsvRef) hasSID(sid int) bool {
	_, ok := c.SIDs[sid]
	return ok
}

func (c tsvRef) hasTitle(title string) (int, bool) {
	for k, v := range c.SIDs {
		if v.SIDTitle == title {
			return k, true
		}
	}

	return 0, false
}

func (c tsvRef) count() int {
	return len(c.SIDs)
}

// upsert store the plugin to the TSV reference if the plugin with the same Title and same Signature ID doesn't exist yet.
// returns true if the plugin is added to the end of current plugin list, means that caller should use next (incremented)
// Signature ID (SID) for the next upsert.
func (c *tsvRef) upsert(pluginName string, pluginID int,
	pluginSID *int, category, sidTitle string) (lastEntry bool) {

	// replace character " in title and category, if any.
	sidTitle = strings.ReplaceAll(sidTitle, "\"", "'")
	category = strings.ReplaceAll(category, "\"", "'")

	// First check the title, exit if already exist.
	if sid, ok := c.hasTitle(sidTitle); ok && sid != 0 {
		// tell caller to increase SID number if there's a stored plugin with same SID and title.
		return sid == *pluginSID
	}

	// here plugin with the same title doesn't exist yet in internal plugin list,
	// so we add it, first find available SID number.
	for {
		if !c.hasSID(*pluginSID) {
			break
		}

		*pluginSID++
	}

	// then, we add the plugin under the available SID number.
	c.addPlugin(PluginSID{
		Name:     pluginName,
		SID:      *pluginSID,
		ID:       pluginID,
		SIDTitle: sidTitle,
		Category: category,
	})

	// and tell the caller that we should increase the SID number.
	return true
}

func (c tsvRef) save() error {
	f, err := os.OpenFile(c.filename, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.WriteString("plugin\tid\tsid\ttitle\tcategory\tkingdom\n"); err != nil {
		return err
	}
	// use slice to get a sorted keys, ikr
	var keys []int
	for k := range c.SIDs {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		v := c.SIDs[k]
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
