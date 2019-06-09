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
	"sync"

	"github.com/defenxor/dsiem/internal/pkg/shared/idgen"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/fasthttp-contrib/websocket"
)

type client struct {
	id string
	ws *websocket.Conn
}

type message struct {
	Eps int64 `json:"eps"`
}

type wsServer struct {
	sync.Mutex
	clients      map[string]*client
	sendAllCh    chan *message
	cConnectedCh chan bool
}

func newWSServer() *wsServer {
	clients := make(map[string]*client)
	sendAllCh := make(chan *message)
	cConnectedCh := make(chan bool)
	return &wsServer{
		clients:      clients,
		sendAllCh:    sendAllCh,
		cConnectedCh: cConnectedCh,
	}
}

func (s *wsServer) add(ws *websocket.Conn) (id string) {
	id = "static"
	id, _ = idgen.GenerateID()
	log.Debug(log.M{Msg: "adding WS client " + id})
	c := client{}
	c.id = id
	c.ws = ws
	s.clients[c.id] = &c
	// non-blocking signal
	select {
	case s.cConnectedCh <- true:
	default:
	}
	return c.id
}

func (s *wsServer) del(cID string) {
	s.Lock()
	delete(s.clients, cID)
	s.Unlock()
}

func (s *wsServer) sendAll(msg *message) {
	s.sendAllCh <- msg
}

func (s *wsServer) onClientConnected(ws *websocket.Conn) {
	defer func() {
		_ = ws.Close()
	}()
	id := s.add(ws)

	for msg := range s.sendAllCh {
		/*
			_, err := json.Marshal(msg)
			if err != nil {
				log.Debug(log.M{Msg: "failed to marshal msg"})
				continue
			}
		*/
		if err := ws.WriteJSON(msg); err != nil {
			log.Debug(log.M{Msg: "failed to write to " + id + ", assuming client is disconnected."})
			s.del(id)
			return
		}
	}
}
