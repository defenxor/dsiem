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
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/str"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/alarm"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"

	"github.com/jonhoo/drwmutex"
	"github.com/spf13/viper"

	"github.com/elastic/apm-agent-go"
)

var (
	bLogFileMutex = sync.Mutex{}
	bLogFile      string
)

var txl = sync.Mutex{}

type backLog struct {
	drwmutex.DRWMutex
	ID           string    `json:"backlog_id"`
	StatusTime   int64     `json:"status_time"`
	Risk         int       `json:"risk"`
	CurrentStage int       `json:"current_stage"`
	HighestStage int       `json:"highest_stage"`
	Directive    directive `json:"directive"`
	SrcIPs       []string  `json:"src_ips"`
	DstIPs       []string  `json:"dst_ips"`
	LastEvent    event.NormalizedEvent
	chData       chan event.NormalizedEvent
	chDone       chan struct{}
	chFound      chan bool
	deleted      bool      // flag for deletion process
	bLogs        *backlogs // pointer to parent, for locking delete operation?
}

type siemAlarmEvents struct {
	ID    string `json:"alarm_id"`
	Stage int    `json:"stage"`
	Event string `json:"event_id"`
}

func (b *backLog) start(initialEvent event.NormalizedEvent) {
	b.processMatchedEvent(initialEvent, 0)
	go b.newEventProcessor()
	go b.expirationChecker()
}

func (b *backLog) newEventProcessor() {
	maxDelay := viper.GetInt("maxDelay")
	for {
		evt, ok := <-b.chData
		if !ok {
			l := b.RLock()
			b.debug("worker chData closed, exiting", 0)
			l.Unlock()
			return
		}
		l := b.RLock()
		b.debug("backlog incoming event", evt.ConnID)
		cs := b.CurrentStage
		if cs <= 1 {
			l.Unlock()
			continue
		}
		// should check for currentStage rule match with event
		// heuristic, we know stage starts at 1 but rules start at 0
		idx := cs - 1
		currRule := b.Directive.Rules[idx]
		currSDiff := &b.Directive.StickyDiffs[idx]
		if !rule.DoesEventMatch(evt, currRule, currSDiff, evt.ConnID) {
			b.debug("backlog doeseventmatch false", evt.ConnID)
			b.chFound <- false
			l.Unlock()
			continue
		}
		b.chFound <- true // answer quickly

		// validate date conversion
		ts, err := str.TimeStampToUnix(evt.Timestamp)
		if err != nil {
			b.warn("cannot parse event timestamp, discarding it", evt.ConnID)
			l.Unlock()
			continue
		}
		// discard out of order event
		if !b.isTimeInOrder(idx, ts) {
			b.warn("event timestamp out of order, discarding it", evt.ConnID)
			l.Unlock()
			continue
		}

		if b.isUnderPressure(evt.RcvdTime, int64(maxDelay)) {
			b.warn("backlog is under pressure", evt.ConnID)
			select {
			case b.bLogs.bpCh <- true:
			default:
			}
		}

		b.debug("processing incoming event", evt.ConnID)
		l.Unlock()
		// this should be under go routine, but chFound need sync access (for first match, backlog creation)
		if cs == 1 {
			b.processMatchedEvent(evt, idx)
		} else {
			runtime.Gosched() // let the main go routine work
			go b.processMatchedEvent(evt, idx)
		}
		// b.info("setting found to true", evt.ConnID)
	}
}

func (b *backLog) expirationChecker() {
	ticker := time.NewTicker(time.Second * 10)
	for {
		<-ticker.C
		select {
		case <-b.chDone:
			l := b.RLock()
			b.debug("backlog tick exiting, chDone.", 0)
			l.Unlock()
			ticker.Stop()
			return
		default:
		}
		l := b.RLock()
		if !b.isExpired() {
			l.Unlock()
			continue
		}
		l.Unlock()
		ticker.Stop() // prevent next signal, we're exiting the go routine
		b.info("backlog expired, deleting it", 0)
		b.setRuleStatus("timeout", 0)
		b.updateAlarm(0, false, nil)
		b.delete()
		return
	}

}

func (b backLog) isUnderPressure(rcvd int64, maxDelay int64) bool {
	if maxDelay == 0 {
		return false
	}
	now := time.Now().Unix()
	return now-rcvd > maxDelay
}

// no modification so use value receiver
func (b backLog) isTimeInOrder(idx int, ts int64) bool {
	// exit if in first stage
	if idx == 0 {
		return true
	}
	prevStageTime := b.Directive.Rules[idx-1].EndTime
	ts = ts + 5 // allow up to 5 seconds diff to compensate for concurrent write
	if prevStageTime > ts {
		return false
	}
	return true
}

