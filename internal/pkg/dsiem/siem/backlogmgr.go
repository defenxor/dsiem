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
	"strconv"
	"sync/atomic"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/alarm"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"
	"github.com/defenxor/dsiem/internal/pkg/shared/idgen"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"

	"sync"
	"time"

	"github.com/jonhoo/drwmutex"
)

type backlogs struct {
	drwmutex.DRWMutex
	id   int
	bl   map[string]*backLog
	bpCh chan bool
}

var (
	// protects allBacklogs
	allBacklogsMu sync.RWMutex

	allBacklogs []backlogs
	fWriter     fs.FileWriter
)

const (
	maxFileQueueLength = 10000
)

// InitBackLogManager initialize backlog and ticker
func InitBackLogManager(logFile string, bpChan chan<- bool, holdDuration int) (err error) {

	err = fWriter.Init(logFile, maxFileQueueLength)

	go func() { bpChan <- false }() // set initial state
	go initBpTicker(bpChan, holdDuration)
	return
}

func initBpTicker(bpChan chan<- bool, holdDuration int) {
	prevState := false
	sl := sync.Mutex{}

	sWait := time.Duration(holdDuration)
	timer := time.NewTimer(time.Second * sWait)
	go func() {
		for {
			<-timer.C
			// send false (reset signal) only if prev state is true
			sl.Lock()
			timer.Reset(time.Second * sWait)
			if prevState {
				select {
				case bpChan <- false:
					prevState = false
					log.Debug(log.M{Msg: "Overload=false signal sent from backend"})
				default:
				}
			}
			sl.Unlock()
		}
	}()

	// get a merged channel consisting of true signal from all
	// backlogs

	out := merge()
	for range out {
		// set the timer again
		// send true only if prev state is false
		sl.Lock()
		timer.Reset(time.Second * sWait)
		if !prevState {
			select {
			case bpChan <- true:
				log.Debug(log.M{Msg: "Overload=true signal sent from backend"})
				prevState = true

			default:
			}
		}
		sl.Unlock()
	}
}

func merge() <-chan bool {
	allBacklogsMu.RLock()
	defer allBacklogsMu.RUnlock()

	out := make(chan bool)
	for _, v := range allBacklogs {
		go func(ch chan bool) {
			for v := range ch {
				// v will only contain true
				out <- v
			}
		}(v.bpCh)
	}
	return out
}

// CountBackLogs returns the number of active backlogs
func CountBackLogs() (sum int, activeDirectives int, ttlDirectives int) {

	ttlDirectives = len(allBacklogs)
	for i := range allBacklogs {
		l := allBacklogs[i].RLock()
		nBlogs := len(allBacklogs[i].bl)
		sum += nBlogs
		if nBlogs > 0 {
			activeDirectives++
		}
		l.Unlock()
	}
	return
}

