package worker

import (
	"dsiem/internal/dsiem/pkg/event"
	log "dsiem/internal/shared/pkg/logger"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/francoispqt/gojay"

	"dsiem/internal/vice/pkg/nats"
)

var transport nats.Transport
var receiver <-chan []byte
var errchan <-chan error

type configFile struct {
	Filename string `json:"filename"`
}
type configFiles struct {
	Files []configFile `json:"files"`
}

func initMsgQueue(msqURL string, msq string, prefix string, nodeName string) {
	opt := nats.WithStreaming(msq, prefix+"-"+nodeName)
	transport := nats.New(opt)
	transport.NatsAddr = msqURL
	transport.NatsStreamingQGroup = nodeName
	// transport := nats.New()
	receiver = transport.Receive(prefix + "_" + "events")
	errchan = transport.ErrChan()
}

func getConfigFileList(frontendAddr string) (*configFiles, error) {
	c := http.Client{Timeout: time.Second * 5}
	url := frontendAddr + "/config"
	resp, err := c.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(body))
	if err != nil {
		return nil, err
	}
	cf := configFiles{}
	err = json.Unmarshal(body, &cf)
	if err != nil {
		return nil, err
	}
	return &cf, nil
}

func downloadConfigFiles(confDir string, frontendAddr string, node string) error {
	cfg, err := getConfigFileList(frontendAddr)
	if err != nil {
		return err
	}
	for _, v := range cfg.Files {
		f := v.Filename
		if !strings.HasPrefix(f, "assets_") &&
			!strings.HasPrefix(f, "vuln_") &&
			!strings.HasPrefix(f, "intel_") &&
			!strings.HasPrefix(f, "directives_"+node+"_") {
			continue
		}
		p := path.Join(confDir, f)
		url := frontendAddr + "/config/" + f
		log.Info(log.M{Msg: "downloading " + url})
		if err := downloadFile(p, url); err != nil {
			return err
		}
	}
	return nil
}

// InitWorker start worker
func InitWorker(ch chan<- event.NormalizedEvent, msqURL string, msq string, msqPrefix string,
	nodeName string, confDir string, frontend string) error {
	if err := downloadConfigFiles(confDir, frontend, nodeName); err != nil {
		return err
	}

	initMsgQueue(msqURL, msq, msqPrefix, nodeName)

	go func() {
		defer transport.Stop()
		for {
			msg := <-receiver
			evt := event.NormalizedEvent{}
			// err := json.NewDecoder(bytes.NewReader(msg)).Decode(&evt)
			// err := evt.FromBytes(msg)
			err := gojay.Unmarshal(msg, &evt)
			if err != nil {
				// log.Warn(log.M{Msg: "Error decoding event from message queue: " + err.Error()})
				fmt.Println("Error decoding json on receiver: ", err.Error())
				fmt.Println(string(msg))
				continue
			}
			// fmt.Println("msg recevd:\n", string(msg))
			ch <- evt
		}
	}()
	go func() {
		for err := range errchan {
			log.Warn(log.M{Msg: "Error received from receive message queue: " + err.Error()})
		}
	}()

	return nil
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}
