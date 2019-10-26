// Copyright (c) 2018 PT Defender Nusa Semesta and contributors, All rights reserved.
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

package expcounter

import (
	"expvar"
	"runtime"
	"strconv"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/siem"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/server"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
)

var goRoutineCounter = expvar.NewInt("goroutine_counter")
var epsCounter *expvar.Int
var alarmCounter *expvar.Int
var backlogCounter *expvar.Int

// Init starts the counters
func Init(mode string) {
	if mode == "standalone" || mode == "cluster-frontend" {
		if epsCounter == nil {
			epsCounter = expvar.NewInt("eps_counter")
		}
		go setEPSTicker()
	}
	if mode == "standalone" || mode == "cluster-backend" {
		if alarmCounter == nil {
			alarmCounter = expvar.NewInt("alarm_counter")
		}
		if backlogCounter == nil {
			backlogCounter = expvar.NewInt("backlog_counter")
		}
	}
	go startTicker(mode, false)
}

func startTicker(mode string, once bool) {
	ticker := time.NewTicker(time.Second * 10)
	// once is a flag for test
	if once {
		ticker = time.NewTicker(time.Second * 1)
	} else {
		// start first counting 5 seconds later to avoid data race with server
		time.Sleep(5 * time.Second)
	}
	countGoroutine() // not used in the loop for now
	for {
		var e, b, m string
		<-ticker.C
		// g = countGoroutine()
		switch {
		case mode == "standalone":
			// a = countAlarm()
			e = countEPS()
			b = countBacklogs()
			m = "# of " + b + " events/sec: " + e
		case mode == "cluster-frontend":
			e = countEPS()
			m = "events/sec: " + e
		case mode == "cluster-backend":
			// a = countAlarm()
			b = countBacklogs()
			m = "# of " + b
		}
		// m = m + " goroutines: " + g
		log.Info(log.M{Msg: "Watchdog tick ended, " + m})
		if once {
			return
		}
	}
}

func countGoroutine() string {
	r := runtime.NumGoroutine()
	goRoutineCounter.Set(int64(r))
	return strconv.Itoa(r)
}

/*
func countAlarm() string {
	a := alarm.Count()
	alarmCounter.Set(int64(a))
	return strconv.Itoa(a)
}
*/

func countBacklogs() string {
	b, act, ttl := siem.CountBackLogs()
	backlogCounter.Set(int64(b))
	return "backlogs: " +
		strconv.Itoa(b) + " directives (in-use/total): " + strconv.Itoa(act) + "/" +
		strconv.Itoa(ttl)
}

func countEPS() (res string) {
	if x := expvar.Get("eps_counter"); x != nil {
		res = x.String()
	}
	return
}

func setEPSTicker() {
	// sleep 5 seconds at the beginning to avoid data race with server
	time.Sleep(5 * time.Second)
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			<-ticker.C
			epsCounter.Set(server.CounterRate())
		}
	}()
}
