// package vice is a modification of Vice (https://github.com/matryer/vice)
// to directly use event.NormalizedEvent instead of []byte for the chans.

// this is done to address what perhaps to be a buffer re-use issue on the underlying
// nats library, which causes new message to still contain left-overs from older
// message under heavy load. If this is no longer the case, we should switch to use
// the main vice library again.

package vice

import (
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"

	"fmt"
)

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
