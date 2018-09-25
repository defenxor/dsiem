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

	"github.com/fasthttp-contrib/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"

	"github.com/buaazp/fasthttprouter"

	rc "github.com/paulbellamy/ratecounter"
)

var connCounter uint64
var webDir, confDir string
var rateCounter *rc.RateCounter
var wss *wsServer
var upgrader websocket.Upgrader

type configFiles struct {
	FileName string `json:"filename"`
}

var epsCounter = expvar.NewInt("eps_counter")
var eventChannel chan<- event.NormalizedEvent

// StartFastHTTP start the server
func StartFastHTTP(ch chan<- event.NormalizedEvent, confd string, webd string, addr string, port int) error {

	if a := net.ParseIP(addr); a == nil {
		return errors.New(addr + " is not a valid IP address")
	}
	if port < 1 || port > 65535 {
		return errors.New("Invalid TCP port number")
	}

	// no need to check this, toctou issue
	confDir = confd
	webDir = webd

	eventChannel = ch
	rateCounter = rc.NewRateCounter(1 * time.Second)
	p := strconv.Itoa(port)

	for {
		log.Info(log.M{Msg: "Server listening on " + addr + ":" + p})
		initWSServer()
		router := fasthttprouter.New()
		router.POST("/events", handleEvents)
		router.GET("/config/:filename", handleConfFileDownload)
		router.GET("/config/", handleConfFileList)
		router.GET("/debug/vars/", expVarHandler)
		router.POST("/config/:filename", handleConfFileUpload)
		router.GET("/eps/", wsHandler)
		router.ServeFiles("/ui/*filepath", webDir)
		fasthttp.ListenAndServe(addr+":"+p, router.Handler)
	}
	return nil
}

func wsHandler(ctx *fasthttp.RequestCtx) {
	upgrader = websocket.New(wss.onClientConnected)
	err := upgrader.Upgrade(ctx)
	if err != nil {
		log.Warn(log.M{Msg: "error returned from websocket: " + err.Error()})
	}
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
	c := []configFiles{}

	for _, f := range files {
		c = append(c, configFiles{f.Name()})
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
	evt := event.NormalizedEvent{}
	connID := increaseConnCounter()
	rateCounter.Incr(1)
	epsCounter.Set(rateCounter.Rate())

	err := evt.FromBytes(ctx.PostBody())
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
	log.Debug(log.M{Msg: "Received event ID: " + evt.EventID, CId: connID})
	evt.ConnID = connID
	// push the event
	eventChannel <- evt
	log.Debug(log.M{Msg: "Event pushed", CId: connID})
}
