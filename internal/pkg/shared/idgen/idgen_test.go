package idgen

import (
	"testing"
)

func TestIdgen(t *testing.T) {
	_, err := GenerateID()
	if err != nil {
		t.Fatal(err)
	}
}
