package gui

import (
	"badgermaps/app/action"
	appserver "badgermaps/app/server"
	"reflect"
	"strings"
	"testing"
)

func TestWorkflowBuilderSyncStepRoundTrip(t *testing.T) {
	draft := workflowBuilderDraftStep{
		ID:       "pull_accounts",
		Name:     "Pull Accounts",
		Type:     appserver.WorkflowStepTypeSync,
		SyncMode: appserver.SyncModePullAccounts,
	}

	steps, err := workflowBuilderDraftsToSteps([]workflowBuilderDraftStep{draft})
	if err != nil {
		t.Fatalf("expected builder draft to compile: %v", err)
	}
	if len(steps) != 1 {
		t.Fatalf("expected one step, got %d", len(steps))
	}
	if steps[0].SyncMode != appserver.SyncModePullAccounts {
		t.Fatalf("expected sync mode %q, got %q", appserver.SyncModePullAccounts, steps[0].SyncMode)
	}

	roundTrip, err := workflowBuilderDraftsFromSteps(steps)
	if err != nil {
		t.Fatalf("expected steps to convert back to builder draft: %v", err)
	}
	if len(roundTrip) != 1 {
		t.Fatalf("expected one roundtrip draft, got %d", len(roundTrip))
	}
	if roundTrip[0].Type != appserver.WorkflowStepTypeSync || roundTrip[0].SyncMode != appserver.SyncModePullAccounts {
		t.Fatalf("unexpected roundtrip draft: %+v", roundTrip[0])
	}
}

func TestWorkflowBuilderExecActionRoundTrip(t *testing.T) {
	disabled := false
	step := appserver.WorkflowStep{
		ID:   "exec_step",
		Type: appserver.WorkflowStepTypeAction,
		Action: action.ActionConfig{
			Type: workflowActionTypeExec,
			Args: map[string]interface{}{
				"command":   "echo hello",
				"use_shell": false,
				"args":      []interface{}{"alpha", "beta"},
			},
			Enabled: &disabled,
		},
	}

	drafts, err := workflowBuilderDraftsFromSteps([]appserver.WorkflowStep{step})
	if err != nil {
		t.Fatalf("expected exec action step to be representable: %v", err)
	}
	if len(drafts) != 1 {
		t.Fatalf("expected one draft, got %d", len(drafts))
	}
	if drafts[0].ActionType != workflowActionTypeExec {
		t.Fatalf("expected exec action type, got %q", drafts[0].ActionType)
	}
	if drafts[0].ActionEnabled {
		t.Fatalf("expected disabled action")
	}
	if drafts[0].ExecCommand != "echo hello" || drafts[0].ExecUseShell {
		t.Fatalf("unexpected exec draft values: %+v", drafts[0])
	}
	if drafts[0].ExecArgsText != "alpha\nbeta" {
		t.Fatalf("unexpected exec args text %q", drafts[0].ExecArgsText)
	}

	compiled, err := workflowBuilderDraftsToSteps(drafts)
	if err != nil {
		t.Fatalf("expected drafts to compile back to step: %v", err)
	}
	if len(compiled) != 1 {
		t.Fatalf("expected one compiled step, got %d", len(compiled))
	}
	if compiled[0].Action.Type != workflowActionTypeExec {
		t.Fatalf("expected exec action, got %q", compiled[0].Action.Type)
	}
	if compiled[0].Action.IsEnabled() {
		t.Fatalf("expected action to remain disabled")
	}
	args, err := toStringSlice(compiled[0].Action.Args["args"])
	if err != nil {
		t.Fatalf("expected exec args to be string slice: %v", err)
	}
	if !reflect.DeepEqual(args, []string{"alpha", "beta"}) {
		t.Fatalf("unexpected exec args: %#v", args)
	}
}

