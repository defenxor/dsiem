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
	//"runtime"
	"strings"
	"testing"
	"time"

	"fmt"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

func TestQueue(t *testing.T) {

	deadlockLimit = 1 * time.Second
	boundedLength := 3

	if !log.TestMode {
		t.Logf("Enabling log test mode")
		log.EnableTestingMode()
	}

	for queueLength := 0; queueLength < 2; queueLength++ {

		evt := event.NormalizedEvent{}
		target := []event.Channel{}

		for i := 0; i < 2; i++ {
			target = append(target, event.Channel{
				DirID: i,
				Ch:    make(chan event.NormalizedEvent),
			})
		}

		eq := EventQueue{}

		vql := boundedLength - 1
		vql = vql * queueLength

		fmt.Println("Using queue length of", vql)
		eq.Init(target, vql, 3000)
		eq.oneTimeRun = true
		eq.reporter.oneTimeRun = true

		fmt.Println("enqueing on a locked queue, should increase discarded count")
		eq.q.Lock()
		eq.Enqueue(evt)
		time.Sleep(time.Second)
		if eq.reporter.getDiscardedCount() != 1 {
			t.Fatal("discarded count expected to be 1")
		}

		fmt.Println("dequeing on a locked queue")
		verifyFuncOutput(t, func() {
			go eq.Dequeue()
			time.Sleep(time.Second)
			eq.q.Unlock()
		}, "queue is locked", true)

		fmt.Println("enqueing 1st event")
		eq.Enqueue(evt)

		errs := make(chan error, 1)
		go func() {
			for i := range target {
				select {
				case <-target[i].Ch:
				case <-time.After(3 * time.Second):
					errs <- fmt.Errorf("cannot read from channel %d", i)
					return
				}
			}
			errs <- nil
		}()

		eq.Dequeue()

		res := <-errs
		if res != nil {
			t.Fatal(res)
		}

		fmt.Println("simulating timeout")
		ttl := 0
		for i := 0; i < 6; i++ {
			eq.Enqueue(evt)
			eq.Enqueue(evt)
			eq.Dequeue()
			n := eq.q.GetLen()
			ttl = eq.reporter.countStatus()
			fmt.Println("count status:", ttl, "queue length:", n)
		}

		eq.reporter.Lock()
		actual := 0
		switch queueLength {
		case 0:
			actual = eq.reporter.nDeadlock
		case 1:
			if eq.reporter.nProcessing == 0 || eq.reporter.nZero == 0 {
				actual = 0 // fail on purpose
			} else {
				actual = eq.reporter.nProcessing + eq.reporter.nZero
			}
		}
		if ttl != actual {
			t.Fatalf("Expected # of timeout to be %d, actual is %d", ttl, actual)
		}
		eq.reporter.Unlock()

		f := eq.GetReporter()
		str := "Backend queue discarded:"
		fmt.Println("verifying output")
		verifyFuncOutput(t, func() {
			f(time.Second)
		}, str, true)

		time.Sleep(time.Second)
	}
}

func verifyFuncOutput(t *testing.T, f func(), expected string, expectMatch bool) {
	out := log.CaptureZapOutput(f)
	t.Log("out: ", out)
	if !strings.Contains(out, expected) == expectMatch {
		t.Fatalf("Cannot find '%s' in output: %s", expected, out)
	} else {
		fmt.Println("OK")
	}
}
