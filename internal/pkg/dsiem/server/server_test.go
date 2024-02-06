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

package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/limiter"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"
	"github.com/valyala/fasthttp"

	gnatsd "github.com/nats-io/nats-server/v2/server"
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
	natsMu.Lock()
	defer natsMu.Unlock()
	natsServer = gnatsd.New(opts)
	if natsServer == nil {
		panic("No NATS Server object returned.")
	}

	// Run server in Go routine.
	go natsServer.Start()

	// Wait for accept loop(s) to be started
	if !natsServer.ReadyForConnections(5 * time.Second) {
		panic("Unable to start NATS Server in Go Routine")
	}
	return natsServer
}

var natsServer *gnatsd.Server
var natsMu sync.Mutex

var testErrChan chan error
var testBpChan chan bool

func initServer(cfg Config, t *testing.T, expectError bool) {
	// use bidirectional channel to simulate err
	cmu.Lock()
	testErrChan = make(chan error, 1)
	testBpChan = make(chan bool, 1)
	cmu.Unlock()
	cfg.ErrChan = testErrChan
	cfg.BpChan = testBpChan
	err := Start(cfg)
	if !expectError && err != nil {
		// t.Fatal("server start return error: " + err.Error())
		// fmt.Println("server start return error: " + err.Error())
		t.Fatal("server start return error:", err)
	}
	if expectError && err == nil {
		// t.Fatal("error expected but start returns nil")
		t.Fatal("error expected but start returns nil")
	}
}

func stopServer(t *testing.T) {
	err := Stop()
	if err != nil {
		t.Fatal("cannot stop server, receive err: ", err)
	}
	fmt.Println("server stopped.")
}

func stopNats(t *testing.T) {
	natsMu.Lock()
	defer natsMu.Unlock()
	if natsServer != nil {
		natsServer.Shutdown()
	}
	return
}

