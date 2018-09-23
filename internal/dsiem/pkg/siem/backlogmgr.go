package siem

import (
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/shared/pkg/idgen"
	log "dsiem/internal/shared/pkg/logger"
	"errors"
	"expvar"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/elastic/apm-agent-go"
)

type backlogs struct {
	sync.RWMutex
	bl map[string]*backLog
}

// allBacklogs doesnt need a lock, its size is fixed to the number
// of all loaded directives
var allBacklogs []backlogs

var backlogCounter = expvar.NewInt("backlog_counter")
var alarmCounter = expvar.NewInt("alarm_counter")

type removalChannelMsg struct {
	blogs *backlogs
	ID    string
}

var backLogRemovalChannel chan removalChannelMsg
var ticker *time.Ticker

// InitBackLog initialize backlog and ticker
func InitBackLog(logFile string) (err error) {
	bLogFile = logFile
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

func removeBackLog(m removalChannelMsg) {
	m.blogs.Lock()
	defer m.blogs.Unlock()
	log.Debug(log.M{Msg: "Lock obtained. Removing backlog", BId: m.ID})
	delete(m.blogs.bl, m.ID)
}

// this checks for timed-out backlog and discard it
func startBackLogTicker() {
	ticker = time.NewTicker(time.Second * 10)
	go func() {
		for {
			<-ticker.C
			log.Debug(log.M{Msg: "Ticker started."})
			alarms.RLock()
			aLen := len(alarms.al)
			alarmCounter.Set(int64(aLen))
			alarms.RUnlock()

			bLen := 0
			for i := range allBacklogs {
				allBacklogs[i].RLock()
				l := len(allBacklogs[i].bl)
				if l == 0 {
					allBacklogs[i].RUnlock()
					continue
				}
				bLen += l
				now := time.Now().Unix()
				for _, v := range allBacklogs[i].bl {
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
					v.delete()
				}
				allBacklogs[i].RUnlock()
			}
			log.Debug(log.M{Msg: "Ticker ends, # of backlogs checked: " + strconv.Itoa(bLen)})
			backlogCounter.Set(int64(bLen))
		}
	}()
}

func (blogs *backlogs) manager(d *directive, ch <-chan event.NormalizedEvent) {
	for {
		// handle incoming event
		e := <-ch
		// first check existing backlog
		found := false
		blogs.RLock()
		for _, v := range blogs.bl {
			v.RLock()
			cs := v.CurrentStage
			// only applicable for non-stage 1,
			// where there's more specific identifier like IP address to match
			// by convention, stage 1 rule *must* always have occurrence = 1
			if v.Directive.ID != d.ID || cs <= 1 {
				v.RUnlock()
				continue
			}
			// should check for currentStage rule match with event
			// heuristic, we know stage starts at 1 but rules start at 0
			idx := cs - 1
			currRule := v.Directive.Rules[idx]
			if !doesEventMatchRule(&e, &currRule, e.ConnID) {
				v.RUnlock()
				continue
			}
			log.Debug(log.M{Msg: " Event match with existing backlog. CurrentStage is " +
				strconv.Itoa(v.CurrentStage), DId: v.Directive.ID, BId: v.ID, CId: e.ConnID})
			v.RUnlock()
			found = true
			go blogs.bl[v.ID].processMatchedEvent(&e, idx)
			break
		}
		if found {
			blogs.RUnlock()
			continue // back to chan loop
		}

		// now for new backlog
		if !doesEventMatchRule(&e, &d.Rules[0], e.ConnID) {
			blogs.RUnlock()
			continue // back to chan loop
		}

		b, err := createNewBackLog(d, &e)
		if err != nil {
			log.Warn(log.M{Msg: "Fail to create new backlog", DId: d.ID, CId: e.ConnID})
			blogs.RUnlock()
			continue
		}
		b.bLogs = blogs
		blogs.RUnlock()
		blogs.Lock()
		blogs.bl[b.ID] = b
		blogs.Unlock()
		go blogs.bl[b.ID].processMatchedEvent(&e, 0)
	}
}

func createNewBackLog(d *directive, e *event.NormalizedEvent) (bp *backLog, err error) {
	// create new backlog here, passing the event as the 1st event for the backlog
	bid, err := idgen.GenerateID()
	if err != nil {
		return nil, err
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
	return &b, nil
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

func reftoDigit(v string) (int64, error) {
	i := strings.Index(v, ":")
	if i == -1 {
		return 0, errors.New("not a reference")
	}
	v = strings.Trim(v, ":")
	return strconv.ParseInt(v, 10, 64)
}
