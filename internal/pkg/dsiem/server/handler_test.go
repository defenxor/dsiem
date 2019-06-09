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
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fasthttp-contrib/websocket"
	uuid "github.com/satori/go.uuid"
	"github.com/valyala/fasthttp"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/test"
)

func TestServerHandlers(t *testing.T) {
	d, err := test.DirEnv(true)
	if err != nil {
		t.Fatal(err)
	}

	if !log.TestMode {
		t.Logf("Enabling log test mode")
		log.EnableTestingMode()
	}

	apm.Enable(true)
	evtChanTimeoutSecond = time.Duration(time.Second)

	fServer.ReadTimeout = time.Second * 3
	fixDir := path.Join(d, "internal", "pkg", "dsiem", "server", "fixtures")

	var cfg Config
	cfg.EvtChan = make(chan event.NormalizedEvent, 1)
	cfg.NodeName = "nodename"
	cfg.Addr = "127.0.0.1"
	cfg.Port = 8080
	cfg.BpChan = make(chan bool)
	cfg.Webd = path.Join(fixDir, "web")
	cfg.WriteableConfig = true
	cfg.Pprof = true
	cfg.WebSocket = true

	cfg.Mode = "standalone"
	initServer(cfg, t, false)

	url := "http://" + cfg.Addr + ":" + strconv.Itoa(cfg.Port)

	verifyFuncOutput(t, func() {
		wsHandler(&fasthttp.RequestCtx{})
	}, "error returned from websocket", true)

	if err = testWs(); err != nil {
		t.Error("websocket client error:", err)
	}

	httpTest(t, url+"/debug/vars", "GET", "", 200)
	httpTest(t, url+"/debug/pprof", "GET", "", 200)

	e := &event.NormalizedEvent{}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}

	url = "http://127.0.0.1:8080/events/"

	httpTest(t, url, "POST", "zpl017", 500)
	httpTest(t, url, "POST", string(b), 418)

	e.EventID = genUUID()
	e.Sensor = "sensor1"
	e.SrcIP = "10.0.0.1"
	e.DstIP = "8.8.8.8"
	e.PluginID = 1001
	e.PluginSID = 1

	e.Timestamp = "im a string"
	b, err = json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	httpTest(t, url, "POST", string(b), 400)

	e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	b, err = json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	httpTest(t, url, "POST", string(b), 200)

	// should timeout, no more buffer/listener to receive the channel
	httpTest(t, url, "POST", string(b), 408)

	// should return service, unavailable due to msq error
	testErrChan <- errors.New("test")
	httpTest(t, url, "POST", string(b), 503)

	stopServer(t)
}

func genUUID() string {
	u, err := uuid.NewV4()
	if err != nil {
		return "static-id-doesnt-really-matter"
	}
	return u.String()
}

func testWs() (err error) {
	u, err := url.Parse("ws://127.0.0.1:8080/eps/")
	if err != nil {
		return
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return
	}
	defer c.Close()
	_, msg, err := c.ReadMessage()
	if err != nil {
		return
	}
	var e message
	err = json.Unmarshal(msg, &e)
	return
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
