// Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.
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

const (
	timeoutNone = iota
	timeoutDeadlock
	timeoutZero
	timeoutProcessing
)

// EventQueue represents queue obj for Normalized Events
type EventQueue struct {
	q                  lgc.Queue
	qMode              queueMode
	bufChans           []bufChan
	maxDequeueDuration time.Duration
	maxWait            time.Duration
	reporter           Reporter
	oneTimeRun         bool
	limitCap           int
}

type bufEvent struct {
	evt     event.NormalizedEvent
	maxWait time.Duration
}

type bufChan struct {
	dirID int
	ch    chan bufEvent
	sync.RWMutex
}

var deadlockLimit = 10 * time.Second

// Init setup the eventqueue
func (eq *EventQueue) Init(target []event.Channel, maxQueueLength int, eps int) {

	eq.q = lgc.NewQueue(maxQueueLength * 2) // doubled to allow short bursts
	eq.maxDequeueDuration = time.Second / time.Duration(eps)

	if maxQueueLength > 0 {
		eq.qMode = bound
		// take 90% of the duration allocated
		eq.maxWait = eq.maxDequeueDuration * 9 / 10
		eq.limitCap = eq.q.GetCap() * 5 / 10 // 50%
	} else {
		eq.qMode = unbound
		eq.maxWait = deadlockLimit
	}

	eq.reporter.Init(len(target), eq.maxWait, eq.maxDequeueDuration, eps, eq.q.GetLen)

	listener := func(i int) {
		for {
			be := <-eq.bufChans[i].ch
			timeoutStatus := timeoutNone
			switch eq.qMode {
			case unbound:
				select {
				case target[i].Ch <- be.evt:
				case <-time.After(deadlockLimit):
					log.Warn(log.M{Msg: "Directive " + strconv.Itoa(target[i].DirID) + " timed out! potential deadlock detected"})
					timeoutStatus = timeoutDeadlock
				}
			case bound:
				if be.maxWait == 0 {
					select {
					case target[i].Ch <- be.evt:
					default:
						timeoutStatus = timeoutZero
					}
				} else {
					select {
					case target[i].Ch <- be.evt:
					case <-time.After(be.maxWait):
						timeoutStatus = timeoutProcessing
					}
				}
			}
			select {
			case eq.reporter.statusChan[i] <- timeoutStatus:
			default:
			}
		}
	}
	for i := range target {
		eq.bufChans = append(eq.bufChans, bufChan{
			ch:    make(chan bufEvent),
			dirID: target[i].DirID,
		})
	}
	for i := range target {
		go listener(i)
	}
}

// Dequeue reads event from queue
func (eq *EventQueue) Dequeue() {
	bEvt := bufEvent{}

	for {
		res, err := eq.q.DequeueOrWaitForNextElement()
		if err != nil {
			log.Warn(log.M{Msg: "Error occur while dequeing event: " + err.Error()})
			if eq.q.IsLocked() {
				time.Sleep(time.Second)
			}
			if eq.oneTimeRun {
				return
			}
			continue
		}
		bEvt.evt = res
		sTime := time.Now()
		// set maxwait to zero unbounded queue, or if capacity is > 50% for bounded queue
		if eq.qMode == unbound || eq.q.GetLen() > eq.limitCap {
			bEvt.maxWait = 0
		} else {
			bEvt.maxWait = eq.maxWait
		}
		for i := range eq.bufChans {
			eq.bufChans[i].ch <- bEvt
		}
		eq.reporter.recordDequeueTime(sTime)
		if eq.oneTimeRun {
			return
		}
	}
}

// Enqueue writes event to queue
func (eq *EventQueue) Enqueue(evt event.NormalizedEvent) {
	err := eq.q.Enqueue(evt)
	if err != nil {
		eq.reporter.increaseDiscardedCount(err.Error())
	}
}

// GetReporter return the function to print reports
func (eq *EventQueue) GetReporter() func(time.Duration) {
	return eq.reporter.PrintReport
}
