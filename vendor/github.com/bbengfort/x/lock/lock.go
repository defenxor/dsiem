/*
Package lock provides lock contention diagnostics for go mutexes.

This package provides a write-only mutex as well as a read and write mutex
that can be swapped for the mutexes in the sync package. They extend the sync
package by tracking the caller when they request a lock or unlock.

If lock contention  is experienced in such a way that the program doesn't
panic (primarily when  there are many go routines all making progress in
different parts of the program), you can print a report of which callers are
attempting to acquire locks, and which callers have not released their locks
yet, helping to diagnose contention issues.
*/
package lock

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

// UnknownCaller is used as the default caller if we cannot query it.
const UnknownCaller = "unknown caller"

// Caller returns the caller of the function that calls this method.
func caller() string {
	pc, _, _, ok := runtime.Caller(2)
	details := runtime.FuncForPC(pc)
	if ok && details != nil {
		return details.Name()
	}
	return UnknownCaller
}

// MutexD wraps sync.Mutex to provide tracking for methods that call the lock
// object. Use the same way you would use a Mutex!
type MutexD struct {
	sync.Mutex
	initialized bool
	locks       map[string]int64
	signals     chan *lockSignal
}

// Init the lock and internal data structures like the map. No need to Init()
// manually though as the lock methods do a check to ensure that it's ready.
func (l *MutexD) Init() {
	if !l.initialized {
		l.locks = make(map[string]int64)
		l.signals = make(chan *lockSignal, 1000)
		go l.listner()
		l.initialized = true
	}
}

// The listener grabs lock signals from the channels and updates the map
// accordingly. This is done to avoid concurrent map reads and writes.
func (l *MutexD) listner() {
	for s := range l.signals {
		if s.locked {
			l.locks[s.caller]++
		} else {
			l.locks[s.caller]--
		}
	}
}

// Lock the data structure, blocking all other calls that are requesting a
// lock until unlock is called. This method provides diagnostic information
// by recording the caller of the lock in an internal map. You can print a
// report to see who is attempting to acquire a lock and who is still holding
// any locks in the system.
func (l *MutexD) Lock() {
	l.Init()
	l.signals <- &lockSignal{lock: writeLock, locked: true, caller: caller()}
	l.Mutex.Lock()
}

// Unlock the data structure, allowing any other blocked calls that have
// requested a lock to acquire it. This method removes the caller from the
// internal map so it's easy to see who still is attempting to acquire locks
// and who has released them (or hasn't released them yet).
func (l *MutexD) Unlock() {
	l.Init()
	l.signals <- &lockSignal{lock: writeLock, locked: false, caller: caller()}
	l.Mutex.Unlock()
}

// String returns a report about who is attempting to acquire locks and which
// callers currently hold locks. E.g. if more than one lock is in the lock
// map than the first one is holding the lock and the others are awaiting it.
func (l *MutexD) String() string {
	output := make([]string, 0)
	for key, val := range l.locks {
		msg := fmt.Sprintf("%d locks requested by %s", val, key)
		output = append(output, msg)
	}
	return strings.Join(output, "\n")
}

//===========================================================================
// RW Mutex Diagnostics
//===========================================================================

// RWMutexD wraps a sync.RWMutex to provide tracking for methods that call the
// lock object. Use the same way you would use a mutex in order to diagnose
// requested read locks, write locks, and currently held read and write locks.
type RWMutexD struct {
	sync.RWMutex
	initialized bool
	wlocks      map[string]int64
	rlocks      map[string]int64
	signals     chan *lockSignal
}

// Init the lock and internal data structures like the maps. No need to call
// Init() manually, though, as the lock methods do a check beforehand.
func (l *RWMutexD) Init() {
	if !l.initialized {
		l.wlocks = make(map[string]int64)
		l.rlocks = make(map[string]int64)
		l.signals = make(chan *lockSignal, 1000)

		go l.listner()

		l.initialized = true
	}
}

// The listener grabs lock signals from the channels and updates the map
// accordingly. This is done to avoid concurrent map reads and writes. This
// listener specializes itself by detecting the lock type.
func (l *RWMutexD) listner() {
	for s := range l.signals {
		switch s.lock {
		case writeLock:
			if s.locked {
				l.wlocks[s.caller]++
			} else {
				l.wlocks[s.caller]--
			}
		case readLock:
			if s.locked {
				l.rlocks[s.caller]++
			} else {
				l.rlocks[s.caller]--
			}
		}

	}
}

// Lock the data structure, blocking all other calls that are requesting a
// lock until unlock is called. This method provides diagnostic information
// by recording the caller of the lock in an internal map. You can print a
// report to see who is attempting to acquire a lock and who is still holding
// any locks in the system.
func (l *RWMutexD) Lock() {
	l.Init()
	l.signals <- &lockSignal{lock: writeLock, locked: true, caller: caller()}
	l.RWMutex.Lock()
}

// Unlock the data structure, allowing any other blocked calls that have
// requested a lock to acquire it. This method removes the caller from the
// internal map so it's easy to see who still is attempting to acquire locks
// and who has released them (or hasn't released them yet).
func (l *RWMutexD) Unlock() {
	l.Init()
	l.signals <- &lockSignal{lock: writeLock, locked: false, caller: caller()}
	l.RWMutex.Unlock()
}

// RLock the data structure, blocking all other calls that are requesting a
// lock until unlock is called. This method provides diagnostic information
// by recording the caller of the lock in an internal map. You can print a
// report to see who is attempting to acquire a lock and who is still holding
// any locks in the system.
func (l *RWMutexD) RLock() {
	l.Init()
	l.signals <- &lockSignal{lock: readLock, locked: true, caller: caller()}
	l.RWMutex.RLock()
}

// RUnlock the data structure, allowing any other blocked calls that have
// requested a lock to acquire it. This method removes the caller from the
// internal map so it's easy to see who still is attempting to acquire locks
// and who has released them (or hasn't released them yet).
func (l *RWMutexD) RUnlock() {
	l.Init()
	l.signals <- &lockSignal{lock: readLock, locked: false, caller: caller()}
	l.RWMutex.RUnlock()
}

// String returns a report about who is attempting to acquire locks and which
// callers currently hold locks. E.g. if more than one lock is in the lock
// map than the first one is holding the lock and the others are awaiting it.
func (l *RWMutexD) String() string {
	output := make([]string, 0)

	// Write locks
	for key, val := range l.wlocks {
		msg := fmt.Sprintf("%d locks requested by %s", val, key)
		output = append(output, msg)
	}

	// Read locks
	for key, val := range l.rlocks {
		msg := fmt.Sprintf("%d read locks requested by %s", val, key)
		output = append(output, msg)
	}

	return strings.Join(output, "\n")
}

//===========================================================================
// Lock signals to the listeners
//===========================================================================

const (
	readLock lockType = iota
	writeLock
)

type lockType uint8

// LockSignal is used to ensure that there are no concurrent reads and writes
// to the internal maps of the Lock object. LockSignal objects are sent to a
// channel that serializes the lock information.
type lockSignal struct {
	locked bool     // true for lock false for unlock
	lock   lockType // either ReadLock or WriteLock
	caller string   // name of the calling function
}
