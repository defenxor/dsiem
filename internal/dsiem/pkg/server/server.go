package server

import (
	"dsiem/internal/dsiem/pkg/event"
	"dsiem/internal/shared/pkg/fs"
	log "dsiem/internal/shared/pkg/logger"
	"encoding/json"
	"errors"
	"expvar"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync/atomic"
	"time"

	rc "github.com/paulbellamy/ratecounter"
	"golang.org/x/net/websocket"

	"github.com/elastic/apm-agent-go/module/apmhttprouter"
	"github.com/julienschmidt/httprouter"
)

var connCounter uint64
var webDir, confDir string
var rateCounter *rc.RateCounter
var wss *wsServer

type configFiles struct {
	FileName string `json:"filename"`
}

var epsCounter = expvar.NewInt("eps_counter")
var eventChannel chan<- event.NormalizedEvent

// Start the HTTP server on addr:port, writing incoming event to ch, reading/writing
// conf files to confd, serving ws connection for eps rate, serving /debug/vars, and the angular UI
func Start(ch chan<- event.NormalizedEvent, confd string, webd string, addr string, port int) error {
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
		router := apmhttprouter.New()
		router.POST("/events", handleEvents)
		router.GET("/config/:filename", handleConfFileDownload)
		router.GET("/config/", handleConfFileList)
		router.GET("/debug/vars/", expvarHandler)
		router.POST("/config/:filename", handleConfFileUpload)
		router.GET("/eps/", wsHandler)
		router.ServeFiles("/ui/*filepath", http.Dir(webDir))
		log.Info(log.M{Msg: "Server listening on " + addr + ":" + p})
		initWSServer()
		// we're expecting fast clients only
		srv := &http.Server{
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
			IdleTimeout:  5 * time.Second,
			Addr:         addr + ":" + p,
			Handler:      router,
		}
		err := srv.ListenAndServe()
		if err != nil {
			log.Warn(log.M{Msg: "error from ListenAndServe: " + err.Error()})
		}
	}
	return nil
}

func expvarHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	expvar.Handler().ServeHTTP(w, r)
}

func wsHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	s := websocket.Server{Handler: websocket.Handler(wss.onClientConnected)}
	s.ServeHTTP(w, r)
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

func handleConfFileList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientAddr := r.RemoteAddr
	log.Info(log.M{Msg: "Request for list of configuration files from " + clientAddr})

	files, err := ioutil.ReadDir(confDir)
	if err != nil {
		http.Error(w, "Error reading config directory.", 500)
		return
	}
	c := []configFiles{}

	for _, f := range files {
		c = append(c, configFiles{f.Name()})
	}
	byteVal, err := json.MarshalIndent(&c, "", "  ")
	if err != nil {
		http.Error(w, "Error reading config file names.", 500)
		return
	}
	_, err = w.Write(byteVal)
	return
}

func handleConfFileDownload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientAddr := r.RemoteAddr
	filename := ps.ByName("filename")
	if filename == "" {
		http.Error(w, "requires /config/filename", 400)
		return
	}
	log.Info(log.M{Msg: "Request for file '" + filename + "' from " + clientAddr})
	f := path.Join(confDir, filename)
	log.Info(log.M{Msg: "Getting file " + f})

	if !fs.FileExist(f) {
		http.Error(w, filename+" doesnt exist", 404)
		return
	}
	file, err := os.Open(f)
	if err != nil {
		http.Error(w, "cannot open "+filename, 500)
		return
	}
	defer file.Close()

	byteValue, err := ioutil.ReadAll(file)
	if err != nil {
		http.Error(w, "cannot read "+filename, 500)
		return
	}
	_, err = w.Write(byteValue)
	return
}

func handleConfFileUpload(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientAddr := r.RemoteAddr
	filename := ps.ByName("filename")
	if filename == "" {
		http.Error(w, "requires /config/filename", 500)
		return
	}
	log.Info(log.M{Msg: "Upload file request for '" + filename + "' from " + clientAddr})
	file := path.Join(confDir, filename)
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warn(log.M{Msg: "Error reading message from " + clientAddr + ". Returning HTTP 500."})
		http.Error(w, "Cannot read posted body content", 500)
		return
	}
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	defer f.Close()
	if err != nil {
		http.Error(w, "Cannot open target file location", 500)
		return
	}
	_, err = f.Write(b)
	if err != nil {
		http.Error(w, "Cannot write to target file location", 500)
		return
	}
	w.Write([]byte("File " + filename + " uploaded successfully\n"))
	return
}

func handleEvents(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	clientAddr := r.RemoteAddr
	evt := event.NormalizedEvent{}
	connID := increaseConnCounter()
	rateCounter.Incr(1)
	epsCounter.Set(rateCounter.Rate())

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Warn(log.M{Msg: "Error reading message from " + clientAddr + ". Returning HTTP 500.", CId: connID})
		http.Error(w, "Cannot read posted body content", 500)
		return
	}
	err = evt.FromBytes(b)
	if err != nil {
		log.Warn(log.M{Msg: "Cannot parse normalizedEvent from " + clientAddr + ". err: " + err.Error(), CId: connID})
		http.Error(w, "Cannot parse the submitted event", 400)
		return
	}
	if !evt.Valid() {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding.", CId: connID})
		http.Error(w, "Not a valid event", 418)
		return
	}
	log.Debug(log.M{Msg: "Received event ID: " + evt.EventID, CId: connID})
	evt.ConnID = connID
	// push the event
	eventChannel <- evt
}
