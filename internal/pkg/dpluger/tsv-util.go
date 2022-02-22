package dpluger

import "sort"

type PluginSIDWithCustomData struct {
	PluginSID
	CustomDataSet
}

func (c *PluginSIDWithCustomData) removeIncompleteCustomData() {
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
