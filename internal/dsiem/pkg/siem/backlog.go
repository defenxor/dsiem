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
	"time"

	"github.com/elastic/apm-agent-go"
)

var bLogFile string

type backLog struct {
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

var bMap map[string]*backLog
var mu sync.RWMutex
var backlogCounter = expvar.NewInt("backlog_counter")

var backLogRemovalChannel chan removalChannelMsg
var ticker *time.Ticker

// InitBackLog initialize backlog and ticker
func InitBackLog(logFile string) (err error) {
	bLogFile = logFile
	bMap = make(map[string]*backLog)
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
			bLen := len(bMap)
			backlogCounter.Set(int64(bLen))
			log.Debug("Ticker started, # of backlogs to check: "+strconv.Itoa(bLen), 0)
			now := time.Now().Unix()
			mu.RLock()
			for _, v := range bMap {
				cs := v.CurrentStage
				idx := cs - 1
				start := v.Directive.Rules[idx].StartTime
				timeout := v.Directive.Rules[idx].Timeout
				maxTime := start + timeout
				if maxTime > now {
					continue
				}
				tx := elasticapm.DefaultTracer.StartTransaction("Directive Event Processing", "SIEM")
				tx.Context.SetCustom("backlog_id", v.ID)
				tx.Context.SetCustom("directive_id", v.Directive.ID)
				tx.Context.SetCustom("backlog_stage", v.CurrentStage)
				defer elasticapm.DefaultTracer.Recover(tx)
				defer tx.End()

				log.Info("directive "+strconv.Itoa(v.Directive.ID)+" backlog "+v.ID+" expired.", 0)
				v.setStatus("timeout", 0, tx)
				v.delete(0)
			}
			mu.RUnlock()
		}
	}()
}

func removeBackLog(m removalChannelMsg) {
	log.Debug("Trying to obtain write lock to remove backlog "+m.ID, m.connID)
	mu.Lock()
	defer mu.Unlock()
	log.Debug("Lock obtained. Removing backlog "+m.ID, m.connID)
	delete(bMap, m.ID)
}

func backlogManager(e *event.NormalizedEvent, d *directive) {
	found := false
	mu.RLock()
	for _, v := range bMap {
		cs := v.CurrentStage
		// only applicable for non-stage 1, where there's more specific identifier like IP address to match
		if v.Directive.ID != d.ID || cs <= 1 {
			continue
		}
		// should check for currentStage rule match with event
		// heuristic, we know stage starts at 1 but rules start at 0
		idx := cs - 1
		currRule := v.Directive.Rules[idx]
		if !doesEventMatchRule(e, &currRule, e.ConnID) {
			continue
		}
		log.Debug("Directive "+strconv.Itoa(d.ID)+" backlog "+v.ID+" matched. Not creating new backlog. CurrentStage is "+
			strconv.Itoa(v.CurrentStage), e.ConnID)
		found = true
		bMap[v.ID].processMatchedEvent(e, idx)
	}
	mu.RUnlock()

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
	log.Info("Directive "+strconv.Itoa(d.ID)+" creating new backlog "+bid, e.ConnID)
	b := backLog{}
	b.ID = bid
	b.Directive = directive{}

	copyDirective(&b.Directive, d, e)
	initBackLogRules(&b.Directive, e)
	b.Directive.Rules[0].StartTime = time.Now().Unix()

	b.CurrentStage = 1
	b.HighestStage = len(d.Rules)
	b.processMatchedEvent(e, 0)
	log.Debug("Trying to obtain write lock to create backlog "+bid, e.ConnID)
	mu.Lock()
	bMap[b.ID] = &b
	mu.Unlock()
	log.Debug("Lock obtained/released for backlog "+bid+" creation.", e.ConnID)
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
	s := b.CurrentStage
	idx := s - 1
	if b.Directive.Rules[idx].Status == "timeout" || b.Directive.Rules[idx].Status == "finished" {
		return
	}
	allowed := []string{"timeout", "finished"}
	if b.Directive.Rules[idx].Status == "inactive" {
		allowed = append(allowed, "active")
	}
	for i := range allowed {
		if allowed[i] == status {
			b.Directive.Rules[idx].Status = status
			upsertAlarmFromBackLog(b, connID, tx)
			break
		}
	}
}

func (b *backLog) ensureStatusAndStartTime(idx int, connID uint64, tx *elasticapm.Transaction) {
	// this reinsert status and startDate for the currentStage rule if the first attempt failed
	updateFlag := false
	if b.Directive.Rules[idx].StartTime == 0 {
		b.Directive.Rules[idx].StartTime = time.Now().Unix()
		updateFlag = true
	}
	if b.Directive.Rules[idx].Status != "active" {
		b.setStatus("active", connID, tx)
		updateFlag = true
	}
	if updateFlag {
		upsertAlarmFromBackLog(b, connID, tx)
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

	log.Debug("Incoming event for backlog "+b.ID+" with idx: "+strconv.Itoa(idx), e.ConnID)
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
		log.Info("Backlog "+b.ID+" has reached its max stage and occurrence. Deleting it.", e.ConnID)
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
		b.LastEvent = *e
		upsertAlarmFromBackLog(b, e.ConnID, tx)
	}
}

func (b *backLog) appendandWriteEvent(e *event.NormalizedEvent, idx int, tx *elasticapm.Transaction) {

	b.Directive.Rules[idx].Events = append(b.Directive.Rules[idx].Events, e.EventID)
	b.SrcIPs = str.AppendUniq(b.SrcIPs, e.SrcIP)
	b.DstIPs = str.AppendUniq(b.DstIPs, e.DstIP)

	if err := b.updateElasticsearch(e); err != nil {
		log.Warn("Backlog "+b.ID+" failed to update Elasticsearch! "+err.Error(), e.ConnID)
		e := elasticapm.DefaultTracer.NewError(err)
		e.Transaction = tx
		e.Send()
		tx.Result = "Failed to append and write event"
	} else {
		tx.Result = "Event appended to backlog"
	}
	return
}

func (b *backLog) isLastStage() bool {
	return b.CurrentStage == b.HighestStage
}

func (b *backLog) isStageReachMaxEvtCount() (reachMaxEvtCount bool) {
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
	b.CurrentStage++
	// make sure its not over the higheststage, concurrency may cause this
	if b.CurrentStage > b.HighestStage {
		b.CurrentStage = b.HighestStage
	}
	log.Info("directive "+strconv.Itoa(b.Directive.ID)+" backlog "+b.ID+" increased stage to "+strconv.Itoa(b.CurrentStage), connID)
	idx := b.CurrentStage - 1
	b.Directive.Rules[idx].StartTime = time.Now().Unix()
	b.StatusTime = b.Directive.Rules[idx].StartTime
	return
}

func (b *backLog) calcRisk(connID uint64) (riskChanged bool) {
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
	risk := priority * reliability * value / 25

	if risk != pRisk {
		b.Risk = risk
		log.Info("directive "+strconv.Itoa(b.Directive.ID)+" backlog "+
			b.ID+" risk changed from "+strconv.Itoa(pRisk)+" to "+strconv.Itoa(risk), connID)
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
	log.Debug("directive "+strconv.Itoa(b.Directive.ID)+" backlog "+b.ID+" updating Elasticsearch.", e.ConnID)
	b.StatusTime = time.Now().Unix()

	f, err := os.OpenFile(bLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	v := siemAlarmEvents{b.ID, b.CurrentStage, e.EventID}
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
