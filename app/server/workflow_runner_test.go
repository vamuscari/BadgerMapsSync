package server

import (
	"badgermaps/app/action"
	"badgermaps/app/state"
	"badgermaps/events"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecuteWorkflowStepsSkipsDisabledActionSteps(t *testing.T) {
	disabled := false
	runActionCalls := 0

	actionFailures, err := ExecuteWorkflowSteps(WorkflowExecutionOptions{
		Steps: []WorkflowStep{
			{
				ID:   "disabled_exec",
				Type: WorkflowStepTypeAction,
				Action: action.ActionConfig{
					Type:    "exec",
					Args:    map[string]interface{}{"command": "echo disabled"},
					Enabled: &disabled,
				},
			},
		},
		RunSync: func(mode SyncMode, resourceID int) error {
			return nil
		},
		RunAction: func(cfg action.ActionConfig, step WorkflowStep) error {
			runActionCalls++
			return nil
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if actionFailures != 0 {
		t.Fatalf("expected 0 action failures, got %d", actionFailures)
	}
	if runActionCalls != 0 {
		t.Fatalf("expected disabled action step to be skipped, got %d run calls", runActionCalls)
	}
}

func TestExecuteWorkflowStepsPropagatesActionMetadataToChildJobs(t *testing.T) {
	s := state.NewState()
	*s.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	dispatcher := events.NewEventDispatcher()
	queue := NewSyncJobCoordinator(s, dispatcher)
	defer queue.Stop()

	parentJob, err := queue.Submit(SyncJobRequest{
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
					{
						ID:   "action_exec",
						Type: WorkflowStepTypeAction,
						Action: action.ActionConfig{
							Type: "exec",
							Args: map[string]interface{}{
								"command": "echo hello",
								"args":    []interface{}{"-n", "world"},
							},
						},
					},
				},
				RunSync: func(mode SyncMode, resourceID int) error {
					return nil
				},
				RunAction: func(cfg action.ActionConfig, step WorkflowStep) error {
					return nil
				},
			})
			return runErr
		},
	})
	if err != nil {
		t.Fatalf("failed to queue parent workflow job: %v", err)
	}

	waitForTerminalJob(t, queue, parentJob.ID)
	if drained := dispatcher.WaitForDrain(2 * time.Second); !drained {
		t.Fatalf("timed out waiting for event dispatcher to drain")
	}

	var actionChild *SyncJob
	for _, job := range queue.ListJobs() {
		if job == nil {
			continue
		}
		if job.ParentJobID == parentJob.ID && job.Kind == SyncJobKindAction {
			actionChild = job
			break
		}
	}
	if actionChild == nil {
		t.Fatalf("expected action child job for parent %s", parentJob.ID)
	}
	if actionChild.ActionType != "exec" {
		t.Fatalf("expected action child action type %q, got %q", "exec", actionChild.ActionType)
	}
	if !strings.Contains(actionChild.CommandText, "echo hello") {
		t.Fatalf("expected command text to include %q, got %q", "echo hello", actionChild.CommandText)
	}
	if !strings.Contains(actionChild.CommandText, "-n") {
		t.Fatalf("expected command text to include %q, got %q", "-n", actionChild.CommandText)
	}
	if !strings.Contains(actionChild.CommandText, "world") {
		t.Fatalf("expected command text to include %q, got %q", "world", actionChild.CommandText)
	}
}
