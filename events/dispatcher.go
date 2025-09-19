package events

import (
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
	// We don't dispatch a debug event here to avoid an infinite loop of dispatching debug events.
	if d.app.GetState() != nil && d.app.GetState().Debug && e.Type != Debug {
		// Dispatching this in a goroutine to avoid blocking the main event dispatch
		go d.Dispatch(Debugf("dispatcher", "Dispatching event: %s (Source: %s)", e.Type.String(), e.Source))
	}

	// Execute configured event actions
	if d.app.GetConfig() != nil {
		for _, eventAction := range d.app.GetConfig().Events {
			if eventAction.Event == e.Type.String() && (eventAction.Source == "" || eventAction.Source == e.Source) {
				for _, actionConfig := range eventAction.Run {
					go d.executeAction(e, eventAction, actionConfig)
				}
			}
		}
	}

	// Copy listeners to call them outside the lock
	d.mu.Lock()
	listenersToCall := make([]EventListener, 0, len(d.listeners[e.Type]))
	if listeners, found := d.listeners[e.Type]; found {
		listenersToCall = append(listenersToCall, listeners...)
	}
	d.mu.Unlock()

	// Notify listeners asynchronously
	for _, listener := range listenersToCall {
		go listener(e)
	}
}

func (d *EventDispatcher) executeAction(e Event, eventAction EventAction, actionConfig ActionConfig) {
	action, err := NewActionFromConfig(actionConfig)
	if err != nil {
		d.Dispatch(ActionErrorf(e.Source, "error creating action: %w", err))
		return
	}

	if err := action.Validate(); err != nil {
		d.Dispatch(ActionErrorf(e.Source, "invalid action configuration: %w", err))
		return
	}

	d.Dispatch(ActionStartf(e.Source, "Executing action type '%s' from '%s'", actionConfig.Type, eventAction.Name))

	if err := action.Execute(d.app); err != nil {
		d.Dispatch(ActionErrorf(e.Source, "action '%s' from '%s' failed: %w", actionConfig.Type, eventAction.Name, err))
	} else {
		d.Dispatch(ActionSuccessf(e.Source, "Action '%s' from '%s' completed successfully", actionConfig.Type, eventAction.Name))
	}
}


