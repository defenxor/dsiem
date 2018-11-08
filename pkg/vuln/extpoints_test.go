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

package vuln

import (
	"context"
	"reflect"
	"testing"
)

type Dummy struct{}

func (d Dummy) Initialize(b []byte) (err error) {
	return
}

func (d Dummy) CheckIPPort(ctx context.Context, ip string, port int) (found bool, results []Result, err error) {
	return
}

func TestExtPoints(t *testing.T) {
	ext1 := RegisterExtension(new(Dummy), "Dummy")
	if !reflect.DeepEqual(ext1, []string{"Checker"}) {
		t.Fatal("Cannot register extension")
	}

	var cs = Checkers

	m := cs.All()
	m2 := make(map[string]Checker)
	if reflect.DeepEqual(m, m2) {
		t.Fatal("Expect a registered extension")
	}
	names := cs.Names()
	if !reflect.DeepEqual(names, []string{"Dummy"}) {
		t.Fatal("Expect a registered extension")
	}
	c := cs.Select(names)
	if c == nil {
		t.Fatal("Expect a registered extension")
	}

	c1 := cs.Lookup("Dummy")
	if c1 == nil {
		t.Fatal("Cannot lookup extension")
	}
	c2 := cs.Lookup("NA")
	if c2 != nil {
		t.Fatal("Expect c equals nil")
	}

	if !cs.Register(c1, "Dummy2") {
		t.Fatal("Cannot register new extension")
	}
	if cs.Register(c1, "Dummy2") {
		t.Fatal("Expected to fail on registering existing extension")
	}
	if cs.Register(c1, "") {
		t.Fatal("Expected to fail on registering existing extension")
	}
	if !cs.Unregister("Dummy2") {
		t.Fatal("Cannot unregister extension")
	}
	if cs.Unregister("Dummy2") {
		t.Fatal("Expected to fail on unregistering non-existent extension")
	}

	ext := UnregisterExtension("Dummy")
	if !reflect.DeepEqual(ext, []string{"Checker"}) {
		t.Fatal("Cannot unregister extension")
	}

}
