package server

import (
	"badgermaps/app/action"
	"badgermaps/app/state"
	"badgermaps/events"
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSyncJobCoordinatorSerializesJobs(t *testing.T) {
	s := state.NewState()
	*s.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	queue := NewSyncJobCoordinator(s, nil)
	defer queue.Stop()

	var (
		mu    sync.Mutex
		order []string
	)

	job1, err := queue.Submit(SyncJobRequest{
		Name:   "first",
		Source: "test",
		Mode:   SyncModePull,
		Run: func(_ context.Context) error {
			time.Sleep(150 * time.Millisecond)
			mu.Lock()
			order = append(order, "first")
			mu.Unlock()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to queue first job: %v", err)
	}

	job2, err := queue.Submit(SyncJobRequest{
		Name:   "second",
		Source: "test",
		Mode:   SyncModePush,
		Run: func(_ context.Context) error {
			mu.Lock()
			order = append(order, "second")
			mu.Unlock()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to queue second job: %v", err)
	}

	waitForTerminalJob(t, queue, job1.ID)
	waitForTerminalJob(t, queue, job2.ID)

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 {
		t.Fatalf("expected 2 job executions, got %d", len(order))
	}
	if order[0] != "first" || order[1] != "second" {
		t.Fatalf("expected serialized execution order [first second], got %v", order)
	}
}

func TestSyncJobCoordinatorTracksActiveJobAction(t *testing.T) {
	s := state.NewState()
	*s.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	queue := NewSyncJobCoordinator(s, nil)
	defer queue.Stop()

	runGate := make(chan struct{})
	job, err := queue.Submit(SyncJobRequest{
		Name:   "tracked",
		Source: "test",
		Mode:   SyncModePull,
		Run: func(_ context.Context) error {
			<-runGate
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to queue job: %v", err)
	}

	waitForRunningJob(t, queue, job.ID)

	queue.SetActiveJobAction("Pulling accounts")

	activeJob, exists := queue.GetJob(job.ID)
	if !exists {
		t.Fatalf("expected job %s to exist", job.ID)
	}
	if activeJob.CurrentAction != "Pulling accounts" {
		t.Fatalf("expected current action to be tracked, got %q", activeJob.CurrentAction)
	}

	activity := queue.GetActivity()
	if activity.ActiveJobAction != "Pulling accounts" {
		t.Fatalf("expected runtime activity action to match, got %q", activity.ActiveJobAction)
	}

	close(runGate)
	waitForTerminalJob(t, queue, job.ID)

	completedJob, exists := queue.GetJob(job.ID)
	if !exists {
		t.Fatalf("expected completed job %s to exist", job.ID)
	}
	if completedJob.CurrentAction != "" {
		t.Fatalf("expected completed job action to be cleared, got %q", completedJob.CurrentAction)
	}

	activity = queue.GetActivity()
	if activity.ActiveJobAction != "" {
		t.Fatalf("expected runtime activity action to be cleared, got %q", activity.ActiveJobAction)
	}
}

func TestRunChildJobErrorEventUsesFinalizedStatus(t *testing.T) {
	s := state.NewState()
	*s.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	dispatcher := events.NewEventDispatcher()
	queue := NewSyncJobCoordinator(s, dispatcher)
	defer queue.Stop()

	childErrorEvents := make(chan events.Event, 1)
	dispatcher.Subscribe("sync.job.error", func(e events.Event) {
		payload, ok := e.Payload.(events.GenericPayload)
		if !ok {
			return
		}
		kind, _ := payload.Data["job_kind"].(string)
		if kind == string(SyncJobKindSync) {
			select {
			case childErrorEvents <- e:
			default:
			}
		}
	})

	job, err := queue.Submit(SyncJobRequest{
		Name:   "parent",
		Source: "test",
		Mode:   SyncModeWorkflow,
		Kind:   SyncJobKindWorkflow,
		Run: func(_ context.Context) error {
			_, runErr := ExecuteWorkflowSteps(WorkflowExecutionOptions{
				Queue:      queue,
				Source:     "test",
				ParentMode: SyncModeWorkflow,
				Steps: []WorkflowStep{
					{ID: "failing_sync", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts},
				},
				RunSync: func(mode SyncMode, resourceID int) error {
					return fmt.Errorf("expected child failure")
				},
				RunAction: func(cfg action.ActionConfig, step WorkflowStep) error {
					return nil
				},
			})
			return runErr
		},
	})
	if err != nil {
		t.Fatalf("failed to queue parent job: %v", err)
	}

	waitForTerminalJob(t, queue, job.ID)
	if drained := dispatcher.WaitForDrain(2 * time.Second); !drained {
		t.Fatalf("timed out waiting for event dispatcher to drain")
	}

	var childErrEvent events.Event
	select {
	case childErrEvent = <-childErrorEvents:
	default:
		t.Fatalf("expected child sync.job.error event")
	}

	payload, ok := childErrEvent.Payload.(events.GenericPayload)
	if !ok {
		t.Fatalf("expected generic payload for child error event")
	}

	status, _ := payload.Data["status"].(string)
	if status != string(SyncJobFailed) {
		t.Fatalf("expected child error status %q, got %q", SyncJobFailed, status)
	}
}

func TestRunChildJobActionEventIncludesActionMetadata(t *testing.T) {
	s := state.NewState()
	*s.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	dispatcher := events.NewEventDispatcher()
	queue := NewSyncJobCoordinator(s, dispatcher)
	defer queue.Stop()

	actionStartEvents := make(chan events.GenericPayload, 1)
	dispatcher.Subscribe("sync.job.start", func(e events.Event) {
		payload, ok := e.Payload.(events.GenericPayload)
		if !ok {
			return
		}
		if kind, _ := payload.Data["job_kind"].(string); kind != string(SyncJobKindAction) {
			return
		}
		select {
		case actionStartEvents <- payload:
		default:
		}
	})

	parentJob, err := queue.Submit(SyncJobRequest{
		Name:   "parent",
		Source: "test",
		Mode:   SyncModeWorkflow,
		Kind:   SyncJobKindWorkflow,
		Run: func(_ context.Context) error {
			_, runErr := queue.RunChildJob(SyncChildJobRequest{
				Name:        "action_step",
				Source:      "test",
				Mode:        SyncModeWorkflow,
				Kind:        SyncJobKindAction,
				StepID:      "action_step",
				StepIndex:   1,
				TotalSteps:  1,
				ActionType:  "exec",
				CommandText: "echo hello world",
				Run: func(_ context.Context) error {
					return nil
				},
			})
			return runErr
		},
	})
	if err != nil {
		t.Fatalf("failed to queue parent job: %v", err)
	}

	waitForTerminalJob(t, queue, parentJob.ID)
	if drained := dispatcher.WaitForDrain(2 * time.Second); !drained {
		t.Fatalf("timed out waiting for event dispatcher to drain")
	}

	var actionPayload events.GenericPayload
	select {
	case actionPayload = <-actionStartEvents:
	default:
		t.Fatalf("expected action sync.job.start payload")
	}

	if got, _ := actionPayload.Data["action_type"].(string); got != "exec" {
		t.Fatalf("expected action_type %q, got %q", "exec", got)
	}
	if got, _ := actionPayload.Data["command_text"].(string); got != "echo hello world" {
		t.Fatalf("expected command_text %q, got %q", "echo hello world", got)
	}
}

func waitForRunningJob(t *testing.T, queue *SyncJobCoordinator, jobID string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, exists := queue.GetJob(jobID)
		if exists && job.Status == SyncJobRunning {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to start running", jobID)
}

func waitForTerminalJob(t *testing.T, queue *SyncJobCoordinator, jobID string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, exists := queue.GetJob(jobID)
		if exists && job.Status.IsTerminal() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to complete", jobID)
}
