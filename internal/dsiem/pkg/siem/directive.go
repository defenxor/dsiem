package siem

import (
	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/dsiem/pkg/rule"

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

	"github.com/jonhoo/drwmutex"
)

const (
	directiveFileGlob = "directives_*.json"
)

type directive struct {
	ID       int                  `json:"id"`
	Name     string               `json:"name"`
	Priority int                  `json:"priority"`
	Kingdom  string               `json:"kingdom"`
	Category string               `json:"category"`
	Rules    []rule.DirectiveRule `json:"rules"`
}

// Directives group directive together
type Directives struct {
	Dirs []directive `json:"directives"`
}

var uCases Directives
var eventChannel chan event.NormalizedEvent

// InitDirectives initialize directive from directive_*.json files in confDir then start
// backlog manager for each directive
func InitDirectives(confDir string, ch <-chan event.NormalizedEvent) error {
	uCases, totalFromFile, err := LoadDirectivesFromFile(confDir, directiveFileGlob)
	if err != nil {
		return err
	}
	total := len(uCases.Dirs)
	if total == 0 {
		return errors.New("Cannot find valid directive from " + confDir)
	}
	log.Info(log.M{Msg: "Successfully Loaded " + strconv.Itoa(total) + "/" + strconv.Itoa(totalFromFile) + " defined directives."})

	var dirchan []chan event.NormalizedEvent
	for i := 0; i < total; i++ {
		dirchan = append(dirchan, make(chan event.NormalizedEvent, 10)) // allow lagging behind up to 10 events
		blogs := backlogs{}
		blogs.DRWMutex = drwmutex.New()
		blogs.id = i
		blogs.bl = make(map[string]*backLog) // have to do it here before the append
		allBacklogs = append(allBacklogs, blogs)
		go allBacklogs[i].manager(uCases.Dirs[i], dirchan[i])
	}

	// copy incoming events to all directive channels
	copier := func() {
		for {
			evt := <-ch
			// running under go routine easily bottleneck under heavy load
			// this however will cause single dirchan to block the loop
			// should investigate to use buffered channel here
			// go func() {
			for i := range dirchan {
				dirchan[i] <- evt
			}
			// }()
		}
	}
	go copier()
	return nil
}

// LoadDirectivesFromFile load directive from namePattern (glob) files in confDir
func LoadDirectivesFromFile(confDir string, namePattern string) (res Directives, totalFromFile int, err error) {
	p := path.Join(confDir, namePattern)
	files, err := filepath.Glob(p)
	if err != nil {
		return res, 0, err
	}
	totalFromFile = 0
	for i := range files {
		var d Directives
		if !fs.FileExist(files[i]) {
			return res, 0, errors.New("Cannot find " + files[i])
		}
		file, err := os.Open(files[i])
		if err != nil {
			return res, 0, err
		}
		defer file.Close()

		byteValue, _ := ioutil.ReadAll(file)
		err = json.Unmarshal(byteValue, &d)
		if err != nil {
			return res, 0, err
		}
		totalFromFile += len(d.Dirs)
		for j := range d.Dirs {
			err = validateDirective(&d.Dirs[j])
			if err != nil {
				log.Warn(log.M{Msg: "Skipping directive ID " +
					strconv.Itoa(d.Dirs[j].ID) +
					" '" + d.Dirs[j].Name + "' due to error: " + err.Error()})
				continue
			}
			res.Dirs = append(res.Dirs, d.Dirs[j])
		}
	}
	if len(res.Dirs) == 0 {
		return res, 0, errors.New("Cannot load any directive from " + path.Join(confDir, namePattern))
	}
	return
}