func (blogs *backlogs) manager(d Directive, ch <-chan event.NormalizedEvent, minAlarmLifetime int) {

	sidPairs, taxoPairs := rule.GetQuickCheckPairs(d.Rules)

	isPluginRule := false
	isTaxoRule := false
	if len(sidPairs) > 0 {
		isPluginRule = true
	}
	if len(taxoPairs) > 0 {
		isTaxoRule = true
	}

mainLoop:
	for {
		evt := <-ch

		var tx *apm.Transaction
		if apm.Enabled() {
			th := apm.TraceHeader{
				Traceparent: evt.TraceParent,
				TraceState:  evt.TraceState,
			}
			tx = apm.StartTransaction("Directive Evaluation", "Event Correlation", nil, &th)
			tx.SetCustom("event_id", evt.EventID)
			tx.SetCustom("directive_id", strconv.Itoa(d.ID))
			// make this parent of downstream transactions
			thisTh := tx.GetTraceContext()
			evt.TraceParent = thisTh.Traceparent
			evt.TraceState = thisTh.TraceState
		}

		if isPluginRule {
			if rule.QuickCheckPluginRule(sidPairs, &evt) == false {
				if apm.Enabled() {
					tx.Result("Event doesn't match directive plugin rules")
					tx.End()
				}
				continue mainLoop
			}
		} else if isTaxoRule {
			if rule.QuickCheckTaxoRule(taxoPairs, &evt) == false {
				if apm.Enabled() {
					tx.Result("Event doesn't match directive taxo rules")
					tx.End()
				}
				continue mainLoop
			}
		}

		// found := false
		// zero means false
		var found uint32
		l := blogs.RLock() // to prevent concurrent r/w with delete()

		wg := &sync.WaitGroup{}

		for k := range blogs.bl {
			wg.Add(1)
			go func(k string) {
				// this first select is required, see #2 on https://go101.org/article/channel-closing.html
				select {
				// exit early if done, this should be the case while backlog in waiting for deletion mode
				case <-blogs.bl[k].chDone:
					wg.Done()
					return
				default:
				}
				select {
				case <-blogs.bl[k].chDone: // exit early if done
					wg.Done()
					return
				case blogs.bl[k].chData <- evt: // fwd to backlog
					select {
					case <-blogs.bl[k].chDone: // exit early if done
						wg.Done()
						return
					// wait for the result
					case f := <-blogs.bl[k].chFound:
						if f {
							// found = true
							atomic.AddUint32(&found, 1)
						}
					}
				}
				wg.Done()
			}(k)
		}
		wg.Wait()
		l.Unlock()

		if found > 0 {
			if apm.Enabled() && tx != nil {
				tx.Result("Event consumed by backlog")
				tx.End()
			}
			continue mainLoop
		}
		// now for new backlog
		// stickydiff cannot be used on 1st rule, so we pass nil
		if !rule.DoesEventMatch(evt, d.Rules[0], nil, evt.ConnID) {
			if apm.Enabled() && tx != nil {
				tx.Result("Event doesn't match rule")
				tx.End()
			}
			continue mainLoop // back to chan loop
		}

		// compare the event against all backlogs event ID to prevent duplicates
		// due to concurrency
		blogs.Lock()
		for _, v := range blogs.bl {
			for _, y := range v.Directive.Rules {
				for _, j := range y.Events {
					if j == evt.EventID {
						log.Info(log.M{Msg: "skipping backlog creation for event " + j +
							", it's already used in backlog " + v.ID})
						if apm.Enabled() && tx != nil {
							tx.Result("Event already used in backlog" + v.ID)
							tx.End()
						}
						blogs.Unlock()
						continue mainLoop // back to chan loop
					}
				}
			}
		}
		blogs.Unlock()

		// lock from here also to prevent duplicates
		blogs.Lock()
		b, err := createNewBackLog(d, evt)
		if err != nil {
			log.Warn(log.M{Msg: "Fail to create new backlog", DId: d.ID, CId: evt.ConnID})
			if apm.Enabled() && tx != nil {
				tx.Result("Fail to create new backlog")
				tx.End()
			}
			blogs.Unlock()
			continue mainLoop
		}
		blogs.bl[b.ID] = b
		blogs.bl[b.ID].bLogs = blogs
		blogs.Unlock()
		if apm.Enabled() && tx != nil {
			tx.Result("Event created a new backlog")
			tx.End()
		}
		blogs.bl[b.ID].start(evt, minAlarmLifetime)
	}
}

func (blogs *backlogs) delete(b *backLog) {
	go func() {
		// first prevent another blogs.delete to enter here
		blogs.Lock()
		b.Lock()
		if b.deleted {
			// already in the closing process
			log.Debug(log.M{Msg: "backlog is already in the process of being deleted"})
			b.Unlock()
			blogs.Unlock()
			return
		}
		log.Info(log.M{Msg: "backlog manager removing backlog in < 10s", DId: b.Directive.ID, BId: b.ID})
		log.Debug(log.M{Msg: "backlog manager setting status to deleted", DId: b.Directive.ID, BId: b.ID})
		b.deleted = true
		// prevent further event write by manager, and stop backlog ticker
		close(b.chDone)
		b.Unlock()
		blogs.Unlock()
		time.Sleep(3 * time.Second)
		// signal backlog worker to exit
		log.Debug(log.M{Msg: "backlog manager closing data channel", DId: b.Directive.ID, BId: b.ID})
		close(b.chData)
		time.Sleep(3 * time.Second)
		log.Debug(log.M{Msg: "backlog manager deleting backlog from map", DId: b.Directive.ID, BId: b.ID})
		blogs.Lock()
		blogs.bl[b.ID].Lock()
		blogs.bl[b.ID].bLogs = nil
		blogs.bl[b.ID].Unlock()
		delete(blogs.bl, b.ID)
		blogs.Unlock()
		ch := alarm.RemovalChannel()
		ch <- b.ID
	}()
}

