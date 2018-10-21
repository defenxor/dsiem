package test

import (
	"testing"
)

func TestTestDir(t *testing.T) {
	d, err := DirEnv()
	if err != nil || d == "" {
		t.Fatal(err)
	}
}
