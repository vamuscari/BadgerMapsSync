package gui

import (
	"badgermaps/events"
	"sync/atomic"
	"testing"
	"time"
)

func TestDispatchServerStatusTransitionEmitsOnlyOnChange(t *testing.T) {
	dispatcher := events.NewEventDispatcher()
	var eventCount atomic.Int32

	dispatcher.Subscribe("server.status.changed", func(e events.Event) {
		eventCount.Add(1)
	})

	previous := serverStatusSnapshot{PID: 1234, Running: true}
	same := serverStatusSnapshot{PID: 1234, Running: true}
	changed := serverStatusSnapshot{PID: 0, Running: false}

	dispatched := dispatchServerStatusTransition(dispatcher, previous, same, "test")
	if dispatched {
		t.Fatalf("expected no dispatch when server status did not change")
	}
	if !dispatcher.WaitForDrain(100 * time.Millisecond) {
		t.Fatalf("event dispatcher did not drain after no-op transition")
	}
	if got := eventCount.Load(); got != 0 {
		t.Fatalf("expected 0 events after no-op transition, got %d", got)
	}

	dispatched = dispatchServerStatusTransition(dispatcher, previous, changed, "test")
	if !dispatched {
		t.Fatalf("expected dispatch when server status changed")
	}
	if !dispatcher.WaitForDrain(1 * time.Second) {
		t.Fatalf("event dispatcher did not drain after transition")
	}
	if got := eventCount.Load(); got != 1 {
		t.Fatalf("expected 1 event after transition, got %d", got)
	}
}

func TestShouldDispatchServerStatusChanged(t *testing.T) {
	cases := []struct {
		name     string
		previous serverStatusSnapshot
		current  serverStatusSnapshot
		want     bool
	}{
		{
			name:     "unchanged_running_status",
			previous: serverStatusSnapshot{PID: 11, Running: true},
			current:  serverStatusSnapshot{PID: 11, Running: true},
			want:     false,
		},
		{
			name:     "running_flag_changed",
			previous: serverStatusSnapshot{PID: 11, Running: true},
			current:  serverStatusSnapshot{PID: 11, Running: false},
			want:     true,
		},
		{
			name:     "pid_changed",
			previous: serverStatusSnapshot{PID: 11, Running: true},
			current:  serverStatusSnapshot{PID: 12, Running: true},
			want:     true,
		},
		{
			name:     "service_state_changed",
			previous: serverStatusSnapshot{RuntimeMode: "service", Installed: false, State: "stopped"},
			current:  serverStatusSnapshot{RuntimeMode: "service", Installed: true, State: "stopped"},
			want:     true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := shouldDispatchServerStatusChanged(tc.previous, tc.current)
			if got != tc.want {
				t.Fatalf("expected %t, got %t", tc.want, got)
			}
		})
	}
}
