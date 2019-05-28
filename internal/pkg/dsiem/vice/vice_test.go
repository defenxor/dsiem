// Copyright (c) 2019 PT Defender Nusa Semesta and contributors, All rights reserved.
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

package vice

import (
	"errors"
	"testing"
)

func TestError(t *testing.T) {
	type errTest struct {
		e        Err
		expected string
	}
	var tbl = []errTest{
		{Err{[]byte("Msg"), "Name1", errors.New("Name1")}, "Name1: |Name1| <- `Msg`"},
		{Err{[]byte{}, "Name2", errors.New("Name2")}, "Name2: |Name2|"},
	}
	for _, tt := range tbl {
		actual := tt.e.Error()
		if actual != tt.expected {
			t.Errorf("Error message for %s is %s. Expected %s.", tt.e.Name, actual, tt.expected)
		}
	}
}
