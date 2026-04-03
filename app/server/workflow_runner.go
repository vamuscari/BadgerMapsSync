package server

import (
	"badgermaps/app/action"
	"context"
	"fmt"
	"strings"
)

type WorkflowExecutionOptions struct {
	Queue      *SyncJobCoordinator
	Source     string
	ParentMode SyncMode
	ResourceID int
	Steps      []WorkflowStep
	RunSync    func(mode SyncMode, resourceID int) error
	RunAction  func(cfg action.ActionConfig, step WorkflowStep) error
}

func ExecuteWorkflowSteps(opts WorkflowExecutionOptions) (int, error) {
	if len(opts.Steps) == 0 {
		return 0, fmt.Errorf("no workflow steps defined")
	}
	if opts.RunSync == nil {
		return 0, fmt.Errorf("sync step runner is required")
	}
	if opts.RunAction == nil {
		return 0, fmt.Errorf("action step runner is required")
	}

	source := strings.TrimSpace(opts.Source)
	if source == "" {
		source = "manual"
	}

	actionFailures := 0
	totalSteps := len(opts.Steps)
	for idx, step := range opts.Steps {
		stepName := step.EffectiveName()
		if opts.Queue != nil {
			opts.Queue.SetActiveJobAction(stepName)
		}

		switch step.Type {
		case WorkflowStepTypeSync:
			resourceID := resourceIDForSyncStep(step.SyncMode, opts.ResourceID)
			runSync := func() error {
				return opts.RunSync(step.SyncMode, resourceID)
			}
			if opts.Queue != nil {
				_, err := opts.Queue.RunChildJob(SyncChildJobRequest{
					Name:       stepName,
					Source:     source,
					Mode:       step.SyncMode,
					Kind:       SyncJobKindSync,
					StepID:     step.ID,
					StepIndex:  idx + 1,
					TotalSteps: totalSteps,
					Run: func(_ context.Context) error {
						return runSync()
					},
				})
				if err != nil {
					return actionFailures, err
				}
			} else if err := runSync(); err != nil {
				return actionFailures, err
			}
		case WorkflowStepTypeAction:
			if !step.Action.IsEnabled() {
				continue
			}
			actionType, commandText := actionLogMetadata(step.Action)
			runAction := func() error {
				return opts.RunAction(step.Action, step)
			}
			if opts.Queue != nil {
				_, err := opts.Queue.RunChildJob(SyncChildJobRequest{
					Name:        stepName,
					Source:      source,
					Mode:        opts.ParentMode,
					Kind:        SyncJobKindAction,
					StepID:      step.ID,
					StepIndex:   idx + 1,
					TotalSteps:  totalSteps,
					ActionType:  actionType,
					CommandText: commandText,
					Run: func(_ context.Context) error {
						return runAction()
					},
				})
				if err != nil {
					actionFailures++
					opts.Queue.IncrementActiveJobErrorCount(1)
					continue
				}
			} else if err := runAction(); err != nil {
				actionFailures++
			}
		default:
			return actionFailures, fmt.Errorf("unsupported workflow step type: %s", step.Type)
		}
	}

	return actionFailures, nil
}

func resourceIDForSyncStep(mode SyncMode, fallback int) int {
	switch mode {
	case SyncModePullAccount, SyncModePullCheckin, SyncModePullRoute:
		return fallback
	default:
		return 0
	}
}

func actionLogMetadata(cfg action.ActionConfig) (string, string) {
	actionType := strings.TrimSpace(cfg.Type)
	if actionType == "" {
		return "", ""
	}

	pickByPriority := func(keys ...string) string {
		for _, key := range keys {
			value := actionArgToString(cfg.Args[key])
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
		return ""
	}

	var commandText string
	switch strings.ToLower(actionType) {
	case "exec":
		command := actionArgToString(cfg.Args["command"])
		args := actionArgToSlice(cfg.Args["args"])
		commandText = strings.TrimSpace(command)
		if len(args) > 0 {
			if commandText != "" {
				commandText += " "
			}
			commandText += strings.Join(args, " ")
		}
	case "db":
		commandText = pickByPriority("command", "query", "procedure", "function")
	default:
		commandText = pickByPriority("command", "query", "procedure", "function")
	}

	return actionType, strings.TrimSpace(commandText)
}

func actionArgToString(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case []byte:
		return string(typed)
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func actionArgToSlice(value any) []string {
	switch typed := value.(type) {
	case nil:
		return nil
	case []string:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			result = append(result, item)
		}
		return result
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(actionArgToString(item))
			if text == "" {
				continue
			}
			result = append(result, text)
		}
		return result
	case string:
		trimmed := strings.TrimSpace(typed)
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	default:
		text := strings.TrimSpace(actionArgToString(typed))
		if text == "" {
			return nil
		}
		return []string{text}
	}
}
