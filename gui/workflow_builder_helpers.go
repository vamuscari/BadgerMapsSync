package gui

import (
	"badgermaps/app/action"
	appserver "badgermaps/app/server"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	workflowEditorModeBuilder  = "Builder"
	workflowEditorModeAdvanced = "Advanced JSON"
	workflowProfileCustom      = "(custom)"

	workflowActionTypeExec = "exec"
	workflowActionTypeDB   = "db"
)

var workflowDBOperationKeys = []string{"command", "function", "procedure", "query"}

type workflowBuilderDraftStep struct {
	ID   string
	Name string
	Type appserver.WorkflowStepType

	SyncMode appserver.SyncMode

	ActionEnabled bool
	ActionType    string

	ExecCommand  string
	ExecUseShell bool
	ExecArgsText string

	DBOperation string
	DBValue     string
	DBArgsText  string
}

func cloneWorkflowSteps(steps []appserver.WorkflowStep) []appserver.WorkflowStep {
	if len(steps) == 0 {
		return []appserver.WorkflowStep{}
	}
	cloned := make([]appserver.WorkflowStep, len(steps))
	for i, step := range steps {
		cloned[i] = appserver.WorkflowStep{
			ID:       step.ID,
			Name:     step.Name,
			Type:     step.Type,
			SyncMode: step.SyncMode,
			Action: action.ActionConfig{
				Type:    step.Action.Type,
				Args:    cloneBuilderArgs(step.Action.Args),
				Enabled: cloneEnabledPtr(step.Action.Enabled),
			},
		}
	}
	return cloned
}

func cloneBuilderArgs(src map[string]interface{}) map[string]interface{} {
	if src == nil {
		return nil
	}
	bytes, err := json.Marshal(src)
	if err != nil {
		dst := make(map[string]interface{}, len(src))
		for k, v := range src {
			dst[k] = v
		}
		return dst
	}
	var dst map[string]interface{}
	if err := json.Unmarshal(bytes, &dst); err != nil || dst == nil {
		dst = make(map[string]interface{}, len(src))
		for k, v := range src {
			dst[k] = v
		}
	}
	return dst
}

func cloneEnabledPtr(enabled *bool) *bool {
	if enabled == nil {
		return nil
	}
	v := *enabled
	return &v
}

func supportedWorkflowBuilderSyncModes() []appserver.SyncMode {
	return appserver.WorkflowSyncModes()
}

func workflowBuilderSyncModeOptions() []string {
	modes := supportedWorkflowBuilderSyncModes()
	out := make([]string, 0, len(modes))
	for _, mode := range modes {
		out = append(out, string(mode))
	}
	return out
}

func workflowBuilderDraftsFromSteps(steps []appserver.WorkflowStep) ([]workflowBuilderDraftStep, error) {
	drafts := make([]workflowBuilderDraftStep, 0, len(steps))
	for _, step := range steps {
		draft, err := workflowBuilderDraftFromStep(step)
		if err != nil {
			return nil, err
		}
		drafts = append(drafts, draft)
	}
	return drafts, nil
}

func workflowBuilderDraftFromStep(step appserver.WorkflowStep) (workflowBuilderDraftStep, error) {
	draft := workflowBuilderDraftStep{
		ID:            strings.TrimSpace(step.ID),
		Name:          strings.TrimSpace(step.Name),
		Type:          step.Type,
		SyncMode:      step.SyncMode,
		ActionEnabled: step.Action.IsEnabled(),
		ExecUseShell:  true,
	}

	switch step.Type {
	case appserver.WorkflowStepTypeSync:
		if _, err := appserver.ParseSyncMode(string(step.SyncMode)); err != nil {
			return draft, fmt.Errorf("step %q has invalid sync_type: %w", step.ID, err)
		}
		if !appserver.IsWorkflowSyncMode(step.SyncMode) {
			return draft, fmt.Errorf("step %q uses unsupported sync_type %q for Builder", step.ID, step.SyncMode)
		}
	case appserver.WorkflowStepTypeAction:
		actionType := strings.TrimSpace(step.Action.Type)
		switch actionType {
		case workflowActionTypeExec:
			draft.ActionType = workflowActionTypeExec
			if err := populateExecDraft(&draft, step); err != nil {
				return draft, err
			}
		case workflowActionTypeDB:
			draft.ActionType = workflowActionTypeDB
			if err := populateDBDraft(&draft, step); err != nil {
				return draft, err
			}
		default:
			return draft, fmt.Errorf("step %q uses unsupported action type %q for Builder", step.ID, actionType)
		}
	default:
		return draft, fmt.Errorf("step %q uses unsupported type %q for Builder", step.ID, step.Type)
	}

	return draft, nil
}

