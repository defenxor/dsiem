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

// TODO consolidate all APM related stuff here

package apm

import (
	"sync"
	"time"

	"go.elastic.co/apm"
	"go.elastic.co/apm/module/apmhttp"
)

var enabled bool
var distributed bool
var mu = sync.RWMutex{}

//Enabled returns whether apm is enabled
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled
}

//Enable set apm status
func Enable(e bool) {
	mu.Lock()
	enabled = e
	mu.Unlock()
}

// TraceHeader defines structure for distributed tracing headers
type TraceHeader struct {
	Traceparent string
	TraceState  string
}

// Transaction wraps transaction from apm Default tracer and make it concurrency safe
type Transaction struct {
	sync.Mutex
	Tx    *apm.Transaction
	ended bool
}

// StartTransaction returns a mutex protected apm.Transaction with optional starting time.
func StartTransaction(name, transactionType string, startTime *time.Time, parentHeader *TraceHeader) (tx *Transaction) {
	txObj := Transaction{}
	opts := apm.TransactionOptions{}
	if startTime != nil {
		opts.Start = *startTime
	}
	if parentHeader != nil {
		tc := apm.TraceContext{}
		tc, _ = apmhttp.ParseTraceparentHeader(parentHeader.Traceparent)
		tc.State, _ = apmhttp.ParseTracestateHeader(parentHeader.TraceState)
		opts.TraceContext = tc
	}

	txObj.Tx = apm.DefaultTracer.StartTransactionOptions(name, transactionType, opts)
	tx = &txObj
	return
}

// Recover returns an apm.DefaultTracer.Recover function to be deferred
func (t *Transaction) Recover() {
	// this is copied from apm.DefaultTracer.Recover(t.Tx)
	v := recover()
	if v == nil {
		return
	}
	e := apm.DefaultTracer.Recovered(v)
	e.SetTransaction(t.Tx)
	e.Send()
}

// SetCustom set custom value for the transaction
func (t *Transaction) SetCustom(key string, value string) {
	t.Lock()
	defer t.Unlock()
	if t.ended {
		return
	}
	defer t.Recover()
	t.Tx.Context.SetTag(key, value)
}

// Result set the result for the transaction
func (t *Transaction) Result(value string) {
	t.Lock()
	defer t.Unlock()
	if t.ended {
		return
	}
	t.Tx.Result = value
}

// SetError set and send error
func (t *Transaction) SetError(err error) {
	e := apm.DefaultTracer.NewError(err)
	e.SetTransaction(t.Tx)
	e.Send()
}

// End completes the transaction
func (t *Transaction) End() {
	t.Lock()
	defer t.Unlock()
	if t.ended {
		return
	}
	t.ended = true
	t.Tx.End()
}

// GetTraceContext gets info for distributed transaction
func (t *Transaction) GetTraceContext() (th *TraceHeader) {
	t.Lock()
	defer t.Unlock()
	thObj := TraceHeader{}
	traceContext := t.Tx.TraceContext()
	thObj.Traceparent = apmhttp.FormatTraceparentHeader(traceContext)
	thObj.TraceState = traceContext.State.String()
	th = &thObj
	return
}
