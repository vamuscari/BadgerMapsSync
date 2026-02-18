package app

import (
	"errors"
	"testing"
	"time"

	"badgermaps/events"
)

func TestShouldSuppressSyncHistoryFinalizeError(t *testing.T) {
	a := NewApp()

	closedErr := errors.New("sql: database is closed")
	if a.shouldSuppressSyncHistoryFinalizeError(closedErr) {
		t.Fatal("expected no suppression before shutdown")
	}

	a.shuttingDown.Store(true)

	if !a.shouldSuppressSyncHistoryFinalizeError(closedErr) {
		t.Fatal("expected shutdown database-closed errors to be suppressed")
	}
	if !a.shouldSuppressSyncHistoryFinalizeError(errors.New("database connection is not initialized")) {
		t.Fatal("expected shutdown database-not-initialized errors to be suppressed")
	}
	if a.shouldSuppressSyncHistoryFinalizeError(errors.New("write failed")) {
		t.Fatal("expected unrelated errors to remain visible")
	}
}

func TestAppCloseWaitsForEventDrain(t *testing.T) {
	a := NewApp()
	block := make(chan struct{})

	a.Events.Subscribe("pull.start", func(e events.Event) {
		<-block
	})
	a.Events.Dispatch(events.Event{Type: "pull.start"})

	closed := make(chan struct{})
	go func() {
		a.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("app close should wait for queued event handlers")
	case <-time.After(50 * time.Millisecond):
		// expected while listener is blocked
	}

	close(block)

	select {
	case <-closed:
		// expected
	case <-time.After(1 * time.Second):
		t.Fatal("app close timed out waiting for event drain")
	}
}
