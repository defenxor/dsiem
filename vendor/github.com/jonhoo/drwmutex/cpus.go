// +build !linux

package drwmutex

func map_cpus() (cpus map[uint64]int) {
	cpus = make(map[uint64]int)
	return
}
