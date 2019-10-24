package goconcurrentqueue

import (
	"github.com/defenxor/dsiem/internal/pkg/dsiem/event"
	"sync"
)

const (
	// WaitForNextElementChanCapacity specifies waitchan capacity
	WaitForNextElementChanCapacity           = 1000
	dequeueOrWaitForNextElementInvokeGapTime = 10
)

// FixedFIFO is a fixed capacity FIFO (First In First Out) concurrent queue
type FixedFIFO struct {
	queue    chan event.NormalizedEvent
	lockChan chan struct{}
	// queue for watchers that will wait for next elements (if queue is empty at DequeueOrWaitForNextElement execution )
	waitForNextElementChan chan chan event.NormalizedEvent
	queueLength            int
	lenLock                sync.RWMutex
}

// NewFixedFIFO creates new obj
func NewFixedFIFO(capacity int) *FixedFIFO {
	queue := &FixedFIFO{}
	queue.initialize(capacity)

	return queue
}

func (st *FixedFIFO) initialize(capacity int) {
	st.queue = make(chan event.NormalizedEvent, capacity)
	st.lockChan = make(chan struct{}, 1)
	st.waitForNextElementChan = make(chan chan event.NormalizedEvent, WaitForNextElementChanCapacity)
}

// Enqueue enqueues an element. Returns error if queue is locked or it is at full capacity.
func (st *FixedFIFO) Enqueue(value event.NormalizedEvent) error {
	if st.IsLocked() {
		return NewQueueError(QueueErrorCodeLockedQueue, "The queue is locked")
	}

	// check if there is a listener waiting for the next element (this element)
	select {
	case listener := <-st.waitForNextElementChan:
		// send the element through the listener's channel instead of enqueue it
		listener <- value

	default:
		// enqueue the element following the "normal way"
		select {
		case st.queue <- value:
		default:
			return NewQueueError(QueueErrorCodeFullCapacity, "FixedFIFO queue is at full capacity")
		}
	}

	return nil
}

// Dequeue dequeues an element. Returns error if: queue is locked, queue is empty or internal channel is closed.
func (st *FixedFIFO) Dequeue() (event.NormalizedEvent, error) {
	if st.IsLocked() {
		return event.NormalizedEvent{}, NewQueueError(QueueErrorCodeLockedQueue, "The queue is locked")
	}

	select {
	case value, ok := <-st.queue:
		if ok {
			return value, nil
		}
		return event.NormalizedEvent{}, NewQueueError(QueueErrorCodeInternalChannelClosed, "internal channel is closed")
	default:
		return event.NormalizedEvent{}, NewQueueError(QueueErrorCodeEmptyQueue, "empty queue")
	}
}

// DequeueOrWaitForNextElement dequeues an element (if exist) or waits until the next element gets enqueued and returns it.
// Multiple calls to DequeueOrWaitForNextElement() would enqueue multiple "listeners" for future enqueued elements.
func (st *FixedFIFO) DequeueOrWaitForNextElement() (event.NormalizedEvent, error) {
	if st.IsLocked() {
		return event.NormalizedEvent{}, NewQueueError(QueueErrorCodeLockedQueue, "The queue is locked")
	}
	st.lenLock.Lock()
	st.queueLength = len(st.queue)
	st.lenLock.Unlock()

	select {
	case value, ok := <-st.queue:
		if ok {
			return value, nil
		}
		return event.NormalizedEvent{}, NewQueueError(QueueErrorCodeInternalChannelClosed, "internal channel is closed")

	// queue is empty, add a listener to wait until next enqueued element is ready
	default:
		// channel to wait for next enqueued element
		waitChan := make(chan event.NormalizedEvent)

		select {
		// enqueue a watcher into the watchForNextElementChannel to wait for the next element
		case st.waitForNextElementChan <- waitChan:
			// return the next enqueued element, if any
			return <-waitChan, nil
		default:
			// too many watchers (waitForNextElementChanCapacity) enqueued waiting for next elements
			return event.NormalizedEvent{}, NewQueueError(QueueErrorCodeEmptyQueue, "empty queue and can't wait for next element")
		}

		//return nil, NewQueueError(QueueErrorCodeEmptyQueue, "empty queue")
	}
}

// GetLen returns queue's length (total enqueued elements)
func (st *FixedFIFO) GetLen() (v int) {
	st.lenLock.RLock()
	v = st.queueLength
	st.lenLock.RUnlock()
	return
}

// GetCap returns the queue's capacity
func (st *FixedFIFO) GetCap() int {
	st.Lock()
	defer st.Unlock()

	return cap(st.queue)
}

// Lock the queue
func (st *FixedFIFO) Lock() {
	// non-blocking fill the channel
	select {
	case st.lockChan <- struct{}{}:
	default:
	}
}

// Unlock queue
func (st *FixedFIFO) Unlock() {
	// non-blocking flush the channel
	select {
	case <-st.lockChan:
	default:
	}
}

// IsLocked check if the queue is locked
func (st *FixedFIFO) IsLocked() bool {
	return len(st.lockChan) >= 1
}
