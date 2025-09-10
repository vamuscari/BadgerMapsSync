package app

import "sync"

// EventType defines the type of an event using a custom type for type safety.
type EventType int

const (
	// Pull Events
	PullStart EventType = iota
	ResourceIDsFetched
	FetchDetailStart
	FetchDetailSuccess
	StoreSuccess
	PullError
	PullComplete

	// Push Events
	PushScanStart
	PushScanComplete
	PushItemStart
	PushItemSuccess
	PushItemError
	PushError
	PushComplete
)

// Event represents a dispatched event with a type and associated data.
type Event struct {
	Type    EventType
	Source  string      // e.g., "accounts", "checkins"
	Payload interface{} // Can be an error, a data object, an ID, etc.
}

// EventListener is a function that can handle an event.
type EventListener func(e Event)

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
	d.listeners[eventType] = append(d.listeners[eventType], listener)
}

// Dispatch sends an event to all registered listeners of its type.
func (d *EventDispatcher) Dispatch(e Event) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if listeners, found := d.listeners[e.Type]; found {
		for _, listener := range listeners {
			// Run listeners in a separate goroutine so they don't block the main process.
			go listener(e)
		}
	}
}
