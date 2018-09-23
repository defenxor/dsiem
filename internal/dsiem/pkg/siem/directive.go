package siem

import (
	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"dsiem/internal/shared/pkg/str"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	directiveFileGlob = "directives_*.json"
)

type directiveRule struct {
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
	Events      []string `json:"events,omitempty"`
	Status      string   `json:"status"`
}

type directive struct {
	ID       int             `json:"id"`
	Name     string          `json:"name"`
	Priority int             `json:"priority"`
	Kingdom  string          `json:"kingdom"`
	Category string          `json:"category"`
	Rules    []directiveRule `json:"rules"`
}

type directives struct {
	Directives []directive `json:"directives"`
}

var uCases directives
var eventChannel chan event.NormalizedEvent

// InitDirectives initialize directive from directive_*.json files in confDir
func InitDirectives(confDir string, ch <-chan event.NormalizedEvent) error {
	p := path.Join(confDir, directiveFileGlob)
	files, err := filepath.Glob(p)
	if err != nil {
		return err
	}

	for i := range files {
		var d directives
		if !fs.FileExist(files[i]) {
			return errors.New("Cannot find " + files[i])
		}
		file, err := os.Open(files[i])
		if err != nil {
			return err
		}
		defer file.Close()

		byteValue, _ := ioutil.ReadAll(file)
		err = json.Unmarshal(byteValue, &d)
		if err != nil {
			return err
		}
		for j := range d.Directives {
			uCases.Directives = append(uCases.Directives, d.Directives[j])
		}
	}

	total := len(uCases.Directives)
	if total == 0 {
		return errors.New("cannot find directory to load from " + confDir)
	}
	log.Info(log.M{Msg: "Loaded " + strconv.Itoa(total) + " directives."})

	var dirchan []chan event.NormalizedEvent

	for i := 0; i < total; i++ {
		dirchan = append(dirchan, make(chan event.NormalizedEvent))
		blogs := backlogs{}
		blogs.bl = make(map[string]*backLog) // have to do it here before the append
		allBacklogs = append(allBacklogs, blogs)
		go blogs.manager(&uCases.Directives[i], dirchan[i])

		// copy incoming events to all directive channels
		go func() {
			for {
				evt := <-ch
				for i := range dirchan {
					dirchan[i] <- evt
				}
			}
		}()
	}
	return nil
}

func doesEventMatchRule(e *event.NormalizedEvent, r *directiveRule, connID uint64) bool {
	if r.Type == "PluginRule" {
		return pluginRuleCheck(e, r, connID)
	}
	if r.Type == "TaxonomyRule" {
		return taxonomyRuleCheck(e, r, connID)
	}
	return false
}

func taxonomyRuleCheck(e *event.NormalizedEvent, r *directiveRule, connID uint64) (ret bool) {
	// product is required and category is required

	if r.Category != e.Category {
		return
	}

	prodMatch := false
	for i := range r.Product {
		if r.Product[i] == e.Product {
			prodMatch = true
			break
		}
	}
	if !prodMatch {
		return
	}

	l := len(r.SubCategory)
	if l == 0 {
		return
	}

	scMatch := false
	for i := range r.SubCategory {
		if r.SubCategory[i] == e.SubCategory {
			scMatch = true
			break
		}
	}

	if !scMatch {
		return
	}
	ret = ipPortCheck(e, r, connID)
	return
}

func pluginRuleCheck(e *event.NormalizedEvent, r *directiveRule, connID uint64) (ret bool) {

	if e.PluginID != r.PluginID {
		return
	}

	sidMatch := false
	for i := range r.PluginSID {
		if r.PluginSID[i] == e.PluginSID {
			sidMatch = true
			break
		}
	}
	if !sidMatch {
		return
	}

	ret = ipPortCheck(e, r, connID)
	return
}

func ipPortCheck(e *event.NormalizedEvent, r *directiveRule, connID uint64) (ret bool) {

	eSrcInHomeNet := e.SrcIPInHomeNet()
	if r.From == "HOME_NET" && eSrcInHomeNet == false {
		return
	}
	if r.From == "!HOME_NET" && eSrcInHomeNet == true {
		return
	}
	// covers  r.From == "IP", r.From == "IP1, IP2", r.From == CIDR-netaddr, r.From == "CIDR1, CIDR2"
	if r.From != "HOME_NET" && r.From != "!HOME_NET" && r.From != "ANY" &&
		!str.IsInCSVList(r.From, e.SrcIP) && !isIPinCIDR(e.SrcIP, r.From) {
		return
	}

	eDstInHomeNet := e.DstIPInHomeNet()
	if r.To == "HOME_NET" && eDstInHomeNet == false {
		return
	}
	if r.To == "!HOME_NET" && eDstInHomeNet == true {
		return
	}
	// covers  r.To == "IP", r.To == "IP1, IP2", r.To == CIDR-netaddr, r.To == "CIDR1, CIDR2"
	if r.To != "HOME_NET" && r.To != "!HOME_NET" && r.To != "ANY" &&
		!str.IsInCSVList(r.To, e.DstIP) && !isIPinCIDR(e.DstIP, r.To) {
		return
	}

	if r.PortFrom != "ANY" && !str.IsInCSVList(r.PortFrom, strconv.Itoa(e.SrcPort)) {
		return
	}
	if r.PortTo != "ANY" && !str.IsInCSVList(r.PortTo, strconv.Itoa(e.DstPort)) {
		return
	}

	// SrcIP, DstIP, SrcPort, DstPort all match
	return true
}

func copyDirective(dst *directive, src *directive, e *event.NormalizedEvent) {
	dst.ID = src.ID
	dst.Priority = src.Priority
	dst.Kingdom = src.Kingdom
	dst.Category = src.Category

	// replace SRC_IP and DST_IP with the asset name or IP address
	title := src.Name
	if strings.Contains(title, "SRC_IP") {
		srcHost := asset.GetName(e.SrcIP)
		if srcHost != "" {
			title = strings.Replace(title, "SRC_IP", srcHost, -1)
		} else {
			title = strings.Replace(title, "SRC_IP", e.SrcIP, -1)
		}
	}
	if strings.Contains(title, "DST_IP") {
		dstHost := asset.GetName(e.DstIP)
		if dstHost != "" {
			title = strings.Replace(title, "DST_IP", dstHost, -1)
		} else {
			title = strings.Replace(title, "DST_IP", e.DstIP, -1)
		}
	}
	dst.Name = title

	for i := range src.Rules {
		r := src.Rules[i]
		dst.Rules = append(dst.Rules, r)
	}
}

func isIPinCIDR(ip string, netcidr string) (found bool) {
	// first convert to slice, because netcidr maybe in a form of "cidr1,cidr2..."
	cleaned := strings.Replace(netcidr, ",", " ", -1)
	cidrSlice := strings.Fields(cleaned)

	found = false
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}
	ipB, _, err := net.ParseCIDR(ip)
	if err != nil {
		log.Warn(log.M{Msg: "Unable to parse IP address: " + ip + ". Make sure the plugin is configured correctly!"})
		return
	}

	for _, v := range cidrSlice {
		if !strings.Contains(v, "/") {
			v = v + "/32"
		}
		_, ipnetA, err := net.ParseCIDR(v)
		if err != nil {
			log.Warn(log.M{Msg: "Unable to parse CIDR address: " + v + ". Make sure the directive is configured correctly!"})
			return
		}
		if ipnetA.Contains(ipB) {
			found = true
			break
		}
	}
	return
}
