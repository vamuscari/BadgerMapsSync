package server

import (
	"badgermaps/app/state"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func LoadScheduledJobs(s *state.State) (map[string]*ScheduledJob, error) {
	configDir := schedulerConfigDir(s)
	jobsFile := filepath.Join(configDir, "scheduled_jobs.json")

	if _, err := os.Stat(jobsFile); os.IsNotExist(err) {
		return map[string]*ScheduledJob{}, nil
	}

	data, err := os.ReadFile(jobsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read jobs file: %w", err)
	}

	var jobs map[string]*ScheduledJob
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse jobs file: %w", err)
	}
	if jobs == nil {
		return map[string]*ScheduledJob{}, nil
	}

	for id, job := range jobs {
		if job == nil {
			delete(jobs, id)
			continue
		}
		job.ID = id
		job.cronID = 0
		if err := ValidateScheduledJobDefinition(job, nil); err != nil {
			return nil, fmt.Errorf("scheduled job %q is invalid: %w", id, err)
		}
	}

	return jobs, nil
}

func SaveScheduledJobs(s *state.State, jobs map[string]*ScheduledJob) error {
	configDir := schedulerConfigDir(s)
	jobsFile := filepath.Join(configDir, "scheduled_jobs.json")

	if jobs == nil {
		jobs = map[string]*ScheduledJob{}
	}

	jobsToSave := make(map[string]*ScheduledJob, len(jobs))
	for id, job := range jobs {
		if job == nil {
			continue
		}
		if err := ValidateScheduledJobDefinition(job, nil); err != nil {
			return fmt.Errorf("scheduled job %q is invalid: %w", id, err)
		}
		jobCopy := *job
		jobCopy.ID = id
		jobCopy.cronID = 0
		jobsToSave[id] = &jobCopy
	}

	data, err := json.MarshalIndent(jobsToSave, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create jobs directory: %w", err)
	}

	if err := os.WriteFile(jobsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}
	return nil
}

func GenerateScheduledJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

func ValidateScheduledJobDefinition(job *ScheduledJob, workflowProfiles map[string]WorkflowProfile) error {
	if job == nil {
		return fmt.Errorf("job is required")
	}
	if job.LegacySyncType != "" || len(job.LegacyActions) > 0 {
		return fmt.Errorf("legacy scheduled job fields are not supported; manually rewrite this job to explicit workflow steps (remove sync_type/actions and define steps)")
	}

	if strings.TrimSpace(job.Name) == "" {
		return fmt.Errorf("job name is required")
	}
	if strings.TrimSpace(job.Schedule) == "" {
		return fmt.Errorf("job schedule is required")
	}

	steps := job.Steps
	if len(steps) == 0 {
		return fmt.Errorf("scheduled job must define explicit steps; manually rewrite legacy sync_type/actions jobs to steps")
	}
	if err := ValidateWorkflowSteps(steps); err != nil {
		return err
	}

	profileName := strings.TrimSpace(job.WorkflowProfile)
	if profileName != "" && workflowProfiles != nil {
		if _, exists := workflowProfiles[profileName]; !exists {
			return fmt.Errorf("workflow_profile %q not found", profileName)
		}
	}
	return nil
}
