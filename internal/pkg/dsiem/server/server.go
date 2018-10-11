package server

import (
	"dsiem/internal/pkg/dsiem/event"
	log "dsiem/internal/pkg/shared/logger"
	"errors"
	"expvar"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"dsiem/internal/pkg/dsiem/limiter"
	"dsiem/internal/pkg/dsiem/vice/nats"

	"github.com/fasthttp-contrib/websocket"
	"github.com/valyala/fasthttp"

	"github.com/valyala/fasthttp/reuseport"

	"context"

	"github.com/buaazp/fasthttprouter"

	rc "github.com/paulbellamy/ratecounter"
)

var connCounter uint64
var webDir, confDir string
var rateCounter *rc.RateCounter
var wss *wsServer
var upgrader websocket.Upgrader
var mode string
var transport nats.Transport
var epsLimiter *limiter.Limiter

var errChan <-chan error
var eventChan chan<- event.NormalizedEvent
var bpChan <-chan bool

// var msq string
var epsCounter = expvar.NewInt("eps_counter")
var overloadFlag bool

// Start starts the server
func Start(ch chan<- event.NormalizedEvent, bpCh <-chan bool, confd string, webd string,
	serverMode string, maxEPS int, minEPS int, msqCluster string,
	msqPrefix string, nodeName string, addr string, port int) error {

	if a := net.ParseIP(addr); a == nil {
		return errors.New(addr + " is not a valid IP address")
	}
	if port < 1 || port > 65535 {
		return errors.New("Invalid TCP port number")
	}

	mode = serverMode
	// msq = msqCluster

	if mode == "cluster-frontend" {
		initMsgQueue(msqCluster, msqPrefix, nodeName)
	} else {
		eventChan = ch
		bpChan = bpCh
		errChan = nil
	}

	// no need to check this, toctou issue
	confDir = confd
	webDir = webd

	rateCounter = rc.NewRateCounter(1 * time.Second)
	p := strconv.Itoa(port)

	log.Info(log.M{Msg: "Server listening on " + addr + ":" + p})

	router := fasthttprouter.New()
	router.GET("/config/:filename", handleConfFileDownload)
	router.GET("/config/", handleConfFileList)
	router.GET("/debug/vars/", expVarHandler)
	router.GET("/debug/pprof/:name", pprofHandler)
	router.GET("/debug/pprof/", pprofHandler)
	router.POST("/config/:filename", handleConfFileUpload)
	router.DELETE("/config/:filename", handleConfFileDelete)

	if mode != "cluster-backend" {

		initWSServer()
		initEPSTicker()

		if maxEPS == 0 || minEPS == 0 {
			router.POST("/events", handleEvents)
		} else {
			var err error
			epsLimiter, err = limiter.New(maxEPS, minEPS)
			if err != nil {
				return err
			}
			router.POST("/events", rateLimit(epsLimiter.Limit(), 3*time.Second, handleEvents))
		}
		router.GET("/eps/", wsHandler)
		router.ServeFiles("/ui/*filepath", webDir)

		overloadManager()
	}
	ln, err := reuseport.Listen("tcp4", addr+":"+p)
	if err != nil {
		return err
	}

	err = fasthttp.Serve(ln, router.Handler)
	return err
}

func initEPSTicker() {
	ticker := time.NewTicker(time.Second)
	go func() {
		for {
			<-ticker.C
			epsCounter.Set(rateCounter.Rate())
		}
	}()
}

func increaseConnCounter() uint64 {
	atomic.AddUint64(&connCounter, 1)
	i := atomic.LoadUint64(&connCounter)
	return i
}

func overloadManager() {
	mu := sync.RWMutex{}
	detector := func() {
		for {
			m := <-bpChan
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
			for {
				res, current := 0, 0
				<-ticker.C
				if epsLimiter == nil {
					continue
				}
				current = epsLimiter.Limit()
				mu.RLock()
				if overloadFlag {
					res = epsLimiter.Lower()
				} else {
					res = epsLimiter.Raise()
				}
				if current != res {
					log.Info(log.M{Msg: "Overload status is " + strconv.FormatBool(overloadFlag) +
						", EPS limit changed from " + strconv.Itoa(current) + " to " + strconv.Itoa(res)})
				}
				mu.RUnlock()
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
		transport := nats.New()
		transport.NatsAddr = msq
		eventChan = transport.Send(prefix + "_" + "events")
		errChan = transport.ErrChan()
		bpChan = transport.ReceiveBool(prefix + "_" + "overload_signals")
		select {
		case err = <-errChan:
		default:
		}
		return err
	}
	for {
		err := initMsq()
		if err == nil {
			log.Info(log.M{Msg: "Successfully connected to message queue " + msq})
			break
		}
		log.Info(log.M{Msg: "Error from message queue " + err.Error()})
		log.Info(log.M{Msg: "Reconnecting in " + strconv.Itoa(reconnectSecond) + " seconds.."})
		time.Sleep(reconnectSecond * time.Second)
	}
}

func initWSServer() {
	wss = newWSServer()
	go func() {
		var c int
		for {
			c = len(wss.clients)
			if c == 0 {
				log.Debug(log.M{Msg: "WS server waiting for client connection."})
				// wait until new client connected
				<-wss.cConnectedCh
			}
			wss.sendAll(&message{rateCounter.Rate()})
			time.Sleep(250 * time.Millisecond)
		}
	}()
}
