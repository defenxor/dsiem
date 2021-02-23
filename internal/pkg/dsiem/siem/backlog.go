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
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/shared/str"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/alarm"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/asset"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/rule"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/spf13/viper"
)

type backLog struct {
	// deadlock.RWMutex
	sync.RWMutex
	ID               string            `json:"backlog_id"`
	StatusTime       int64             `json:"status_time"`
	Risk             int               `json:"risk"`
	CurrentStage     int               `json:"current_stage"`
	HighestStage     int               `json:"highest_stage"`
	Directive        Directive         `json:"directive"`
	SrcIPs           []string          `json:"src_ips"`
	DstIPs           []string          `json:"dst_ips"`
	CustomData       []rule.CustomData `json:"custom_data"`
	LastEvent        event.NormalizedEvent
	chData           chan event.NormalizedEvent
	chDone           chan struct{}
	chFound          chan bool
	deleted          bool      // flag for deletion process
	bLogs            *backlogs // pointer to parent, for locking delete operation?
	minAlarmLifetime int64
	logger           backlogLogger
}

type siemAlarmEvents struct {
	ID    string `json:"alarm_id"`
	Stage int    `json:"stage"`
	Event string `json:"event_id"`
}

func (b *backLog) start(initialEvent event.NormalizedEvent, minAlarmLifetime int) {
	// convert to int64 and to seconds
	b.logger = newBacklogLogger(b.ID)
	b.minAlarmLifetime = int64(minAlarmLifetime * 60)
	b.processMatchedEvent(initialEvent, 0)
	go b.newEventProcessor()
	go b.expirationChecker()
}

func (b *backLog) newEventProcessor() {
	maxDelay := viper.GetInt("maxDelay")
	for {
		evt, ok := <-b.chData
		if !ok {
			b.logger.debug("worker chData closed, exiting", evt.ConnID)
			return
		}
		b.RLock()
		cs := b.CurrentStage
		idx := cs - 1
		b.logger.debug("backlog incoming event on idx "+strconv.Itoa(idx)+": "+evt.Title, evt.ConnID)
		currRule := b.Directive.Rules[idx]
		currSDiff := &b.Directive.StickyDiffs[idx]
		lenSDiffString := len(b.Directive.StickyDiffs[idx].SDiffString)
		lenSDiffInt := len(b.Directive.StickyDiffs[idx].SDiffInt)
		checkAllRulesFlag := b.Directive.AllRulesAlwaysActive
		b.RUnlock()
		if !rule.DoesEventMatch(evt, currRule, currSDiff, evt.ConnID) {
			// if flag is set, check if event match previous stage
			if checkAllRulesFlag && idx != 0 {
				prevFound := false
				for i := 0; i < idx; i++ {
					b.RLock()
					prevRule := b.Directive.Rules[i]
					prevSDiff := &b.Directive.StickyDiffs[i]
					b.RUnlock()
					if rule.DoesEventMatch(evt, prevRule, prevSDiff, evt.ConnID) {
						b.logger.debug("backlog previous rule "+strconv.Itoa(i)+" consumes matching event", evt.ConnID)
						// just add the event to the stage, no need to process other steps in processMatchedEvent
						b.appendandWriteEvent(evt, i, nil)
						// also update alarm to sync any changes to customData
						b.updateAlarm(evt.ConnID, false, nil)
						prevFound = true
						break
					}
				}
				b.chFound <- prevFound
			} else {
				b.logger.debug("backlog doeseventmatch false", evt.ConnID)
				b.chFound <- false
			}
			continue // main for loop
		}
		b.chFound <- true // answer quickly

		// if stickydiff is set, there must be added member to sDiffString
		// or sDiffInt, otherwise skip further processing
		if currRule.StickyDiff != "" {
			nString := len(b.Directive.StickyDiffs[idx].SDiffString)
			nInt := len(b.Directive.StickyDiffs[idx].SDiffInt)
			if nString == lenSDiffString && nInt == lenSDiffInt {
				b.logger.debug("backlog can't find new unique value in stickydiff field "+currRule.StickyDiff, evt.ConnID)
				continue
			}
		}

		// validate date conversion
		ts, err := str.TimeStampToUnix(evt.Timestamp)
		if err != nil {
			b.logger.warn("cannot parse event timestamp, discarding it", evt.ConnID)
			continue
		}
		// discard out of order event
		if !b.isTimeInOrder(idx, ts) {
			b.logger.warn("event timestamp out of order, discarding it", evt.ConnID)
			continue
		}

		if b.isUnderPressure(evt.RcvdTime, int64(maxDelay)) {
			b.logger.warn("backlog is under pressure", evt.ConnID)
			select {
			case b.bLogs.bpCh <- true:
			default:
			}
		}

		b.logger.debug("processing incoming event for idx "+strconv.Itoa(idx), evt.ConnID)
		runtime.Gosched() // let the main go routine work
		go b.processMatchedEvent(evt, idx)
	}
}

