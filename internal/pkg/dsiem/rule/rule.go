package rule

import "sync"

// DirectiveRule defines the struct for directive rules, this is read-only
// struct.
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
}

// StickyDiffData hold the previous data for stickydiff rule
// This is mutable, so its separated from DirectiveRule
type StickyDiffData struct {
	sync.RWMutex
	SDiffString []string
	SDiffInt    []int
}
