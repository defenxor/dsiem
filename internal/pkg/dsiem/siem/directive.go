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

package siem

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/queue"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"

	"github.com/jonhoo/drwmutex"
)

const (
	directiveFileGlob = "directives_*.json"
)

// Directive represents a SIEM use case that has several correlation rules
type Directive struct {
	ID                   int                   `json:"id"`
	Name                 string                `json:"name"`
	Priority             int                   `json:"priority"`
	Disabled             bool                  `json:"disabled"`
	AllRulesAlwaysActive bool                  `json:"all_rules_always_active"`
	Kingdom              string                `json:"kingdom"`
	Category             string                `json:"category"`
	Rules                []rule.DirectiveRule  `json:"rules"`
	StickyDiffs          []rule.StickyDiffData `json:"-"`
}

// Directives group directive together
type Directives struct {
	Dirs []Directive `json:"directives"`
}

var uCases Directives

// InitDirectives initialize directive from directive_*.json files in confDir then start
// backlog manager for each directive
func InitDirectives(confDir string, ch <-chan event.NormalizedEvent, minAlarmLifetime, maxEPS, maxEventQueueLength int) error {

	var dirchan []event.Channel
	uCases, totalFromFile, err := LoadDirectivesFromFile(confDir, directiveFileGlob, false)

	if err != nil {
		return err
	}

	total := len(uCases.Dirs)
	log.Info(log.M{Msg: "Successfully Loaded " + strconv.Itoa(total) + "/" + strconv.Itoa(totalFromFile) + " defined directives."})

	for i := 0; i < total; i++ {
		dirchan = append(dirchan, event.Channel{
			Ch:    make(chan event.NormalizedEvent),
			DirID: uCases.Dirs[i].ID,
		})
		blogs := backlogs{}
		blogs.DRWMutex = drwmutex.New()
		blogs.id = i
		blogs.bpCh = make(chan bool)
		blogs.bl = make(map[string]*backLog) // have to do it here before the append
		l := blogs.RLock()
		allBacklogs = append(allBacklogs, blogs)
		l.Unlock()
		go allBacklogs[i].manager(uCases.Dirs[i], dirchan[i].Ch, minAlarmLifetime)
	}

	eq := queue.EventQueue{}
	eq.Init(dirchan, maxEventQueueLength, maxEPS)

	copier := func() {
		for {
			evt := <-ch
			if isWhitelisted(evt.SrcIP) {
				continue
			}
			if apm.Enabled() {
				if evt.RcvdTime == 0 {
					log.Warn(log.M{Msg: "Cannot parse event received time, skipping event", CId: evt.ConnID})
					continue
				}
				tStart := time.Unix(0, evt.RcvdTime)
				th := apm.TraceHeader{
					Traceparent: evt.TraceParent,
					TraceState:  evt.TraceState,
				}
				tx := apm.StartTransaction("Frontend to Backend", "Network", &tStart, &th)
				tx.SetCustom("event_id", evt.EventID)
				tx.Result("Received from frontend")
				tx.End()
			}
			eq.Enqueue(evt)
		}
	}

	go eq.Dequeue()
	go copier()
	f := eq.GetReporter()
	go f(30 * time.Second)
	return nil
}

func isWhitelisted(ip string) (ret bool) {
	whitelisted, err := asset.IsWhiteListed(ip)
	if err != nil {
		log.Warn(log.M{Msg: "Fail to check if source IP " + ip + " is whitelisted"})
		return
	}
	if whitelisted {
		log.Debug(log.M{Msg: "Skipping event, Source IP " + ip + " is whitelisted."})
		ret = true
	}
	return
}

// LoadDirectivesFromFile load directive from namePattern (glob) files in confDir
func LoadDirectivesFromFile(confDir string, namePattern string, includeDisabled bool) (res Directives, totalFromFile int, err error) {
	p := path.Join(confDir, namePattern)
	files, err := filepath.Glob(p)
	if err != nil {
		return res, 0, err
	}
	totalFromFile = 0
	for i := range files {
		var d Directives
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
			if d.Dirs[j].Disabled && !includeDisabled {
				log.Warn(log.M{Msg: "Skipping disabled directive ID " +
					strconv.Itoa(d.Dirs[j].ID)})
				continue
			}
			err = validateDirective(&d.Dirs[j], &res)
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

func validateDirective(d *Directive, res *Directives) (err error) {
	for _, v := range res.Dirs {
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
	if len(d.Rules) <= 1 {
		return errors.New(strconv.Itoa(d.ID) + " has no rule therefore has no effect, or only 1 rule and therefore will never expire")
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
		// reliability maybe 0 for the first rule!
		if v.Reliability < 0 {
			log.Warn(log.M{Msg: "Directive " + strconv.Itoa(d.ID) + " rule " + strconv.Itoa(v.Stage) +
				" has wrong reliability set (" + strconv.Itoa(v.Reliability) + "), configuring it to 0"})
			d.Rules[j].Reliability = 0
		}
		if v.Reliability > 10 {
			log.Warn(log.M{Msg: "Directive " + strconv.Itoa(d.ID) + " rule " + strconv.Itoa(v.Stage) +
				" has wrong reliability set (" + strconv.Itoa(v.Reliability) + "), configuring it to 10"})
			d.Rules[j].Reliability = 10
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
		stages = append(stages, v.Stage)
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

	if s == "" {
		return errors.New("From/To cannot be empty")
	}

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

func copyDirective(dst *Directive, src Directive, e event.NormalizedEvent) {
	dst.ID = src.ID
	dst.Priority = src.Priority
	dst.Kingdom = src.Kingdom
	dst.Category = src.Category
	dst.AllRulesAlwaysActive = src.AllRulesAlwaysActive

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

	l := len(src.Rules)
	dst.Rules = make([]rule.DirectiveRule, l)
	copy(dst.Rules, src.Rules)
	dst.StickyDiffs = make([]rule.StickyDiffData, l)
	for i := range dst.StickyDiffs {
		dst.StickyDiffs[i] = rule.StickyDiffData{}
	}
}
