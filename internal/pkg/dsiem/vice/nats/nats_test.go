package nats

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	gnatsd "github.com/nats-io/gnatsd/server"
)

// DefaultTestOptions are default options for the unit tests.
var DefaultTestOptions = gnatsd.Options{
	Host:           "127.0.0.1",
	Port:           4224,
	NoLog:          true,
	NoSigs:         true,
	MaxControlLine: 256,
}

// https://github.com/nats-io/gnatsd/blob/master/test/test.go
// RunDefaultServer starts a new Go routine based server using the default options
func RunDefaultServer() *gnatsd.Server {
	return RunServer(&DefaultTestOptions)
}

// RunServer starts a new Go routine based server
func RunServer(opts *gnatsd.Options) *gnatsd.Server {
	if opts == nil {
		opts = &DefaultTestOptions
	}
	natsMu.Lock()
	defer natsMu.Unlock()
	natsServer = gnatsd.New(opts)
	if natsServer == nil {
		panic("No NATS Server object returned.")
	}
	// Run server in Go routine.
	go natsServer.Start()
	// Wait for accept loop(s) to be started
	if !natsServer.ReadyForConnections(3 * time.Second) {
		panic("Unable to start NATS Server in Go Routine")
	}
	return natsServer
}

var natsServer *gnatsd.Server
var natsMu sync.Mutex

func stopNATS() {
	natsMu.Lock()
	defer natsMu.Unlock()
	if natsServer != nil {
		natsServer.Shutdown()
	}
}
func TestNATS(t *testing.T) {

	time.AfterFunc(time.Second*3, func() {
		go RunDefaultServer()
		time.Sleep(time.Second)
	})

	natsAddr := "nats://" + DefaultTestOptions.Host + ":" +
		strconv.Itoa(DefaultTestOptions.Port)
	natsEvt := "dsiem_events"
	natsBp := "dsiem_overload_signals"

	fmt.Println("using natsAddr:", natsAddr)
	var err error
	// receiver
	var (
		r        *Transport
		rEvtChan <-chan event.NormalizedEvent
		rErrChan <-chan error
		rBpChan  chan<- bool
	)
	for i := 0; i < 10; i++ {
		r = New()
		r.NatsAddr = natsAddr
		rEvtChan = r.Receive(natsEvt)
		rErrChan = r.ErrChan()
		rBpChan = r.SendBool(natsBp)
		select {
		case err = <-rErrChan:
		default:
		}
		if err == nil {
			break
		}
		fmt.Println("error while initializing receiver:", err.Error(), "attempted #", i+1)
		time.Sleep(time.Second)
		err = nil
	}
	if err != nil {
		t.Fatal("Error in initializing receiver: ", err)
	}
	stopNATS()
	time.AfterFunc(time.Second*3, func() {
		go RunDefaultServer()
		time.Sleep(time.Second)
	})
	defer stopNATS()

	// sender
	var (
		s        *Transport
		sEvtChan chan<- event.NormalizedEvent
		sErrChan <-chan error
		sBpChan  <-chan bool
	)
	for i := 0; i < 10; i++ {
		s = New()
		s.NatsAddr = natsAddr
		sEvtChan = s.Send(natsEvt)
		sErrChan = s.ErrChan()
		sBpChan = s.ReceiveBool(natsBp)
		select {
		case err = <-sErrChan:
		default:
		}
		if err == nil {
			break
		}
		fmt.Println("error while initializing sender:", err.Error(), "attempted #", i+1)
		time.Sleep(time.Second)
		err = nil
	}
	if err != nil {
		t.Fatal("Error in initializing sender: ", err)
	}

	sEvt := event.NormalizedEvent{ConnID: 1}
	fmt.Println("Sending to bool chan")
	rBpChan = r.SendBool(natsBp)
	rBpChan <- true
	fmt.Println("Receiving from bool chan")
	sBpChan = s.ReceiveBool(natsBp)
	sBp := <-sBpChan
	fmt.Println("Sending to evt chan")
	sEvtChan = s.Send(natsEvt)
	sEvtChan <- sEvt
	fmt.Println("Receiving from evt chan")
	rEvtChan = r.Receive(natsEvt)
	rEvt := <-rEvtChan

	if rEvt.ConnID != 1 {
		t.Fatal("Expected ConnID: 1, actual:", rEvt.ConnID)
	}
	if !sBp {
		t.Fatal("Expected sBp: true, actual:", sBp)
	}

	rDone := r.Done()
	r.Stop()
	<-rDone
	sDone := s.Done()
	s.Stop()
	<-sDone

}
