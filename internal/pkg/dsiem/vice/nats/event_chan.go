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
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"github.com/defenxor/dsiem/internal/pkg/dsiem/vice"
)

// Receive gets a channel on which to receive messages
func (t *Transport) Receive(name string) <-chan event.NormalizedEvent {
	t.Lock()
	defer t.Unlock()

	ch, ok := t.receiveChans[name]
	if ok {
		return ch
	}

	ch, err := t.makeSubscriber(name)
	if err != nil {
		t.errChan <- vice.Err{Name: name, Err: err}
		return make(chan event.NormalizedEvent)
	}

	t.receiveChans[name] = ch
	return ch
}

func (t *Transport) makeSubscriber(name string) (chan event.NormalizedEvent, error) {

	s, err := t.newEncodedConnection()
	if err != nil {
		return nil, err
	}
	ch := make(chan event.NormalizedEvent, 1024)
	var sub unsubscriber

	sub, err = s.QueueSubscribe(name, t.NatsQueueGroup, func(e *event.NormalizedEvent) {
		ch <- *e
	})

	if err == nil {
		t.subscriptions = append(t.subscriptions, sub)
	}
	return ch, err
}

// Send gets a channel on which messages with the
// specified name may be sent.
func (t *Transport) Send(name string) chan<- event.NormalizedEvent {
	t.Lock()
	defer t.Unlock()

	ch, ok := t.sendChans[name]
	if ok {
		return ch
	}

	ch, err := t.makePublisher(name)
	if err != nil {
		t.errChan <- vice.Err{Name: name, Err: err}
		return make(chan event.NormalizedEvent)
	}

	t.sendChans[name] = ch
	return ch
}

func (t *Transport) makePublisher(name string) (chan event.NormalizedEvent, error) {
	var (
		c   publisher
		err error
	)

	c, err = t.newEncodedConnection()
	if err != nil {
		return nil, err
	}

	ch := make(chan event.NormalizedEvent, 1024)

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
