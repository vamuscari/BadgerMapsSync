package server

import (
	"badgermaps/app/action"
	"badgermaps/app/state"
	"testing"
	"time"
)

type mockActionExecutor struct {
	executed chan bool
}

func (m *mockActionExecutor) ExecuteAction(actionConfig action.ActionConfig) error {
	m.executed <- true
	return nil
}

func TestServerManager_Start(t *testing.T) {
	cronJobs := []CronJob{
		{
			Name:     "test_job",
			Schedule: "@every 1s",
			Action:   action.ActionConfig{Type: "test"},
		},
	}
	sm := NewServerManager(state.NewState())
	executor := &mockActionExecutor{executed: make(chan bool, 1)}

	if err := sm.Start(cronJobs, executor); err != nil {
		t.Fatalf("unexpected error starting server manager: %v", err)
	}

	select {
	case <-executor.executed:
		// success
	case <-time.After(2 * time.Second):
		t.Error("timed out waiting for cron job to execute")
	}

	sm.StopServer()
}
