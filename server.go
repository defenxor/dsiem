package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync/atomic"

	"github.com/julienschmidt/httprouter"
)

var taskCounter uint64

const (
	progName = "siem"
	port     = "8080"
)

func startServer() {
	logger.Info("Starting " + progName)
	router := httprouter.New()
	router.POST("/*file", handle)
	logger.Info("Server listening on port: ", port)
	logger.Fatal(http.ListenAndServe(":"+port, router))
}

func handle(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	clientAddr := r.RemoteAddr
	evt := normalizedEvent{}

	// increase counter to differentiate entries in log
	atomic.AddUint64(&taskCounter, 1)
	myID := atomic.LoadUint64(&taskCounter)
	sMyID := strconv.Itoa(int(myID))

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logger.Warn("[" + sMyID + "] Error reading message from " + clientAddr + ". Ignoring it.")
		return
	}
	err = json.Unmarshal(b, &evt)
	fmt.Printf("%+v\n", evt)

	if !evt.valid() {
		logger.Warn("[" + sMyID + "] l337 or epic fail attempt from " + clientAddr + " detected. Responding with UNKNOWN status")
		return
	}

	// just show the program name, parameters may contain sensitive info
	logger.Info("[" + sMyID + "] Receive event from " + clientAddr + " for timestamp: " + evt.Timestamp + " pluginID: " + strconv.Itoa(evt.PluginID) + " sensor: " + evt.Sensor)

	// push the event
	eventChannel <- evt

	// n := executeSSH(c)
	logger.Info("[" + sMyID + "] Done.")
}
