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
	c, err := New("CacheName", 0, 3)
	if err == nil {
		t.Error("Expected error for shard eq. 3")
	}

	c, err = New("CacheName", 0, 0)
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
