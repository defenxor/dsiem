package siem

import (
	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/shared/pkg/idgen"
	log "dsiem/internal/shared/pkg/logger"
	"dsiem/internal/shared/pkg/str"
	"encoding/json"
	"errors"
	"expvar"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

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
}
type siemAlarmEvents struct {
	ID    string `json:"alarm_id"`
	Stage int    `json:"stage"`
	Event string `json:"event_id"`
}

type removalChannelMsg struct {
	ID     string
	connID uint64
}

var backlogs struct {
	sync.RWMutex
	bl map[string]*backLog
}

var backlogCounter = expvar.NewInt("backlog_counter")
var alarmCounter = expvar.NewInt("alarm_counter")

var backLogRemovalChannel chan removalChannelMsg
var ticker *time.Ticker

// InitBackLog initialize backlog and ticker
func InitBackLog(logFile string) (err error) {
	bLogFile = logFile
	backlogs.bl = make(map[string]*backLog)
	backLogRemovalChannel = make(chan removalChannelMsg)
	startBackLogTicker()

	go func() {
		for {
			// handle incoming event, id should be the ID to remove
			msg := <-backLogRemovalChannel
			go removeBackLog(msg)
		}
	}()
	return
}

// this checks for timed-out backlog and discard it
func startBackLogTicker() {
	ticker = time.NewTicker(time.Second * 10)
	go func() {
		for {
			<-ticker.C
			alarms.RLock()
			aLen := len(alarms.al)
			alarmCounter.Set(int64(aLen))
			alarms.RUnlock()
			backlogs.RLock()
			bLen := len(backlogs.bl)
			backlogCounter.Set(int64(bLen))
			log.Debug(log.M{Msg: "Ticker started, # of backlogs to check: " + strconv.Itoa(bLen)})
			now := time.Now().Unix()
			for _, v := range backlogs.bl {
				v.RLock()
				cs := v.CurrentStage
				idx := cs - 1
				start := v.Directive.Rules[idx].StartTime
				timeout := v.Directive.Rules[idx].Timeout
				maxTime := start + timeout
				if maxTime > now {
					v.RUnlock()
					continue
				}
				tx := elasticapm.DefaultTracer.StartTransaction("Directive Event Processing", "SIEM")
				tx.Context.SetCustom("backlog_id", v.ID)
				tx.Context.SetCustom("directive_id", v.Directive.ID)
				tx.Context.SetCustom("backlog_stage", v.CurrentStage)
				defer elasticapm.DefaultTracer.Recover(tx)
				defer tx.End()
				v.RUnlock()
				v.info("expired", 0)
				v.setStatus("timeout", 0, tx)
				v.delete(0)
			}
			backlogs.RUnlock()
		}
	}()
}

func removeBackLog(m removalChannelMsg) {
	backlogs.Lock()
	defer backlogs.Unlock()
	log.Debug(log.M{Msg: "Lock obtained. Removing backlog", BId: m.ID, CId: m.connID})
	delete(backlogs.bl, m.ID)
}

func backlogManager(e *event.NormalizedEvent, d *directive) {
	found := false
	//	log.Debug("blogmgr trying lock", 0)
	backlogs.RLock()
	//	log.Debug("blogmgr success", 0)
	for _, v := range backlogs.bl {
		//		log.Debug("trying to lock v", 0)
		v.RLock()
		//		log.Debug("v success", 0)
		cs := v.CurrentStage
		// only applicable for non-stage 1, where there's more specific identifier like IP address to match
		if v.Directive.ID != d.ID || cs <= 1 {
			v.RUnlock()
			continue
		}
		v.RUnlock()
		// should check for currentStage rule match with event
		// heuristic, we know stage starts at 1 but rules start at 0
		idx := cs - 1
		//		log.Debug("trying to lock v again", 0)
		v.RLock()
		//		log.Debug("v success again", 0)
		currRule := v.Directive.Rules[idx]
		v.RUnlock()
		if !doesEventMatchRule(e, &currRule, e.ConnID) {
			continue
		}
		//		log.Debug("trying to lock v again 2", 0)
		v.RLock()
		//		log.Debug("v success again 2", 0)
		log.Debug(log.M{Msg: " Event match with existing backlog. CurrentStage is " + strconv.Itoa(v.CurrentStage),
			DId: v.Directive.ID, BId: v.ID, CId: e.ConnID})
		v.RUnlock()
		found = true
		backlogs.bl[v.ID].processMatchedEvent(e, idx)
	}
	backlogs.RUnlock()

	if found {
		return
	}
	createNewBackLog(d, e)
}

func createNewBackLog(d *directive, e *event.NormalizedEvent) error {
	// create new backlog here, passing the event as the 1st event for the backlog
	bid, err := idgen.GenerateID()
	if err != nil {
		return err
	}
	log.Info(log.M{Msg: "Creating new backlog", DId: d.ID, CId: e.ConnID})
	b := backLog{}
	b.ID = bid
	b.Directive = directive{}

	copyDirective(&b.Directive, d, e)
	initBackLogRules(&b.Directive, e)
	b.Directive.Rules[0].StartTime = time.Now().Unix()

	b.CurrentStage = 1
	b.HighestStage = len(d.Rules)
	b.processMatchedEvent(e, 0)
	backlogs.Lock()
	backlogs.bl[b.ID] = &b
	backlogs.Unlock()
	return nil
}

