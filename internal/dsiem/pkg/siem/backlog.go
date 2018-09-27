package siem

import (
	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	log "dsiem/internal/shared/pkg/logger"
	"dsiem/internal/shared/pkg/str"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/viper"

	"github.com/elastic/apm-agent-go"
)

var bLogFile string

type backLog struct {
	sync.RWMutex
	ID           string    `json:"backlog_id"`
	StatusTime   int64     `json:"status_time"`
	Risk         int       `json:"risk"`
	CurrentStage int       `json:"current_stage"`
	HighestStage int       `json:"highest_stage"`
	Directive    directive `json:"directive"`
	SrcIPs       []string  `json:"src_ips"`
	DstIPs       []string  `json:"dst_ips"`
	LastEvent    event.NormalizedEvent
	chData       chan *event.NormalizedEvent
	chDone       chan struct{}
	chFound      chan bool
	bLogs        *backlogs // pointer to parent, for locking delete operation?
}

type siemAlarmEvents struct {
	ID    string `json:"alarm_id"`
	Stage int    `json:"stage"`
	Event string `json:"event_id"`
}

func (b *backLog) worker(initialEvent *event.NormalizedEvent) {

	debug := viper.GetBool("debug")
	// first, process the initial event
	b.processMatchedEvent(initialEvent, 0)
	// b.info("after processmatchedevent", initialEvent.ConnID)

	go func() {
		for {
			evt, ok := <-b.chData
			if !ok {
				b.debug("worker chData closed, exiting", 0)
				return
			}
			// b.info("backlog incoming event", evt.ConnID)
			cs := b.CurrentStage
			if cs <= 1 {
				// b.info("backlog cs <= 1", evt.ConnID)
				continue
			}
			// should check for currentStage rule match with event
			// heuristic, we know stage starts at 1 but rules start at 0
			idx := cs - 1
			currRule := b.Directive.Rules[idx]
			if !doesEventMatchRule(evt, &currRule, evt.ConnID) {
				// b.info("backlog doeseventmatch false", evt.ConnID)
				b.chFound <- false
				continue
			}
			b.chFound <- true // answer quickly
			// this should be under go routine, but chFound need sync access
			b.processMatchedEvent(evt, idx)
			// b.info("setting found to true", evt.ConnID)
		}
	}()

	go func() {
		// create own ticker here
		ticker := time.NewTicker(time.Second * 10)
		for {
			<-ticker.C
			b.debug("backlog tick started.", 0)
			select {
			case <-b.chDone:
				b.debug("backlog tick exiting, chDone.", 0)
				return
			default:
			}
			if debug {
				b.dumpCurrentRule(false)
			}
			if !b.isExpired() {
				continue
			}
			ticker.Stop() // prevent next signal, we're exiting the go routine
			b.info("backlog expired, deleting it", 0)
			tx := elasticapm.DefaultTracer.StartTransaction("Directive Event Processing", "SIEM")
			b.RLock()
			tx.Context.SetCustom("backlog_id", b.ID)
			tx.Context.SetCustom("directive_id", b.Directive.ID)
			tx.Context.SetCustom("backlog_stage", b.CurrentStage)
			b.RUnlock()
			b.setStatus("timeout", 0, tx)
			b.delete()
		}
	}()
	b.debug("exiting worker, leaving routine behind", initialEvent.ConnID)

}

func (b *backLog) dumpCurrentRule(listEvent bool) {
	fmt.Println("Directive ID:", b.Directive.ID, "backlog ID: ", b.ID)
	for i := range b.Directive.Rules {
		if listEvent {
			fmt.Println("Rule:", i, ":", b.Directive.Rules[i].Events)
		} else {
			fmt.Println("Rule:", i, "length:", len(b.Directive.Rules[i].Events))
		}
	}
}

