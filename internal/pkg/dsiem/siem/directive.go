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
	"fmt"
	"io"
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

var (
	ErrNoDirectiveLoaded = errors.New("no directive loaded from the file")
)

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

		byteValue, _ := io.ReadAll(file)
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
			err = ValidateDirective(&d.Dirs[j], &res)
			if err != nil {
				log.Warn(log.M{Msg: fmt.Sprintf("Skipping directive ID %d '%s' due to error: %s", d.Dirs[j].ID, d.Dirs[j].Name, err.Error())})
				continue
			}
			res.Dirs = append(res.Dirs, d.Dirs[j])
		}
	}
	if len(res.Dirs) == 0 {
		return res, 0, ErrNoDirectiveLoaded
	}

	return
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