func populateExecDraft(draft *workflowBuilderDraftStep, step appserver.WorkflowStep) error {
	args := cloneBuilderArgs(step.Action.Args)
	allowed := map[string]struct{}{"command": {}, "args": {}, "use_shell": {}}
	for key := range args {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("step %q exec action contains unsupported arg key %q for Builder", step.ID, key)
		}
	}

	if rawCommand, ok := args["command"]; ok {
		command, ok := rawCommand.(string)
		if !ok {
			return fmt.Errorf("step %q exec command must be a string", step.ID)
		}
		draft.ExecCommand = command
	}

	draft.ExecUseShell = true
	if rawUseShell, ok := args["use_shell"]; ok {
		useShell, ok := rawUseShell.(bool)
		if !ok {
			return fmt.Errorf("step %q exec use_shell must be a boolean", step.ID)
		}
		draft.ExecUseShell = useShell
	}

	if rawArgs, ok := args["args"]; ok {
		parsedArgs, err := toStringSlice(rawArgs)
		if err != nil {
			return fmt.Errorf("step %q exec args: %w", step.ID, err)
		}
		if draft.ExecUseShell && len(parsedArgs) > 0 {
			return fmt.Errorf("step %q exec args are only supported when use_shell is false", step.ID)
		}
		draft.ExecArgsText = strings.Join(parsedArgs, "\n")
	}

	return nil
}

func populateDBDraft(draft *workflowBuilderDraftStep, step appserver.WorkflowStep) error {
	args := cloneBuilderArgs(step.Action.Args)
	allowed := map[string]struct{}{"command": {}, "function": {}, "procedure": {}, "query": {}, "args": {}}
	for key := range args {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("step %q db action contains unsupported arg key %q for Builder", step.ID, key)
		}
	}

	foundOps := make([]string, 0, 1)
	for _, key := range workflowDBOperationKeys {
		if rawValue, ok := args[key]; ok {
			value, ok := rawValue.(string)
			if !ok {
				return fmt.Errorf("step %q db %s must be a string", step.ID, key)
			}
			foundOps = append(foundOps, key)
			draft.DBOperation = key
			draft.DBValue = value
		}
	}
	if len(foundOps) == 0 {
		return fmt.Errorf("step %q db action has no operation", step.ID)
	}
	if len(foundOps) > 1 {
		return fmt.Errorf("step %q db action has multiple operations (%s)", step.ID, strings.Join(foundOps, ", "))
	}

	if rawDBArgs, ok := args["args"]; ok {
		payload, err := json.MarshalIndent(rawDBArgs, "", "  ")
		if err != nil {
			return fmt.Errorf("step %q db args cannot be encoded as JSON: %w", step.ID, err)
		}
		draft.DBArgsText = string(payload)
	}

	return nil
}