func TestWorkflowBuilderDBActionRoundTrip(t *testing.T) {
	step := appserver.WorkflowStep{
		ID:   "db_step",
		Type: appserver.WorkflowStepTypeAction,
		Action: action.ActionConfig{
			Type: workflowActionTypeDB,
			Args: map[string]interface{}{
				"query": "select 1",
				"args":  map[string]interface{}{"tenant": "acme"},
			},
		},
	}

	drafts, err := workflowBuilderDraftsFromSteps([]appserver.WorkflowStep{step})
	if err != nil {
		t.Fatalf("expected db action step to be representable: %v", err)
	}
	if drafts[0].DBOperation != "query" || drafts[0].DBValue != "select 1" {
		t.Fatalf("unexpected db draft: %+v", drafts[0])
	}
	if !strings.Contains(drafts[0].DBArgsText, "tenant") {
		t.Fatalf("expected db args JSON text, got %q", drafts[0].DBArgsText)
	}

	compiled, err := workflowBuilderDraftsToSteps(drafts)
	if err != nil {
		t.Fatalf("expected db draft to compile back to step: %v", err)
	}
	if compiled[0].Action.Type != workflowActionTypeDB {
		t.Fatalf("expected db action type, got %q", compiled[0].Action.Type)
	}
	if compiled[0].Action.Args["query"] != "select 1" {
		t.Fatalf("unexpected db query: %#v", compiled[0].Action.Args["query"])
	}
}

func TestWorkflowBuilderDraftsToJSONReflectsBuilderChanges(t *testing.T) {
	drafts := []workflowBuilderDraftStep{
		{
			ID:       "step_1",
			Type:     appserver.WorkflowStepTypeSync,
			SyncMode: appserver.SyncModePullAccounts,
		},
	}

	jsonText, steps, err := workflowBuilderDraftsToJSON(drafts)
	if err != nil {
		t.Fatalf("expected drafts to serialize: %v", err)
	}
	if !strings.Contains(jsonText, "\"sync_type\": \"pull_accounts\"") {
		t.Fatalf("expected generated JSON to include sync_type pull_accounts, got %s", jsonText)
	}
	if len(steps) != 1 || steps[0].SyncMode != appserver.SyncModePullAccounts {
		t.Fatalf("unexpected steps from builder->json sync: %+v", steps)
	}

	drafts[0].SyncMode = appserver.SyncModePushAccounts
	jsonText, steps, err = workflowBuilderDraftsToJSON(drafts)
	if err != nil {
		t.Fatalf("expected updated drafts to serialize: %v", err)
	}
	if !strings.Contains(jsonText, "\"sync_type\": \"push_accounts\"") {
		t.Fatalf("expected generated JSON to update to push_accounts, got %s", jsonText)
	}
	if steps[0].SyncMode != appserver.SyncModePushAccounts {
		t.Fatalf("expected updated sync mode push_accounts, got %q", steps[0].SyncMode)
	}
}

func TestParseWorkflowStepsJSONForModeSwitch(t *testing.T) {
	valid := `[{"id":"pull_accounts","type":"sync","sync_type":"pull_accounts"}]`
	steps, err := parseWorkflowStepsJSON(valid)
	if err != nil {
		t.Fatalf("expected valid workflow JSON to parse: %v", err)
	}
	if _, err := workflowBuilderDraftsFromSteps(steps); err != nil {
		t.Fatalf("expected parsed workflow JSON to be builder-compatible: %v", err)
	}

	invalid := `[{"id":"bad","type":"sync"}]`
	if _, err := parseWorkflowStepsJSON(invalid); err == nil {
		t.Fatalf("expected invalid workflow JSON to fail parsing/validation")
	}
}

