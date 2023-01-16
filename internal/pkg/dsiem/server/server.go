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

package server

import (
	"errors"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/defenxor/dsiem/internal/pkg/shared/proc"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/limiter"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/vice/nats"

	"github.com/fasthttp-contrib/websocket"
	"github.com/valyala/fasthttp"

	"github.com/valyala/fasthttp/reuseport"

	"context"

	"github.com/buaazp/fasthttprouter"

	rc "github.com/paulbellamy/ratecounter"
)

var connCounter uint64

var wss *wsServer
var upgrader websocket.Upgrader
var transport *nats.Transport
var epsLimiter *limiter.Limiter

var (
	overloadFlag bool
	mu           sync.RWMutex
)

var rateCounter = rc.NewRateCounter(1 * time.Second)

var fServer fasthttp.Server

var (
	cmu sync.RWMutex
	c   Config
)

// Config define the struct used to initialize server
type Config struct {
	EvtChan         chan<- event.NormalizedEvent
	BpChan          <-chan bool
	StopChan        chan struct{}
	ErrChan         <-chan error
	Confd           string
	Webd            string
	WriteableConfig bool
	Pprof           bool
	Mode            string
	MaxEPS          int
	MinEPS          int
	MsqCluster      string
	MsqPrefix       string
	NodeName        string
	Addr            string
	Port            int
	WebSocket       bool
}

func init() {
	wss = newWSServer()
}

// Stop the server
func Stop() (err error) {
	time.Sleep(time.Second)
	cmu.Lock()
	if c.StopChan != nil {
		close(c.StopChan)
	}
	cmu.Unlock()
	err = fServer.Shutdown()
	return
}

// Start the server
func Start(cfg Config) (err error) {
	cmu.Lock()
	// these are used by other functions
	c.EvtChan = cfg.EvtChan
	c.StopChan = make(chan struct{})
	c.Mode = cfg.Mode
	c.Confd = cfg.Confd
	cmu.Unlock()

	if a := net.ParseIP(cfg.Addr); a == nil {
		err = errors.New(cfg.Addr + " is not a valid IP address")
		return
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		err = errors.New("Invalid TCP port number")
		return
	}

	if c.Mode == "cluster-frontend" {
		initMsgQueue(cfg.MsqCluster, cfg.MsqPrefix, cfg.NodeName)
	} else {
		cmu.Lock()
		c.EvtChan = cfg.EvtChan
		c.BpChan = cfg.BpChan
		c.ErrChan = cfg.ErrChan
		cmu.Unlock()
	}

	p := strconv.Itoa(cfg.Port)

	log.Info(log.M{Msg: "Server mode " + c.Mode + " listening on " + cfg.Addr + ":" + p})

	router := fasthttprouter.New()
	router.GET("/config/:filename", handleConfFileDownload)
	router.GET("/config/", handleConfFileList)
	router.GET("/debug/vars/", expVarHandler)
	if cfg.Pprof {
		router.GET("/debug/pprof/:name", pprofHandler)
		router.GET("/debug/pprof/", pprofHandler)
	}
	if cfg.WriteableConfig {
		router.POST("/config/:filename", handleConfFileUpload)
		router.DELETE("/config/:filename", handleConfFileDelete)
	}

	if c.Mode != "cluster-backend" {
		if cfg.WebSocket {
			initWSServer()
			router.GET("/eps/", wsHandler)
		}
		if cfg.MaxEPS == 0 || cfg.MinEPS == 0 {
			router.POST("/events", handleEvents)
		} else {
			// reuse cmu lock
			cmu.Lock()
			epsLimiter, err = limiter.New(cfg.MaxEPS, cfg.MinEPS)
			cmu.Unlock()
			if err != nil {
				return
			}
			router.POST("/events", rateLimit(epsLimiter.Limit(), 3*time.Second, handleEvents))
		}
		router.ServeFiles("/ui/*filepath", cfg.Webd)
		overloadManager()
	}
	// just reuse the lock here
	cmu.Lock()
	fServer.Handler = router.Handler
	fServer.Name = "dsiem"

	// fasthttp default is 4MB, change to 100MB since directive file can be larger than 50MB
	fServer.MaxRequestBodySize = 100 * 1024 * 1024

	cmu.Unlock()

	go func() {
		// ListenAndServe may panic due to already in use port,
		// but its ok to quit ASAP in that case
		defer func() {
			if r := recover(); r != nil {
				log.Error(log.M{Msg: fmt.Sprintf("Unable to listen and serve, perhaps the port is already-in-use?, %s", r)})
				proc.StopProcess(proc.GetProcID())
			}
		}()
		if runtime.GOOS == "windows" {
			if err := fServer.ListenAndServe(cfg.Addr + ":" + p); err != nil {
				log.Error(log.M{Msg: fmt.Sprintf("serve error, %s", err)})
				return
			}

		} else {
			ln, err := reuseport.Listen("tcp4", cfg.Addr+":"+p)
			if err != nil {
				log.Error(log.M{Msg: fmt.Sprintf("unable to reuse port %s, %s", cfg.Addr+":"+p, err)})
				if isAlreadyInUseError(err) {
					proc.StopProcess(proc.GetProcID())
				}

				return
			}

			if err := fServer.Serve(ln); err != nil {
				log.Error(log.M{Msg: fmt.Sprintf("serve error, %s", err)})
				return
			}
		}
		// for some reason, using err.Error() here causes fasthttp.server Shutdown in Stop() to exit
		// during test
		log.Info(log.M{Msg: "Server process exited."})
	}()
	time.Sleep(time.Second)
	return
}

