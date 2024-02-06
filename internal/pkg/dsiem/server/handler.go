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

package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"time"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"

	"github.com/defenxor/dsiem/internal/pkg/shared/apm"

	"github.com/fasthttp-contrib/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/expvarhandler"
	"github.com/valyala/fasthttp/pprofhandler"
)

type configFile struct {
	Filename string `json:"filename"`
}
type configFiles struct {
	Files []configFile `json:"files"`
}

var evtChanTimeoutSecond = time.Duration(10 * time.Second)

func pprofHandler(ctx *fasthttp.RequestCtx) {
	pprofhandler.PprofHandler(ctx)
}

func expVarHandler(ctx *fasthttp.RequestCtx) {
	expvarhandler.ExpvarHandler(ctx)
}

func wsHandler(ctx *fasthttp.RequestCtx) {
	upgrader = websocket.New(wss.onClientConnected)
	err := upgrader.Upgrade(ctx)
	if err != nil {
		log.Warn(log.M{Msg: "error returned from websocket: " + err.Error()})
	}
}

func handleConfFileList(ctx *fasthttp.RequestCtx) {
	clientAddr := ctx.RemoteAddr().String()
	log.Info(log.M{Msg: "Request for list of configuration files from " + clientAddr + ". Using config dir: " + c.Confd})

	files, err := os.ReadDir(c.Confd)
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
	_, _ = ctx.Write(byteVal)
}

func handleConfFileDelete(ctx *fasthttp.RequestCtx) {
	clientAddr := ctx.RemoteAddr().String()
	filename := ctx.UserValue("filename").(string)

	if !isCfgFileNameValid(filename) {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding."})
		fmt.Fprintf(ctx, "Not a valid filename, should be in any_N4m3-that_you_want.json format\n")
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		return
	}

	log.Info(log.M{Msg: "Delete request for file '" + filename + "' from " + clientAddr})
	f := path.Join(c.Confd, filename)
	log.Info(log.M{Msg: "Deleting file " + f})

	var err = os.Remove(f)
	if err != nil {
		fmt.Fprintf(ctx, "cannot delete "+filename)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	}
}

func handleConfFileDownload(ctx *fasthttp.RequestCtx) {
	clientAddr := ctx.RemoteAddr().String()
	filename := ctx.UserValue("filename").(string)

	if !isCfgFileNameValid(filename) {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding."})
		fmt.Fprintf(ctx, "Not a valid filename, should be in any_N4m3-that_you_want.json format\n")
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		return
	}

	log.Info(log.M{Msg: "Request for file '" + filename + "' from " + clientAddr})
	f := path.Join(c.Confd, filename)
	log.Info(log.M{Msg: "Getting file " + f})

	file, err := os.Open(f)
	if err != nil {
		fmt.Fprintf(ctx, "cannot open "+filename)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}
	defer file.Close()

	byteValue, err := io.ReadAll(file)
	if err != nil {
		fmt.Fprintf(ctx, "cannot read "+filename)
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	_, _ = ctx.Write(byteValue)
}

func handleConfFileUpload(ctx *fasthttp.RequestCtx) {
	clientAddr := ctx.RemoteAddr().String()
	filename := ctx.UserValue("filename").(string)

	if !isCfgFileNameValid(filename) {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding."})
		fmt.Fprintf(ctx, "Not a valid filename, should be in any_N4m3_you_want.json format\n")
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		return
	}

	content := ctx.PostBody()
	log.Info(log.M{Msg: "Upload file request for '" + filename + "' from " + clientAddr})
	err := isUploadContentValid(filename, content)
	if err != nil {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding."})
		fmt.Fprintf(ctx, "Invalid content detected, parsing error message is: %s\n", err.Error())
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		return
	}

	file := path.Join(c.Confd, filename)
	f, err := os.OpenFile(file, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		fmt.Fprintf(ctx, "Cannot open target file location\n")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	defer f.Close()
	_, err = f.Write(content)
	if err != nil {
		fmt.Fprintf(ctx, "Cannot write to target file location\n")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	fmt.Fprintf(ctx, "File "+filename+" uploaded successfully\n")
	ctx.SetStatusCode(fasthttp.StatusCreated)
}

func handleEvents(ctx *fasthttp.RequestCtx) {

	clientAddr := ctx.RemoteAddr().String()
	connID := increaseConnCounter()
	rateCounter.Incr(1)

	evt := &event.NormalizedEvent{}

	msg := ctx.PostBody()
	err := evt.FromBytes(msg)

	if err != nil {
		log.Warn(log.M{Msg: "Cannot parse normalizedEvent from " + clientAddr + ". err: " + err.Error(), CId: connID})
		log.Warn(log.M{Msg: "The failed message is: " + string(msg)})
		fmt.Fprintf(ctx, "Cannot parse the submitted event\n")
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		return
	}
	if !evt.Valid() {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding.", CId: connID})
		fmt.Fprintf(ctx, "Not a valid event\n")
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		log.Debug(log.M{
			Msg: "ts:" + evt.Timestamp + ",sensor:" + evt.Sensor + ",Id:" + evt.EventID +
				",srcIP:" + evt.SrcIP + ",dstIP:" + evt.DstIP + ",plugID:" + strconv.Itoa(evt.PluginID) +
				",SID:" + strconv.Itoa(evt.PluginSID) + ",product:" + evt.Product + ",category: " + evt.Category +
				",subcat:" + evt.SubCategory})
		return
	}

	now := time.Now()
	evt.RcvdTime = now.UnixNano()
	evt.ConnID = connID

	var tx *apm.Transaction

	if apm.Enabled() {
		tStart, err := time.Parse(time.RFC3339, evt.Timestamp)
		if err != nil {
			log.Warn(log.M{Msg: "Cannot parse event timestamp, skipping event", CId: evt.ConnID})
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}
		tx = apm.StartTransaction("Log Source to Frontend", "Network", &tStart, nil)
		th := tx.GetTraceContext()
		evt.TraceParent = th.Traceparent
		evt.TraceState = th.TraceState
		tx.SetCustom("event_id", evt.EventID)
		duration := now.Sub(tStart)
		tx.Tx.Duration = duration
		tx.Result("Event received from log source")
		defer tx.End()
	}

	log.Debug(log.M{Msg: "Received event ID: " + evt.EventID, CId: connID})

	// push the event, timeout to avoid open fd overload
	select {
	case <-time.After(evtChanTimeoutSecond):
		log.Info(log.M{Msg: "event channel timed out!", CId: connID})
		ctx.SetStatusCode(fasthttp.StatusRequestTimeout)
		if apm.Enabled() {
			tx.Result("Event channel timed out")
		}
	case c.EvtChan <- *evt:
		log.Debug(log.M{Msg: "Event pushed", CId: connID})
		if apm.Enabled() {
			tx.Result("Event sent to backend")
		}
	case err := <-c.ErrChan:
		log.Info(log.M{Msg: "Error from message queue:" + err.Error(), CId: connID})
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		if apm.Enabled() {
			tx.SetError(err)
			tx.Result("Error received from message queue")
		}
	}
}
