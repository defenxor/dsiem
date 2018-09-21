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

func startDirective(d directive, c chan event.NormalizedEvent) {
	for {
		// handle incoming event
		evt := <-c
		if !doesEventMatchRule(&evt, &d.Rules[0], 0) {
			continue
		}

		log.Info("directive "+strconv.Itoa(d.ID)+" found matched event.", evt.ConnID)
		go backlogManager(&evt, &d)
	}
}

func doesEventMatchRule(e *event.NormalizedEvent, r *directiveRule, connID uint64) bool {
	if r.Type == "PluginRule" {
		return pluginRuleCheck(e, r, connID)
	}
	if r.Type == "TaxonomyRule" {
		return taxonomyRuleCheck(e, r)
	}
	return false
}

func taxonomyRuleCheck(e *event.NormalizedEvent, r *directiveRule) bool {
	// product is required and category is required

	if r.Category != e.Category {
		return false
	}

	prodMatch := false
	for i := range r.Product {
		if r.Product[i] == e.Product {
			prodMatch = true
			break
		}
	}
	if !prodMatch {
		return false
	}

	l := len(r.SubCategory)
	if l == 0 {
		return true
	}

	scMatch := false
	for i := range r.SubCategory {
		if r.SubCategory[i] == e.SubCategory {
			scMatch = true
			break
		}
	}
	return scMatch
}

func pluginRuleCheck(e *event.NormalizedEvent, r *directiveRule, connID uint64) bool {

	if e.PluginID != r.PluginID {
		return false
	}

	sidMatch := false
	for i := range r.PluginSID {
		if r.PluginSID[i] == e.PluginSID {
			sidMatch = true
			break
		}
	}
	if sidMatch == false {
		return false
	}

	eSrcInHomeNet := e.SrcIPInHomeNet()
	if r.From == "HOME_NET" && eSrcInHomeNet == false {
		return false
	}
	if r.From == "!HOME_NET" && eSrcInHomeNet == true {
		return false
	}
	// covers  r.From == "IP", r.From == "IP1, IP2", r.From == CIDR-netaddr, r.From == "CIDR1, CIDR2"
	if r.From != "HOME_NET" && r.From != "!HOME_NET" && r.From != "ANY" &&
		!str.CaseInsensitiveContains(r.From, e.SrcIP) && !isIPinCIDR(e.SrcIP, r.From, connID) {
		return false
	}

	eDstInHomeNet := e.DstIPInHomeNet()
	if r.To == "HOME_NET" && eDstInHomeNet == false {
		return false
	}
	if r.To == "!HOME_NET" && eDstInHomeNet == true {
		return false
	}
	// covers  r.To == "IP", r.To == "IP1, IP2", r.To == CIDR-netaddr, r.To == "CIDR1, CIDR2"
	if r.To != "HOME_NET" && r.To != "!HOME_NET" && r.To != "ANY" &&
		!str.CaseInsensitiveContains(r.To, e.DstIP) && !isIPinCIDR(e.DstIP, r.To, connID) {
		return false
	}

	if r.PortFrom != "ANY" && !str.CaseInsensitiveContains(r.PortFrom, strconv.Itoa(e.SrcPort)) {
		return false
	}
	if r.PortTo != "ANY" && !str.CaseInsensitiveContains(r.PortTo, strconv.Itoa(e.DstPort)) {
		return false
	}

	// PluginID, PluginSID, SrcIP, DstIP, SrcPort, DstPort all match
	return true
}

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
		return errors.New("cannot find any directive to load from conf dir")
	}
	log.Info("Loaded "+strconv.Itoa(total)+" directives.", 0)

	/*
		for i := range uCases.Directives {
			printDirective(uCases.Directives[i])
		}
	*/

	var dirchan []chan event.NormalizedEvent

	for i := 0; i < total; i++ {
		dirchan = append(dirchan, make(chan event.NormalizedEvent))
		go startDirective(uCases.Directives[i], dirchan[i])

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

func printDirective(d directive) {
	log.Info("Directive ID: "+strconv.Itoa(d.ID)+" Name: "+d.Name, 0)
	for j := 0; j < len(d.Rules); j++ {
		log.Info("- Rule "+strconv.Itoa(d.Rules[j].Stage)+" Name: "+d.Rules[j].Name+
			" From: "+d.Rules[j].From+" To: "+d.Rules[j].To, 0)
	}
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

func isIPinCIDR(ip string, netcidr string, connID uint64) (found bool) {
	// first convert to slice, because netcidr maybe in a form of "cidr1,cidr2..."
	cleaned := strings.Replace(netcidr, ",", " ", -1)
	cidrSlice := strings.Fields(cleaned)

	found = false
	if !strings.Contains(ip, "/") {
		ip = ip + "/32"
	}
	ipB, _, err := net.ParseCIDR(ip)
	if err != nil {
		log.Warn("Unable to parse IP address: "+ip+". Make sure the plugin is configured correctly!", connID)
		return
	}

	for _, v := range cidrSlice {
		if !strings.Contains(v, "/") {
			v = v + "/32"
		}
		_, ipnetA, err := net.ParseCIDR(v)
		if err != nil {
			log.Warn("Unable to parse CIDR address: "+v+". Make sure the directive is configured correctly!", connID)
			return
		}
		if ipnetA.Contains(ipB) {
			found = true
			break
		}
	}
	return
}