func validateDirective(d *directive) (err error) {
	for _, v := range uCases.Dirs {
		if v.ID == d.ID {
			return errors.New(strconv.Itoa(d.ID) + " is already used as an ID by other directive")
		}
	}
	if d.Name == "" || d.Kingdom == "" || d.Category == "" {
		return errors.New("Name, Kingdom, and Category cannot be empty")
	}
	if d.Priority < 1 || d.Priority > 5 {
		// return errors.New("Priority must be between 1 - 5")
		log.Warn(log.M{Msg: "Directive " + strconv.Itoa(d.ID) +
			" has wrong priority set (" + strconv.Itoa(d.Priority) + "), configuring it to 1"})
		d.Priority = 1
	}
	if len(d.Rules) == 1 {
		return errors.New(strconv.Itoa(d.ID) + " has only 1 rule and therefore will never expire")
	}

	stages := []int{}
	for j, v := range d.Rules {
		if v.Stage == 0 {
			return errors.New("rule stage should start from 1, cannot use 0")
		}
		for i := range stages {
			if stages[i] == v.Stage {
				return errors.New("duplicate rule stage " + strconv.Itoa(v.Stage) + " found.")
			}
		}
		if v.Stage == 1 {
			if v.Occurrence != 1 {
				// return errors.New("Stage 1 rule occurrence is configured to " + strconv.Itoa(v.Occurrence) + ". It must be set to 1")
				log.Warn(log.M{Msg: "Directive " + strconv.Itoa(d.ID) + " rule " + strconv.Itoa(v.Stage) +
					" has wrong occurrence set (" + strconv.Itoa(v.Occurrence) + "), configuring it to 1"})
				d.Rules[j].Occurrence = 1
			}
		}
		if v.Type != "PluginRule" && v.Type != "TaxonomyRule" {
			return errors.New("Rule Type must be PluginRule or TaxonomyRule")
		}
		if v.Reliability < 1 || v.Reliability > 10 {
			// return errors.New("Reliability must be defined between 1 to 10")
			log.Warn(log.M{Msg: "Directive " + strconv.Itoa(d.ID) + " rule " + strconv.Itoa(v.Stage) +
				" has wrong reliability set (" + strconv.Itoa(v.Reliability) + "), configuring it to 1"})
			d.Rules[j].Reliability = 1
		}
		isFirstStage := v.Stage == 1
		if err := validateFromTo(v.From, isFirstStage); err != nil {
			return err
		}
		if err := validateFromTo(v.To, isFirstStage); err != nil {
			return err
		}
		if err := validatePort(v.PortFrom); err != nil {
			return err
		}
		if err := validatePort(v.PortTo); err != nil {
			return err
		}
		if v.Type == "PluginRule" {
			if v.PluginID < 1 {
				return errors.New("PluginRule requires PluginID to be 1 or higher")
			}
			if len(v.PluginSID) == 0 {
				return errors.New("PluginRule requires PluginSID to be defined")
			}
			for i := range v.PluginSID {
				if v.PluginSID[i] < 1 {
					return errors.New("PluginRule requires PluginSID to be 1 or higher")
				}
			}
		}
		if v.Type == "TaxonomyRule" {
			if len(v.Product) == 0 {
				return errors.New("TaxonomyRule requires Product to be defined")
			}
			if v.Category == "" {
				return errors.New("TaxonomyRule requires Category to be defined")
			}
		}
	}
	return nil
}

func validatePort(s string) error {
	if s == "ANY" {
		return nil
	}
	if _, ok := str.RefToDigit(s); ok {
		return nil
	}
	sSlice := str.CsvToSlice(s)
	for _, v := range sSlice {
		n, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		if n <= 1 || n >= 65535 {
			return errors.New(v + " is not a valid TCP/IP port number")
		}
	}
	return nil
}

func validateFromTo(s string, isFirstRule bool) (err error) {
	if s == "ANY" || s == "HOME_NET" || s == "!HOME_NET" {
		return nil
	}
	if !isFirstRule {
		if _, ok := str.RefToDigit(s); ok {
			return nil
		}
	}
	// covers  r.To == "IP", r.To == "IP1, IP2", r.To == CIDR-netaddr, r.To == "CIDR1, CIDR2"
	// first convert to slice, because netcidr maybe in a form of "cidr1,cidr2..."
	sSlice := str.CsvToSlice(s)
	for i, v := range sSlice {
		if !strings.Contains(v, "/") {
			v = v + "/32"
		}
		if _, _, err := net.ParseCIDR(v); err != nil {
			return errors.New(sSlice[i] + " is not a valid IPv4 address or CIDR")
		}
	}
	return nil
}

func doesEventMatchRule(e event.NormalizedEvent, r rule.DirectiveRule, connID uint64) bool {

	if r.Type == "PluginRule" {
		return pluginRuleCheck(e, r, connID)
	}
	if r.Type == "TaxonomyRule" {
		return taxonomyRuleCheck(e, r, connID)
	}
	return false
}

func taxonomyRuleCheck(e event.NormalizedEvent, r rule.DirectiveRule, connID uint64) (ret bool) {
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

func pluginRuleCheck(e event.NormalizedEvent, r rule.DirectiveRule, connID uint64) (ret bool) {
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

func ipPortCheck(e event.NormalizedEvent, r rule.DirectiveRule, connID uint64) (ret bool) {
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

func copyDirective(dst *directive, src directive, e event.NormalizedEvent) {
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
