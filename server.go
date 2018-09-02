/* The following are expected:

 - env var:
		- ICINGA_WEBHOOK_ACCESS_TOKEN: required - token or password to use by submitter to access webhook
		- ICINGA_WEBHOOK_SSH_USER: optional - default to root
 - mounted volume:
		- ./keys/sshpriv.key: required - should be mounted from k8s secret
		- ./keys/tls.crt: optional - will be created if doesnt exist
		- ./keys/tls.key: optional - will be created if doesnt exist
*/
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"

	"github.com/julienschmidt/httprouter"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	progName  = "siem"
	port      = "8080"
	rulesFile = "rules/rules.json"
)

var logger = logrus.New()
var taskCounter uint64

// { "timestamp": "2018-08-09T00:00:01Z", "sensor": "sensor1", "plugin_id": 1001, "plugin_sid": 2002,
// "priority": 3, "reliability": 1, "src_ip": "10.73.255.1", "src_port": "51231",
// "dst_ip": "10.73.255.10", "dst_port": 80, "protocol": "TCP", "userdata1": "ponda", "userdata2": "rossa" }

type (
	normalizedEvent struct {
		Timestamp    string `json:"timestamp"`
		Sensor       string `json:"sensor"`
		PluginID     int    `json:"plugin_id"`
		PluginSID    int    `json:"plugin_sid"`
		Priority     int    `json:"priority"`
		Reliability  int    `json:"reliability"`
		SrcIP        string `json:"src_ip"`
		SrcPort      int    `json"src_port"`
		DstIP        string `json:"dst_ip"`
		DstPort      int    `json:"dst_port"`
		Protocol     string `json:"protocol"`
		CustomData1  string `json:"custom_data1"`
		CustomLabel1 string `json:"custom_label1"`
		CustomData2  string `json:"custom_data2"`
		CustomLabel2 string `json:"custom_label2"`
		CustomData3  string `json:"custom_data3"`
		CustomLabel3 string `json:"custom_label3"`
	}
)

var eventChannel chan normalizedEvent

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

	err = verify(evt)
	if err != nil {
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

func directiveChanController() {
	var total = 3000
	var dirchan []chan normalizedEvent
	logger.Info("Creating ", total, " directives.")
	eventChannel = make(chan normalizedEvent)
	for i := 0; i < total; i++ {
		dirchan = append(dirchan, make(chan normalizedEvent))
		go directive(i, dirchan[i])
		go func() {
			for {
				evt := <-eventChannel
				for i := range dirchan {
					dirchan[i] <- evt
				}
			}
		}()
	}
}

func directive(id int, c chan normalizedEvent) {
	logger.Info("started directive ", id)

	// should setup pipeline here with first input from chan c
	//stdout := processors.NewIoWriter(os.Stdout)
	//upperCaser := processors.NewFuncTransformer(func(d data.JSON) data.JSON {
	//	return data.JSON(strings.ToUpper(string(d)))
	//})
	//	pipeline := ratchet.NewPipeline(upperCaser, stdout)

	// Finally, run the Pipeline and wait for either an error or nil to be returned
	//	err := <-pipeline.Run()
	//	if err != nil {
	//		return
	//	}

	for {
		evt := <-c
		logger.Info("directive ", id, " received data from dirchan: ", evt)
	}
}

func main() {
	setupLogger()
	directiveChanController()
	logger.Info("Starting " + progName)
	router := httprouter.New()
	router.POST("/*file", handle)
	logger.Info("Server listening on port: ", port)
	logger.Fatal(http.ListenAndServe(":"+port, router))
}

func setupLogger() {
	formatter := &logrus.TextFormatter{
		FullTimestamp: true,
	}
	// formatter := &logrus.JSONFormatter{}
	logger.Formatter = formatter
	logger.Out = os.Stdout

	// use logrus for standard log output, those chatty 3rd-party libs ..
	log.SetOutput(logger.Writer())
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
}

func logAndExit(err error) {
	// time.Sleep(5 * time.Minute)
	logger.Fatalf("%+v", errors.Wrap(err, ""))
}

func getDir() (string, error) {
	dir, err := osext.ExecutableFolder()
	return dir, err
}

func fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func verify(e normalizedEvent) error {
	if e.Timestamp == "" || e.Sensor == "" || e.PluginID == 0 {
		return errors.New("missing required parameters")
	}
	return nil
}
