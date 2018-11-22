package server

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/limiter"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"
	"github.com/valyala/fasthttp"

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
	natsMu.Lock()
	defer natsMu.Unlock()
	natsServer = gnatsd.New(opts)
	if natsServer == nil {
		panic("No NATS Server object returned.")
	}

	// Run server in Go routine.
	go natsServer.Start()

	// Wait for accept loop(s) to be started
	if !natsServer.ReadyForConnections(15 * time.Second) {
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
		t.Fatal("server start return error: " + err.Error())
	}
	if expectError && err == nil {
		t.Fatal("error expected but start returns nil")
	}
}

func stopServer(t *testing.T) {
	err := Stop()
	if err != nil {
		t.Fatal(err)
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

	var cfg Config
	cfg.EvtChan = make(chan event.NormalizedEvent)
	cfg.MsqCluster = "nats://127.0.0.1:4222"
	cfg.MsqPrefix = "dsiem"
	cfg.NodeName = "nodename"
	cfg.Addr = "127.0.0.1"
	cfg.Port = 8080
	cfg.Webd = path.Join(fixDir, "web")
	cfg.WriteableConfig = true
	cfg.Pprof = true

	time.AfterFunc(time.Second, func() {
		fmt.Println("TIMEAFTER FUNC RUNNING NATS")
		go RunDefaultServer()
	})
	defer stopNats(t)

	fmt.Println("TESTING FAILED CONFIG")
	cfg.Confd = `/\/\/\/`
	cfg.Mode = "cluster-frontend"

	go initServer(cfg, t, false)
	// wait for msg queue to be connected
	time.Sleep(time.Second * 4)

	url := "http://" + cfg.Addr + ":" + strconv.Itoa(cfg.Port)

	// test for few errs first
	httpTest(t, url+"/config", "GET", "", 500)
	httpTest(t, url, "GET", "", 404)
	httpTest(t, url+"/config/intel_wise.json", "GET", "", 400)
	httpTest(t, url+"/config/valid.json", "POST", "{}", 500)

	stopServer(t)

	fmt.Println("REINIT WITH EXTRA PARAMS")
	// reinit frontend with extra params
	cfg.Confd = path.Join(fixDir, "configs")
	cfg.MaxEPS = 1000
	cfg.MinEPS = 100
	go initServer(cfg, t, false)
	time.Sleep(time.Second * 4)

	fmt.Println("ABT TO TEST PAYLOAD.EXE")
	httpTest(t, url+"/config/payload.exe", "POST", "zpl01t", 418)
	httpTest(t, url+"/config/valid.json", "POST", "{}", 201)
	httpTest(t, url+"/config/payload.exe", "DELETE", "", 418)
	httpTest(t, url+"/config/valid.json", "DELETE", "", 200)
	httpTest(t, url+"/config/doesntexist.json", "DELETE", "", 400)

	httpTest(t, url+"/config/", "GET", "", 200)
	httpTest(t, url+"/config/intel_wise.json", "GET", "", 200)
	httpTest(t, url+"/config/doesntexist.json", "GET", "", 400)
	httpTest(t, url+"/config/payload.exe", "GET", "", 418)
	httpTest(t, url+"/config/dir/asdad/", "GET", "", 404)

	stopServer(t)

	cfg.Mode = "cluster-backend"
	go initServer(cfg, t, false)
	stopServer(t)

	cfg.Mode = "standalone"
	go initServer(cfg, t, false)
	fmt.Println("sending true to bpCh")
	time.Sleep(time.Second)

	cmu.Lock()
	testBpChan <- true
	cmu.Unlock()
	time.Sleep(time.Second)

	stopServer(t)

	// expected errors on server startup

	cfg.MinEPS = 2000
	go initServer(cfg, t, true)
	time.Sleep(time.Second)

	cfg.Port = 0
	go initServer(cfg, t, true)
	time.Sleep(time.Second)

	cfg.Addr = "wrong"
	go initServer(cfg, t, true)
	time.Sleep(time.Second)
}

func httpTest(t *testing.T, url, method, data string, expectedStatusCode int) {
	_, code, err := httpClient(url, method, data)
	if err != nil {
		t.Fatal("Error received from httpClient", url, ":", err)
	}
	if code != expectedStatusCode {
		t.Fatal("Received", code, "from", url, "expected", expectedStatusCode)
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

	body, err := ioutil.ReadAll(resp.Body)
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
