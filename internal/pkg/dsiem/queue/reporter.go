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

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

// Reporter represent reporter object
type Reporter struct {
	statusChan         []chan int
	dequeueDuration    time.Duration
	nDeadlock          int
	nZero              int
	nProcessing        int
	maxWait            time.Duration
	maxDequeueDuration time.Duration
	eps                string
	nDiscarded         int
	discardedMsg       string
	queueLengthFunc    func() int
	sync.RWMutex
	oneTimeRun bool
}

// Init initialises Reporter
func (r *Reporter) Init(nChans int, maxWait, maxDequeueDuration time.Duration, eps int, queueLengthFunc func() int) {
	r.queueLengthFunc = queueLengthFunc
	r.maxDequeueDuration = maxDequeueDuration
	r.maxWait = maxWait
	r.eps = strconv.Itoa(eps)
	for i := 0; i <= nChans; i++ {
		r.statusChan = append(r.statusChan, make(chan int, 1))
	}
}

func (r *Reporter) recordDequeueTime(tm time.Time) {
	go func() {
		r.Lock()
		r.dequeueDuration = time.Since(tm)
		r.Unlock()
	}()
}

func (r *Reporter) increaseDiscardedCount(reason string) {
	go func() {
		r.Lock()
		r.nDiscarded++
		r.discardedMsg = reason
		r.Unlock()
	}()
}

func (r *Reporter) getDiscardedCount() (n int) {
	r.RLock()
	n = r.nDiscarded
	r.RUnlock()
	return
}

func (r *Reporter) countStatus() (total int) {
	r.Lock()
	defer r.Unlock()
	for i := range r.statusChan {
		select {
		case s := <-r.statusChan[i]:
			switch s {
			case timeoutDeadlock:
				r.nDeadlock++
			case timeoutZero:
				r.nZero++
			case timeoutProcessing:
				r.nProcessing++
			}
		default:
		}
	}
	return r.nDeadlock + r.nZero + r.nProcessing
}

// PrintReport regularly prints report content
func (r *Reporter) PrintReport(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		ttl := r.countStatus()
		r.RLock()
		log.Info(log.M{Msg: "Backend queue length: " + strconv.Itoa(r.queueLengthFunc()) +
			" dequeue duration: " + r.dequeueDuration.String() +
			" timed-out directives: " + strconv.Itoa(ttl) + "(" +
			strconv.Itoa(r.nDeadlock) + "/" + strconv.Itoa(r.nZero) + "/" + strconv.Itoa(r.nProcessing) +
			") max-proc.time/directive: " + r.maxWait.String()})
		if r.dequeueDuration > r.maxDequeueDuration {
			log.Warn(log.M{Msg: "Single event processing took " + r.dequeueDuration.String() +
				", may not be able to sustain the target " + r.eps + " events/sec (" + r.maxDequeueDuration.String() + "/event)"})
		}
		if r.nDiscarded != 0 {
			log.Warn(log.M{Msg: "Backend queue discarded: " + strconv.Itoa(r.nDiscarded) +
				" events. Reason: " + r.discardedMsg})
		}
		oneTime := r.oneTimeRun
		r.RUnlock()
		r.resetDiscardedCount()
		if oneTime {
			return
		}
	}
}

func (r *Reporter) resetDiscardedCount() {
	r.Lock()
	r.nDiscarded = 0
	r.Unlock()
}
