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
	Tx *elasticapm.Transaction
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

// Recover returns a elasticapm.DefaultTracer.Recover function to be deferred
func (t *Transaction) Recover() func() {
	f := func() {
		elasticapm.DefaultTracer.Recover(t.Tx)
	}
	return f
}

// SetCustom set custom value for the transaction
func (t *Transaction) SetCustom(key string, value interface{}) {
	t.Lock()
	t.Tx.Context.SetCustom(key, value)
	t.Unlock()
}

// Result set the result for the transaction
func (t *Transaction) Result(value string) {
	t.Lock()
	t.Tx.Result = value
	t.Unlock()
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
	t.Tx.End()
	t.Unlock()
}
