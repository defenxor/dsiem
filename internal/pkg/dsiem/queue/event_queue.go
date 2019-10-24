package queue

import (
	"strconv"
	"sync"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	lgc "github.com/defenxor/dsiem/internal/pkg/dsiem/queue/goconcurrentqueue"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

type queueMode int

// these constants represent different types of summarizer
const (
	bound queueMode = iota
	unbound
)

// EventQueue represents queue obj for Normalized Events
type EventQueue struct {
	q                  lgc.Queue
	qMode              queueMode
	bufChans           []bufChan
	discardedCount     int
	dcLock             sync.RWMutex
	dcLastErrMsg       string
	dequeueDuration    time.Duration
	maxDequeueDuration time.Duration
	timedOut           int
	maxWait            time.Duration
	eps                string
}

type bufEvent struct {
	evt     event.NormalizedEvent
	maxWait time.Duration
}

type bufChan struct {
	dirID     int
	ch        chan bufEvent
	isTimeOut bool
	sync.RWMutex
}

const deadlockLimit = 10 * time.Second

// Init setup the eventqueue
func (eq *EventQueue) Init(target []event.Channel, maxQueueLength int, eps int) {

	eq.q = lgc.NewQueue(maxQueueLength * 2) // doubled to allow short bursts
	eq.maxDequeueDuration = time.Second / time.Duration(eps)
	eq.eps = strconv.Itoa(eps)

	if maxQueueLength > 0 {
		eq.qMode = bound
		// take 90% of the duration allocated
		eq.maxWait = eq.maxDequeueDuration * 9 / 10
	} else {
		eq.qMode = unbound
		eq.maxWait = deadlockLimit
	}

	listener := func(i int) {
		for {
			be := <-eq.bufChans[i].ch
			timedOut := false
			switch eq.qMode {
			case unbound:
				select {
				case target[i].Ch <- be.evt:
				case <-time.After(deadlockLimit):
					log.Warn(log.M{Msg: "Directive " + strconv.Itoa(target[i].DirID) + " timed out! potential deadlock detected"})
					timedOut = true
				}
			case bound:
				if be.maxWait == 0 {
					select {
					case target[i].Ch <- be.evt:
					default:
						timedOut = true
					}
				} else {
					select {
					case target[i].Ch <- be.evt:
					case <-time.After(be.maxWait):
						timedOut = true
					}
				}
			}
			eq.bufChans[i].Lock()
			eq.bufChans[i].isTimeOut = timedOut
			eq.bufChans[i].Unlock()
		}
	}
	for i := range target {
		eq.bufChans = append(eq.bufChans, bufChan{
			ch:    make(chan bufEvent),
			dirID: target[i].DirID,
		})
		go listener(i)
	}
}

// Dequeue reads event from queue
func (eq *EventQueue) Dequeue() {
	limitCap := 0
	if eq.qMode == bound {
		limitCap = eq.q.GetCap() * 5 / 10 // 50%
	}
	bEvt := bufEvent{}
	for {
		res, err := eq.q.DequeueOrWaitForNextElement()
		if err != nil {
			log.Warn(log.M{Msg: "Error occur while dequeing event: " + err.Error()})
			continue
		}
		bEvt.evt = res
		sTime := time.Now()
		if eq.qMode == unbound || eq.q.GetLen() > limitCap {
			bEvt.maxWait = 0
		} else {
			bEvt.maxWait = eq.maxWait
		}

		for i := range eq.bufChans {
			eq.bufChans[i].ch <- bEvt
		}

		sStop := time.Since(sTime)
		eq.dcLock.Lock()
		eq.dequeueDuration = sStop
		eq.dcLock.Unlock()
	}
}

// Enqueue writes event to queue
func (eq *EventQueue) Enqueue(evt event.NormalizedEvent) {
	err := eq.q.Enqueue(evt)
	if err != nil {
		eq.dcLock.Lock()
		eq.discardedCount++
		eq.dcLastErrMsg = err.Error()
		eq.dcLock.Unlock()
	}
}

// Reporter regularly prints out queue overview
func (eq *EventQueue) Reporter(d time.Duration) {
	ticker := time.NewTicker(d)
	for {
		<-ticker.C
		eq.dcLock.RLock()
		c := 0
		for i := range eq.bufChans {
			eq.bufChans[i].RLock()
			if eq.bufChans[i].isTimeOut {
				c++
			}
			eq.bufChans[i].RUnlock()
		}
		log.Info(log.M{Msg: "Backend queue length: " + strconv.Itoa(eq.q.GetLen()) +
			" dequeue duration: " + eq.dequeueDuration.String() +
			" timed-out directives: " + strconv.Itoa(c) + " max-proc.time/directive: " + eq.maxWait.String()})
		if eq.dequeueDuration > eq.maxDequeueDuration {
			log.Warn(log.M{Msg: "Single event processing took " + eq.dequeueDuration.String() +
				", may not be able to sustain the target " + eq.eps + " events/sec (" + eq.maxDequeueDuration.String() + "/event)"})
		}
		if eq.discardedCount != 0 {
			log.Warn(log.M{Msg: "Backend queue discarded: " + strconv.Itoa(eq.discardedCount) +
				" events. Reason: " + eq.dcLastErrMsg})
			eq.dcLock.RUnlock()
			eq.dcLock.Lock()
			eq.discardedCount = 0
			eq.dcLock.Unlock()
		} else {
			eq.dcLock.RUnlock()
		}
	}
}