func toStringSlice(value interface{}) ([]string, error) {
	switch typed := value.(type) {
	case []string:
		out := make([]string, len(typed))
		copy(out, typed)
		return out, nil
	case []interface{}:
		out := make([]string, 0, len(typed))
		for idx, v := range typed {
			s, ok := v.(string)
			if !ok {
				return nil, fmt.Errorf("argument %d is not a string", idx)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("args must be a string array")
	}
}

func workflowBuilderDraftsToSteps(drafts []workflowBuilderDraftStep) ([]appserver.WorkflowStep, error) {
	steps := make([]appserver.WorkflowStep, 0, len(drafts))
	for _, draft := range drafts {
		step, err := workflowStepFromBuilderDraft(draft)
		if err != nil {
			return nil, err
		}
		steps = append(steps, step)
	}
	if err := appserver.ValidateWorkflowSteps(steps); err != nil {
		return nil, err
	}
	return steps, nil
}

func workflowStepFromBuilderDraft(draft workflowBuilderDraftStep) (appserver.WorkflowStep, error) {
	step := appserver.WorkflowStep{
		ID:   strings.TrimSpace(draft.ID),
		Name: strings.TrimSpace(draft.Name),
		Type: draft.Type,
	}

	switch draft.Type {
	case appserver.WorkflowStepTypeSync:
		step.SyncMode = appserver.SyncMode(strings.TrimSpace(string(draft.SyncMode)))
	case appserver.WorkflowStepTypeAction:
		actionType := strings.TrimSpace(draft.ActionType)
		switch actionType {
		case workflowActionTypeExec:
			args := map[string]interface{}{
				"command":   strings.TrimSpace(draft.ExecCommand),
				"use_shell": draft.ExecUseShell,
			}
			if !draft.ExecUseShell {
				execArgs := parseMultilineArgs(draft.ExecArgsText)
				if len(execArgs) > 0 {
					args["args"] = execArgs
				}
			}
			step.Action = action.ActionConfig{Type: workflowActionTypeExec, Args: args}
		case workflowActionTypeDB:
			op := strings.TrimSpace(draft.DBOperation)
			if op == "" {
				op = "command"
			}
			if !containsString(workflowDBOperationKeys, op) {
				return step, fmt.Errorf("workflow step %q has unsupported db operation %q", step.ID, op)
			}
			args := map[string]interface{}{op: strings.TrimSpace(draft.DBValue)}
			if strings.TrimSpace(draft.DBArgsText) != "" {
				var parsed interface{}
				if err := json.Unmarshal([]byte(draft.DBArgsText), &parsed); err != nil {
					return step, fmt.Errorf("workflow step %q db args JSON is invalid: %w", step.ID, err)
				}
				args["args"] = parsed
			}
			step.Action = action.ActionConfig{Type: workflowActionTypeDB, Args: args}
		default:
			return step, fmt.Errorf("workflow step %q has unsupported action type %q", step.ID, actionType)
		}
		if !draft.ActionEnabled {
			step.Action.SetEnabled(false)
		}
	default:
		return step, fmt.Errorf("workflow step %q has unsupported type %q", step.ID, step.Type)
	}

	return step, nil
}

func parseMultilineArgs(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	lines := strings.Split(trimmed, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

func workflowStepsToJSON(steps []appserver.WorkflowStep) (string, error) {
	payload, err := json.MarshalIndent(steps, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func parseWorkflowStepsJSON(raw string) ([]appserver.WorkflowStep, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("workflow steps JSON is required")
	}
	var steps []appserver.WorkflowStep
	if err := json.Unmarshal([]byte(trimmed), &steps); err != nil {
		return nil, fmt.Errorf("invalid workflow steps JSON: %w", err)
	}
	if err := appserver.ValidateWorkflowSteps(steps); err != nil {
		return nil, err
	}
	return steps, nil
}

func workflowBuilderDraftsToJSON(drafts []workflowBuilderDraftStep) (string, []appserver.WorkflowStep, error) {
	steps, err := workflowBuilderDraftsToSteps(drafts)
	if err != nil {
		return "", nil, err
	}
	jsonText, err := workflowStepsToJSON(steps)
	if err != nil {
		return "", nil, err
	}
	return jsonText, steps, nil
}

func sanitizeWorkflowBuilderStepIDSegment(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return "step"
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range raw {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlphaNum {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if b.Len() == 0 || prevUnderscore {
			continue
		}
		b.WriteByte('_')
		prevUnderscore = true
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "step"
	}
	return out
}

func workflowBuilderStepIDPrefix(draft workflowBuilderDraftStep) string {
	switch draft.Type {
	case appserver.WorkflowStepTypeSync:
		mode := appserver.NormalizeSyncMode(string(draft.SyncMode))
		if appserver.IsWorkflowSyncMode(mode) {
			return sanitizeWorkflowBuilderStepIDSegment(string(mode))
		}
		return "sync"
	case appserver.WorkflowStepTypeAction:
		actionType := strings.TrimSpace(strings.ToLower(draft.ActionType))
		if actionType == "" {
			actionType = workflowActionTypeExec
		}
		return sanitizeWorkflowBuilderStepIDSegment("action_" + actionType)
	default:
		return "step"
	}
}

func randomWorkflowBuilderStepCode(length int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if length <= 0 {
		return ""
	}
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err == nil {
		out := make([]byte, length)
		for i, b := range randomBytes {
			out[i] = alphabet[int(b)%len(alphabet)]
		}
		return string(out)
	}
	fallback := strings.ToUpper(strconv.FormatInt(time.Now().UnixNano(), 36))
	if len(fallback) >= length {
		return fallback[len(fallback)-length:]
	}
	return strings.Repeat("X", length-len(fallback)) + fallback
}

func nextWorkflowBuilderStepID(drafts []workflowBuilderDraftStep, draft workflowBuilderDraftStep, currentIndex int) string {
	used := map[string]struct{}{}
	for idx, existing := range drafts {
		if idx == currentIndex {
			continue
		}
		id := strings.TrimSpace(existing.ID)
		if id != "" {
			used[strings.ToLower(id)] = struct{}{}
		}
	}
	prefix := workflowBuilderStepIDPrefix(draft)
	for attempt := 0; attempt < 512; attempt++ {
		candidate := fmt.Sprintf("%s_%s", prefix, randomWorkflowBuilderStepCode(5))
		if _, exists := used[strings.ToLower(candidate)]; !exists {
			return candidate
		}
	}
	return fmt.Sprintf("%s_%s", prefix, strings.ToUpper(strconv.FormatInt(time.Now().UnixNano(), 36)))
}

func applyWorkflowProfileTemplateSelection(currentProfile, selectedProfile string, currentSteps []appserver.WorkflowStep, profiles map[string]appserver.WorkflowProfile, confirmed bool) (string, []appserver.WorkflowStep, bool, error) {
	currentProfile = strings.TrimSpace(currentProfile)
	selectedProfile = strings.TrimSpace(selectedProfile)

	if selectedProfile == "" {
		selectedProfile = workflowProfileCustom
	}
	if currentProfile == "" {
		currentProfile = workflowProfileCustom
	}

	if selectedProfile == currentProfile {
		return currentProfile, cloneWorkflowSteps(currentSteps), false, nil
	}
	if selectedProfile == workflowProfileCustom {
		return workflowProfileCustom, cloneWorkflowSteps(currentSteps), false, nil
	}
	if !confirmed {
		return currentProfile, cloneWorkflowSteps(currentSteps), false, nil
	}

	profile, ok := profiles[selectedProfile]
	if !ok {
		return currentProfile, cloneWorkflowSteps(currentSteps), false, fmt.Errorf("workflow profile %q not found", selectedProfile)
	}
	return selectedProfile, cloneWorkflowSteps(profile.Steps), true, nil
}

func sortedWorkflowProfileOptions(profiles map[string]appserver.WorkflowProfile, selectedProfile string) []string {
	options := make([]string, 0, len(profiles)+2)
	options = append(options, workflowProfileCustom)
	for name := range profiles {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		options = append(options, name)
	}
	selectedProfile = strings.TrimSpace(selectedProfile)
	if selectedProfile != "" && selectedProfile != workflowProfileCustom && !containsString(options, selectedProfile) {
		options = append(options, selectedProfile)
	}
	sort.Strings(options[1:])
	return options
}

func defaultBuilderDraftStep(existingDrafts []workflowBuilderDraftStep) workflowBuilderDraftStep {
	draft := workflowBuilderDraftStep{
		Type:          appserver.WorkflowStepTypeSync,
		SyncMode:      appserver.SyncModePullAccounts,
		ActionEnabled: true,
		ActionType:    workflowActionTypeExec,
		ExecUseShell:  true,
		DBOperation:   "command",
	}
	draft.ID = nextWorkflowBuilderStepID(existingDrafts, draft, -1)
	return draft
}
