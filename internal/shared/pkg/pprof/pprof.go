package pprof

import (
	"github.com/pkg/profile"
)

// GetProfiler returns func to start pprof for a given profile
func GetProfiler(p string) interface{ Stop() } {
	switch p {
	case "memory":
		return profile.Start(profile.MemProfile)
	case "mutex":
		return profile.Start(profile.MutexProfile)
	case "block":
		return profile.Start(profile.MutexProfile)
	default:
		return profile.Start(profile.CPUProfile)
	}
}
