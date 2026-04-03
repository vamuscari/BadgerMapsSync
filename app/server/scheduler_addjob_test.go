package server

import (
	"badgermaps/app/action"
	"badgermaps/app/state"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

func TestAddJobRollsBackWhenSaveFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not reliably enforced on windows")
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	st := state.NewState()
	*st.ConfigFile = configPath

	s := NewScheduler(st, nil, nil, nil, nil, nil, nil, nil, "")

	job := &ScheduledJob{
		ID:       "job_rollback",
		Name:     "Rollback Test",
		Schedule: "0 0 * * * *",
		Steps: []WorkflowStep{
			{ID: "pull_accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts},
		},
		Enabled: true,
	}

	if err := os.Chmod(tempDir, 0555); err != nil {
		t.Fatalf("failed to lock directory permissions: %v", err)
	}
	defer func() {
		_ = os.Chmod(tempDir, 0755)
	}()

	err := s.AddJob(job)
	if err == nil {
		t.Fatalf("expected AddJob to fail when persistence is unavailable")
	}

	if _, exists := s.jobs[job.ID]; exists {
		t.Fatalf("expected in-memory job rollback after save failure")
	}
	if job.cronID != 0 {
		t.Fatalf("expected cron entry rollback after save failure, got cron id %d", job.cronID)
	}
	if entries := s.cron.Entries(); len(entries) != 0 {
		t.Fatalf("expected no cron entries after rollback, got %d", len(entries))
	}
}

func TestRemoveJobRollsBackWhenSaveFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not reliably enforced on windows")
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	st := state.NewState()
	*st.ConfigFile = configPath

	s := NewScheduler(st, nil, nil, nil, nil, nil, nil, nil, "")

	job := &ScheduledJob{
		ID:       "job_remove_rollback",
		Name:     "Remove Rollback Test",
		Schedule: "0 0 * * * *",
		Steps: []WorkflowStep{
			{ID: "pull_accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts},
		},
		Enabled: true,
	}

	if err := s.AddJob(job); err != nil {
		t.Fatalf("failed to add seed job: %v", err)
	}
	if _, exists := s.jobs[job.ID]; !exists {
		t.Fatalf("expected seed job to be present")
	}

	jobsFile := filepath.Join(tempDir, "scheduled_jobs.json")
	if err := os.Chmod(jobsFile, 0444); err != nil {
		t.Fatalf("failed to lock jobs file permissions: %v", err)
	}
	defer func() {
		_ = os.Chmod(jobsFile, 0644)
	}()

	err := s.RemoveJob(job.ID)
	if err == nil {
		t.Fatalf("expected RemoveJob to fail when persistence is unavailable")
	}

	rolledBack, exists := s.jobs[job.ID]
	if !exists || rolledBack == nil {
		t.Fatalf("expected removed job to be restored in memory after save failure")
	}
	if rolledBack.cronID == 0 {
		t.Fatalf("expected restored job to be reattached to cron after rollback")
	}
	if entries := s.cron.Entries(); len(entries) != 1 {
		t.Fatalf("expected one cron entry after rollback, got %d", len(entries))
	}
}

func TestUpdateJobRollsBackWhenSaveFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not reliably enforced on windows")
	}

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	st := state.NewState()
	*st.ConfigFile = configPath

	s := NewScheduler(st, nil, nil, nil, nil, nil, nil, nil, "")

	job := &ScheduledJob{
		ID:       "job_update_rollback",
		Name:     "Original Name",
		Schedule: "0 0 * * * *",
		Steps: []WorkflowStep{
			{ID: "pull_accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts},
		},
		Enabled: true,
	}
	if err := s.AddJob(job); err != nil {
		t.Fatalf("failed to add seed job: %v", err)
	}

	before := cloneScheduledJob(s.jobs[job.ID])
	if before == nil {
		t.Fatalf("expected seed job snapshot")
	}

	jobsFile := filepath.Join(tempDir, "scheduled_jobs.json")
	if err := os.Chmod(jobsFile, 0444); err != nil {
		t.Fatalf("failed to lock jobs file permissions: %v", err)
	}
	defer func() {
		_ = os.Chmod(jobsFile, 0644)
	}()

	err := s.UpdateJob(job.ID, &ScheduledJob{
		Name:     "Updated Name",
		Schedule: "0 30 * * * *",
		Steps: []WorkflowStep{
			{ID: "push_accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePushAccounts},
		},
		Enabled: true,
	})
	if err == nil {
		t.Fatalf("expected UpdateJob to fail when persistence is unavailable")
	}

	after := s.jobs[job.ID]
	if after == nil {
		t.Fatalf("expected updated job to be restored in memory after save failure")
	}
	if after.Name != before.Name || after.Schedule != before.Schedule {
		t.Fatalf("expected rollback to restore original name/schedule, got name=%q schedule=%q", after.Name, after.Schedule)
	}
	if !reflect.DeepEqual(after.Steps, before.Steps) {
		t.Fatalf("expected rollback to restore original steps")
	}
	if after.cronID == 0 {
		t.Fatalf("expected restored job to be reattached to cron after rollback")
	}
	if entries := s.cron.Entries(); len(entries) != 1 {
		t.Fatalf("expected one cron entry after rollback, got %d", len(entries))
	}
}

func TestRunScheduledJobFailurePreservesLastSuccess(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	st := state.NewState()
	*st.ConfigFile = configPath

	s := NewScheduler(st, nil, nil, nil, nil, failingSchedulerSyncExecutor{}, nil, nil, "")

	lastSuccess := time.Now().Add(-1 * time.Hour).UTC().Truncate(time.Second)
	job := &ScheduledJob{
		ID:       "job_last_success",
		Name:     "Last Success Preserve",
		Schedule: "0 0 * * * *",
		Steps: []WorkflowStep{
			{ID: "pull_accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts},
		},
		Enabled:     false,
		LastSuccess: &lastSuccess,
	}
	s.jobs[job.ID] = job

	if err := s.runScheduledJob(job.ID); err == nil {
		t.Fatalf("expected scheduled job failure")
	}

	preserved := s.jobs[job.ID].LastSuccess
	if preserved == nil {
		t.Fatalf("expected LastSuccess to be preserved after failure")
	}
	if !preserved.Equal(lastSuccess) {
		t.Fatalf("expected LastSuccess %s, got %s", lastSuccess, preserved.Format(time.RFC3339Nano))
	}
}

func TestRunScheduledJobPreservesNonZeroNextRunWhenCronNotStarted(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	st := state.NewState()
	*st.ConfigFile = configPath

	s := NewScheduler(st, nil, nil, nil, nil, nil, nil, nil, "")

	job := &ScheduledJob{
		ID:       "job_next_run_non_zero",
		Name:     "Next Run Non-Zero",
		Schedule: "0 0 * * * *",
		Steps: []WorkflowStep{
			{
				ID:   "action_echo",
				Type: WorkflowStepTypeAction,
				Action: action.ActionConfig{
					Type: "exec",
					Args: map[string]interface{}{
						"command": "echo scheduler-next-run",
					},
				},
			},
		},
		Enabled: true,
	}

	if err := s.prepareJobForWrite(job); err != nil {
		t.Fatalf("failed to prepare scheduled job: %v", err)
	}
	if err := s.attachJobToCron(job); err != nil {
		t.Fatalf("failed to attach scheduled job: %v", err)
	}
	if job.NextRun == nil || job.NextRun.IsZero() {
		t.Fatal("expected initial next run to be populated")
	}
	s.jobs[job.ID] = job

	if err := s.runScheduledJob(job.ID); err != nil {
		t.Fatalf("expected scheduled job run to succeed, got %v", err)
	}

	updated := s.jobs[job.ID]
	if updated.NextRun == nil {
		t.Fatal("expected next run to stay populated after manual run")
	}
	if updated.NextRun.IsZero() {
		t.Fatal("expected non-zero next run after manual run")
	}
}

func TestQueueStoredJobNowLoadsJobsFromPersistence(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	st := state.NewState()
	*st.ConfigFile = configPath

	persisted := map[string]*ScheduledJob{
		"job_queue_persisted": {
			ID:       "job_queue_persisted",
			Name:     "Queue Persisted Job",
			Schedule: "0 0 * * * *",
			Steps: []WorkflowStep{
				{
					ID:   "action_echo",
					Type: WorkflowStepTypeAction,
					Action: action.ActionConfig{
						Type: "exec",
						Args: map[string]interface{}{
							"command": "echo queued-from-store",
						},
					},
				},
			},
			Enabled: true,
		},
	}
	if err := SaveScheduledJobs(st, persisted); err != nil {
		t.Fatalf("failed to seed persisted scheduled jobs: %v", err)
	}

	queue := NewSyncJobCoordinator(st, nil)
	defer queue.Stop()

	s := NewScheduler(st, nil, nil, nil, nil, nil, queue, nil, "")
	if err := s.QueueStoredJobNow("job_queue_persisted"); err != nil {
		t.Fatalf("expected persisted job queueing to succeed, got %v", err)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if len(queue.ListJobs()) > 0 {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("expected queue entry after QueueStoredJobNow")
}

type failingSchedulerSyncExecutor struct{}

func (failingSchedulerSyncExecutor) PullAccounts() error   { return os.ErrPermission }
func (failingSchedulerSyncExecutor) PullCheckins() error   { return nil }
func (failingSchedulerSyncExecutor) PullRoutes() error     { return nil }
func (failingSchedulerSyncExecutor) PullProfile() error    { return nil }
func (failingSchedulerSyncExecutor) PushAll() error        { return nil }
func (failingSchedulerSyncExecutor) PushAccounts() error   { return nil }
func (failingSchedulerSyncExecutor) PushCheckins() error   { return nil }
func (failingSchedulerSyncExecutor) PullAccount(int) error { return nil }
func (failingSchedulerSyncExecutor) PullCheckin(int) error { return nil }
func (failingSchedulerSyncExecutor) PullRoute(int) error   { return nil }