func (b backLog) isExpired() bool {
	now := time.Now().Unix()
	cs := b.CurrentStage
	idx := cs - 1
	start := b.Directive.Rules[idx].StartTime
	timeout := b.Directive.Rules[idx].Timeout
	maxTime := start + timeout
	if maxTime >= now {
		return false
	}
	return true
}

func (b *backLog) setRuleEndTime(e event.NormalizedEvent) {
	b.Lock()
	s := b.CurrentStage
	idx := s - 1
	ts, _ := str.TimeStampToUnix(e.Timestamp)
	b.Directive.Rules[idx].EndTime = ts
	b.Unlock()
}

func (b *backLog) processMatchedEvent(e event.NormalizedEvent, idx int) {

	var tx *elasticapm.Transaction
	var l sync.Locker

	if apm.Enabled() {
		txl.Lock()
		tx = elasticapm.DefaultTracer.StartTransaction("Directive Event Processing", "SIEM")
		tx.Context.SetCustom("event_id", e.EventID)
		l = b.RLock()
		tx.Context.SetCustom("backlog_id", b.ID)
		tx.Context.SetCustom("directive_id", b.Directive.ID)
		tx.Context.SetCustom("backlog_stage", b.CurrentStage)
		l.Unlock()
		txl.Unlock()
		// defer tx.End()
		defer elasticapm.DefaultTracer.Recover(tx)
	}

	l = b.RLock()
	b.debug("Incoming event with idx: "+strconv.Itoa(idx), e.ConnID)
	// concurrent write may make events count overflow, so dont append current stage unless needed
	if !b.isStageReachMaxEvtCount(idx) {
		l.Unlock()
		b.appendandWriteEvent(e, idx, tx)
		// exit early if the newly added event hasnt caused events_count == occurrence
		// for the current stage
		l = b.RLock()
		if !b.isStageReachMaxEvtCount(idx) {
			l.Unlock()
			return
		}
		l.Unlock()
	} else {
		l.Unlock()
	}
	// the new event has caused events_count == occurrence
	b.setRuleStatus("finished", e.ConnID)
	b.setRuleEndTime(e)
	b.updateAlarm(e.ConnID, true, tx)

	// if it causes the last stage to reach events_count == occurrence, delete it
	if b.isLastStage() {
		b.info("reached max stage and occurrence, deleting.", e.ConnID)
		b.delete()
		if apm.Enabled() {
			txl.Lock()
			tx.Result = "Backlog removed (max reached)"
			tx.End()
			txl.Unlock()
		}
		return
	}

	// reach max occurrence, but not in last stage. Increase stage.
	b.increaseStage(e)
	// set rule startTime for the new stage
	b.setRuleStartTime(e)

	// stageIncreased, update alarm to publish new stage startTime
	b.updateAlarm(e.ConnID, true, tx)

	// b.setStatus("active", e.ConnID, tx)
	if apm.Enabled() {
		txl.Lock()
		l = b.RLock()
		tx.Context.SetCustom("backlog_stage", b.CurrentStage)
		l.Unlock()
		tx.Result = "Stage increased"
		tx.End()
		txl.Unlock()
	}

	// recalc risk, the new stage will have a different reliability
	riskChanged := b.calcRisk(e.ConnID)
	if riskChanged {
		// this LastEvent is used to get ports by alarm
		b.setLastEvent(e)
		b.updateAlarm(e.ConnID, true, tx)
	}
}

func (b backLog) info(msg string, connID uint64) {
	log.Info(log.M{Msg: msg, BId: b.ID, CId: connID})
}

func (b backLog) warn(msg string, connID uint64) {
	log.Warn(log.M{Msg: msg, BId: b.ID, CId: connID})
}

func (b backLog) debug(msg string, connID uint64) {
	log.Debug(log.M{Msg: msg, BId: b.ID, CId: connID})
}

func (b *backLog) setLastEvent(e event.NormalizedEvent) {
	b.Lock()
	b.LastEvent = e
	b.Unlock()
}

func (b backLog) updateAlarm(connID uint64, checkIntelVuln bool, tx *elasticapm.Transaction) {
	if b.Risk == 0 {
		return
	}
	l := b.RLock()
	tmp := make([]rule.DirectiveRule, len(b.Directive.Rules))
	copy(tmp, b.Directive.Rules)
	l.Unlock()
	go alarm.Upsert(b.ID, b.Directive.Name, b.Directive.Kingdom,
		b.Directive.Category, b.SrcIPs, b.DstIPs, b.LastEvent.SrcPort,
		b.LastEvent.DstPort, b.Risk, b.StatusTime, tmp,
		connID, checkIntelVuln, tx)
}

func (b *backLog) setRuleStatus(status string, connID uint64) {
	b.Lock()
	s := b.CurrentStage
	idx := s - 1
	b.Directive.Rules[idx].Status = status
	b.Unlock()
}

