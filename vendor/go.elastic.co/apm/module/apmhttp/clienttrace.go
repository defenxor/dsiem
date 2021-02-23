// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package apmhttp // import "go.elastic.co/apm/module/apmhttp"

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http/httptrace"
	"sync"

	"go.elastic.co/apm"
)

// WithClientTrace returns a ClientOption for
// tracing events within HTTP client requests.
func WithClientTrace() ClientOption {
	return func(rt *roundTripper) {
		rt.traceRequests = true
	}
}

type connectKey struct {
	network, addr string
}

type requestTracer struct {
	DNS,
	TLS,
	Request,
	Response *apm.Span

	mu       sync.RWMutex
	Connects map[connectKey]*apm.Span
}

func withClientTrace(ctx context.Context, tx *apm.Transaction, parent *apm.Span) (context.Context, *requestTracer) {
	r := requestTracer{
		Connects: make(map[connectKey]*apm.Span),
	}

	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		DNSStart: func(i httptrace.DNSStartInfo) {
			r.DNS = tx.StartSpan(fmt.Sprintf("DNS %s", i.Host), "external.http.dns", parent)
		},

		DNSDone: func(i httptrace.DNSDoneInfo) {
			r.DNS.End()
		},

		ConnectStart: func(network, addr string) {
			span := tx.StartSpan(fmt.Sprintf("Connect %s", addr), "external.http.connect", parent)
			r.mu.Lock()
			r.Connects[connectKey{network: network, addr: addr}] = span
			r.mu.Unlock()
		},

		ConnectDone: func(network, addr string, err error) {
			r.mu.RLock()
			span := r.Connects[connectKey{network: network, addr: addr}]
			r.mu.RUnlock()
			span.End()
		},

		GotConn: func(info httptrace.GotConnInfo) {
			r.Request = tx.StartSpan("Request", "external.http.request", parent)
		},

		TLSHandshakeStart: func() {
			r.TLS = tx.StartSpan("TLS", "external.http.tls", parent)
		},

		TLSHandshakeDone: func(_ tls.ConnectionState, _ error) {
			r.TLS.End()
		},

		GotFirstResponseByte: func() {
			r.Request.End()
			r.Response = tx.StartSpan("Response", "external.http.response", parent)
		},
	}), &r
}

func (r *requestTracer) end() {
	if r.Response != nil {
		r.Response.End()
	}
}
