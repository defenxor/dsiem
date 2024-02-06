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

package worker

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/server"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/vice/nats"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"

	gnatsd "github.com/nats-io/nats-server/v2/server"
)

// DefaultTestOptions are default options for the unit tests.
var DefaultTestOptions = gnatsd.Options{
	Host:           "127.0.0.1",
	Port:           4223,
	NoLog:          true,
	NoSigs:         true,
	MaxControlLine: 256,
}

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

var (
	ch        chan event.NormalizedEvent
	msq       string
	msqPrefix string
)

func initFrontend(d string, t *testing.T) {
	fixturesDir := path.Join(d, "internal", "pkg", "dsiem", "worker", "fixtures")

	c := server.Config{}
	c.BpChan = make(chan bool)
	c.MsqCluster = "nats://" + DefaultTestOptions.Host + ":" + strconv.Itoa(DefaultTestOptions.Port)
	c.Confd = path.Join(fixturesDir, "configs")
	c.Webd = path.Join()
	c.WriteableConfig = true
	c.Pprof = true
	c.Mode = "cluster-frontend"
	c.MaxEPS = 1000
	c.MinEPS = 100
	c.NodeName = "frontend"
	c.Addr = "127.0.0.1"
	c.Port = 8090

	if err := server.Start(c); err != nil {
		t.Fatal("Cannot start frontend server:", err)
	}
	time.Sleep(time.Second)
}

func cleanUp(t *testing.T) {
	fmt.Println("worker cleaning up..")
	natsMu.Lock()
	defer natsMu.Unlock()
	if natsServer != nil {
		//fmt.Println("Shutting down NATS server")
		natsServer.Shutdown()
		//fmt.Println("Done shutting down NATS server")
	}
	fmt.Println("Stopping server")
	if err := server.Stop(); err != nil {
		t.Fatal(err)
	}
	fmt.Println("Server stopped")
	return
}

func TestWorker(t *testing.T) {
	d, err := test.DirEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	ch = make(chan event.NormalizedEvent)

	msq = "nats://127.0.0.1:4223"
	msqPrefix = "dsiem"

	time.AfterFunc(time.Second, func() {
		go RunDefaultServer()
	})
	initFrontend(d, t)
	defer cleanUp(t)

	nodeName := "dsiem-backend-0"
	wd, err := os.MkdirTemp(os.TempDir(), "dsiem-worker")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(wd)

	frontend := "foo"
	if err := Start(ch, msq, msqPrefix, nodeName, wd, frontend); err == nil {
		t.Fatal("expect an error due to failure to download from a wrong frontend address:", frontend)
	}

	// trigger err first
	transport = nats.New()
	transport.SimulateError(errors.New("error during NATS initialization"))
	if errOccurred := initMsgQueue(msq, msqPrefix, nodeName); !errOccurred {
		t.Fatal("expect error to occur during NATS initialization")
	}
	transport = nil

	// use different port from the server_test
	frontend = "http://127.0.0.1:8090"
	// flaky test when run together with server test, so retry it 3x
	for i := 0; i < 10; i++ {
		err = Start(ch, msq, msqPrefix, nodeName, wd, frontend)
		if err != nil {
			fmt.Println("Flaky worker test result detected, attempted start so far:", i+1, "..")
			time.Sleep(time.Second * 2)
		} else {
			break
		}
	}
	if err != nil {
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

	// start testing for errors

	err = downloadFile(`/\/\/\/`, "http://127.0.0.1:8090/config/assets_testing.json")
	if err == nil {
		t.Error("expected error due to wrong filepath")
	}
	err = downloadFile(os.TempDir(), "http://127.0.0.1:31337")
	if err == nil {
		t.Error("expected error due to wrong URL")
	}

	go func() {
		transport.SimulateError(errors.New("test"))
	}()

	start := time.Now()
	handleMsqError(err)
	elapsed := time.Since(start)
	if elapsed < time.Second*3 {
		t.Error("expected to wait at least 3 seconds")
	}
}
