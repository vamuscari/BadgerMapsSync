package app

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// EventDispatcher manages listeners and dispatches events.
type EventDispatcher struct {
	app       *App
	listeners map[EventType][]EventListener
	mu        sync.Mutex
}

// NewEventDispatcher creates a new EventDispatcher.
func NewEventDispatcher(a *App) *EventDispatcher {
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

	if d.app.State.Debug {
		fmt.Printf("Dispatching event: %s (Source: %s)\n", e.Type.String(), e.Source)
	}

	key := fmt.Sprintf("on_%s", e.Type.String())
	actions := d.app.Config.Events[key]
	for _, action := range actions {
		parts := strings.SplitN(action, ":", 2)
		if len(parts) != 2 {
			continue
		}
		actionType := parts[0]
		actionValue := parts[1]

		switch actionType {
		case "db":
			go func(functionName string) {
					if err := d.app.DB.RunFunction(functionName); err != nil {
						// Handle error appropriately, e.g., log it
						fmt.Printf("Error executing db function '%s': %v\n", functionName, err)
					}
			}(actionValue)
		case "api":
			go func(endpoint string) {
					if _, err := d.app.API.GetRaw(endpoint); err != nil {
						// Handle error appropriately, e.g., log it
						fmt.Printf("Error executing api action '%s': %v\n", endpoint, err)
					}
			}(actionValue)
		case "exec":
			go func(command string) {
					cmd := exec.Command("sh", "-c", command)
					cmd.Run()
				}(actionValue)
		}
	}

	if listeners, found := d.listeners[e.Type]; found {
		for _, listener := range listeners {
			// Listeners are called synchronously.
			// If a listener needs to perform a long-running task, it should do so in its own goroutine.
			listener(e)
		}
	}
}
