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
	"github.com/defenxor/dsiem/internal/pkg/shared/idgen"
	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"

	"github.com/fasthttp-contrib/websocket"
	// "golang.org/x/net/websocket"
)

type client struct {
	id string
	ws *websocket.Conn
}

type message struct {
	Eps int64 `json:"eps"`
}

type wsServer struct {
	clients      map[string]*client
	sendAllCh    chan *message
	cConnectedCh chan bool
}

func newWSServer() *wsServer {
	clients := make(map[string]*client)
	sendAllCh := make(chan *message)
	cConnectedCh := make(chan bool)
	return &wsServer{
		clients,
		sendAllCh,
		cConnectedCh,
	}
}

func (s *wsServer) add(ws *websocket.Conn) (id string, err error) {
	id, err = idgen.GenerateID()
	if err != nil {
		log.Debug(log.M{Msg: "cannot create an ID for WS client!" + id})
		return "", err
	}
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
	return c.id, nil
}

func (s *wsServer) del(cID string) {
	delete(s.clients, cID)
}

func (s *wsServer) sendAll(msg *message) {
	s.sendAllCh <- msg
}

func (s *wsServer) onClientConnected(ws *websocket.Conn) {
	defer func() {
		_ = ws.Close()
	}()
	id, err := s.add(ws)
	if err != nil {
		return
	}
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
