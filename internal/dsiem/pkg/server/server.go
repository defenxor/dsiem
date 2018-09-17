package server

import (
	"dsiem/internal/dsiem/pkg/event"
	log "dsiem/internal/dsiem/pkg/logger"
	"dsiem/internal/shared/pkg/fs"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync/atomic"

	"github.com/julienschmidt/httprouter"
)

var connCounter uint64
var progDir, confDir string

const (
	progName = "SIEM"
	port     = "8080"
)

type configFiles struct {
	FileName string `json:"filename"`
}

var eventChannel chan<- event.NormalizedEvent

// Start the server
func Start(ch chan<- event.NormalizedEvent) {
	eventChannel = ch
	for {
		log.Info("Starting "+progName, 0)
		router := httprouter.New()
		router.POST("/events", handleEvents)
		router.GET("/config/:filename", handleConfFileDownload)
		router.GET("/config/", handleConfFileList)
		router.POST("/config/:filename", handleConfFileUpload)
		log.Info("Server listening on port: "+port, 0)
		err := http.ListenAndServe(":"+port, router)
		if err != nil {
			log.Warn("Error from http.ListenAndServe: "+err.Error(), 0)
		}
	}
}

func increaseConnCounter() uint64 {
	// increase counter to differentiate entries in log
	atomic.AddUint64(&connCounter, 1)
	myID := atomic.LoadUint64(&connCounter)
	return myID
}

func handleConfFileList(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientAddr := r.RemoteAddr
	log.Info("Request for list of configuration files from "+clientAddr, 0)

	dir := path.Join(progDir, confDir)
	files, err := ioutil.ReadDir(dir)
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
		http.Error(w, "requires /config/filename", 500)
		return
	}
	log.Info("Request for file '"+filename+"' from "+clientAddr, 0)
	f := path.Join(progDir, confDir, filename)
	log.Info("Getting file "+f, 0)
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
	log.Info("Upload file request for '"+filename+"' from "+clientAddr, 0)
	file := path.Join(progDir, confDir, filename)
	b, err := ioutil.ReadAll(r.Body)
	// bstr := string(b)
	// logger.Info(bstr)
	if err != nil {
		log.Warn("Error reading message from "+clientAddr+". Ignoring it.", 0)
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

	b, err := ioutil.ReadAll(r.Body)
	// bstr := string(b)
	// logger.Info(bstr)
	if err != nil {
		log.Warn("Error reading message from "+clientAddr+". Ignoring it", connID)
		return
	}
	err = evt.FromBytes(b)
	if err != nil {
		log.Warn("Cannot parse normalizedEvent from "+clientAddr+". Ignoring it. err: "+err.Error(), connID)
		// bstr := string(b)
		// log.Warn(bstr,connID)
		return
	}
	if !evt.Valid() {
		log.Warn("l337 or epic fail attempt from "+clientAddr+" detected. Discarding.", connID)
		return
	}

	log.Debug("Received event ID: "+evt.EventID, connID)
	evt.ConnID = connID
	// push the event
	eventChannel <- evt
}
