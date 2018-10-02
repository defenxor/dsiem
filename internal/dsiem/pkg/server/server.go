package server

import (
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"encoding/json"
	"errors"
	"expvar"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"regexp"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/matryer/vice/queues/nats"

	"github.com/fasthttp-contrib/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"
	"github.com/valyala/fasthttp/pprofhandler"

	"github.com/valyala/fasthttp/reuseport"

	"context"

	"github.com/buaazp/fasthttprouter"
	"golang.org/x/time/rate"

	rc "github.com/paulbellamy/ratecounter"
)

var connCounter uint64
var webDir, confDir string
var rateCounter *rc.RateCounter
var wss *wsServer
var upgrader websocket.Upgrader
var mode string
var transport nats.Transport
var sender chan<- []byte
var errchan <-chan error
var msq string
var epsCounter = expvar.NewInt("eps_counter")
var eventChannel chan<- event.NormalizedEvent
var overloadFlag bool

type configFile struct {
	Filename string `json:"filename"`
}
type configFiles struct {
	Files []configFile `json:"files"`
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

// Start starts the server
func Start(ch chan<- event.NormalizedEvent, bpCh <-chan bool, confd string, webd string,
	serverMode string, maxEPS int, msqURL string, msqCluster string, msqPrefix string, nodeName string, addr string, port int) error {

	if a := net.ParseIP(addr); a == nil {
		return errors.New(addr + " is not a valid IP address")
	}
	if port < 1 || port > 65535 {
		return errors.New("Invalid TCP port number")
	}

	mode = serverMode
	msq = msqCluster

	if mode == "cluster-frontend" {
		initMsgQueue(msqURL, msq, msqPrefix, nodeName)
	}

	// no need to check this, toctou issue
	confDir = confd
	webDir = webd

	eventChannel = ch
	rateCounter = rc.NewRateCounter(1 * time.Second)
	p := strconv.Itoa(port)

	log.Info(log.M{Msg: "Server listening on " + addr + ":" + p})
	initWSServer()
	initEPSTicker()
	initOverLoadDetector(bpCh)

	router := fasthttprouter.New()
	router.GET("/config/:filename", handleConfFileDownload)
	router.GET("/config/", handleConfFileList)
	router.GET("/debug/vars/", expVarHandler)
	router.GET("/debug/pprof/:name", pprofHandler)
	router.GET("/debug/pprof/", pprofHandler)
	router.POST("/config/:filename", handleConfFileUpload)
	if mode != "cluster-backend" {
		if maxEPS == 0 {
			router.POST("/events", handleEvents)
		} else {
			router.POST("/events", rateLimit(maxEPS, maxEPS, 3*time.Second, handleEvents))
		}
		router.GET("/eps/", wsHandler)
		router.ServeFiles("/ui/*filepath", webDir)
	}
	ln, err := reuseport.Listen("tcp4", addr+":"+p)
	if err != nil {
		return err
	}

	err = fasthttp.Serve(ln, router.Handler)
	return err
}

func initOverLoadDetector(ch <-chan bool) {
	go func() {
		for {
			m := <-ch
			if m != overloadFlag {
				log.Info(log.M{Msg: "Received overload status change from " + strconv.FormatBool(overloadFlag) +
					" to " + strconv.FormatBool(m) + " from backend"})
			}
			overloadFlag = m
		}
	}()
}

func rateLimit(rps, burst int, wait time.Duration, h fasthttp.RequestHandler) fasthttp.RequestHandler {
	l := rate.NewLimiter(rate.Limit(rps), burst)

	return func(c *fasthttp.RequestCtx) {
		// create a new context from the request with the wait timeout
		ctx, cancel := context.WithTimeout(context.Background(), wait)
		defer cancel() // always cancel the context!

		// Wait errors out if the request cannot be processed within
		// the deadline. This is preemptive, instead of waiting the
		// entire duration.
		if err := l.Wait(ctx); err != nil {
			fmt.Fprintf(c, "Too many requests\n")
			c.SetStatusCode(fasthttp.StatusTooManyRequests)
			return
		}
		h(c)
	}
}

func initMsgQueue(msqURL string, msq string, prefix, nodeName string) {
	opt := nats.WithStreaming(msq, prefix+"-"+nodeName)
	transport := nats.New(opt)
	transport.NatsAddr = msqURL
	sender = transport.Send(prefix + "_" + "events")
	errchan = transport.ErrChan()
}

func wsHandler(ctx *fasthttp.RequestCtx) {
	upgrader = websocket.New(wss.onClientConnected)
	err := upgrader.Upgrade(ctx)
	if err != nil {
		log.Warn(log.M{Msg: "error returned from websocket: " + err.Error()})
	}
}

func pprofHandler(ctx *fasthttp.RequestCtx) {
	pprofhandler.PprofHandler(ctx)
}

func expVarHandler(ctx *fasthttp.RequestCtx) {
	expvarhandler.ExpvarHandler(ctx)
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

func increaseConnCounter() uint64 {
	atomic.AddUint64(&connCounter, 1)
	i := atomic.LoadUint64(&connCounter)
	return i
}

func handleConfFileList(ctx *fasthttp.RequestCtx) {
	clientAddr := ctx.RemoteAddr().String()
	log.Info(log.M{Msg: "Request for list of configuration files from " + clientAddr})

	files, err := ioutil.ReadDir(confDir)
	if err != nil {
		fmt.Fprintf(ctx, "Error reading config directory")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	c := configFiles{}

	for _, f := range files {
		c.Files = append(c.Files, configFile{f.Name()})
	}
	byteVal, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		fmt.Fprintf(ctx, "Error reading config file names")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	_, err = ctx.Write(byteVal)
	return
}

func handleConfFileDownload(ctx *fasthttp.RequestCtx) {
	clientAddr := ctx.RemoteAddr().String()
	filename := ctx.UserValue("filename").(string)
	if filename == "" {
		fmt.Fprintf(ctx, "requires /config/filename\n")
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	if !isCfgFileNameValid(filename) {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding."})
		fmt.Fprintf(ctx, "Not a valid filename, should be in any_N4m3_you_want.json format\n")
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		return
	}

	log.Info(log.M{Msg: "Request for file '" + filename + "' from " + clientAddr})
	f := path.Join(confDir, filename)
	log.Info(log.M{Msg: "Getting file " + f})

	if !fs.FileExist(f) {
		fmt.Fprintf(ctx, filename+" doesn't exist\n")
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}
	file, err := os.Open(f)
	if err != nil {
		fmt.Fprintf(ctx, "cannot open "+filename)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Fprintf(ctx, "cannot open "+filename)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	_, err = ctx.Write(byteValue)
	return
}

func handleConfFileUpload(ctx *fasthttp.RequestCtx) {
	clientAddr := ctx.RemoteAddr().String()
	filename := ctx.UserValue("filename").(string)
	if filename == "" {
		fmt.Fprintf(ctx, "requires /config/filename\n")
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	if !isCfgFileNameValid(filename) {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding."})
		fmt.Fprintf(ctx, "Not a valid filename, should be in any_N4m3_you_want.json format\n")
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		return
	}

	log.Info(log.M{Msg: "Upload file request for '" + filename + "' from " + clientAddr})
	file := path.Join(confDir, filename)
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	defer f.Close()
	if err != nil {
		fmt.Fprintf(ctx, "Cannot open target file location\n")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	_, err = f.Write(ctx.PostBody())
	if err != nil {
		fmt.Fprintf(ctx, "Cannot write to target file location\n")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	fmt.Fprintf(ctx, "File "+filename+" uploaded successfully\n")
	ctx.SetStatusCode(fasthttp.StatusCreated)
	return
}

func isCfgFileNameValid(filename string) (ok bool) {
	r, err := regexp.Compile(`[a-zA-Z0-9_]+.json`)
	if err != nil {
		return
	}
	ok = r.MatchString(filename)
	return
}

func handleEvents(ctx *fasthttp.RequestCtx) {

	clientAddr := ctx.RemoteAddr().String()
	connID := increaseConnCounter()
	rateCounter.Incr(1)

	if overloadFlag {
		log.Info(log.M{Msg: "Overload condition, rejecting request from " + clientAddr, CId: connID})
		fmt.Fprintf(ctx, "backend overloaded\n")
		ctx.SetStatusCode(fasthttp.StatusTooManyRequests)
		return
	}

	evt := event.NormalizedEvent{}

	msg := ctx.PostBody()
	err := evt.FromBytes(msg)
	// err := gojay.Unmarshal(ctx.PostBody(), &evt)

	if err != nil {
		log.Warn(log.M{Msg: "Cannot parse normalizedEvent from " + clientAddr + ". err: " + err.Error(), CId: connID})
		fmt.Fprintf(ctx, "Cannot parse the submitted event\n")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	if !evt.Valid() {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding.", CId: connID})
		fmt.Fprintf(ctx, "Not a valid event\n")
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		return
	}

	evt.ConnID = connID
	evt.RcvdTime = time.Now().Unix()

	log.Debug(log.M{Msg: "Received event ID: " + evt.EventID, CId: connID})

	if mode == "standalone" {
		// push the event, timeout in 10s to avoid open fd overload
		select {
		case <-time.After(10 * time.Second):
			log.Info(log.M{Msg: "event channel timed out!", CId: connID})
			ctx.SetStatusCode(fasthttp.StatusRequestTimeout)
		case eventChannel <- evt:
			log.Debug(log.M{Msg: "Event pushed", CId: connID})
		}
		return
	}

	// mode = cluster-frontend

	// TODO: replace this, inefficient but needed to keep connID and rcvdTime in msg
	bEvt, err := evt.ToBytes()
	if err != nil {
		log.Warn(log.M{Msg: "Cannot convert event from " + clientAddr + " to message queue format. err: " + err.Error(), CId: connID})
		fmt.Fprintf(ctx, "Cannot parse the submitted event\n")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}

	select {
	case <-time.After(10 * time.Second):
		log.Info(log.M{Msg: "message queue timed out!", CId: connID})
		ctx.SetStatusCode(fasthttp.StatusRequestTimeout)
	case sender <- bEvt:
		log.Debug(log.M{Msg: "Event pushed", CId: connID})
		//fmt.Println("msg pushed:\n", string(msg))
	case err := <-errchan:
		log.Info(log.M{Msg: "Error from message queue:" + err.Error(), CId: connID})
		// fmt.Println("msg queue error on sender: ", err)
	}
}
