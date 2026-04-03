package server

import (
	"badgermaps/app/action"
	"fmt"
	"sort"
	"strings"
)

type WorkflowStepType string

const (
	WorkflowStepTypeSync   WorkflowStepType = "sync"
	WorkflowStepTypeAction WorkflowStepType = "action"
)

type WorkflowStep struct {
	ID       string              `yaml:"id" json:"id"`
	Name     string              `yaml:"name,omitempty" json:"name,omitempty"`
	Type     WorkflowStepType    `yaml:"type" json:"type"`
	SyncMode SyncMode            `yaml:"sync_type,omitempty" json:"sync_type,omitempty"`
	Action   action.ActionConfig `yaml:"action,omitempty" json:"action,omitempty"`
}

type WorkflowProfile struct {
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description,omitempty" json:"description,omitempty"`
	Steps       []WorkflowStep `yaml:"steps" json:"steps"`
}

func (s WorkflowStep) EffectiveName() string {
	trimmed := strings.TrimSpace(s.Name)
	if trimmed != "" {
		return trimmed
	}
	switch s.Type {
	case WorkflowStepTypeSync:
		if s.SyncMode != "" {
			return fmt.Sprintf("sync:%s", s.SyncMode)
		}
		return "sync"
	case WorkflowStepTypeAction:
		actionType := strings.TrimSpace(s.Action.Type)
		if actionType == "" {
			actionType = "action"
		}
		return fmt.Sprintf("action:%s", actionType)
	default:
		return string(s.Type)
	}
}

func ValidateWorkflowStep(step WorkflowStep) error {
	if strings.TrimSpace(step.ID) == "" {
		return fmt.Errorf("workflow step id is required")
	}

	switch step.Type {
	case WorkflowStepTypeSync:
		if step.SyncMode == "" {
			return fmt.Errorf("workflow step %s requires sync_type", step.ID)
		}
		normalizedMode := NormalizeSyncMode(string(step.SyncMode))
		if !IsWorkflowSyncMode(normalizedMode) {
			return fmt.Errorf("workflow step %s has unsupported sync_type %q", step.ID, step.SyncMode)
		}
	case WorkflowStepTypeAction:
		if strings.TrimSpace(step.Action.Type) == "" {
			return fmt.Errorf("workflow step %s requires action.type", step.ID)
		}
		actionInstance, err := action.NewActionFromConfig(step.Action)
		if err != nil {
			return fmt.Errorf("workflow step %s has invalid action config: %w", step.ID, err)
		}
		if err := actionInstance.Validate(); err != nil {
			return fmt.Errorf("workflow step %s action failed validation: %w", step.ID, err)
		}
	default:
		return fmt.Errorf("workflow step %s has unsupported type %q", step.ID, step.Type)
	}

	return nil
}

func ValidateWorkflowSteps(steps []WorkflowStep) error {
	if len(steps) == 0 {
		return fmt.Errorf("at least one workflow step is required")
	}

	seen := map[string]struct{}{}
	for _, step := range steps {
		if err := ValidateWorkflowStep(step); err != nil {
			return err
		}
		if _, exists := seen[step.ID]; exists {
			return fmt.Errorf("duplicate workflow step id %q", step.ID)
		}
		seen[step.ID] = struct{}{}
	}

	return nil
}

func ValidateWorkflowProfile(profile WorkflowProfile) error {
	if strings.TrimSpace(profile.Name) == "" {
		return fmt.Errorf("workflow profile name is required")
	}
	if err := ValidateWorkflowSteps(profile.Steps); err != nil {
		return fmt.Errorf("workflow profile %q invalid: %w", profile.Name, err)
	}
	return nil
}

func ValidateWorkflowProfiles(profiles map[string]WorkflowProfile) error {
	if len(profiles) == 0 {
		return fmt.Errorf("at least one workflow profile is required")
	}

	keys := make([]string, 0, len(profiles))
	for key := range profiles {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		profile := profiles[key]
		if strings.TrimSpace(profile.Name) == "" {
			profile.Name = key
		}
		if err := ValidateWorkflowProfile(profile); err != nil {
			return err
		}
	}

	return nil
}

func NormalizeWorkflowProfiles(profiles map[string]WorkflowProfile) map[string]WorkflowProfile {
	if profiles == nil {
		profiles = map[string]WorkflowProfile{}
	}

	normalized := make(map[string]WorkflowProfile, len(profiles))
	for key, profile := range profiles {
		name := strings.TrimSpace(key)
		if name == "" {
			name = strings.TrimSpace(profile.Name)
		}
		if name == "" {
			continue
		}
		profile.Name = name
		profile.Description = strings.TrimSpace(profile.Description)
		normalized[name] = profile
	}
	return normalized
}

