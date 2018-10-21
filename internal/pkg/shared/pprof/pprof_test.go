package pprof

import (
	"testing"
)

func TestPProf(t *testing.T) {
	prof := []string{"cpu", "memory", "mutex", "block"}

	for _, p := range prof {
		f, err := GetProfiler(p)
		if err != nil {
			t.Fatal("Cannot start profiler: " + err.Error())
		}
		f.Stop()
	}
	_, err := GetProfiler("invalid")
	if err == nil {
		t.Fatal("invalid profiler should results in non-nil err")
	}
}
