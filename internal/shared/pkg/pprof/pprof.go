package pprof

import (
	"errors"

	"github.com/pkg/profile"
)

// GetProfiler returns func to start pprof for a given profile
func GetProfiler(p string) (i interface{ Stop() }, err error) {
	switch p {
	case "cpu":
		i = profile.Start(profile.CPUProfile)
	case "memory":
		i = profile.Start(profile.MemProfile)
	case "mutex":
		i = profile.Start(profile.MutexProfile)
	case "block":
		i = profile.Start(profile.MutexProfile)
	default:
		i = nil
		err = errors.New("invalid profiler, valid option is cpu|memory|mutex|block")
	}
	return
}
