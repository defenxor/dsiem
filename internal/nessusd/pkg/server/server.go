package server

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"

	log "dsiem/internal/shared/pkg/logger"

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
		log.Info("Server listening on "+addr+":"+p, 0)
		err := http.ListenAndServe(addr+":"+p, router)
		if err != nil {
			log.Warn("Error from http.ListenAndServe: "+err.Error(), 0)
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
		return
	}
	if a := net.ParseIP(ip); a == nil {
		http.Error(w, "ip parameter only accept... ip address", 418)
		return
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		http.Error(w, "port parameter only accept... valid port number", 418)
		return
	}
	log.Info("Incoming query for "+ip+":"+port+" from "+clientAddr, 0)

	found, vulns := findMatch(ip, p)
	if !found {
		w.Write([]byte("no vulnerability found\n"))
		log.Debug("No vulnerability found for "+ip+":"+port, 0)
		return
	}

	b, err := json.Marshal(&vulns.v)
	if err != nil {
		log.Warn("Cannot encode result for "+ip+":"+port+". Error: "+err.Error(), 0)
		http.Error(w, "Cannot encode result", 500)
		return
	}
	n := strconv.Itoa(len(vulns.v))
	log.Info("Returning "+n+" positive result for "+ip+":"+port+" to "+clientAddr, 0)
	_, err = w.Write(b)
	if err != nil {
		log.Warn("Cannot return positive result for "+ip+":"+port+" to "+clientAddr+". Error: "+err.Error(), 0)
	}
	return
}
