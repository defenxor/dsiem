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
	l, err := New(500, 100)
	if err != nil {
		t.Fatal("CAnnot create new limiter")
	}

	a := l.Limit()
	b := l.Lower()
	c := l.Raise()
	if b >= a || a < b {
		t.Errorf("r: %d %d %d", a, b, c)
	}
	for i := 0; i <= 60; i++ {
		l.Lower()
	}

	for i := 0; i <= 60; i++ {
		l.Raise()
	}
}

func TestWait(t *testing.T) {
	l, err := New(50, 1)
	if err != nil {
		t.Fatal("CAnnot create new limiter")
	}
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	err = l.Wait(ctx)
	if err != nil {
		t.Error(err)
	}

}
