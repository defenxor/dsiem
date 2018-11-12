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

	"github.com/defenxor/dsiem/internal/pkg/dsiem/alarm"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/idgen"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"

	"github.com/elastic/apm-agent-go"

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

var allBacklogs []backlogs

// InitBackLogManager initialize backlog and ticker
func InitBackLogManager(logFile string, bpChan chan<- bool, holdDuration int) (err error) {
	// bLogFile is defined in backlog.go
	bLogFile = logFile

	go func() { bpChan <- false }() // set initial state
	go initBpTicker(bpChan, holdDuration)
	return
}

func initBpTicker(bpChan chan<- bool, holdDuration int) {
	prevState := false
	sl := sync.RWMutex{}

	sWait := time.Duration(holdDuration)
	timer := time.NewTimer(time.Second * sWait)
	go func() {
		for {
			<-timer.C
			// send false (reset signal) only if prev state is true
			timer.Reset(time.Second * sWait)
			sl.RLock()
			if prevState == true {
				sl.RUnlock()
				bpChan <- false
				sl.Lock()
				prevState = false
				sl.Unlock()
			} else {
				sl.RUnlock()
			}
		}
	}()
	// get a merged channel consisting of true signal from all
	// backlogs
	out := merge()
	for range out {
		<-out
		// set the timer again
		timer.Reset(time.Second * sWait)
		// send true only if prev state is false
		sl.RLock()
		if prevState == false {
			sl.RUnlock()
			bpChan <- true
			sl.Lock()
			prevState = true
			sl.Unlock()
		} else {
			sl.RUnlock()
		}
	}
}

func merge() <-chan bool {
	out := make(chan bool)
	for i := range allBacklogs {
		go func(i int) {
			for v := range allBacklogs[i].bpCh {
				// v will only contain true
				out <- v
			}
		}(i)
	}
	return out
}

func (blogs *backlogs) manager(d directive, ch <-chan event.NormalizedEvent) {

	for {
		evt := <-ch

		var tx *elasticapm.Transaction
		if apm.Enabled() {
			if evt.RcvdTime == 0 {
				log.Warn(log.M{Msg: "Cannot parse event received time, skipping event", CId: evt.ConnID})
				continue
			}
			tStart := time.Unix(evt.RcvdTime, 0)
			opts := elasticapm.TransactionOptions{TraceContext: elasticapm.TraceContext{}, Start: tStart}
			tx = elasticapm.DefaultTracer.StartTransactionOptions("Frontend to Backend", "SIEM", opts)
			tx.Context.SetCustom("event_id", evt.EventID)
			tx.Context.SetCustom("directive_id", d.ID)
		}

		found := false
		l := blogs.RLock() // to prevent concurrent r/w with delete()
		// TODO:
		// maybe check event against all rules here, if non match then continue
		// this will avoid checking against all backlogs which could be in 1000s compared to
		// # of rules which in the 10s
		wg := &sync.WaitGroup{}

		for k := range blogs.bl {
			wg.Add(1)
			go func(k string) {
				// go try-receive pattern
				select {
				case <-blogs.bl[k].chDone: // exit early if done, this should be the case while backlog in waiting for deletion mode
					wg.Done()
					return
					// continue
				default:
				}

				select {
				case <-blogs.bl[k].chDone: // exit early if done
					wg.Done()
					return
					// continue
				case blogs.bl[k].chData <- evt: // fwd to backlog
					select {
					case <-blogs.bl[k].chDone: // exit early if done
						wg.Done()
						return
						// continue
					// wait for the result
					case f := <-blogs.bl[k].chFound:
						if f {
							found = true
						}
					}
				}
				wg.Done()
			}(k)
		}
		wg.Wait()
		l.Unlock()

		if found {
			if apm.Enabled() {
				tx.Result = "Event consumed by backlog"
				tx.End()
			}
			continue
		}
		// now for new backlog
		// stickydiff cannot be used on 1st rule, so we pass nil
		if !rule.DoesEventMatch(evt, d.Rules[0], nil, evt.ConnID) {
			if apm.Enabled() {
				tx.Result = "Event doesnt match rule"
				tx.End()
			}
			continue // back to chan loop
		}
		b, err := createNewBackLog(d, evt)
		if err != nil {
			log.Warn(log.M{Msg: "Fail to create new backlog", DId: d.ID, CId: evt.ConnID})
			if apm.Enabled() {
				tx.Result = "Fail to create new backlog"
				tx.End()
			}
			continue
		}
		blogs.Lock()
		blogs.bl[b.ID] = b
		blogs.bl[b.ID].DRWMutex = drwmutex.New()
		blogs.bl[b.ID].bLogs = blogs
		blogs.Unlock()
		blogs.bl[b.ID].start(evt)
	}
}

func (blogs *backlogs) delete(b *backLog) {
	log.Info(log.M{Msg: "backlog manager removing backlog in 20s", DId: b.Directive.ID, BId: b.ID})
	go func() {
		// first prevent another blogs.delete to enter here
		blogs.Lock() // to protect bl.Lock??
		b.Lock()
		if b.deleted {
			// already in the closing process
			b.Unlock()
			blogs.Unlock()
			return
		}
		log.Debug(log.M{Msg: "backlog manager setting status to deleted", DId: b.Directive.ID, BId: b.ID})
		b.deleted = true
		b.Unlock()
		blogs.Unlock()
		// prevent further event write by manager, and stop backlog ticker
		close(b.chDone)
		time.Sleep(10 * time.Second)
		// signal backlog worker to exit
		log.Debug(log.M{Msg: "backlog manager closing data channel", DId: b.Directive.ID, BId: b.ID})
		close(b.chData)
		time.Sleep(10 * time.Second)
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

func createNewBackLog(d directive, e event.NormalizedEvent) (bp *backLog, err error) {
	bid, err := idgen.GenerateID()
	if err != nil {
		return
	}
	log.Info(log.M{Msg: "Creating new backlog", DId: d.ID, CId: e.ConnID})
	b := backLog{}
	b.ID = bid
	b.Directive = directive{}

	copyDirective(&b.Directive, d, e)
	initBackLogRules(&b.Directive, e)
	t, err := time.Parse(time.RFC3339, e.Timestamp)
	if err != nil {
		return
	}
	b.Directive.Rules[0].StartTime = t.Unix()
	b.Directive.Rules[0].RcvdTime = e.RcvdTime
	// b.chData = make(chan event.NormalizedEvent)
	b.chData = make(chan event.NormalizedEvent)
	b.chFound = make(chan bool)
	b.chDone = make(chan struct{}, 1)

	b.CurrentStage = 1
	b.HighestStage = len(d.Rules)
	bp = &b

	return
}

func initBackLogRules(d *directive, e event.NormalizedEvent) {
	for i := range d.Rules {
		// the first rule cannot use reference to other
		if i == 0 {
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
	}
}
