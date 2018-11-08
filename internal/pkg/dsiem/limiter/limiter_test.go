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

package limiter

import (
	"context"
	"testing"
	"time"
)

func TestNew(t *testing.T) {

	type limTests struct {
		min      int
		max      int
		err      bool
		expected int
	}

	var tbl = []limTests{
		{0, 0, false, 0}, {0, 1, false, 1}, {1, 0, true, 0},
	}

	for _, tt := range tbl {
		actual, err := New(tt.max, tt.min)
		if (!tt.err && err != nil) || (tt.err && err == nil) {
			t.Errorf("Limit(%d,%d): expected err %v, actual err %v", tt.min, tt.max, tt.err, err)
		}
		if err == nil && actual.Limit() != tt.expected {
			t.Errorf("Limit(%d,%d): expected %v, actual limit %v", tt.min, tt.max, tt.expected, actual)
		}
	}
}

func TestModif(t *testing.T) {
	min := 100
	max := 500
	iter := 100
	l, err := New(max, min)
	if err != nil {
		t.Fatal("Cannot create new limiter")
	}

	a := l.Limit()
	b := l.Lower()
	c := l.Raise()
	if b >= a || a < b {
		t.Errorf("r: %d %d %d", a, b, c)
	}
	for i := 0; i <= iter; i++ {
		l.Lower()
	}
	if res := l.Limit(); res != min {
		t.Errorf("expected %v, received %v", min, res)
	}

	for i := 0; i <= iter; i++ {
		l.Raise()
	}
	if res := l.Limit(); res != max {
		t.Errorf("expected %v, received %v", max, res)
	}

}

func TestWait(t *testing.T) {
	l, err := New(50, 1)
	if err != nil {
		t.Fatal("CAnnot create new limiter")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = l.Wait(ctx)
	if err != nil {
		t.Error(err)
	}

}
