package nesd

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/julienschmidt/httprouter"
)

var csvDir string

// Start the server
func Start(addr string, port int) error {
	if a := net.ParseIP(addr); a == nil {
		return errors.New(addr + " is not a valid IP address")
	}
	if port < 1 || port > 65535 {
		return errors.New("Invalid TCP port number")
	}

	p := strconv.Itoa(port)
	for {
		router := httprouter.New()
		router.GET("/", handler)
		log.Info(log.M{Msg: "Server listening on " + addr + ":" + p})
		err := http.ListenAndServe(addr+":"+p, router)
		if err != nil {
			log.Warn(log.M{Msg: "Error from http.ListenAndServe: " + err.Error()})
		}
	}
	return nil
}

func handler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientAddr := r.RemoteAddr
	qv := r.URL.Query()
	ip := qv.Get("ip")
	port := qv.Get("port")
	if ip == "" || port == "" {
		http.Error(w, "requires ip and port parameter", 400)
		log.Info(log.M{Msg: "returning 400-1"})
		return
	}
	if a := net.ParseIP(ip); a == nil {
		http.Error(w, "ip parameter only accept... ip address", 418)
		log.Info(log.M{Msg: "returning 418-1, provided adress: " + ip})
		return
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		http.Error(w, "port parameter only accept... valid port number", 418)
		log.Info(log.M{Msg: "returning 418-2"})
		return
	}
	log.Info(log.M{Msg: "Incoming query for " + ip + ":" + port + " from " + clientAddr})

	found, vulns := findMatch(ip, p)
	if !found {
		w.Write([]byte("no vulnerability found\n"))
		log.Debug(log.M{Msg: "No vulnerability found for " + ip + ":" + port})
		return
	}

	b, err := json.MarshalIndent(&vulns.v, "", "  ")
	if err != nil {
		log.Warn(log.M{Msg: "Cannot encode result for " + ip + ":" + port + ". Error: " + err.Error()})
		http.Error(w, "Cannot encode result", 500)
		return
	}
	n := strconv.Itoa(len(vulns.v))
	log.Info(log.M{Msg: "Returning " + n + " positive result for " + ip + ":" + port + " to " + clientAddr})
	_, err = w.Write(b)
	if err != nil {
		log.Warn(log.M{Msg: "Cannot return positive result for " + ip + ":" + port + " to " + clientAddr + ". Error: " + err.Error()})
	}
	return
}
