package events

import (
	"fmt"
	"sync"
)

// EventDispatcher manages listeners and dispatches events.
type EventDispatcher struct {
	app       AppInterface
	listeners map[EventType][]EventListener
	mu        sync.Mutex
}

// NewEventDispatcher creates a new EventDispatcher.
func NewEventDispatcher(a AppInterface) *EventDispatcher {
	return &EventDispatcher{
		app:       a,
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

	// We don't dispatch a debug event here to avoid an infinite loop of dispatching debug events.
	if d.app.GetState().Debug && e.Type != Debug {
		go d.Dispatch(Debugf("dispatcher", "Dispatching event: %s (Source: %s)", e.Type.String(), e.Source))
	}

	// Execute configured event actions
	for _, eventAction := range d.app.GetConfig().Events {
		// Check if the event type matches
		if eventAction.Event != e.Type.String() {
			continue
		}

		// Check if the source matches (if a source is specified in the action)
		if eventAction.Source != "" && eventAction.Source != e.Source {
			continue
		}

		// If we reach here, the event matches, so run the actions
		for _, actionConfig := range eventAction.Run {
			action, err := NewActionFromConfig(actionConfig)
			if err != nil {
				d.Dispatch(Event{Type: ActionError, Source: e.Source, Payload: fmt.Errorf("error creating action: %w", err)})
				continue
			}

			if err := action.Validate(); err != nil {
				d.Dispatch(Event{Type: ActionError, Source: e.Source, Payload: fmt.Errorf("invalid action configuration: %w", err)})
				continue
			}

			go d.Dispatch(Event{Type: ActionStart, Source: e.Source, Payload: fmt.Sprintf("Executing action type '%s' from '%s'", actionConfig.Type, eventAction.Name)})

			go func(action Action, actionConfig ActionConfig) {
				if err := action.Execute(d.app); err != nil {
					go d.Dispatch(Event{Type: ActionError, Source: e.Source, Payload: err})
				} else {
					go d.Dispatch(Event{Type: ActionSuccess, Source: e.Source, Payload: fmt.Sprintf("Action '%s' from '%s' completed successfully", actionConfig.Type, eventAction.Name)})
				}
			}(action, actionConfig)
		}
	}

	// Notify listeners
	if listeners, found := d.listeners[e.Type]; found {
		for _, listener := range listeners {
			// Listeners are called synchronously.
			// If a listener needs to perform a long-running task, it should do so in its own goroutine.
			listener(e)
		}
	}
}


