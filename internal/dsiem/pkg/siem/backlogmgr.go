package siem

import (
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/shared/pkg/idgen"
	log "dsiem/internal/shared/pkg/logger"
	"dsiem/internal/shared/pkg/str"
	"expvar"
	"strconv"

	"sync"
	"time"
)

type backlogs struct {
	sync.RWMutex
	id int
	bl map[string]*backLog
}

// allBacklogs doesnt need a lock, its size is fixed to the number
// of all loaded directives
var allBacklogs []backlogs

var backlogCounter = expvar.NewInt("backlog_counter")
var alarmCounter = expvar.NewInt("alarm_counter")

// InitBackLog initialize backlog and ticker
func InitBackLog(logFile string) (err error) {
	bLogFile = logFile
	startWatchdogTicker()
	return
}

func updateAlarmCounter() (count int) {
	alarms.RLock()
	log.Debug(log.M{Msg: "counter obtained alarm lock"})
	count = len(alarms.al)
	alarmCounter.Set(int64(count))
	alarms.RUnlock()
	return
}

// this checks for timed-out backlog and discard it
func startWatchdogTicker() {
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		for {
			<-ticker.C
			log.Debug(log.M{Msg: "Watchdog tick started."})
			aLen := updateAlarmCounter()
			log.Info(log.M{Msg: "Watchdog tick ended, # alarms:" + strconv.Itoa(aLen)})
		}
	}()
}

func readEPS() (res string) {
	if x := expvar.Get("eps_counter"); x != nil {
		res = " events/sec:" + x.String()
	}
	return
}

func (blogs *backlogs) delete(b *backLog) {
	log.Info(log.M{Msg: "backlog manager removing backlog in 10s", DId: b.Directive.ID, BId: b.ID})
	go func() {
		time.Sleep(3 * time.Second)
		log.Debug(log.M{Msg: "backlog manager closing data channel", DId: b.Directive.ID, BId: b.ID})
		close(b.chData)
		time.Sleep(5 * time.Second)
		blogs.Lock()
		delete(blogs.bl, b.ID)
		blogs.Unlock()
		alarmRemovalChannel <- removalChannelMsg{b.ID}
	}()
}

func (blogs *backlogs) manager(d *directive, ch <-chan *event.NormalizedEvent) {
	blogs.bl = make(map[string]*backLog)

	for {
		evt := <-ch
		found := false
		blogs.RLock() // to prevent concurrent r/w with delete()
		for k := range blogs.bl {
			// go try-receive pattern
			select {
			case <-blogs.bl[k].chDone: // exit early if done, this should be the case while backlog in waiting for deletion mode
				continue
			default:
			}

			select {
			case <-blogs.bl[k].chDone: // exit early if done
				continue
			case blogs.bl[k].chData <- evt: // fwd to backlog
				select {
				case <-blogs.bl[k].chDone: // exit early if done
					continue
				// wait for the result
				case f := <-blogs.bl[k].chFound:
					if f {
						found = true
					}
				}
			}
		}
		blogs.RUnlock()

		if found {
			continue
		}
		// now for new backlog
		if !doesEventMatchRule(evt, &d.Rules[0], evt.ConnID) {
			continue // back to chan loop
		}

		b, err := createNewBackLog(d, evt)
		if err != nil {
			log.Warn(log.M{Msg: "Fail to create new backlog", DId: d.ID, CId: evt.ConnID})
			continue
		}
		blogs.Lock() // got hit with concurrent map write here
		blogs.bl[b.ID] = &b
		blogs.bl[b.ID].bLogs = blogs
		blogs.Unlock()
		blogs.bl[b.ID].worker(evt)
	}
}

func createNewBackLog(d *directive, e *event.NormalizedEvent) (bp backLog, err error) {
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
	b.Directive.Rules[0].StartTime = time.Now().Unix()
	b.chData = make(chan *event.NormalizedEvent)
	b.chFound = make(chan bool)
	b.chDone = make(chan struct{}, 1)

	b.CurrentStage = 1
	b.HighestStage = len(d.Rules)
	bp = b

	return
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
