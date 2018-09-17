package siem

import (
	"dsiem/internal/dsiem/pkg/asset"
	"dsiem/internal/dsiem/pkg/event"
	log "dsiem/internal/dsiem/pkg/logger"
	"dsiem/internal/shared/pkg/str"
	"encoding/json"
	"errors"

	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/teris-io/shortid"
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
}
type siemAlarmEvents struct {
	ID    string `json:"alarm_id"`
	Stage int    `json:"stage"`
	Event string `json:"event_id"`
}
type backLogs struct {
	mu       sync.RWMutex
	BackLogs []backLog `json:"backlogs"`
}
type removalChannelMsg struct {
	ID     string
	connID uint64
}

var backLogRemovalChannel chan removalChannelMsg
var bLogs backLogs
var sid *shortid.Shortid
var ticker *time.Ticker

func initShortID() (err error) {
	sid, err = shortid.New(1, shortid.DEFAULT_ABC, 2342)
	return
}

// InitBackLog initialize backlog and ticker
func InitBackLog(logFile string) (err error) {
	bLogFile = logFile
	if err = initShortID(); err != nil {
		return
	}
	startBackLogTicker()
	backLogRemovalChannel = make(chan removalChannelMsg)
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
			log.Debug("Ticker started, # of backlogs to check: "+strconv.Itoa(len(bLogs.BackLogs)), 0)
			now := time.Now().Unix()
			bLogs.mu.RLock()
			for i := range bLogs.BackLogs {
				cs := bLogs.BackLogs[i].CurrentStage
				idx := cs - 1
				start := bLogs.BackLogs[i].Directive.Rules[idx].StartTime
				timeout := bLogs.BackLogs[i].Directive.Rules[idx].Timeout
				maxTime := start + timeout
				if maxTime > now {
					continue
				}
				log.Info("directive "+strconv.Itoa(bLogs.BackLogs[i].Directive.ID)+" backlog "+bLogs.BackLogs[i].ID+" expired.", 0)
				bLogs.BackLogs[i].setStatus("timeout", 0)
				bLogs.BackLogs[i].delete(0)
			}
			bLogs.mu.RUnlock()
		}
	}()
}

func removeBackLog(m removalChannelMsg) {
	log.Debug("Trying to obtain write lock to remove backlog "+m.ID, m.connID)
	bLogs.mu.Lock()
	defer bLogs.mu.Unlock()
	log.Debug("Lock obtained. Removing backlog "+m.ID, m.connID)
	idx := -1
	for i := range bLogs.BackLogs {
		if bLogs.BackLogs[i].ID == m.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return
	}
	// copy last element to idx location
	bLogs.BackLogs[idx] = bLogs.BackLogs[len(bLogs.BackLogs)-1]
	// write empty to last element
	bLogs.BackLogs[len(bLogs.BackLogs)-1] = backLog{}
	// truncate slice
	bLogs.BackLogs = bLogs.BackLogs[:len(bLogs.BackLogs)-1]
}

// BacklogManager
func backlogManager(e *event.NormalizedEvent, d *directive) {
	found := false
	bLogs.mu.RLock()
	for i := range bLogs.BackLogs {
		cs := bLogs.BackLogs[i].CurrentStage
		// only applicable for non-stage 1, where there's more specific identifier like IP address to match
		if bLogs.BackLogs[i].Directive.ID != d.ID || cs <= 1 {
			continue
		}
		// should check for currentStage rule match with event
		// heuristic, we know stage starts at 1 but rules start at 0
		idx := cs - 1
		currRule := bLogs.BackLogs[i].Directive.Rules[idx]
		if !doesEventMatchRule(e, &currRule, e.ConnID) {
			continue
		}
		log.Debug("Directive "+strconv.Itoa(d.ID)+" backlog "+bLogs.BackLogs[i].ID+" matched. Not creating new backlog.", e.ConnID)
		found = true
		bLogs.BackLogs[i].processMatchedEvent(e, idx)
	}
	bLogs.mu.RUnlock()

	if found {
		return
	}
	createNewBackLog(d, e)
}

func createNewBackLog(d *directive, e *event.NormalizedEvent) {
	// create new backlog here, passing the event as the 1st event for the backlog
	bid, _ := sid.Generate()
	log.Info("Directive "+strconv.Itoa(d.ID)+" created new backlog "+bid, e.ConnID)
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
	bLogs.mu.Lock()
	bLogs.BackLogs = append(bLogs.BackLogs, b)
	bLogs.mu.Unlock()
	log.Debug("Lock obtained/released for backlog "+bid+" creation.", e.ConnID)
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

func (b *backLog) setStatus(status string, connID uint64) {
	s := b.CurrentStage
	idx := s - 1
	b.Directive.Rules[idx].Status = status
	upsertAlarmFromBackLog(b, connID)
}

func (b *backLog) ensureStatusAndStartTime(idx int, connID uint64) {
	// this reinsert status and startDate for the currentStage rule if the first attempt failed
	updateFlag := false
	if b.Directive.Rules[idx].StartTime == 0 {
		b.Directive.Rules[idx].StartTime = time.Now().Unix()
		updateFlag = true
	}
	if b.Directive.Rules[idx].Status != "active" {
		b.setStatus("active", connID)
		updateFlag = true
	}
	if updateFlag {
		upsertAlarmFromBackLog(b, connID)
	}
}

func (b *backLog) processMatchedEvent(e *event.NormalizedEvent, idx int) {

	b.appendandWriteEvent(e, idx)

	// exit early if the newly added event hasnt caused events_count == occurrence
	// for the current stage
	if !b.isStageReachMaxEvtCount() {
		b.ensureStatusAndStartTime(idx, e.ConnID)
		return
	}

	// the new event has caused events_count == occurrence
	b.setStatus("finished", e.ConnID)

	// if it causes the last stage to reach events_count == occurrence, delete it
	if b.isLastStage() {
		log.Info("Backlog "+b.ID+" has reached its max stage and occurrence. Deleting it.", e.ConnID)
		b.delete(e.ConnID)
		return
	}

	// reach max occurrence, but not in last stage. Increase stage.
	b.increaseStage(e.ConnID)
	b.setStatus("active", e.ConnID)

	// recalc risk, the new stage will have a different reliability
	riskChanged := b.calcRisk(e.ConnID)
	if riskChanged {
		upsertAlarmFromBackLog(b, e.ConnID)
	}
}

func (b *backLog) appendandWriteEvent(e *event.NormalizedEvent, idx int) {
	b.Directive.Rules[idx].Events = append(b.Directive.Rules[idx].Events, e.EventID)
	b.SrcIPs = str.AppendUniq(b.SrcIPs, e.SrcIP)
	b.DstIPs = str.AppendUniq(b.DstIPs, e.DstIP)

	if err := b.updateElasticsearch(e); err != nil {
		log.Warn("Backlog "+b.ID+" failed to update Elasticsearch! "+err.Error(), e.ConnID)
	}
}

func (b *backLog) isLastStage() bool {
	return b.CurrentStage == b.HighestStage
}

func (b *backLog) isStageReachMaxEvtCount() (reachMaxEvtCount bool) {
	s := b.CurrentStage
	idx := s - 1
	currRule := b.Directive.Rules[idx]
	nEvents := len(b.Directive.Rules[idx].Events)
	if nEvents == currRule.Occurrence {
		reachMaxEvtCount = true
	}
	return
}

func (b *backLog) increaseStage(connID uint64) {
	b.CurrentStage++
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
