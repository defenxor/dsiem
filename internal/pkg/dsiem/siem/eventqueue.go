package siem

import (
	"strconv"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/enriquebris/goconcurrentqueue"
)

type evtChan struct {
	dirID int
	ch    chan event.NormalizedEvent
}

type eventQueue struct {
	q        *goconcurrentqueue.FIFO
	bufChans []evtChan
}

func (eq *eventQueue) init(target []evtChan) {
	eq.q = goconcurrentqueue.NewFIFO()
	listener := func(i int) {
		for {
			e := <-eq.bufChans[i].ch
			select {
			case target[i].ch <- e:
			case <-time.After(10 * time.Second):
				log.Error(log.M{Msg: "Timeout sending event to directive " + strconv.Itoa(eq.bufChans[i].dirID) + "!"})
			}
		}
	}
	for i := range target {
		eq.bufChans = append(eq.bufChans, evtChan{
			ch:    make(chan event.NormalizedEvent),
			dirID: target[i].dirID,
		})
		go listener(i)
	}
}

func (eq *eventQueue) dequeue() {
	for {
		res, err := eq.q.DequeueOrWaitForNextElement()
		if err != nil {
			log.Error(log.M{Msg: "Error occur while dequeing event"})
			continue
		}
		evt := res.(event.NormalizedEvent)
		for i := range eq.bufChans {
			eq.bufChans[i].ch <- evt
		}
	}
}

func (eq *eventQueue) enqueue(evt event.NormalizedEvent) {
	err := eq.q.Enqueue(evt)
	if err != nil {
		log.Error(log.M{Msg: "Cannot enqueue event " + evt.EventID})
	}
}

func (eq *eventQueue) reporter() {
	ticker := time.NewTicker(30 * time.Second)
	for {
		<-ticker.C
		log.Debug(log.M{Msg: "Queue length: " + strconv.Itoa(eq.q.GetLen())})
	}
}
