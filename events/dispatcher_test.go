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

func TestEventDispatcher_NonBlockingDispatchMaintainsListenerOrder(t *testing.T) {
	dispatcher := NewEventDispatcher()
	blocker := make(chan struct{})
	received := make(chan EventType, 2)

	dispatcher.Subscribe("pull.*", func(e Event) {
		if e.Type == "pull.start" {
			<-blocker
		}
		received <- e.Type
	})

	start := time.Now()
	dispatcher.Dispatch(Event{Type: "pull.start"})
	dispatcher.Dispatch(Event{Type: "pull.complete"})

	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("dispatch should be non-blocking, took %v", elapsed)
	}

	close(blocker)

	var got []EventType
	timeout := time.After(1 * time.Second)
	for len(got) < 2 {
		select {
		case e := <-received:
			got = append(got, e)
		case <-timeout:
			t.Fatalf("timed out waiting for listener events, got %v", got)
		}
	}

	if got[0] != "pull.start" || got[1] != "pull.complete" {
		t.Fatalf("expected in-order delivery [pull.start pull.complete], got %v", got)
	}
}

func TestEventDispatcher_WaitForDrain(t *testing.T) {
	dispatcher := NewEventDispatcher()
	release := make(chan struct{})

	dispatcher.Subscribe("pull.start", func(e Event) {
		<-release
	})

	dispatcher.Dispatch(Event{Type: "pull.start"})

	done := make(chan bool, 1)
	go func() {
		done <- dispatcher.WaitForDrain(1 * time.Second)
	}()

	select {
	case <-done:
		t.Fatal("wait should not complete before listener drains")
	case <-time.After(50 * time.Millisecond):
		// expected
	}

	close(release)

	select {
	case ok := <-done:
		if !ok {
			t.Fatal("expected wait to succeed after listener drained")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for dispatcher to drain")
	}
}

func TestEventDispatcher_WaitForDrainTimeout(t *testing.T) {
	dispatcher := NewEventDispatcher()
	block := make(chan struct{})

	dispatcher.Subscribe("pull.start", func(e Event) {
		<-block
	})

	dispatcher.Dispatch(Event{Type: "pull.start"})

	if ok := dispatcher.WaitForDrain(20 * time.Millisecond); ok {
		t.Fatal("expected wait to time out while listener is blocked")
	}

	close(block)

	if ok := dispatcher.WaitForDrain(1 * time.Second); !ok {
		t.Fatal("expected wait to succeed once listener unblocks")
	}
}
