package session

import (
	"sync"
	"sync/atomic"
)

// EventBus is a lightweight in-process pub/sub used to broadcast thread-local events.
// It is intentionally thread-scoped to avoid cross-thread state leaks.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string]map[int64]chan Event
	nextID      atomic.Int64
	bufferSize  int
}

// NewEventBus constructs a new EventBus.
func NewEventBus(bufferSize int) *EventBus {
	if bufferSize <= 0 {
		bufferSize = 64
	}
	return &EventBus{
		subscribers: make(map[string]map[int64]chan Event),
		bufferSize:  bufferSize,
	}
}

// Publish delivers the event to all subscribers for the given thread.
// It never blocks; when a subscriber buffer is full, the oldest event is dropped.
// It returns true when any subscriber had to drop an event due to backpressure.
func (b *EventBus) Publish(threadID string, e Event) (dropped bool) {
	if b == nil {
		return false
	}

	b.mu.RLock()
	subs := b.subscribers[threadID]
	b.mu.RUnlock()
	if len(subs) == 0 {
		return false
	}

	// Snapshot channels to avoid holding the lock while sending.
	chans := make([]chan Event, 0, len(subs))
	for _, ch := range subs {
		chans = append(chans, ch)
	}

	for _, ch := range chans {
		// Fast path: try send.
		select {
		case ch <- e:
			continue
		default:
		}

		// Drop oldest then try once more.
		select {
		case <-ch:
			dropped = true
		default:
			// Still full; nothing we can do without blocking.
			dropped = true
		}
		select {
		case ch <- e:
		default:
			// If still full, drop the new event for this subscriber.
			dropped = true
		}
	}

	return dropped
}

// Subscribe returns a channel that receives future events for the given thread,
// plus a cancel function to unsubscribe.
func (b *EventBus) Subscribe(threadID string) (<-chan Event, func()) {
	if b == nil {
		ch := make(chan Event)
		close(ch)
		return ch, func() {}
	}

	id := b.nextID.Add(1)
	ch := make(chan Event, b.bufferSize)

	b.mu.Lock()
	if b.subscribers[threadID] == nil {
		b.subscribers[threadID] = make(map[int64]chan Event)
	}
	b.subscribers[threadID][id] = ch
	b.mu.Unlock()

	cancel := func() {
		b.mu.Lock()
		subs := b.subscribers[threadID]
		if subs != nil {
			if existing, ok := subs[id]; ok {
				delete(subs, id)
				close(existing)
			}
			if len(subs) == 0 {
				delete(b.subscribers, threadID)
			}
		}
		b.mu.Unlock()
	}

	return ch, cancel
}

