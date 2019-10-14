package goconcurrentqueue

// Queue interface with basic && common queue functions
type Queue interface {
	// Enqueue element
	Enqueue(interface{}) error
	// Dequeue element
	Dequeue() (interface{}, error)
	// DequeueOrWaitForNextElement dequeues an element (if exist) or waits until the next element gets enqueued and returns it.
	// Multiple calls to DequeueOrWaitForNextElement() would enqueue multiple "listeners" for future enqueued elements.
	DequeueOrWaitForNextElement() (interface{}, error)
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
