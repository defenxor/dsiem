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

package apm

import (
	"errors"
	"testing"
	"time"
)

func TestAPM(t *testing.T) {
	Enable(true)
	if !Enabled() {
		t.Errorf("APM expected to be enabled")
	}

	tx := StartTransaction("test", "testType", nil)
	if tx.Tx == nil {
		t.Fatal("Expected transaction not to be nil")
	}
	if tx.Tx.Name != "test" {
		t.Fatal("Expected tx.Name to be test")
	}
	if tx.Tx.Type != "testType" {
		t.Fatal("Expected tx.Name to be testType")
	}

	tm := time.Now()
	tx = StartTransaction("test", "test", &tm)
	if tx.Tx == nil {
		t.Fatal("Expected transaction not to be nil")
	}

	tx.Result("result test")
	if tx.Tx.Result != "result test" {
		t.Error("Expected result to be 'result test'")
	}

	// dont know how to verify the output of these without checking the output at apm server
	tx.SetCustom("key", "val")
	tx.SetError(errors.New("Test error"))
	tx.Recover()
	tx.End()
	tx.Result("Try to change result")
	// if tx.Tx.Result == "Try to change result" {
	//	 t.Fatal("Expected to not be able to set result after End()")
	// }

	// longtimeAgo := time.Now().AddDate(-30, 0, 0)
	//	tx = StartTransaction("test", "test2", &longtimeAgo)
	//	time.Sleep(time.Second)
	tx.End()
	defer tx.Recover()
	trick := false // this is just a workaround to skip vet on reachable t.Error below
	if !trick {
		panic("panic")
	}
	t.Error("expected to recover from panic and never reach this point")
}
