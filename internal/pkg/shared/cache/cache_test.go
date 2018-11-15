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

package cache

import (
	"bytes"
	"strconv"
	"testing"
)

func TestCache(t *testing.T) {

	type cacheTests struct {
		key      string
		val      interface{}
		expected interface{}
	}
	var tbl = []cacheTests{
		{"key1", "string", "string"},
		{"key2", 1, 1},
	}

	// test for fail init first
	_, err := New("CacheName", 0, 3)
	if err == nil {
		t.Error("Expected error for shard eq. 3")
	}

	c, err := New("CacheName", 0, 0)
	if err != nil {
		t.Error(err)
	}

	for _, tt := range tbl {
		switch v := tt.val.(type) {
		case int:
			c.Set(tt.key, []byte(strconv.Itoa(v)))
		case string:
			c.Set(tt.key, []byte(v))
		}

		actual, err := c.Get(tt.key)
		if err != nil {
			t.Error(err)
		}

		var r []byte
		switch v := tt.expected.(type) {
		case int:
			r = []byte(strconv.Itoa(v))
		case string:
			r = []byte(v)
		}
		if !bytes.Equal(actual, r) {
			t.Errorf("key %v val %v, result is %v expected %v.", tt.key, tt.val, actual, r)
		}
	}

}
