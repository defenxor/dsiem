package goconcurrentqueue

import "github.com/defenxor/dsiem/internal/pkg/dsiem/event"

// Queue interface with basic && common queue functions
type Queue interface {
	// Enqueue element
	Enqueue(event.NormalizedEvent) error
	// Dequeue element
	Dequeue() (event.NormalizedEvent, error)
	// DequeueOrWaitForNextElement dequeues an element (if exist) or waits until the next element gets enqueued and returns it.
	// Multiple calls to DequeueOrWaitForNextElement() would enqueue multiple "listeners" for future enqueued elements.
	DequeueOrWaitForNextElement() (event.NormalizedEvent, error)
	// Get number of enqueued elements
	GetLen() int
	// Get queue's capacity
	GetCap() int

	// Lock the queue. No enqueue/dequeue/remove/get operations will be allowed after this point.
	Lock()
	// Unlock the queue.
	Unlock()
	// Return true whether the queue is locked
	IsLocked() bool
}

// NewQueue creates a new object depending on capacity
func NewQueue(cap int) (q Queue) {
	if cap == 0 {
		q = NewFIFO()
	} else {
		q = NewFixedFIFO(cap)
	}
	return
}