// CounterRate return the rate of EPS
func CounterRate() int64 {
	return rateCounter.Rate()
}

func increaseConnCounter() uint64 {
	atomic.AddUint64(&connCounter, 1)
	i := atomic.LoadUint64(&connCounter)
	return i
}

func overloadManager() {
	detector := func() {
		var m bool
		cmu.RLock()
		stopCh := c.StopChan
		bpCh := c.BpChan
		cmu.RUnlock()
		for {
			select {
			case <-stopCh:
				return
			case m = <-bpCh:
			}
			if m != overloadFlag {
				log.Info(log.M{Msg: "Received overload status change from " + strconv.FormatBool(overloadFlag) +
					" to " + strconv.FormatBool(m) + " from backend"})
			}
			mu.Lock()
			overloadFlag = m
			mu.Unlock()
		}
	}
	modifier := func() {
		ticker := time.NewTicker(5 * time.Second)
		go func() {
			cmu.RLock()
			stopCh := c.StopChan
			cmu.RUnlock()
			for {
				res, current := 0, 0
				select {
				case <-stopCh:
					ticker.Stop()
					return
				case <-ticker.C:
				}
				<-ticker.C
				cmu.RLock()
				if epsLimiter == nil {
					cmu.RUnlock()
					continue
				}
				cmu.RUnlock()
				current = epsLimiter.Limit()
				mu.Lock()
				if overloadFlag {
					res = epsLimiter.Lower()
				} else {
					res = epsLimiter.Raise()
				}
				if current != res {
					log.Info(log.M{Msg: "Overload status is " + strconv.FormatBool(overloadFlag) +
						", EPS limit changed from " + strconv.Itoa(current) + " to " + strconv.Itoa(res)})
				}
				mu.Unlock()
			}
		}()
	}
	go detector()
	go modifier()
}

func rateLimit(rps int, wait time.Duration, h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(c *fasthttp.RequestCtx) {
		// create a new context from the request with the wait timeout
		ctx, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel()
		// Wait errors out if the request cannot be processed within
		// the deadline. This is preemptive, instead of waiting the
		// entire duration.
		if err := epsLimiter.Wait(ctx); err != nil {
			fmt.Fprintf(c, "Too many requests\n")
			c.SetStatusCode(fasthttp.StatusTooManyRequests)
			return
		}
		h(c)
	}
}

func initMsgQueue(msq string, prefix, nodeName string) {
	const reconnectSecond = 3
	initMsq := func() (err error) {
		transport = nats.New()
		transport.NatsAddr = msq
		cmu.Lock()
		c.EvtChan = transport.Send(prefix + "_" + "events")
		c.ErrChan = transport.ErrChan()
		c.BpChan = transport.ReceiveBool(prefix + "_" + "overload_signals")
		errCh := c.ErrChan
		cmu.Unlock()
		select {
		case err = <-errCh:
		default:
		}
		return err
	}
	for {
		err := initMsq()
		if err != nil {
			log.Info(log.M{Msg: "Error from message queue " + err.Error()})
			log.Info(log.M{Msg: "Reconnecting in " + strconv.Itoa(reconnectSecond) + " seconds.."})
			time.Sleep(reconnectSecond * time.Second)
			continue
		}
		log.Info(log.M{Msg: "Successfully connected to message queue " + msq})
		break
	}
}

func initWSServer() {
	go func() {
		var cl int
		for {
			wss.Lock()
			cl = len(wss.clients)
			wss.Unlock()
			cmu.RLock()
			stopCh := c.StopChan
			wsChan := wss.cConnectedCh
			cmu.RUnlock()
			if cl == 0 {
				log.Debug(log.M{Msg: "WS server waiting for client connection."})
				// wait until new client connected
				select {
				case <-stopCh:
					break
				case <-wsChan:
				}
			}
			wss.sendAll(&message{rateCounter.Rate()})
			time.Sleep(250 * time.Millisecond)
		}
	}()
}

// FIXME: this is an inefficient way to check wether the error is address-alread-in-use error.
func isAlreadyInUseError(err error) bool {
	return strings.Contains(err.Error(), "address already in use")
}
