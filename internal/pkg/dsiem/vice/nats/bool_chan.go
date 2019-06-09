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

package nats

import (
	"time"

	"github.com/defenxor/dsiem/internal/pkg/dsiem/vice"
)

// ReceiveBool gets a channel on which to receive messages
func (t *Transport) ReceiveBool(name string) <-chan bool {
	t.Lock()
	defer t.Unlock()

	ch, ok := t.receiveBoolChans[name]
	if ok {
		return ch
	}

	ch, err := t.makeSubscriberBool(name)
	if err != nil {
		t.errChan <- vice.Err{Name: name, Err: err}
		return make(chan bool)
	}

	t.receiveBoolChans[name] = ch
	return ch
}

func (t *Transport) makeSubscriberBool(name string) (chan bool, error) {

	s, err := t.newEncodedConnection()
	if err != nil {
		return nil, err
	}
	ch := make(chan bool, 1024)
	var sub unsubscriber

	sub, err = s.QueueSubscribe(name, t.NatsQueueGroup, func(b *bool) {
		ch <- *b
	})

	if err == nil {
		t.subscriptions = append(t.subscriptions, sub)
	}
	return ch, err
}

// SendBool gets a channel on which messages with the
// specified name may be sent.
func (t *Transport) SendBool(name string) chan<- bool {
	t.Lock()
	defer t.Unlock()

	ch, ok := t.sendBoolChans[name]
	if ok {
		return ch
	}

	ch, err := t.makePublisherBool(name)
	if err != nil {
		t.errChan <- vice.Err{Name: name, Err: err}
		return make(chan bool)
	}

	t.sendBoolChans[name] = ch
	return ch
}

func (t *Transport) makePublisherBool(name string) (chan bool, error) {
	var (
		c   publisher
		err error
	)

	c, err = t.newEncodedConnection()
	if err != nil {
		return nil, err
	}

	ch := make(chan bool, 1024)

	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		for {
			select {
			case <-t.stopPubChan:
				if len(ch) != 0 && t.natsConn.IsConnected() {
					continue
				}
				return
			case msg := <-ch:
				err := c.Publish(name, msg)
				t.handlePublishError(name, err)
			}
		}
	}()

	return ch, nil
}

func (t *Transport) handlePublishError(name string, err error) {
	if err == nil {
		return
	}
	t.errChan <- vice.Err{Name: name, Err: err}
	time.Sleep(1 * time.Second)
}
