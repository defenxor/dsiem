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
	id   int
	bl   map[string]*backLog
	bpCh chan bool
}

var allBacklogs []backlogs

var backlogCounter = expvar.NewInt("backlog_counter")
var alarmCounter = expvar.NewInt("alarm_counter")

// InitBackLog initialize backlog and ticker
func InitBackLog(logFile string, backPressureChannel chan<- bool) (err error) {
	bLogFile = logFile
	startWatchdog(backPressureChannel)
	return
}

func updateAlarmCounter() (count int) {
	alarms.RLock()
	count = len(alarms.al)
	alarms.RUnlock()
	alarmCounter.Set(int64(count))
	return
}

func startWatchdog(backPressureChannel chan<- bool) {
	ticker := time.NewTicker(time.Second * 10)
	go func() {
		for {
			<-ticker.C
			log.Debug(log.M{Msg: "Watchdog tick started."})
			aLen := updateAlarmCounter()
			log.Info(log.M{Msg: "Watchdog tick ended, # alarms:" +
				strconv.Itoa(aLen) + readEPS()})
			// debug.FreeOSMemory()
		}
	}()

	// note, initDirective must have completed before this
	go func() {
		sWait := time.Duration(30)
		timer := time.NewTimer(time.Second * sWait)
		go func() {
			for {
				<-timer.C
				backPressureChannel <- false
				timer.Reset(time.Second * sWait)
			}
		}()
		out := mergeWait()
		for range out {
			n := <-out
			if n == true {
				backPressureChannel <- true
				timer.Reset(time.Second * sWait)
			}
		}
	}()
}

func mergeWait() <-chan bool {
	out := make(chan bool)
	var wg sync.WaitGroup
	l := len(allBacklogs)
	wg.Add(l)
	for i := range allBacklogs {
		go func(i int) {
			for v := range allBacklogs[i].bpCh {
				out <- v
			}
			wg.Done()
		}(i)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}

func readEPS() (res string) {
	if x := expvar.Get("eps_counter"); x != nil {
		res = " events/sec:" + x.String()
	}
	return
}

func (blogs *backlogs) manager(d directive, ch <-chan event.NormalizedEvent) {
	blogs.bpCh = make(chan bool)
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			<-ticker.C
			select {
			case blogs.bpCh <- false:
				ticker.Stop()
			default:
			}
		}
	}()

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
		if !doesEventMatchRule(evt, d.Rules[0], evt.ConnID) {
			continue // back to chan loop
		}

		b, err := createNewBackLog(d, evt)
		if err != nil {
			log.Warn(log.M{Msg: "Fail to create new backlog", DId: d.ID, CId: evt.ConnID})
			continue
		}
		blogs.Lock() // got hit with concurrent map write here
		blogs.bl[b.ID] = b
		blogs.bl[b.ID].bLogs = blogs
		blogs.Unlock()
		blogs.bl[b.ID].worker(evt)
	}
}

func (blogs *backlogs) delete(b *backLog) {
	log.Info(log.M{Msg: "backlog manager removing backlog in 60s", DId: b.Directive.ID, BId: b.ID})
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
		time.Sleep(30 * time.Second)
		// signal backlog worker to exit
		log.Debug(log.M{Msg: "backlog manager closing data channel", DId: b.Directive.ID, BId: b.ID})
		close(b.chData)
		time.Sleep(30 * time.Second)
		log.Debug(log.M{Msg: "backlog manager deleting backlog from map", DId: b.Directive.ID, BId: b.ID})
		blogs.Lock()
		blogs.bl[b.ID].bLogs = nil
		delete(blogs.bl, b.ID)
		blogs.Unlock()
		alarmRemovalChannel <- removalChannelMsg{b.ID}
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
			// d.Rules[i].Status = "active"
			continue
		}

		// d.Rules[i].Status = "inactive"

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
