package server

import (
	"badgermaps/app/state"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
