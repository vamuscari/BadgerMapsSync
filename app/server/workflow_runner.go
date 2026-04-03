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
			runAction := func() error {
				return opts.RunAction(step.Action, step)
			}
			if opts.Queue != nil {
				_, err := opts.Queue.RunChildJob(SyncChildJobRequest{
					Name:       stepName,
					Source:     source,
					Mode:       opts.ParentMode,
					Kind:       SyncJobKindAction,
					StepID:     step.ID,
					StepIndex:  idx + 1,
					TotalSteps: totalSteps,
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
