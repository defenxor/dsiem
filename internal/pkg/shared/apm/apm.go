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

	"github.com/elastic/apm-agent-go"
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

// Transaction wraps transaction from elasticapm Default tracer and make it concurrency safe
type Transaction struct {
	sync.Mutex
	Tx    *elasticapm.Transaction
	ended bool
}

// StartTransaction returns a mutex protected elasticapm.Transaction, with optional starting time
func StartTransaction(name, transactionType string, startTime *time.Time) *Transaction {
	txObj := Transaction{}
	if startTime != nil {
		opts := elasticapm.TransactionOptions{TraceContext: elasticapm.TraceContext{}, Start: *startTime}
		txObj.Tx = elasticapm.DefaultTracer.StartTransactionOptions(name, transactionType, opts)
	} else {
		txObj.Tx = elasticapm.DefaultTracer.StartTransaction(name, transactionType)
	}
	return &txObj
}

// Recover returns an elasticapm.DefaultTracer.Recover function to be deferred
func (t *Transaction) Recover() {
	// this is copied from elasticapm.DefaultTracer.Recover(t.Tx)
	v := recover()
	if v == nil {
		return
	}
	elasticapm.DefaultTracer.Recovered(v, t.Tx).Send()
}

// SetCustom set custom value for the transaction
func (t *Transaction) SetCustom(key string, value interface{}) {
	/*
		if either of the following still occur:
		- index out of range error in Tx.Context.SetCustom
		- concurrent map write in Tx.Context.SetTag
		then this func should be set to no op
	*/
	t.Lock()
	defer t.Unlock()
	defer t.Recover()
	t.Tx.Context.SetCustom(key, value)
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

// SetError set and send error fom the transaction
func (t *Transaction) SetError(err error) {
	t.Lock()
	defer t.Unlock()
	e := elasticapm.DefaultTracer.NewError(err)
	e.Transaction = t.Tx
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
