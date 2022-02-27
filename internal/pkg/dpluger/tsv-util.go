package dpluger

import (
	"sort"

	"github.com/defenxor/dsiem/internal/pkg/shared/tsv"
)

// PluginSIDWithCustomData is extension to original PluginSID with added custom-data set.
// it inherits all PluginSID methods.
type PluginSIDWithCustomData struct {
	PluginSID
	CustomDataSet

	lastIndex int
}

func (p PluginSIDWithCustomData) IsEmpty() bool {
	if p.PluginSID.IsEmpty() {
		return true
	}

	if p.CustomDataSet.IsEmpty() {
		return true
	}

	return false
}

func (p *PluginSIDWithCustomData) Next(b tsv.Castable) bool {
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
	case 6:
		p.CustomLabel1 = b.String()
	case 7:
		p.CustomData1 = b.String()
	case 8:
		p.CustomLabel2 = b.String()
	case 9:
		p.CustomData2 = b.String()
	case 10:
		p.CustomLabel3 = b.String()
	case 11:
		p.CustomData3 = b.String()
	default:
		return false
	}

	p.lastIndex++
	return true
}

func (p *PluginSIDWithCustomData) Defaults(in interface{}) {
	if in == nil {
		return
	}

	v, ok := in.(PluginSIDWithCustomData)
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

	if p.CustomLabel1 == "" {
		p.CustomLabel1 = v.CustomLabel1
	}

	if p.CustomData1 == "" {
		p.CustomData1 = v.CustomData1
	}

	if p.CustomLabel2 == "" {
		p.CustomLabel2 = v.CustomLabel2
	}

	if p.CustomData2 == "" {
		p.CustomData2 = v.CustomData2
	}

	if p.CustomLabel3 == "" {
		p.CustomLabel3 = v.CustomLabel3
	}

	if p.CustomData3 == "" {
		p.CustomData3 = v.CustomData3
	}
}

func (c *CustomDataSet) removeIncompleteCustomData() {
	if !isCompletePair(c.CustomData1, c.CustomLabel1) {
		c.CustomData1 = ""
		c.CustomLabel1 = ""
	}

	if !isCompletePair(c.CustomData2, c.CustomLabel2) {
		c.CustomData2 = ""
		c.CustomLabel2 = ""
	}

	if !isCompletePair(c.CustomData3, c.CustomLabel3) {
		c.CustomData3 = ""
		c.CustomLabel3 = ""
	}
}

func (c CustomDataSet) IsEmpty() bool {
	if c.CustomData1 != "" {
		return false
	}

	if c.CustomLabel1 != "" {
		return false
	}

	if c.CustomData2 != "" {
		return false
	}

	if c.CustomLabel2 != "" {
		return false
	}

	if c.CustomData3 != "" {
		return false
	}

	if c.CustomLabel3 != "" {
		return false
	}

	return true
}

// PluginSIDWithCustomDataGroup is mapping of a CustomDataSet to set of Plugin SID, used to map
// unique custom data set to list of plugin-sid along with its custom-data.
type PluginSIDWithCustomDataGroup struct {
	CustomData CustomDataSet
	Plugins    PluginSIDSet
}

type CustomDataSet struct {
	CustomLabel1 string `json:"custom_label1,omitempty" tsv:"custom_label1" csv:"custom_label1"`
	CustomData1  string `json:"custom_data1,omitempty" tsv:"custom_data1" csv:"custom_data1"`
	CustomLabel2 string `json:"custom_label2,omitempty" tsv:"custom_label2" csv:"custom_label2"`
	CustomData2  string `json:"custom_data2,omitempty" tsv:"custom_data2" csv:"custom_data2"`
	CustomLabel3 string `json:"custom_label3,omitempty" tsv:"custom_label3" csv:"custom_label3"`
	CustomData3  string `json:"custom_data3,omitempty" tsv:"custom_data3" csv:"custom_data3"`
}

type PluginSIDSet []PluginSIDWithCustomData

func (p PluginSIDSet) SID() []int {
	m := make(map[int]struct{})
	for _, ref := range p {
		m[ref.SID] = struct{}{}
	}

	sid := make([]int, 0, len(m))
	for k := range m {
		sid = append(sid, k)
	}

	return sid
}

func (p PluginSIDSet) FirstSID() int {
	sid := p.SID()
	if len(sid) == 0 {
		return 0
	}

	if len(sid) == 1 {
		return sid[0]
	}

	sort.Ints(sid)
	return sid[0]
}

type ByFirstPluginSID []PluginSIDWithCustomDataGroup

func (g ByFirstPluginSID) Len() int { return len(g) }
func (g ByFirstPluginSID) Less(i, j int) bool {
	return g[i].Plugins.FirstSID() < g[j].Plugins.FirstSID()
}
func (g ByFirstPluginSID) Swap(i, j int) { g[i], g[j] = g[j], g[i] }

func GroupByCustomData(ref PluginSIDSet) []PluginSIDWithCustomDataGroup {
	m := make(map[CustomDataSet]PluginSIDSet)

	for _, r := range ref {
		r.removeIncompleteCustomData()
		m[r.CustomDataSet] = append(m[r.CustomDataSet], r)
	}

	group := make([]PluginSIDWithCustomDataGroup, 0, len(m))
	for k, v := range m {
		group = append(group, PluginSIDWithCustomDataGroup{
			CustomData: k,
			Plugins:    v,
		})
	}

	sort.Sort(ByFirstPluginSID(group))

	return group
}

func isCompletePair(s1, s2 string) bool {
	return s1 != "" && s2 != ""
}