func DefaultWorkflowProfiles() map[string]WorkflowProfile {
	return map[string]WorkflowProfile{
		"pull_all": {
			Name:        "pull_all",
			Description: "Pull accounts, then check-ins, routes, and profile.",
			Steps: []WorkflowStep{
				{ID: "pull_accounts", Name: "Pull Accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts},
				{ID: "pull_checkins", Name: "Pull Check-ins", Type: WorkflowStepTypeSync, SyncMode: SyncModePullCheckins},
				{ID: "pull_routes", Name: "Pull Routes", Type: WorkflowStepTypeSync, SyncMode: SyncModePullRoutes},
				{ID: "pull_profile", Name: "Pull Profile", Type: WorkflowStepTypeSync, SyncMode: SyncModePullProfile},
			},
		},
		"push_all": {
			Name:        "push_all",
			Description: "Push accounts, then check-ins.",
			Steps: []WorkflowStep{
				{ID: "push_accounts", Name: "Push Accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePushAccounts},
				{ID: "push_checkins", Name: "Push Check-ins", Type: WorkflowStepTypeSync, SyncMode: SyncModePushCheckins},
			},
		},
		"pull_push": {
			Name:        "pull_push",
			Description: "Pull all resources, then push local account/check-in changes.",
			Steps: []WorkflowStep{
				{ID: "pull_accounts", Name: "Pull Accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts},
				{ID: "pull_checkins", Name: "Pull Check-ins", Type: WorkflowStepTypeSync, SyncMode: SyncModePullCheckins},
				{ID: "pull_routes", Name: "Pull Routes", Type: WorkflowStepTypeSync, SyncMode: SyncModePullRoutes},
				{ID: "pull_profile", Name: "Pull Profile", Type: WorkflowStepTypeSync, SyncMode: SyncModePullProfile},
				{ID: "push_accounts", Name: "Push Accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePushAccounts},
				{ID: "push_checkins", Name: "Push Check-ins", Type: WorkflowStepTypeSync, SyncMode: SyncModePushCheckins},
			},
		},
		"pull_accounts": {
			Name:        "pull_accounts",
			Description: "Pull account records.",
			Steps:       []WorkflowStep{{ID: "pull_accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts}},
		},
		"pull_checkins": {
			Name:        "pull_checkins",
			Description: "Pull check-in records.",
			Steps:       []WorkflowStep{{ID: "pull_checkins", Type: WorkflowStepTypeSync, SyncMode: SyncModePullCheckins}},
		},
		"pull_routes": {
			Name:        "pull_routes",
			Description: "Pull route records.",
			Steps:       []WorkflowStep{{ID: "pull_routes", Type: WorkflowStepTypeSync, SyncMode: SyncModePullRoutes}},
		},
		"pull_profile": {
			Name:        "pull_profile",
			Description: "Pull user profile.",
			Steps:       []WorkflowStep{{ID: "pull_profile", Type: WorkflowStepTypeSync, SyncMode: SyncModePullProfile}},
		},
		"push_accounts": {
			Name:        "push_accounts",
			Description: "Push account changes.",
			Steps:       []WorkflowStep{{ID: "push_accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePushAccounts}},
		},
		"push_checkins": {
			Name:        "push_checkins",
			Description: "Push check-in changes.",
			Steps:       []WorkflowStep{{ID: "push_checkins", Type: WorkflowStepTypeSync, SyncMode: SyncModePushCheckins}},
		},
	}
}

func DefaultWorkflowProfileForMode(mode SyncMode) string {
	switch mode {
	case SyncModePull:
		return "pull_all"
	case SyncModePush:
		return "push_all"
	case SyncModePullPush:
		return "pull_push"
	default:
		return strings.TrimSpace(string(mode))
	}
}

func ActionStepsForMode(profiles map[string]WorkflowProfile, mode SyncMode) []WorkflowStep {
	profileName := DefaultWorkflowProfileForMode(mode)
	if profileName == "" {
		return nil
	}
	profile, ok := profiles[profileName]
	if !ok {
		return nil
	}
	steps := make([]WorkflowStep, 0, len(profile.Steps))
	for _, step := range profile.Steps {
		if step.Type == WorkflowStepTypeAction {
			steps = append(steps, step)
		}
	}
	return steps
}

func ResolveWorkflowSteps(profileName string, inlineSteps []WorkflowStep, profiles map[string]WorkflowProfile) ([]WorkflowStep, error) {
	if len(inlineSteps) > 0 {
		if err := ValidateWorkflowSteps(inlineSteps); err != nil {
			return nil, err
		}
		return inlineSteps, nil
	}

	trimmed := strings.TrimSpace(profileName)
	if trimmed == "" {
		return nil, fmt.Errorf("job must define steps or workflow_profile")
	}
	profile, ok := profiles[trimmed]
	if !ok {
		return nil, fmt.Errorf("workflow profile %q not found", trimmed)
	}
	if err := ValidateWorkflowProfile(profile); err != nil {
		return nil, err
	}
	return profile.Steps, nil
}