func (b *backLog) appendandWriteEvent(e event.NormalizedEvent, idx int, tx *elasticapm.Transaction) {
	b.Lock()
	b.Directive.Rules[idx].Events = append(b.Directive.Rules[idx].Events, e.EventID)
	b.SrcIPs = str.AppendUniq(b.SrcIPs, e.SrcIP)
	b.DstIPs = str.AppendUniq(b.DstIPs, e.DstIP)
	b.Unlock()
	b.setStatusTime()
	// dont wait for I/O
	l := b.RLock()
	go func(b backLog) {
		err := b.updateElasticsearch(e)
		if err != nil {
			b.warn("failed to update Elasticsearch! "+err.Error(), e.ConnID)
			if apm.Enabled() {
				txl.Lock()
				e := elasticapm.DefaultTracer.NewError(err)
				e.Transaction = tx
				e.Send()
				tx.Result = "Failed to append and write event"
				tx.End()
				txl.Unlock()
			}
		} else {
			if apm.Enabled() {
				txl.Lock()
				tx.Result = "Event appended to backlog"
				tx.End()
				txl.Unlock()
			}
		}
	}(*b)
	l.Unlock()
	return
}

func (b backLog) isLastStage() (ret bool) {
	ret = b.CurrentStage == b.HighestStage
	return
}

func (b backLog) isStageReachMaxEvtCount(idx int) (reachMaxEvtCount bool) {
	// still need lock because Rules is a slice
	l := b.RLock()
	currRule := b.Directive.Rules[idx]
	nEvents := len(b.Directive.Rules[idx].Events)
	if nEvents >= currRule.Occurrence {
		reachMaxEvtCount = true
	}
	l.Unlock()
	return
}

func (b *backLog) increaseStage(e event.NormalizedEvent) {
	b.Lock()
	n := int32(b.CurrentStage)
	b.CurrentStage = int(atomic.AddInt32(&n, 1))
	if b.CurrentStage > b.HighestStage {
		b.CurrentStage = b.HighestStage
	}
	b.Unlock()
	b.info("stage increased", e.ConnID)
}

func (b *backLog) setRuleStartTime(e event.NormalizedEvent) {
	b.Lock()
	idx := b.CurrentStage - 1
	t, _ := str.TimeStampToUnix(e.Timestamp)
	b.Directive.Rules[idx].StartTime = t
	b.StatusTime = time.Now().Unix()
	b.Unlock()
}

func (b *backLog) calcRisk(connID uint64) (riskChanged bool) {
	l := b.RLock()
	s := b.CurrentStage
	idx := s - 1
	value := 0
	for i := range b.SrcIPs {
		v := asset.GetValue(b.SrcIPs[i])
		if v > value {
			value = v
		}
	}
	for i := range b.DstIPs {
		v := asset.GetValue(b.DstIPs[i])
		if v > value {
			value = v
		}
	}

	pRisk := b.Risk

	reliability := b.Directive.Rules[idx].Reliability
	priority := b.Directive.Priority
	risk := priority * reliability * value / 25
	//	fmt.Println("directive:", b.Directive.ID, "stage", b.CurrentStage,
	//		"SrcIPs:", b.SrcIPs, "DstIPs:", b.DstIPs, "asset value:", value, "rel:",
	//		reliability, "prio:", priority, "risk:", risk)

	l.Unlock()

	if risk != pRisk {
		b.Lock()
		b.Risk = risk
		b.Unlock()
		b.info("risk changed.", connID)
		riskChanged = true
	}
	return
}

// need to use ptr receiver for bLogs.delete
func (b *backLog) delete() {
	l := b.RLock()
	defer l.Unlock()
	if b.deleted {
		return
	}
	b.debug("delete sending signal to bLogs", 0)
	b.bLogs.delete(b)
}

func (b *backLog) setStatusTime() {
	b.Lock()
	b.StatusTime = time.Now().Unix()
	b.Unlock()
}

func (b backLog) updateElasticsearch(e event.NormalizedEvent) error {
	log.Debug(log.M{Msg: "backlog updating Elasticsearch", DId: b.Directive.ID, BId: b.ID, CId: e.ConnID})
	b.StatusTime = time.Now().Unix()
	bLogFileMutex.Lock()
	f, err := os.OpenFile(bLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	bLogFileMutex.Unlock()
	if err != nil {
		return err
	}
	defer f.Close()
	v := siemAlarmEvents{b.ID, b.CurrentStage, e.EventID}
	vJSON, err := json.Marshal(v)
	if err != nil {
		fmt.Println(v)
		return err
	}
	f.SetDeadline(time.Now().Add(60 * time.Second))
	_, err = f.WriteString(string(vJSON) + "\n")
	return err
}