func createNewBackLog(d Directive, e event.NormalizedEvent) (bp *backLog, err error) {
	bid, err := idgen.GenerateID()
	if err != nil {
		return
	}
	log.Info(log.M{Msg: "Creating new backlog", DId: d.ID, CId: e.ConnID})
	b := backLog{}
	b.ID = bid
	b.Directive = Directive{}

	copyDirective(&b.Directive, d, e)
	initBackLogRules(&b.Directive, e)
	t, err := time.Parse(time.RFC3339, e.Timestamp)
	if err != nil {
		return
	}
	b.Directive.Rules[0].StartTime = t.Unix()
	b.Directive.Rules[0].RcvdTime = e.RcvdTime
	b.chData = make(chan event.NormalizedEvent)
	b.chFound = make(chan bool)
	b.chDone = make(chan struct{}, 1)

	b.CurrentStage = 1
	b.HighestStage = len(d.Rules)
	bp = &b

	return
}

func initBackLogRules(d *Directive, e event.NormalizedEvent) {

	for i := range d.Rules {
		if i == 0 {
			// if flag is active, replace ANY and HOME_NET on the first rule with specific addresses from event
			if d.AllRulesAlwaysActive {
				ref := d.Rules[i].From
				if ref == "ANY" || ref == "HOME_NET" || ref == "!HOME_NET" {
					d.Rules[i].From = e.SrcIP
				}
				ref = d.Rules[i].To
				if ref == "ANY" || ref == "HOME_NET" || ref == "!HOME_NET" {
					d.Rules[i].To = e.DstIP
				}
			}
			// the first rule cannot use reference to other
			continue
		}

		// for the rest, refer to the referenced stage if its not ANY or HOME_NET or !HOME_NET
		// if the reference is ANY || HOME_NET || !HOME_NET then refer to event if its in the format of
		// :ref
		r := d.Rules[i].From
		if v, ok := str.RefToDigit(r); ok {
			vmin1 := v - 1
			ref := d.Rules[vmin1].From
			if ref != "ANY" && ref != "HOME_NET" && ref != "!HOME_NET" {
				d.Rules[i].From = ref
			} else {
				d.Rules[i].From = e.SrcIP
			}
		}

		r = d.Rules[i].To
		if v, ok := str.RefToDigit(r); ok {
			vmin1 := v - 1
			ref := d.Rules[vmin1].To
			if ref != "ANY" && ref != "HOME_NET" && ref != "!HOME_NET" {
				d.Rules[i].To = ref
			} else {
				d.Rules[i].To = e.DstIP
			}
		}

		r = d.Rules[i].PortFrom
		if v, ok := str.RefToDigit(r); ok {
			vmin1 := v - 1
			ref := d.Rules[vmin1].PortFrom
			if ref != "ANY" {
				d.Rules[i].PortFrom = ref
			} else {
				d.Rules[i].PortFrom = strconv.Itoa(e.SrcPort)
			}
		}

		r = d.Rules[i].PortTo
		if v, ok := str.RefToDigit(r); ok {
			vmin1 := v - 1
			ref := d.Rules[vmin1].PortTo
			if ref != "ANY" {
				d.Rules[i].PortTo = ref
			} else {
				d.Rules[i].PortTo = strconv.Itoa(e.DstPort)
			}
		}

		// add reference for custom datas.
		r = d.Rules[i].CustomData1
		if v, ok := str.RefToDigit(r); ok {
			vmin1 := v - 1
			ref := d.Rules[vmin1].CustomData1
			if ref != "ANY" {
				d.Rules[i].CustomData1 = ref
			} else {
				d.Rules[i].CustomData1 = e.CustomData1
			}
		}

		r = d.Rules[i].CustomData2
		if v, ok := str.RefToDigit(r); ok {
			vmin1 := v - 1
			ref := d.Rules[vmin1].CustomData2
			if ref != "ANY" {
				d.Rules[i].CustomData2 = ref
			} else {
				d.Rules[i].CustomData2 = e.CustomData2
			}
		}

		r = d.Rules[i].CustomData3
		if v, ok := str.RefToDigit(r); ok {
			vmin1 := v - 1
			ref := d.Rules[vmin1].CustomData3
			if ref != "ANY" {
				d.Rules[i].CustomData3 = ref
			} else {
				d.Rules[i].CustomData3 = e.CustomData3
			}
		}
	}
}
