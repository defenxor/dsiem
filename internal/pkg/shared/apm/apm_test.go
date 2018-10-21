package apm

import "testing"

func TestAPM(t *testing.T) {
	Enable(true)
	if !Enabled() {
		t.Errorf("APM expected to be enabled")
	}
}
