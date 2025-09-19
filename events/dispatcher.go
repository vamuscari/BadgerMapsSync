package events

import (
	"sync"
)

// EventDispatcher manages listeners and dispatches events.
type EventDispatcher struct {
	listeners map[EventType][]EventListener
	mu        sync.Mutex
}

// NewEventDispatcher creates a new EventDispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		listeners: make(map[EventType][]EventListener),
	}
}

// Subscribe adds a listener for a given event type.
func (d *EventDispatcher) Subscribe(eventType EventType, listener EventListener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if eventType.String() == "*" {
		d.listeners[Wildcard] = append(d.listeners[Wildcard], listener)
		return
	}
	d.listeners[eventType] = append(d.listeners[eventType], listener)
}

// Dispatch sends an event to all registered listeners of its type.
func (d *EventDispatcher) Dispatch(e Event) {
	// Copy listeners to call them outside the lock
	d.mu.Lock()
	listenersToCall := make([]EventListener, 0, len(d.listeners[e.Type]))
	if listeners, found := d.listeners[e.Type]; found {
		listenersToCall = append(listenersToCall, listeners...)
	}
	// Add wildcard listeners
	if listeners, found := d.listeners[Wildcard]; found {
		listenersToCall = append(listenersToCall, listeners...)
	}
	d.mu.Unlock()

	// Notify listeners asynchronously
	for _, listener := range listenersToCall {
		go listener(e)
	}
}
