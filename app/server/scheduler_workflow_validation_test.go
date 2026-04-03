package server

import (
	"badgermaps/app/action"
	"testing"
)

func TestValidateScheduledJobDefinitionRejectsLegacySyncType(t *testing.T) {
	job := &ScheduledJob{
		Name:           "legacy",
		Schedule:       "0 0 * * * *",
		LegacySyncType: SyncTypePull,
	}

	if err := ValidateScheduledJobDefinition(job, DefaultWorkflowProfiles()); err == nil {
		t.Fatalf("expected legacy sync_type validation failure")
	}
}

func TestValidateScheduledJobDefinitionRejectsLegacyActions(t *testing.T) {
	job := &ScheduledJob{
		Name:     "legacy",
		Schedule: "0 0 * * * *",
		LegacyActions: []action.ActionConfig{
			{Type: "exec", Args: map[string]interface{}{"command": "echo hi"}},
		},
	}

	if err := ValidateScheduledJobDefinition(job, DefaultWorkflowProfiles()); err == nil {
		t.Fatalf("expected legacy actions validation failure")
	}
}

func TestValidateScheduledJobDefinitionRequiresSteps(t *testing.T) {
	job := &ScheduledJob{
		Name:     "job",
		Schedule: "0 0 * * * *",
	}

	if err := ValidateScheduledJobDefinition(job, DefaultWorkflowProfiles()); err == nil {
		t.Fatalf("expected missing steps validation failure")
	}
}

func TestValidateWorkflowStepRejectsWorkflowPseudoMode(t *testing.T) {
	step := WorkflowStep{
		ID:       "workflow_mode",
		Type:     WorkflowStepTypeSync,
		SyncMode: SyncModeWorkflow,
	}

	if err := ValidateWorkflowStep(step); err == nil {
		t.Fatalf("expected workflow pseudo-mode validation failure")
	}
}

func TestValidateWorkflowStepRejectsGroupSyncModes(t *testing.T) {
	step := WorkflowStep{
		ID:       "legacy_pull",
		Type:     WorkflowStepTypeSync,
		SyncMode: SyncModePull,
	}

	if err := ValidateWorkflowStep(step); err == nil {
		t.Fatalf("expected group sync_type to be rejected")
	}
}
