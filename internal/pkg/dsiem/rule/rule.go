package rule

// DirectiveRule defines the struct for directive rules
type DirectiveRule struct {
	Name        string   `json:"name"`
	Stage       int      `json:"stage"`
	PluginID    int      `json:"plugin_id"`
	PluginSID   []int    `json:"plugin_sid"`
	Product     []string `json:"product"`
	Category    string   `json:"category"`
	SubCategory []string `json:"subcategory"`
	Occurrence  int      `json:"occurrence"`
	From        string   `json:"from"`
	To          string   `json:"to"`
	Type        string   `json:"type"`
	PortFrom    string   `json:"port_from"`
	PortTo      string   `json:"port_to"`
	Protocol    string   `json:"protocol"`
	Reliability int      `json:"reliability"`
	Timeout     int64    `json:"timeout"`
	StartTime   int64    `json:"start_time"`
	EndTime     int64    `json:"end_time"`
	Status      string   `json:"status"`
	Events      []string `json:"events,omitempty"`
	StickyDiff  string   `json:"sticky_different,omitempty"`
	SDiffString []string `json:"-"`
	SDiffInt    []int    `json:"-"`
}

// IsStringStickyDiff check if v fulfill stickydiff condition
func (r *DirectiveRule) IsStringStickyDiff(v string) bool {
	for i := range r.SDiffString {
		if r.SDiffString[i] == v {
			return false
		}
	}
	// add it to the coll
	r.SDiffString = append(r.SDiffString, v)
	return true
}

// IsIntStickyDiff check if v fulfill stickydiff condition
func (r *DirectiveRule) IsIntStickyDiff(v int) (match bool) {
	for i := range r.SDiffInt {
		if r.SDiffInt[i] == v {
			return false
		}
	}
	// add it to the coll
	r.SDiffInt = append(r.SDiffInt, v)
	return true
}
