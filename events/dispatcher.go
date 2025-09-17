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

	if d.app.GetState().Debug {
		fmt.Printf("Dispatching event: %s (Source: %s)\n", e.Type.String(), e.Source)
	}

	// runActions is a helper closure to execute actions for a given key.
	runActions := func(key string) {
		actionConfigs, ok := d.app.GetConfig().Events[key]
		if !ok {
			return
		}

		for _, actionConfig := range actionConfigs {
			action, err := NewActionFromConfig(actionConfig)
			if err != nil {
				d.Dispatch(Event{Type: ActionError, Source: e.Source, Payload: fmt.Errorf("error creating action: %w", err)})
				continue
			}

			if err := action.Validate(); err != nil {
				d.Dispatch(Event{Type: ActionError, Source: e.Source, Payload: fmt.Errorf("invalid action configuration: %w", err)})
				continue
			}

			d.Dispatch(Event{Type: ActionStart, Source: e.Source, Payload: fmt.Sprintf("Executing action type '%s'", actionConfig.Type)})

			go func(action Action, actionConfig ActionConfig) {
				if err := action.Execute(d.app); err != nil {
					d.Dispatch(Event{Type: ActionError, Source: e.Source, Payload: err})
				} else {
					d.Dispatch(Event{Type: ActionSuccess, Source: e.Source, Payload: fmt.Sprintf("Action '%v' completed successfully", actionConfig.Type)})
				}
			}(action, actionConfig)
		}
	}

	// Run actions for the general event type (e.g., "on_PullComplete")
	generalKey := fmt.Sprintf("on_%s", e.Type.String())
	runActions(generalKey)

	// Run actions for the source-specific event type (e.g., "on_PullComplete_accounts")
	if e.Source != "" {
		sourceKey := fmt.Sprintf("on_%s_%s", e.Type.String(), e.Source)
		runActions(sourceKey)
	}

	if listeners, found := d.listeners[e.Type]; found {
		for _, listener := range listeners {
			// Listeners are called synchronously.
			// If a listener needs to perform a long-running task, it should do so in its own goroutine.
			listener(e)
		}
	}
}

