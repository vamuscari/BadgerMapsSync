package events

import (
	"sync"
	"testing"
	"time"
)

func TestEventDispatcher_SubscribeAndDispatch(t *testing.T) {
	dispatcher := NewEventDispatcher()
	var wg sync.WaitGroup
	wg.Add(1)

	listener := func(e Event) {
		if e.Type != "test.event" {
			t.Errorf("Expected event type 'test.event', got '%s'", e.Type)
		}
		if e.Source != "test_source" {
			t.Errorf("Expected event source 'test_source', got '%s'", e.Source)
		}
		wg.Done()
	}

	dispatcher.Subscribe("test.event", listener)
	dispatcher.Dispatch(Event{
		Type:   "test.event",
		Source: "test_source",
	})

	wg.Wait()
}

func TestEventDispatcher_WildcardSubscription(t *testing.T) {
	dispatcher := NewEventDispatcher()
	var wg sync.WaitGroup
	wg.Add(6)

	listener := func(e Event) {
		wg.Done()
	}

	dispatcher.Subscribe("pull.*", listener)
	dispatcher.Subscribe("*.accounts", listener)
	dispatcher.Subscribe("*", listener)

	dispatcher.Dispatch(Event{Type: "pull.start"})
	dispatcher.Dispatch(Event{Type: "pull.complete.accounts"})
	dispatcher.Dispatch(Event{Type: "push.start"})

	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		// All good
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out, not all listeners were called")
	}
}

func TestMatchFunction(t *testing.T) {
	testCases := []struct {
		pattern   EventType
		eventType EventType
		expected  bool
	}{
		{"pull.start", "pull.start", true},
		{"pull.start", "pull.end", false},
		{"pull.*", "pull.start", true},
		{"pull.*", "pull.end", true},
		{"pull.*", "push.start", false},
		{"*.accounts", "pull.start.accounts", true},
		{"*.accounts", "push.end.accounts", true},
		{"*.accounts", "pull.start.checkins", false},
		{"*", "any.event.whatsoever", true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.pattern), func(t *testing.T) {
			if got := match(tc.pattern, tc.eventType); got != tc.expected {
				t.Errorf("match(%q, %q) = %v; want %v", tc.pattern, tc.eventType, got, tc.expected)
			}
		})
	}
}