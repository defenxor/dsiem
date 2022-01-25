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

package nesd

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/julienschmidt/httprouter"
)

var csvDir string
var httpSrv http.Server
var mu sync.Mutex

// Start the server
func Start(addr string, port int) (err error) {
	if a := net.ParseIP(addr); a == nil {
		err = errors.New(addr + " is not a valid IP address")
	}
	if port < 1 || port > 65535 {
		err = errors.New("Invalid TCP port number")
	}
	if err != nil {
		return
	}
	p := strconv.Itoa(port)
	router := httprouter.New()
	router.GET("/", handler)
	log.Info(log.M{Msg: "Server listening on " + addr + ":" + p})
	httpSrv.Addr = addr + ":" + p
	httpSrv.Handler = router
	err = httpSrv.ListenAndServe()
	return
}

func handler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	clientAddr := r.RemoteAddr
	qv := r.URL.Query()
	ip := qv.Get("ip")
	ip = strings.Replace(ip, "\n", "", -1)
	ip = strings.Replace(ip, "\r", "", -1)

	port := qv.Get("port")
	port = strings.Replace(port, "\n", "", -1)
	port = strings.Replace(port, "\r", "", -1)

	if ip == "" || port == "" {
		http.Error(w, "requires ip and port parameter", 400)
		log.Info(log.M{Msg: "returning 400-1"})
		return
	}
	if a := net.ParseIP(ip); a == nil {
		http.Error(w, "ip parameter only accept... ip address", 418)
		log.Info(log.M{Msg: "returning 418-1, provided address: " + ip})
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

	// can't trigger this error, input from findMatch is already type-checked and correct
	b, _ := json.MarshalIndent(&vulns.V, "", "  ")

	n := strconv.Itoa(len(vulns.V))
	log.Info(log.M{Msg: "Returning " + n + " positive result for " + ip + ":" + port + " to " + clientAddr})
	_, err = w.Write(b)
	if err != nil {
		log.Warn(log.M{Msg: "Cannot return positive result for " + ip + ":" + port + " to " + clientAddr + ". Error: " + err.Error()})
	}
}
