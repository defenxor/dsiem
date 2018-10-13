package expcounter

import (
	"dsiem/internal/pkg/dsiem/alarm"
	"dsiem/internal/pkg/dsiem/server"
	log "dsiem/internal/pkg/shared/logger"
	"expvar"
	"runtime"
	"strconv"
	"time"
)

var goRoutineCounter = expvar.NewInt("goroutine_counter")
var epsCounter *expvar.Int
var alarmCounter *expvar.Int

// Init starts the counters
func Init(mode string) {
	ticker := time.NewTicker(time.Second * 10)
	if mode == "standalone" || mode == "cluster-frontend" {
		setEPSTicker()
		epsCounter = expvar.NewInt("eps_counter")
	}
	if mode == "standalone" || mode == "cluster-backend" {
		alarmCounter = expvar.NewInt("alarm_counter")
	}

	go func() {
		for {
			var a, e, m string
			<-ticker.C
			countGoroutine()
			switch {
			case mode == "standalone":
				a = countAlarm()
				e = countEPS()
				m = "# of alarms: " + a + " events/sec: " + e
			case mode == "cluster-frontend":
				e = countEPS()
				m = "events/sec: " + e
			case mode == "cluster-backend":
				a = countAlarm()
				m = "# of alarms: " + a
			}
			log.Info(log.M{Msg: "Watchdog tick ended, " + m})
			// debug.FreeOSMemory()
		}
	}()
}

func countGoroutine() string {
	r := runtime.NumGoroutine()
	goRoutineCounter.Set(int64(r))
	return strconv.Itoa(r)
}

func countAlarm() string {
	a := alarm.Count()
	alarmCounter.Set(int64(a))
	return strconv.Itoa(a)
}

func countEPS() (res string) {
	if x := expvar.Get("eps_counter"); x != nil {
		res = x.String()
	}
	return
}

func setEPSTicker() {
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			<-ticker.C
			epsCounter.Set(server.CounterRate())
		}
	}()
}
