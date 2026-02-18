package events

import (
	"strings"
	"sync"
)

// EventDispatcher manages listeners and dispatches events.
type EventDispatcher struct {
	listeners map[EventType][]EventListener
	mu        sync.RWMutex
}

// NewEventDispatcher creates a new EventDispatcher.
func NewEventDispatcher() *EventDispatcher {
	return &EventDispatcher{
		listeners: make(map[EventType][]EventListener),
	}
}

// Subscribe adds a listener for a given event type pattern.
// Patterns can include wildcards, e.g., "pull.*" or "*.accounts".
func (d *EventDispatcher) Subscribe(eventType EventType, listener EventListener) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.listeners[eventType] = append(d.listeners[eventType], listener)
}

// Dispatch sends an event to all listeners whose subscribed pattern matches the event type.
func (d *EventDispatcher) Dispatch(e Event) {
	d.mu.RLock()
	var listenersToCall []EventListener

	for pattern, listeners := range d.listeners {
		if match(pattern, e.Type) {
			listenersToCall = append(listenersToCall, listeners...)
		}
	}
	d.mu.RUnlock()

	// Notify listeners synchronously to preserve event order.
	for _, listener := range listenersToCall {
		listener(e)
	}
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