func (b *backLog) setStatus(status string, connID uint64, tx *elasticapm.Transaction) {
	// enforce flow here, cannot go back to active after timeout/finished
	b.RLock()
	s := b.CurrentStage
	idx := s - 1
	if b.Directive.Rules[idx].Status == "timeout" || b.Directive.Rules[idx].Status == "finished" {
		b.RUnlock()
		return
	}
	allowed := []string{"timeout", "finished"}
	if b.Directive.Rules[idx].Status == "inactive" {
		allowed = append(allowed, "active")
	}
	b.RUnlock()
	for i := range allowed {
		if allowed[i] == status {
			b.Lock()
			b.Directive.Rules[idx].Status = status
			b.Unlock()
			b.RLock()
			upsertAlarmFromBackLog(*b, connID, tx)
			b.RUnlock()
			break
		}
	}
}

func (b *backLog) isExpired() bool {
	b.RLock()
	defer b.RUnlock()
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

func (b *backLog) ensureStatusAndStartTime(idx int, connID uint64, tx *elasticapm.Transaction) {
	// this reinsert status and startDate for the currentStage rule if the first attempt failed
	updateFlag := false
	b.Lock()
	if b.Directive.Rules[idx].StartTime == 0 {
		b.Directive.Rules[idx].StartTime = time.Now().Unix()
		updateFlag = true
	}
	s := b.Directive.Rules[idx].Status
	b.Unlock()

	if s == "inactive" {
		b.setStatus("active", connID, tx)
		updateFlag = true
	}

	if updateFlag {
		b.RLock()
		upsertAlarmFromBackLog(*b, connID, tx)
		b.RUnlock()
	}
}

func (b *backLog) processMatchedEvent(e *event.NormalizedEvent, idx int) {

	tx := elasticapm.DefaultTracer.StartTransaction("Directive Event Processing", "SIEM")
	tx.Context.SetCustom("event_id", e.EventID)
	b.RLock()
	tx.Context.SetCustom("backlog_id", b.ID)
	tx.Context.SetCustom("directive_id", b.Directive.ID)
	tx.Context.SetCustom("backlog_stage", b.CurrentStage)
	b.RUnlock()
	defer tx.End()
	defer elasticapm.DefaultTracer.Recover(tx)

	b.debug("Incoming event with idx: "+strconv.Itoa(idx), e.ConnID)
	// concurrent write may make events count overflow, so dont append current stage unless needed
	if !b.isStageReachMaxEvtCount() {
		b.appendandWriteEvent(e, idx, tx)
		// exit early if the newly added event hasnt caused events_count == occurrence
		// for the current stage
		if !b.isStageReachMaxEvtCount() {
			b.ensureStatusAndStartTime(idx, e.ConnID, tx)
			return
		}
	}
	b.info("setting status to finish", e.ConnID)

	// the new event has caused events_count == occurrence
	b.setStatus("finished", e.ConnID, tx)

	// if it causes the last stage to reach events_count == occurrence, delete it
	if b.isLastStage() {
		b.info("reached max stage and occurrence, deleting.", e.ConnID)
		b.delete()
		tx.Result = "Backlog removed (max reached)"
		return
	}

	// reach max occurrence, but not in last stage. Increase stage.
	b.increaseStage(e.ConnID)
	b.setStatus("active", e.ConnID, tx)
	b.RLock()
	tx.Context.SetCustom("backlog_stage", b.CurrentStage) // race b
	b.RUnlock()
	tx.Result = "Stage increased"

	// recalc risk, the new stage will have a different reliability
	riskChanged := b.calcRisk(e.ConnID)
	if riskChanged {
		// this LastEvent is used to get ports by alarm
		b.setLastEvent(e)
		b.updateAlarm(e.ConnID, tx)
	}
}

func (b *backLog) info(msg string, connID uint64) {
	b.RLock()
	defer b.RUnlock()

	log.Info(log.M{Msg: msg, BId: b.ID, CId: connID})
}

func (b *backLog) debug(msg string, connID uint64) {
	b.RLock()
	log.Debug(log.M{Msg: msg, BId: b.ID, CId: connID})
	b.RUnlock()
}

func (b *backLog) setLastEvent(e *event.NormalizedEvent) {
	b.Lock()
	b.LastEvent = *e
	b.Unlock()
}

func (b *backLog) updateAlarm(connID uint64, tx *elasticapm.Transaction) {
	b.RLock()
	upsertAlarmFromBackLog(*b, connID, tx)
	b.RUnlock()
}

func (b *backLog) appendandWriteEvent(e *event.NormalizedEvent, idx int, tx *elasticapm.Transaction) {
	b.Directive.Rules[idx].Events = append(b.Directive.Rules[idx].Events, e.EventID)
	b.SrcIPs = str.AppendUniq(b.SrcIPs, e.SrcIP)
	b.DstIPs = str.AppendUniq(b.DstIPs, e.DstIP)
	if err := b.updateElasticsearch(e); err != nil {
		b.RLock()
		log.Warn(log.M{Msg: "failed to update Elasticsearch! " + err.Error(), BId: b.ID, CId: e.ConnID})
		b.RUnlock()
		e := elasticapm.DefaultTracer.NewError(err)
		e.Transaction = tx
		e.Send()
		tx.Result = "Failed to append and write event"
	} else {
		tx.Result = "Event appended to backlog"
	}
	return
}

func (b *backLog) isLastStage() (ret bool) {
	b.RLock()
	ret = b.CurrentStage == b.HighestStage
	b.RUnlock()
	return
}

func (b *backLog) isStageReachMaxEvtCount() (reachMaxEvtCount bool) {
	//b.RLock()
	//defer b.RUnlock()
	s := b.CurrentStage
	idx := s - 1
	currRule := b.Directive.Rules[idx]
	nEvents := len(b.Directive.Rules[idx].Events)
	if nEvents >= currRule.Occurrence {
		reachMaxEvtCount = true
	}
	return
}

func (b *backLog) increaseStage(connID uint64) {

	b.Lock()
	n := int32(b.CurrentStage)
	b.CurrentStage = int(atomic.AddInt32(&n, 1)) // race b
	if b.CurrentStage > b.HighestStage {
		b.CurrentStage = b.HighestStage
	}
	idx := b.CurrentStage - 1
	b.Directive.Rules[idx].StartTime = time.Now().Unix()
	b.StatusTime = b.Directive.Rules[idx].StartTime
	b.Unlock()
	b.info("stage increased", connID)
	return
}

func (b *backLog) calcRisk(connID uint64) (riskChanged bool) {
	b.RLock()
	s := b.CurrentStage
	idx := s - 1
	from := b.Directive.Rules[idx].From
	to := b.Directive.Rules[idx].To
	value := asset.GetValue(from)
	tval := asset.GetValue(to)
	if tval > value {
		value = tval
	}

	pRisk := b.Risk

	reliability := b.Directive.Rules[idx].Reliability
	priority := b.Directive.Priority
	b.RUnlock()
	risk := priority * reliability * value / 25

	if risk != pRisk {
		b.Lock()
		b.Risk = risk
		b.Unlock()
		b.info("risk changed.", connID)
		riskChanged = true
	}
	return
}

func (b *backLog) delete() {
	b.debug("delete sending chdone", 0)
	// no need to do this, let it gc'ed
	// close(b.chFound)
	close(b.chDone)
	//	b.chDone <- struct{}{}
	b.bLogs.delete(b)
}

func (b *backLog) updateElasticsearch(e *event.NormalizedEvent) error {
	b.Lock()
	log.Debug(log.M{Msg: "backlog updating Elasticsearch", DId: b.Directive.ID, BId: b.ID, CId: e.ConnID})
	b.StatusTime = time.Now().Unix()
	b.Unlock()
	f, err := os.OpenFile(bLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	b.RLock()
	v := siemAlarmEvents{b.ID, b.CurrentStage, e.EventID}
	b.RUnlock()
	vJSON, err := json.Marshal(v)
	if err != nil {
		fmt.Println(v)
		return err
	}

	_, err = f.WriteString(string(vJSON) + "\n") //race a
	return err
}