func TestApplyWorkflowProfileTemplateSelection(t *testing.T) {
	profiles := map[string]appserver.WorkflowProfile{
		"pull_all": {
			Name: "pull_all",
			Steps: []appserver.WorkflowStep{
				{ID: "pull_accounts", Type: appserver.WorkflowStepTypeSync, SyncMode: appserver.SyncModePullAccounts},
			},
		},
	}
	currentSteps := []appserver.WorkflowStep{{ID: "custom", Type: appserver.WorkflowStepTypeSync, SyncMode: appserver.SyncModePushAccounts}}

	stepsAfterCancel, replaced, err := applyWorkflowProfileTemplateSelection("pull_all", currentSteps, profiles, false)
	if err != nil {
		t.Fatalf("expected cancel path without error: %v", err)
	}
	if replaced {
		t.Fatalf("expected no replacement when not confirmed")
	}
	if !reflect.DeepEqual(stepsAfterCancel, currentSteps) {
		t.Fatalf("expected steps unchanged on cancel, got %#v", stepsAfterCancel)
	}

	stepsAfterConfirm, replaced, err := applyWorkflowProfileTemplateSelection("pull_all", currentSteps, profiles, true)
	if err != nil {
		t.Fatalf("expected confirm path without error: %v", err)
	}
	if !replaced {
		t.Fatalf("expected replacement when confirmed")
	}
	if !reflect.DeepEqual(stepsAfterConfirm, profiles["pull_all"].Steps) {
		t.Fatalf("expected steps from selected profile, got %#v", stepsAfterConfirm)
	}
}

func TestWorkflowBuilderSyncModeOptionsOnlyExplicitTypes(t *testing.T) {
	got := workflowBuilderSyncModeOptions()
	want := []string{
		string(appserver.SyncModePullAccounts),
		string(appserver.SyncModePullCheckins),
		string(appserver.SyncModePullRoutes),
		string(appserver.SyncModePullProfile),
		string(appserver.SyncModePushAccounts),
		string(appserver.SyncModePushCheckins),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected sync options %#v, got %#v", want, got)
	}
}

func TestWorkflowBuilderRejectsGroupSyncModes(t *testing.T) {
	_, err := workflowBuilderDraftsFromSteps([]appserver.WorkflowStep{
		{ID: "legacy_pull", Type: appserver.WorkflowStepTypeSync, SyncMode: appserver.SyncModePull},
	})
	if err == nil {
		t.Fatalf("expected builder to reject legacy group sync mode")
	}
	if !strings.Contains(err.Error(), "unsupported sync_type") {
		t.Fatalf("expected unsupported sync_type error, got %v", err)
	}
}

func TestDefaultBuilderDraftStepGeneratesUniqueID(t *testing.T) {
	drafts := []workflowBuilderDraftStep{}
	seen := map[string]struct{}{}
	for i := 0; i < 24; i++ {
		draft := defaultBuilderDraftStep(drafts)
		if !strings.HasPrefix(draft.ID, "pull_accounts_") {
			t.Fatalf("expected default sync-step id prefix pull_accounts_, got %q", draft.ID)
		}
		key := strings.ToLower(strings.TrimSpace(draft.ID))
		if _, exists := seen[key]; exists {
			t.Fatalf("expected generated id to be unique, duplicate %q", draft.ID)
		}
		seen[key] = struct{}{}
		drafts = append(drafts, draft)
	}
}

func TestNextWorkflowBuilderStepIDUsesTypePrefix(t *testing.T) {
	id := nextWorkflowBuilderStepID(nil, workflowBuilderDraftStep{
		Type:       appserver.WorkflowStepTypeAction,
		ActionType: workflowActionTypeDB,
	}, -1)
	if !strings.HasPrefix(id, "action_db_") {
		t.Fatalf("expected action-db step id prefix action_db_, got %q", id)
	}
}

func TestWorkflowBuilderRejectsUnsupportedShape(t *testing.T) {
	step := appserver.WorkflowStep{
		ID:   "exec_with_extra",
		Type: appserver.WorkflowStepTypeAction,
		Action: action.ActionConfig{
			Type: workflowActionTypeExec,
			Args: map[string]interface{}{
				"command": "echo hi",
				"cwd":     "/tmp",
			},
		},
	}
	if _, err := workflowBuilderDraftsFromSteps([]appserver.WorkflowStep{step}); err == nil {
		t.Fatalf("expected unsupported builder shape to be rejected")
	}
}
