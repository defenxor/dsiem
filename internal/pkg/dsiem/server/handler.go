package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"time"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"

	"github.com/elastic/apm-agent-go"

	"github.com/defenxor/dsiem/internal/pkg/shared/apm"
	"github.com/defenxor/dsiem/internal/pkg/shared/fs"

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

func handleConfFileDelete(ctx *fasthttp.RequestCtx) {
	clientAddr := ctx.RemoteAddr().String()
	filename := ctx.UserValue("filename").(string)
	if filename == "" {
		fmt.Fprintf(ctx, "requires /config/filename\n")
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	if !isCfgFileNameValid(filename) {
		log.Warn(log.M{Msg: "l337 or epic fail attempt from " + clientAddr + " detected. Discarding."})
		fmt.Fprintf(ctx, "Not a valid filename, should be in any_N4m3-that_you_want.json format\n")
		ctx.SetStatusCode(fasthttp.StatusTeapot)
		return
	}

	log.Info(log.M{Msg: "Delete request for file '" + filename + "' from " + clientAddr})
	f := path.Join(confDir, filename)
	log.Info(log.M{Msg: "Deleting file " + f})

	if !fs.FileExist(f) {
		fmt.Fprintf(ctx, filename+" doesn't exist\n")
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		return
	}

	// delete file
	var err = os.Remove(f)
	if err != nil {
		fmt.Fprintf(ctx, "cannot delete "+filename)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	}
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
		fmt.Fprintf(ctx, "Not a valid filename, should be in any_N4m3-that_you_want.json format\n")
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
	r, err := regexp.Compile(`[a-zA-Z0-9_-]+.json`)
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

	evt := &event.NormalizedEvent{}

	msg := ctx.PostBody()
	err := evt.FromBytes(msg)
	// works: err := gojay.Unmarshal(ctx.PostBody(), evt)

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
		return
	}

	evt.RcvdTime = time.Now().Unix()
	evt.ConnID = connID

	var tx *elasticapm.Transaction

	if apm.Enabled() {
		tStart, err := time.Parse(time.RFC3339, evt.Timestamp)
		if err != nil {
			log.Warn(log.M{Msg: "Cannot parse event timestamp, skipping event", CId: evt.ConnID})
			ctx.SetStatusCode(fasthttp.StatusBadRequest)
			return
		}
		opts := elasticapm.TransactionOptions{elasticapm.TraceContext{}, tStart}
		tx = elasticapm.DefaultTracer.StartTransactionOptions("Log Source to Frontend", "SIEM", opts)
		tx.Context.SetCustom("event_id", evt.EventID)
		defer tx.End()
	}

	log.Debug(log.M{Msg: "Received event ID: " + evt.EventID, CId: connID})

	// push the event, timeout in 10s to avoid open fd overload
	select {
	case <-time.After(10 * time.Second):
		log.Info(log.M{Msg: "event channel timed out!", CId: connID})
		ctx.SetStatusCode(fasthttp.StatusRequestTimeout)
		if apm.Enabled() {
			tx.Result = "Event channel timed out"
		}
	case eventChan <- *evt:
		log.Debug(log.M{Msg: "Event pushed", CId: connID})
		if apm.Enabled() {
			tx.Result = "Event sent to backend"
		}
	case err := <-errChan:
		log.Info(log.M{Msg: "Error from message queue:" + err.Error(), CId: connID})
		ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
		if apm.Enabled() {
			tx.Result = err.Error()
		}
	}
}
