package worker

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/server"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/vice/nats"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"

	gnatsd "github.com/nats-io/gnatsd/server"
)

// DefaultTestOptions are default options for the unit tests.
var DefaultTestOptions = gnatsd.Options{
	Host:           "127.0.0.1",
	Port:           4222,
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

var (
	ch        chan event.NormalizedEvent
	msq       string
	msqPrefix string
)

func initFrontend(d string, t *testing.T) {
	fixturesDir := path.Join(d, "internal", "pkg", "dsiem", "worker", "fixtures")

	c := server.Config{}
	c.BpChan = make(chan bool)
	c.Confd = path.Join(fixturesDir, "configs")
	c.Webd = path.Join()
	c.WriteableConfig = true
	c.Pprof = true
	c.Mode = "cluster-frontend"
	c.MaxEPS = 1000
	c.MinEPS = 100
	c.NodeName = "frontend"
	c.Addr = "127.0.0.1"
	c.Port = 8080

	go func() {
		if err := server.Start(c); err != nil {
			t.Error(err)
		}
	}()
	//	evt := event.NormalizedEvent{}
	//	ch <- evt
}

func TestWorker(t *testing.T) {
	d, err := test.DirEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	ch = make(chan event.NormalizedEvent)
	errChan = make(chan error)

	msq = "nats://127.0.0.1:4222"
	msqPrefix = "dsiem"

	go initFrontend(d, t)

	nodeName := "dsiem-backend-0"
	wd, err := ioutil.TempDir(os.TempDir(), "dsiem-worker")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(wd)

	frontend := "foo"
	if err := Start(ch, msq, msqPrefix, nodeName, wd, frontend); err == nil {
		t.Fatal("expect an error due to wrong frontend address:", frontend)
	}
	time.AfterFunc(time.Second, func() {
		go RunDefaultServer()
	})

	time.Sleep(time.Second * 5)

	frontend = "http://127.0.0.1:8080"
	if err = Start(ch, msq, msqPrefix, nodeName, wd, frontend); err != nil {
		t.Fatal("error during worker start:", err)
	}

	bp := GetBackPressureChannel()
	select {
	case bp <- true:
	default:
		t.Fatal("Cannot send to bp channel")
	}

	tr := nats.New()
	tr.NatsAddr = msq
	sendCh := tr.Send(msqPrefix + "_" + "events")

	sendCh <- event.NormalizedEvent{}
}

func TestErrors(t *testing.T) {
	err := downloadFile(`/\/\/\/`, "http://127.0.0.1:8080/config/assets_testing.json")
	if err == nil {
		t.Error("expected error due to wrong filepath")
	}
	err = downloadFile(os.TempDir(), "http://127.0.0.1:31337")
	if err == nil {
		t.Error("expected error due to wrong URL")
	}

	start := time.Now()
	handleMsqError(err)
	elapsed := time.Since(start)
	if elapsed < time.Second*3 {
		t.Error("expected to wait at least 3 seconds")
	}
}
