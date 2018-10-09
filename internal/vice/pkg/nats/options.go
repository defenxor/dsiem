package nats

import "github.com/nats-io/go-nats"

// Options can be used to create a customized transport.
type Options struct {
	Conn       *nats.Conn
	UseEncoded bool
}

// Option is a function on the options for a nats transport.
type Option func(*Options)

// WithConnection is an Option to set underlying nats connection.
func WithConnection(c *nats.Conn) Option {
	return func(o *Options) {
		o.Conn = c
	}
}
