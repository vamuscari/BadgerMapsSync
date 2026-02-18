package events

import (
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// EventDispatcher manages listeners and dispatches events.
type EventDispatcher struct {
	listeners map[EventType][]*queuedListener
	mu        sync.RWMutex
	pending   atomic.Int64
}

type queuedListener struct {
	fn EventListener
	d  *EventDispatcher

	mu         sync.Mutex
	queue      []Event
	processing bool
}

// NewEventDispatcher creates a new EventDispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		listeners: make(map[EventType][]*queuedListener),
	}
}

// Subscribe adds a listener for a given event type pattern.
// Patterns can include wildcards, e.g., "pull.*" or "*.accounts".
func (d *EventDispatcher) Subscribe(eventType EventType, listener EventListener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners[eventType] = append(d.listeners[eventType], &queuedListener{
		fn: listener,
		d:  d,
	})
}

// Dispatch sends an event to all listeners whose subscribed pattern matches the event type.
func (d *EventDispatcher) Dispatch(e Event) {
	d.mu.RLock()
	var listenersToCall []*queuedListener

	for pattern, listeners := range d.listeners {
		if match(pattern, e.Type) {
			listenersToCall = append(listenersToCall, listeners...)
		}
	}
	d.mu.RUnlock()

	// Enqueue event delivery per listener so each listener processes events in-order
	// without blocking the caller of Dispatch.
	for _, listener := range listenersToCall {
		listener.enqueue(e)
	}
}

func (l *queuedListener) enqueue(e Event) {
	l.d.pending.Add(1)

	l.mu.Lock()
	l.queue = append(l.queue, e)
	if l.processing {
		l.mu.Unlock()
		return
	}
	l.processing = true
	l.mu.Unlock()

	go l.drain()
}

func (l *queuedListener) drain() {
	for {
		l.mu.Lock()
		if len(l.queue) == 0 {
			l.processing = false
			l.mu.Unlock()
			return
		}

		e := l.queue[0]
		l.queue = l.queue[1:]
		l.mu.Unlock()

		func() {
			defer l.d.pending.Add(-1)
			l.fn(e)
		}()
	}
}

// PendingEvents returns the number of queued or currently running listener calls.
func (d *EventDispatcher) PendingEvents() int64 {
	return d.pending.Load()
}

// WaitForDrain waits for all pending listener work to finish until timeout is reached.
// It returns true when drained, false when the timeout elapsed first.
func (d *EventDispatcher) WaitForDrain(timeout time.Duration) bool {
	if d.PendingEvents() == 0 {
		return true
	}
	if timeout <= 0 {
		return false
	}

	deadline := time.Now().Add(timeout)
	for d.PendingEvents() > 0 {
		if time.Now().After(deadline) {
			return false
		}
		sleep := 10 * time.Millisecond
		if remaining := time.Until(deadline); remaining < sleep {
			sleep = remaining
		}
		if sleep > 0 {
			time.Sleep(sleep)
		}
	}

	return true
}

// match checks if an event type matches a subscription pattern.
// It supports single wildcards at the beginning or end of a pattern segment.
// e.g., "pull.*" matches "pull.start" and "pull.complete"
// e.g., "*.accounts" matches "pull.start.accounts"
func match(pattern, eventType EventType) bool {
	if pattern == "*" {
		return true
	}
	if strings.Contains(string(pattern), "*") {
		patternParts := strings.Split(string(pattern), ".")
		eventParts := strings.Split(string(eventType), ".")

		if len(patternParts) > len(eventParts) {
			return false
		}

		for i, part := range patternParts {
			if part == "*" {
				continue
			}
			if i >= len(eventParts) || part != eventParts[i] {
				// Handle suffix wildcard case like "*.accounts"
				if patternParts[0] == "*" && len(patternParts) == 2 && len(eventParts) > 1 {
					if eventParts[len(eventParts)-1] == patternParts[1] {
						return true
					}
				}
				return false
			}
		}
		// If we've reached here, it's a match (e.g., "pull.*" matches "pull.start")
		return true
	}

	return pattern == eventType
}
