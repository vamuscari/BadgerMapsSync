package server

import (
	"badgermaps/app/state"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type RuntimeActivity struct {
	ActiveJobID         string      `json:"active_job_id,omitempty"`
	ActiveJobName       string      `json:"active_job_name,omitempty"`
	ActiveJobSource     string      `json:"active_job_source,omitempty"`
	ActiveJobMode       SyncMode    `json:"active_job_mode,omitempty"`
	ActiveJobKind       SyncJobKind `json:"active_job_kind,omitempty"`
	ActiveJobParentID   string      `json:"active_job_parent_id,omitempty"`
	ActiveJobRootID     string      `json:"active_job_root_id,omitempty"`
	ActiveJobStepID     string      `json:"active_job_step_id,omitempty"`
	ActiveJobStepIndex  int         `json:"active_job_step_index,omitempty"`
	ActiveJobTotalSteps int         `json:"active_job_total_steps,omitempty"`
	ActiveJobAction     string      `json:"active_job_action,omitempty"`
	ActiveJobErrorCount int         `json:"active_job_error_count,omitempty"`
	QueueDepth          int         `json:"queue_depth"`
	LastHeartbeat       time.Time   `json:"last_heartbeat"`
	UpdatedAt           time.Time   `json:"updated_at"`
}

func (ra RuntimeActivity) IsActive() bool {
	return ra.ActiveJobID != ""
}

func WriteRuntimeActivity(s *state.State, activity RuntimeActivity) error {
	path := runtimeActivityPath(s)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create runtime activity directory: %w", err)
	}

	data, err := json.MarshalIndent(activity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal runtime activity: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write runtime activity file: %w", err)
	}
	return nil
}

func ReadRuntimeActivity(s *state.State) (RuntimeActivity, error) {
	path := runtimeActivityPath(s)
	data, err := os.ReadFile(path)
	if err != nil {
		return RuntimeActivity{}, err
	}

	var activity RuntimeActivity
	if err := json.Unmarshal(data, &activity); err != nil {
		return RuntimeActivity{}, fmt.Errorf("failed to parse runtime activity file: %w", err)
	}
	return activity, nil
}