func TestServerStartupAndFileServer(t *testing.T) {
	d, err := test.DirEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	fixDir := path.Join(d, "internal", "pkg", "dsiem", "server", "fixtures")

	//	fServer.ReadTimeout = time.Second * 1

	var cfg Config
	cfg.EvtChan = make(chan event.NormalizedEvent)
	cfg.MsqCluster = "nats://127.0.0.1:4222"
	cfg.MsqPrefix = "dsiem"
	cfg.NodeName = "nodename"
	cfg.Addr = "127.0.0.1"
	cfg.Port = 8080
	cfg.WebSocket = false
	cfg.Pprof = false
	cfg.Webd = path.Join(fixDir, "web")
	cfg.WriteableConfig = true

	cfg.Confd = `\/\/\/\/`
	cfg.Mode = "cluster-frontend"

	// first reach the server to msq connection error handling code
	time.AfterFunc(time.Second, func() {
		RunDefaultServer()
	})
	initServer(cfg, t, false)
	stopServer(t)
	defer stopNats(t)

	// test port already in use handling
	fakeServer := fasthttp.Server{}
	go func() {
		fakeServer.ListenAndServe(cfg.Addr + ":" + strconv.Itoa(cfg.Port))
	}()
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT)
	go func() {
		<-signalChan
		signalChan <- syscall.SIGINT
		signal.Stop(signalChan)
	}()
	initServer(cfg, t, false)
	if len(signalChan) == 0 {
		t.Error("Expected to receive an interrupt signal due to port already in use")
	}
	stopServer(t)
	fakeServer.Shutdown()
	time.Sleep(time.Second)

	// now start the server for real
	initServer(cfg, t, false)
	time.Sleep(time.Second) // without this test for 500 below often returns 200

	url := "http://" + cfg.Addr + ":" + strconv.Itoa(cfg.Port)

	// test for few errs first
	httpTest(t, url+"/config", "GET", "", 500)
	httpTest(t, url, "GET", "", 404)
	httpTest(t, url+"/config/intel_wise.json", "GET", "", 400)
	httpTest(t, url+"/config/valid.json", "POST", "{}", 500)

	stopServer(t)

	// re-init frontend with extra params
	cfg.Confd = path.Join(fixDir, "configs")
	cfg.MaxEPS = 1000
	cfg.MinEPS = 100
	initServer(cfg, t, false)
	time.Sleep(time.Second * 4)

	httpTest(t, url+"/config/payload.exe", "POST", "zpl01t", 418)
	httpTest(t, url+"/config/valid.json", "POST", "{}", 201)
	httpTest(t, url+"/config/intel_hack.json", "POST", "{ \"foo\":\"bar\"}", 418)
	httpTest(t, url+"/config/vuln_hack.json", "POST", "{ \"foo\":\"bar\"}", 418)
	httpTest(t, url+"/config/directives_hack.json", "POST", "{ \"foo\":\"bar\"}", 418)
	httpTest(t, url+"/config/assets_hack.json", "POST", "{ \"foo\":\"bar\"}", 418)
	httpTest(t, url+"/config/payload.exe", "DELETE", "", 418)
	httpTest(t, url+"/config/valid.json", "DELETE", "", 200)
	httpTest(t, url+"/config/doesntexist.json", "DELETE", "", 400)

	httpTest(t, url+"/config/", "GET", "", 200)
	httpTest(t, url+"/config/intel_wise.json", "GET", "", 200)
	httpTest(t, url+"/config/doesntexist.json", "GET", "", 400)
	httpTest(t, url+"/config/a.json", "GET", "", 400)
	httpTest(t, url+"/config/payload.exe", "GET", "", 418)
	httpTest(t, url+"/config/dir/asdad/", "GET", "", 404)

	httpTest(t, url+"/config/test.json.foo", "GET", "", 418)
	httpTest(t, url+"/config/..dot_in_fname.json", "GET", "", 418)
	httpTest(t, url+"/config/ʘunicode.json", "GET", "", 418)
	httpTest(t, url+"/config/_.json", "GET", "", 418)

	stopServer(t)

	cfg.Mode = "cluster-backend"
	initServer(cfg, t, false)
	stopServer(t)

	cfg.Mode = "standalone"
	initServer(cfg, t, false)
	fmt.Println("sending true to bpCh")
	cmu.Lock()
	testBpChan <- true
	cmu.Unlock()

	stopServer(t)

	// expected errors on server startup

	cfg.MinEPS = 2000
	initServer(cfg, t, true)

	cfg.Port = 0
	initServer(cfg, t, true)

	cfg.Addr = "wrong"
	initServer(cfg, t, true)
	stopServer(t)
}

func httpTest(t *testing.T, url, method, data string, expectedStatusCode int) {
	_, code, err := httpClient(url, method, data)
	if err != nil {
		t.Fatal("Error received from httpClient", url, ":", err)
	}
	if code != expectedStatusCode && expectedStatusCode != 500 {
		t.Fatal("Received", code, "from", url, "expected", expectedStatusCode)
	}
	if code != expectedStatusCode && expectedStatusCode == 500 {
		fmt.Println("Flaky server test result detected, for", url, "retrying for 10 times every 2 sec ..")
		for i := 0; i < 10; i++ {
			fmt.Println("attempt ", i+1, "..")
			_, code, err := httpClient(url, method, data)
			if err != nil {
				t.Fatal("Flaky test workaround receive error from httpClient", url, ":", err)
			}
			if code == expectedStatusCode {
				return
			}
			time.Sleep(time.Second * 2)
		}
		t.Fatal("Flaky test received", code, "from", url, "expected", expectedStatusCode)
	}
}

func httpClient(url, method, data string) (out string, statusCode int, err error) {
	client := &http.Client{}
	r := strings.NewReader(data)
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	out = string(body)
	statusCode = resp.StatusCode
	return
}

func TestRateFuncs(t *testing.T) {
	var c *fasthttp.RequestCtx
	f := func(c *fasthttp.RequestCtx) {}
	cmu.Lock()
	epsLimiter, _ = limiter.New(1, 1)
	cmu.Unlock()
	h := rateLimit(10, time.Duration(time.Second), f)
	h(c)
	v := connCounter
	increaseConnCounter()
	r := connCounter - v
	if r != 1 {
		t.Fatal("Expected connCounter to be increased by 1")
	}
	n := CounterRate()
	if n < 0 {
		t.Fatal("Expected rate to be at least 0")
	}

}