func initBackLogRules(d *directive, e *event.NormalizedEvent) {
	for i := range d.Rules {
		// the first rule cannot use reference to other
		if i == 0 {
			d.Rules[i].Status = "active"
			continue
		}

		d.Rules[i].Status = "inactive"

		// for the rest, refer to the referenced stage if its not ANY or HOME_NET or !HOME_NET
		// if the reference is ANY || HOME_NET || !HOME_NET then refer to event if its in the format of
		// :ref

		r := d.Rules[i].From
		v, err := reftoDigit(r)
		if err == nil {
			vmin1 := v - 1
			ref := d.Rules[vmin1].From
			if ref != "ANY" && ref != "HOME_NET" && ref != "!HOME_NET" {
				d.Rules[i].From = ref
			} else {
				d.Rules[i].From = e.SrcIP
			}
		}
		r = d.Rules[i].To
		v, err = reftoDigit(r)
		if err == nil {
			vmin1 := v - 1
			ref := d.Rules[vmin1].To
			if ref != "ANY" && ref != "HOME_NET" && ref != "!HOME_NET" {
				d.Rules[i].To = ref
			} else {
				d.Rules[i].To = e.DstIP
			}
		}
		r = d.Rules[i].PortFrom
		v, err = reftoDigit(r)
		if err == nil {
			vmin1 := v - 1
			ref := d.Rules[vmin1].PortFrom
			if ref != "ANY" {
				d.Rules[i].PortFrom = ref
			} else {
				d.Rules[i].PortFrom = strconv.Itoa(e.SrcPort)
			}
		}
		r = d.Rules[i].PortTo
		v, err = reftoDigit(r)
		if err == nil {
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
			upsertAlarmFromBackLog(b, connID, tx)
			b.RUnlock()
			break
		}
	}
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
		upsertAlarmFromBackLog(b, connID, tx)
		b.RUnlock()
	}
}

func (b *backLog) processMatchedEvent(e *event.NormalizedEvent, idx int) {

	tx := elasticapm.DefaultTracer.StartTransaction("Directive Event Processing", "SIEM")
	tx.Context.SetCustom("event_id", e.EventID)
	tx.Context.SetCustom("backlog_id", b.ID)
	tx.Context.SetCustom("directive_id", b.Directive.ID)
	tx.Context.SetCustom("backlog_stage", b.CurrentStage)
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
	// the new event has caused events_count == occurrence
	b.setStatus("finished", e.ConnID, tx)

	// if it causes the last stage to reach events_count == occurrence, delete it
	if b.isLastStage() {
		b.info("reached max stage and occurrence, deleting.", e.ConnID)
		b.delete(e.ConnID)
		tx.Result = "Backlog removed (max reached)"
		return
	}

	// reach max occurrence, but not in last stage. Increase stage.
	b.increaseStage(e.ConnID)
	b.setStatus("active", e.ConnID, tx)
	tx.Context.SetCustom("backlog_stage", b.CurrentStage)
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
	log.Info(log.M{Msg: msg, BId: b.ID, CId: connID})
	b.RUnlock()
}

func (b *backLog) debug(msg string, connID uint64) {
	b.RLock()
	log.Debug(log.M{Msg: "Backlog " + b.ID + ": " + msg, BId: b.ID, CId: connID})
	b.RUnlock()
}

func (b *backLog) setLastEvent(e *event.NormalizedEvent) {
	b.Lock()
	b.LastEvent = *e
	b.Unlock()
}

func (b *backLog) updateAlarm(connID uint64, tx *elasticapm.Transaction) {
	b.RLock()
	upsertAlarmFromBackLog(b, connID, tx)
	b.RUnlock()
}

func (b *backLog) appendandWriteEvent(e *event.NormalizedEvent, idx int, tx *elasticapm.Transaction) {
	b.Lock()
	b.Directive.Rules[idx].Events = append(b.Directive.Rules[idx].Events, e.EventID)
	b.SrcIPs = str.AppendUniq(b.SrcIPs, e.SrcIP)
	b.DstIPs = str.AppendUniq(b.DstIPs, e.DstIP)
	b.Unlock()
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
	b.RLock()
	defer b.RUnlock()
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
	b.CurrentStage = int(atomic.AddInt32(&n, 1))
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

func (b *backLog) delete(connID uint64) {
	m := removalChannelMsg{b.ID, connID}
	backLogRemovalChannel <- m
	alarmRemovalChannel <- m
}

func (b *backLog) updateElasticsearch(e *event.NormalizedEvent) error {
	b.Lock()
	log.Debug(log.M{Msg: "updating Elasticsearch.", DId: b.Directive.ID, BId: b.ID, CId: e.ConnID})
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
	vJSON, _ := json.Marshal(v)

	_, err = f.WriteString(string(vJSON) + "\n")
	return err
}

func reftoDigit(v string) (int64, error) {
	i := strings.Index(v, ":")
	if i == -1 {
		return 0, errors.New("not a reference")
	}
	v = strings.Trim(v, ":")
	return strconv.ParseInt(v, 10, 64)
}
