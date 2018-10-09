package nats

import (
	"dsiem/internal/pkg/dsiem/event"
	"dsiem/internal/pkg/dsiem/vice"

	"time"
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
	t.subscriptions = append(t.subscriptions, sub)
	return ch, nil
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
				if err := c.Publish(name, msg); err != nil {
					t.errChan <- vice.Err{Name: name, Err: err}
					time.Sleep(1 * time.Second)
				}
			}
		}
	}()

	return ch, nil
}
