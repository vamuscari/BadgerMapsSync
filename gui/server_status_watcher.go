package gui

import (
	"badgermaps/app"
	"badgermaps/events"
	"time"
)

const serverStatusPollInterval = 2 * time.Second

type serverStatusSnapshot struct {
	PID     int
	Running bool
}

func readServerStatusSnapshot(a *app.App) serverStatusSnapshot {
	if a == nil || a.Server == nil {
		return serverStatusSnapshot{}
	}

	pid, running := a.Server.GetServerStatus()
	return serverStatusSnapshot{
		PID:     pid,
		Running: running,
	}
}

func shouldDispatchServerStatusChanged(previous, current serverStatusSnapshot) bool {
	return previous.PID != current.PID || previous.Running != current.Running
}

func dispatchServerStatusTransition(dispatcher *events.EventDispatcher, previous, current serverStatusSnapshot, source string) bool {
	if dispatcher == nil || !shouldDispatchServerStatusChanged(previous, current) {
		return false
	}

	if source == "" {
		source = "status_watcher"
	}

	dispatcher.Dispatch(events.Event{
		Type:   "server.status.changed",
		Source: source,
	})
	return true
}

func startServerStatusWatcher(a *app.App, stop <-chan struct{}) {
	if a == nil || a.Server == nil || a.Events == nil || stop == nil {
		return
	}

	previous := readServerStatusSnapshot(a)

	go func() {
		ticker := time.NewTicker(serverStatusPollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				current := readServerStatusSnapshot(a)
				dispatchServerStatusTransition(a.Events, previous, current, "status_watcher")
				previous = current
			}
		}
	}()
}
