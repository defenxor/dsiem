package vice

import (
	"dsiem/internal/pkg/dsiem/event"

	"fmt"
)

// Transport is copied from Vice to directly use event.NormalizedEvent
// instead of []byte for the chans

// Transport provides message sending and receiving
// capabilities over a messaging queue technology.
// Clients should always check for errors coming through ErrChan.
type Transport interface {
	// Receive gets a channel on which to receive messages
	// with the specified name.
	Receive(name string) <-chan event.NormalizedEvent
	// Send gets a channel on which messages with the
	// specified name may be sent.
	Send(name string) chan<- event.NormalizedEvent
	// ErrChan gets a channel through which errors
	// are sent.
	// Receive gets a channel on which to receive messages
	// with the specified name.
	ReceiveBool(name string) <-chan bool
	// Send gets a channel on which messages with the
	// specified name may be sent.
	SendBool(name string) chan<- bool
	// ErrChan gets a channel through which errors
	// are sent.
	ErrChan() <-chan error

	// Stop stops the transport. The channel returned from Done() will be closed
	// when the transport has stopped.
	Stop()
	// Done gets a channel which is closed when the
	// transport has successfully stopped.
	Done() chan struct{}
}

// Err represents a vice error.
type Err struct {
	Message []byte
	Name    string
	Err     error
}

func (e Err) Error() string {
	if len(e.Message) > 0 {
		return fmt.Sprintf("%s: |%s| <- `%s`", e.Err, e.Name, string(e.Message))
	}
	return fmt.Sprintf("%s: |%s|", e.Err, e.Name)
}