func (b *backLog) expirationChecker() {
	ticker := time.NewTicker(time.Second * 10)
	for {
		<-ticker.C
		select {
		case <-b.chDone:
			b.logger.debug("backlog tick exiting, chDone.", 0)
			ticker.Stop()
			return
		default:
		}
		if !b.isExpired() {
			continue
		}
		ticker.Stop() // prevent next signal, we're exiting the go routine
		b.logger.info("backlog expired, deleting it", 0)
		b.setRuleStatus("timeout", 0)
		b.updateAlarm(0, false, nil)
		b.delete()
		return
	}

}

func (b *backLog) isUnderPressure(rcvd int64, maxDelay int64) (ret bool) {
	if maxDelay != 0 {
		// rcvd in nanosec
		rcvdSec := rcvd / int64(time.Second)
		now := time.Now().Unix()
		ret = now-rcvdSec > maxDelay
	}
	return
}

// no modification so use value receiver
func (b *backLog) isTimeInOrder(idx int, ts int64) bool {
	b.RLock()
	prevStageTime := b.Directive.Rules[idx-1].EndTime
	b.RUnlock()
	ts = ts + 5 // allow up to 5 seconds diff to compensate for concurrent write
	return prevStageTime < ts
}

func (b *backLog) isExpired() bool {
	limit := time.Now().Unix()
	b.RLock()
	limit = limit - b.minAlarmLifetime
	cs := b.CurrentStage
	idx := cs - 1
	start := b.Directive.Rules[idx].StartTime
	timeout := b.Directive.Rules[idx].Timeout
	b.RUnlock()
	maxTime := start + timeout
	return maxTime < limit
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

	var tx *apm.Transaction
	if apm.Enabled() {
		th := apm.TraceHeader{
			Traceparent: e.TraceParent,
			TraceState:  e.TraceState,
		}
		tx = apm.StartTransaction("Event Processing", "Event Correlation", nil, &th)
		tx.SetCustom("event_id", e.EventID)
		b.RLock()
		tx.SetCustom("backlog_id", b.ID)
		tx.SetCustom("directive_id", strconv.Itoa(b.Directive.ID))
		tx.SetCustom("backlog_stage", strconv.Itoa(b.CurrentStage))
		b.RUnlock()
		defer tx.End()
	}

	b.logger.debug("Incoming event with idx: "+strconv.Itoa(idx), e.ConnID)

	// first append the event
	b.appendandWriteEvent(e, idx, tx)

	// exit early if the newly added event hasnt caused events_count == occurrence
	// for the current stage
	if !b.isStageReachMaxEvtCount(idx) {
		return
	}

	// the new event has caused events_count == occurrence
	b.setRuleStatus("finished", e.ConnID)
	b.setRuleEndTime(e)

	_ = b.calcRisk(e.ConnID)

	// if it causes the last stage to reach events_count == occurrence, delete it
	if b.isLastStage() {
		b.updateAlarm(e.ConnID, true, tx)
		b.logger.info("reached max stage and occurrence, deleting.", e.ConnID)
		b.delete()
		if apm.Enabled() {
			tx.Result("Backlog removed (max reached)")
		}
		return
	}

	// reach max occurrence, but not in last stage.

	// increase stage.
	b.increaseStage(e)

	// set rule startTime for the new stage
	b.setRuleStartTime(e)

	// stageIncreased, update alarm to publish new stage startTime
	b.updateAlarm(e.ConnID, true, tx)

	if apm.Enabled() {
		b.Lock()
		tx.SetCustom("backlog_stage", strconv.Itoa(b.CurrentStage))
		tx.Result("Stage increased")
		b.Unlock()
	}
}

func (b *backLog) updateAlarm(connID uint64, checkIntelVuln bool, tx *apm.Transaction) {
	b.RLock()
	vRules := make([]rule.DirectiveRule, len(b.Directive.Rules))
	copy(vRules, b.Directive.Rules)
	vCustomData := make([]rule.CustomData, len(b.CustomData))
	copy(vCustomData, b.CustomData)
	vSrcIPs := make([]string, len(b.SrcIPs))
	copy(vSrcIPs, b.SrcIPs)
	vDstIPs := make([]string, len(b.DstIPs))
	copy(vDstIPs, b.DstIPs)

	go alarm.Upsert(b.ID,
		b.Directive.Name,
		b.Directive.Kingdom,
		b.Directive.Category,
		vSrcIPs,
		vDstIPs,
		vCustomData,
		b.LastEvent.SrcPort,
		b.LastEvent.DstPort,
		b.Risk,
		b.StatusTime,
		vRules,
		connID,
		checkIntelVuln,
		tx)
	b.RUnlock()
}

