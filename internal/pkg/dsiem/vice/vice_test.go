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
