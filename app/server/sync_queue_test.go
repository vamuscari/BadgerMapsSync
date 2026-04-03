package server

import (
	"badgermaps/app/state"
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSyncJobCoordinatorSerializesJobs(t *testing.T) {
	s := state.NewState()
	*s.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	queue := NewSyncJobCoordinator(s, nil)
	defer queue.Stop()

	var (
		mu    sync.Mutex
		order []string
	)

	job1, err := queue.Submit(SyncJobRequest{
		Name:   "first",
		Source: "test",
		Mode:   SyncModePull,
		Run: func(_ context.Context) error {
			time.Sleep(150 * time.Millisecond)
			mu.Lock()
			order = append(order, "first")
			mu.Unlock()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to queue first job: %v", err)
	}

	job2, err := queue.Submit(SyncJobRequest{
		Name:   "second",
		Source: "test",
		Mode:   SyncModePush,
		Run: func(_ context.Context) error {
			mu.Lock()
			order = append(order, "second")
			mu.Unlock()
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to queue second job: %v", err)
	}

	waitForTerminalJob(t, queue, job1.ID)
	waitForTerminalJob(t, queue, job2.ID)

	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 {
		t.Fatalf("expected 2 job executions, got %d", len(order))
	}
	if order[0] != "first" || order[1] != "second" {
		t.Fatalf("expected serialized execution order [first second], got %v", order)
	}
}

func TestSyncJobCoordinatorTracksActiveJobAction(t *testing.T) {
	s := state.NewState()
	*s.ConfigFile = filepath.Join(t.TempDir(), "config.yaml")

	queue := NewSyncJobCoordinator(s, nil)
	defer queue.Stop()

	runGate := make(chan struct{})
	job, err := queue.Submit(SyncJobRequest{
		Name:   "tracked",
		Source: "test",
		Mode:   SyncModePull,
		Run: func(_ context.Context) error {
			<-runGate
			return nil
		},
	})
	if err != nil {
		t.Fatalf("failed to queue job: %v", err)
	}

	waitForRunningJob(t, queue, job.ID)

	queue.SetActiveJobAction("Pulling accounts")

	activeJob, exists := queue.GetJob(job.ID)
	if !exists {
		t.Fatalf("expected job %s to exist", job.ID)
	}
	if activeJob.CurrentAction != "Pulling accounts" {
		t.Fatalf("expected current action to be tracked, got %q", activeJob.CurrentAction)
	}

	activity := queue.GetActivity()
	if activity.ActiveJobAction != "Pulling accounts" {
		t.Fatalf("expected runtime activity action to match, got %q", activity.ActiveJobAction)
	}

	close(runGate)
	waitForTerminalJob(t, queue, job.ID)

	completedJob, exists := queue.GetJob(job.ID)
	if !exists {
		t.Fatalf("expected completed job %s to exist", job.ID)
	}
	if completedJob.CurrentAction != "" {
		t.Fatalf("expected completed job action to be cleared, got %q", completedJob.CurrentAction)
	}

	activity = queue.GetActivity()
	if activity.ActiveJobAction != "" {
		t.Fatalf("expected runtime activity action to be cleared, got %q", activity.ActiveJobAction)
	}
}

func waitForRunningJob(t *testing.T, queue *SyncJobCoordinator, jobID string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, exists := queue.GetJob(jobID)
		if exists && job.Status == SyncJobRunning {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to start running", jobID)
}

func waitForTerminalJob(t *testing.T, queue *SyncJobCoordinator, jobID string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, exists := queue.GetJob(jobID)
		if exists && job.Status.IsTerminal() {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to complete", jobID)
}
