package server

import (
	"badgermaps/app/action"
	"testing"
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