func (b *backLog) setRuleStatus(status string, connID uint64) {
	b.Lock()
	s := b.CurrentStage
	idx := s - 1
	b.Directive.Rules[idx].Status = status
	b.Unlock()
}

func (b *backLog) appendandWriteEvent(e event.NormalizedEvent, idx int, tx *apm.Transaction) {
	b.Lock()
	b.Directive.Rules[idx].Events = append(b.Directive.Rules[idx].Events, e.EventID)
	b.SrcIPs = str.AppendUniq(b.SrcIPs, e.SrcIP)
	b.DstIPs = str.AppendUniq(b.DstIPs, e.DstIP)
	b.CustomData = rule.AppendUniqCustomData(b.CustomData, e.CustomLabel1, e.CustomData1)
	b.CustomData = rule.AppendUniqCustomData(b.CustomData, e.CustomLabel2, e.CustomData2)
	b.CustomData = rule.AppendUniqCustomData(b.CustomData, e.CustomLabel3, e.CustomData3)

	// remove 0.0.0.0 if any, but only when it's not the only entry
	b.SrcIPs = str.RemoveElementUnlessEmpty(b.SrcIPs, "0.0.0.0")
	b.DstIPs = str.RemoveElementUnlessEmpty(b.DstIPs, "0.0.0.0")

	b.LastEvent = e
	b.Unlock()
	b.setStatusTime()

	err := b.updateElasticsearch(e, idx)
	if err != nil {
		b.logger.warn("failed to update Elasticsearch! "+err.Error(), e.ConnID)
		if apm.Enabled() && tx != nil {
			tx.SetError(err)
			tx.Result("Failed to append and write event")
			tx.End()
		}
	} else {
		if apm.Enabled() && tx != nil {
			tx.Result("Event appended to backlog")
			tx.End()
		}
	}
}

func (b *backLog) isLastStage() (ret bool) {
	b.RLock()
	ret = b.CurrentStage == b.HighestStage
	b.RUnlock()
	return
}

func (b *backLog) isStageReachMaxEvtCount(idx int) (reachMaxEvtCount bool) {
	// still need lock because Rules is a slice
	b.RLock()
	currRule := b.Directive.Rules[idx]
	nEvents := len(b.Directive.Rules[idx].Events)
	if nEvents >= currRule.Occurrence {
		reachMaxEvtCount = true
	}
	b.RUnlock()
	return
}

func (b *backLog) increaseStage(e event.NormalizedEvent) {
	b.Lock()
	increased := false
	if b.CurrentStage < b.HighestStage {
		n := int32(b.CurrentStage)
		b.CurrentStage = int(atomic.AddInt32(&n, 1))
		increased = true
	}
	b.Unlock()
	if increased {
		b.logger.info("stage increased", e.ConnID)
	}
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
	b.RLock()
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

	b.RUnlock()
	if risk != pRisk {
		b.Lock()
		b.Risk = risk
		b.Unlock()
		b.logger.info("risk changed from "+strconv.Itoa(pRisk)+" to "+strconv.Itoa(risk), connID)
		riskChanged = true
	}
	return
}

// need to use ptr receiver for bLogs.delete
func (b *backLog) delete() {
	b.RLock()
	if b.deleted {
		b.RUnlock()
		return
	}
	b.RUnlock()
	b.logger.debug("delete sending signal to bLogs", 0)
	b.bLogs.delete(b)
}

func (b *backLog) setStatusTime() {
	b.Lock()
	b.StatusTime = time.Now().Unix()
	b.Unlock()
}

func (b *backLog) updateElasticsearch(e event.NormalizedEvent, idx int) (err error) {
	b.logger.debug("backlog updating Elasticsearch", e.ConnID)
	b.Lock()
	b.StatusTime = time.Now().Unix()
	stage := idx + 1
	v := siemAlarmEvents{b.ID, stage, e.EventID}
	b.Unlock()

	var vJSON []byte
	vJSON, err = json.Marshal(v)
	if err == nil {
		err = fWriter.EnqueueWrite(string(vJSON))
	}
	return
}
