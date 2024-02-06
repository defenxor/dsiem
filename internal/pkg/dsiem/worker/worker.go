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

package worker

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/vice/nats"
)

// var receiver <-chan []byte
var transport *nats.Transport
var eventChan <-chan event.NormalizedEvent
var bpChan chan<- bool
var errChan <-chan error
var bpChanReady chan struct{}

type configFile struct {
	Filename string `json:"filename"`
}
type configFiles struct {
	Files []configFile `json:"files"`
}

func init() {
	bpChanReady = make(chan struct{}, 1)
}

func getConfigFileList(frontendAddr string) (cf *configFiles, err error) {
	c := http.Client{Timeout: time.Second * 5}
	url := frontendAddr + "/config"
	resp, err := c.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	// fmt.Println(string(body))

	if err == nil {
		cfg := configFiles{}
		err = json.Unmarshal(body, &cfg)
		cf = &cfg
	}
	return
}

func downloadConfigFiles(confDir string, frontendAddr string, node string) error {
	cfg, err := getConfigFileList(frontendAddr)
	if err != nil {
		return err
	}
	for _, v := range cfg.Files {
		f := v.Filename
		if strings.HasPrefix(f, "assets_") || strings.HasPrefix(f, "vuln_") ||
			strings.HasPrefix(f, "intel_") || strings.HasPrefix(f, "directives_"+node+"_") {
			p := path.Join(confDir, f)
			url := frontendAddr + "/config/" + f
			log.Info(log.M{Msg: "downloading " + url})
			// use trick to avoid testing err != nil (test coverage hack)
			if err == nil {
				err = downloadFile(p, url)
			}
		}
	}
	return nil
}

//GetBackPressureChannel returns channel for sending backpressure bool messages
func GetBackPressureChannel() chan<- bool {
	<-bpChanReady
	return bpChan
}

func initMsgQueue(msq string, prefix, nodeName string) (errOccurred bool) {

	initMsq := func() (err error) {
		// reuse existing transport, used during testing
		if transport == nil {
			transport = nats.New()
		}
		transport.NatsAddr = msq
		eventChan = transport.Receive(prefix + "_" + "events")
		errChan = transport.ErrChan()
		bpChan = transport.SendBool(prefix + "_" + "overload_signals")
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
			select {
			case bpChanReady <- struct{}{}:
			default:
			}
			break
		}
		errOccurred = true
		handleMsqError(err)
	}
	return
}

func handleMsqError(err error) {
	const reconnectSecond = 3
	log.Info(log.M{Msg: "Error from message queue " + err.Error()})
	log.Info(log.M{Msg: "Reconnecting in " + strconv.Itoa(reconnectSecond) + " seconds.."})
	time.Sleep(reconnectSecond * time.Second)
}

// Start start worker
func Start(ch chan<- event.NormalizedEvent, msq string, msqPrefix string,
	nodeName string, confDir string, frontend string) error {
	if err := downloadConfigFiles(confDir, frontend, nodeName); err != nil {
		return err
	}

	go func() {
		_ = initMsgQueue(msq, msqPrefix, nodeName)
		for {
			select {
			case evt := <-eventChan:
				ch <- evt
			case err := <-errChan:
				log.Warn(log.M{Msg: "Error received from receive message queue: " + err.Error()})
				_ = initMsgQueue(msq, msqPrefix, nodeName)
			}
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
	return err
}
