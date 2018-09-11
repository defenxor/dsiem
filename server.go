package main

import (
	"io/ioutil"
	"net/http"
	"sync/atomic"

	"github.com/julienschmidt/httprouter"
)

var connCounter uint64

const (
	progName = "SIEM"
	port     = "8080"
)

func startServer() {
	for {
		logInfo("Starting "+progName, 0)
		router := httprouter.New()
		router.POST("/*file", handle)
		logInfo("Server listening on port: "+port, 0)
		err := http.ListenAndServe(":"+port, router)
		if err != nil {
			logWarn("Error from http.ListenAndServe: "+err.Error(), 0)
		}
	}
}

func increaseConnCounter() uint64 {
	// increase counter to differentiate entries in log
	atomic.AddUint64(&connCounter, 1)
	myID := atomic.LoadUint64(&connCounter)
	return myID
}

func handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	clientAddr := r.RemoteAddr
	evt := normalizedEvent{}
	connID := increaseConnCounter()

	b, err := ioutil.ReadAll(r.Body)
	// bstr := string(b)
	// logger.Info(bstr)
	if err != nil {
		logWarn("Error reading message from "+clientAddr+". Ignoring it", connID)
		return
	}
	err = evt.fromBytes(b)
	if err != nil {
		logWarn("Cannot parse normalizedEvent from "+clientAddr+". Ignoring it.", connID)
		bstr := string(b)
		logger.Warn(bstr)
		return
	}
	if !evt.valid() {
		logWarn("l337 or epic fail attempt from "+clientAddr+" detected. Discarding.", connID)
		return
	}

	logInfo("Received event ID: "+evt.EventID, connID)
	evt.ConnID = connID
	// push the event
	eventChannel <- evt

	// logInfo("Done.", connID)
}
