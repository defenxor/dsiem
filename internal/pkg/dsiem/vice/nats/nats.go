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

// Package nats provides a Vice implementation for NATS.
// Copied from the original project and modified to support broadcasting to
// multiple receivers, and encoded data format to avoid marshal/unmarshal error from
// go-nats.
package nats

import (
	"fmt"
	"sync"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/vice"

	log "github.com/defenxor/dsiem/internal/pkg/shared/logger"
	"github.com/nats-io/nats.go"
)

// DefaultAddr is the NATS default TCP address.
const DefaultAddr = nats.DefaultURL

var _ vice.Transport = (*Transport)(nil)

type unsubscriber interface {
	Unsubscribe() error
}

type publisher interface {
	Publish(subject string, data interface{}) error
}

// Transport implement VTransport
type Transport struct {
	sync.Mutex
	wg sync.WaitGroup

	receiveChans     map[string]chan event.NormalizedEvent
	sendChans        map[string]chan event.NormalizedEvent
	receiveBoolChans map[string]chan bool
	sendBoolChans    map[string]chan bool

	errChan chan error
	// stopchan is closed when everything has stopped.
	stopchan    chan struct{}
	stopPubChan chan struct{}

	subscriptions      []unsubscriber
	natsConn           *nats.Conn
	natsEncodedConn    *nats.EncodedConn
	natsEncoded        bool
	streamingClusterID string
	streamingClientID  string

	// exported fields
	NatsAddr       string
	NatsQueueGroup string
}

// New returns a new Transport
func New(opts ...Option) *Transport {
	var options Options

	return &Transport{
		NatsAddr: DefaultAddr,

		receiveChans:     make(map[string]chan event.NormalizedEvent),
		sendChans:        make(map[string]chan event.NormalizedEvent),
		receiveBoolChans: make(map[string]chan bool),
		sendBoolChans:    make(map[string]chan bool),

		errChan:     make(chan error, 10),
		stopchan:    make(chan struct{}),
		stopPubChan: make(chan struct{}),

		subscriptions: []unsubscriber{},

		natsConn:    options.Conn,
		natsEncoded: options.UseEncoded,
	}
}

func (t *Transport) newEncodedConnection() (*nats.EncodedConn, error) {
	var err error
	if t.natsEncodedConn != nil {
		return t.natsEncodedConn, err
	}
	t.natsConn, err = nats.Connect(t.NatsAddr,
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			msg := fmt.Sprintf("%q", err) // err maybe nil
			log.Info(log.M{Msg: "Disconnected from NATS server. Reason: " + msg})
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info(log.M{Msg: "Reconnected to NATS server"})
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			msg := fmt.Sprintf("%q", err)
			log.Info(log.M{Msg: "NATS error occurred: " + msg})
			// this happen quite frequent (NATS slow consumer error) so disable it for now
			// t.errChan <- vice.Err{Name: "Error", Err: errors.New(msg)} // err maybe nil
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			msg := fmt.Sprintf("%q", nc.LastError()) // err maybe nil
			log.Info(log.M{Msg: "NATS connection closed. Reason: " + msg})
		}),
		nats.MaxReconnects(-1))

	if err == nil {
		t.natsEncodedConn, err = nats.NewEncodedConn(t.natsConn, nats.JSON_ENCODER)
	}
	return t.natsEncodedConn, err
}

// ErrChan gets the channel on which errors are sent.
func (t *Transport) ErrChan() <-chan error {
	return t.errChan
}

// SimulateError simulate an error coming from NATS
func (t *Transport) SimulateError(err error) {
	t.errChan <- vice.Err{Name: "Simulator", Err: err}
}

// Stop stops the transport.
// The channel returned from Done() will be closed
// when the transport has stopped.
func (t *Transport) Stop() {
	t.Lock()
	defer t.Unlock()

	for _, s := range t.subscriptions {
		s.Unsubscribe()
	}

	close(t.stopPubChan)
	t.wg.Wait()

	if t.natsEncodedConn != nil {
		t.natsEncodedConn.Close()
	}

	t.natsConn.Flush()
	t.natsConn.Close()
	t.natsConn = nil

	close(t.stopchan)
}

// Done gets a channel which is closed when the
// transport has successfully stopped.
func (t *Transport) Done() chan struct{} {
	return t.stopchan
}
