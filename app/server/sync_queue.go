package server

import (
	"badgermaps/app/state"
	"badgermaps/events"
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultSyncQueueBufferSize = 256
	defaultSyncHistoryLimit    = 256
)

type SyncJob struct {
	ID          string        `json:"id"`
	Name        string        `json:"name,omitempty"`
	Source      string        `json:"source"`
	Mode        SyncMode      `json:"mode"`
	Status      SyncJobStatus `json:"status"`
	QueuedAt    time.Time     `json:"queued_at"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	Error       string        `json:"error,omitempty"`
}

func (j *SyncJob) clone() *SyncJob {
	if j == nil {
		return nil
	}
	clone := *j
	if j.StartedAt != nil {
		started := *j.StartedAt
		clone.StartedAt = &started
	}
	if j.CompletedAt != nil {
		completed := *j.CompletedAt
		clone.CompletedAt = &completed
	}
	return &clone
}

type SyncJobRequest struct {
	Name   string
	Source string
	Mode   SyncMode
	Run    func(context.Context) error
}

type queuedSyncJob struct {
	job *SyncJob
	run func(context.Context) error
}

type SyncJobCoordinator struct {
	state  *state.State
	events *events.EventDispatcher

	mu             sync.RWMutex
	jobs           map[string]*SyncJob
	completedOrder []string
	activeJobID    string
	queuedCount    int
	activity       RuntimeActivity
	stopped        bool

	reqCh          chan queuedSyncJob
	stopCh         chan struct{}
	workerDone     chan struct{}
	heartbeatDone  chan struct{}
	jobCounterSeed atomic.Uint64
}

func NewSyncJobCoordinator(s *state.State, eventBus *events.EventDispatcher) *SyncJobCoordinator {
	coordinator := &SyncJobCoordinator{
		state:          s,
		events:         eventBus,
		jobs:           make(map[string]*SyncJob),
		completedOrder: make([]string, 0, defaultSyncHistoryLimit),
		reqCh:          make(chan queuedSyncJob, defaultSyncQueueBufferSize),
		stopCh:         make(chan struct{}),
		workerDone:     make(chan struct{}),
		heartbeatDone:  make(chan struct{}),
	}

	now := time.Now()
	coordinator.activity = RuntimeActivity{
		QueueDepth:    0,
		LastHeartbeat: now,
		UpdatedAt:     now,
	}
	coordinator.persistActivity(coordinator.activity)

	go coordinator.worker()
	go coordinator.heartbeatLoop()
	return coordinator
}

func (c *SyncJobCoordinator) Stop() {
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return
	}
	c.stopped = true
	c.mu.Unlock()

	close(c.stopCh)
	<-c.workerDone
	<-c.heartbeatDone
}

func (c *SyncJobCoordinator) Submit(req SyncJobRequest) (*SyncJob, error) {
	if req.Run == nil {
		return nil, fmt.Errorf("sync job run function is required")
	}

	if req.Source == "" {
		req.Source = "manual"
	}

	now := time.Now()
	job := &SyncJob{
		ID:       c.nextJobID(now),
		Name:     req.Name,
		Source:   req.Source,
		Mode:     req.Mode,
		Status:   SyncJobQueued,
		QueuedAt: now,
	}

	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return nil, fmt.Errorf("sync job coordinator is stopped")
	}
	c.jobs[job.ID] = job
	c.queuedCount++
	activity := c.snapshotActivityLocked(now)
	c.mu.Unlock()

	select {
	case c.reqCh <- queuedSyncJob{job: job, run: req.Run}:
	case <-c.stopCh:
		return nil, fmt.Errorf("sync job coordinator is stopping")
	default:
		c.mu.Lock()
		delete(c.jobs, job.ID)
		if c.queuedCount > 0 {
			c.queuedCount--
		}
		activity = c.snapshotActivityLocked(time.Now())
		c.mu.Unlock()
		c.persistActivity(activity)
		return nil, fmt.Errorf("sync job queue is full")
	}

	c.persistActivity(activity)
	if c.events != nil {
		c.events.Dispatch(events.Event{
			Type:   "sync.job.queued",
			Source: req.Source,
			Payload: events.GenericPayload{
				Type: "sync.job.queued",
				Data: map[string]interface{}{
					"job_id": job.ID,
					"mode":   string(job.Mode),
					"name":   job.Name,
				},
			},
		})
	}

	return job.clone(), nil
}

func (c *SyncJobCoordinator) GetJob(jobID string) (*SyncJob, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	job, exists := c.jobs[jobID]
	if !exists {
		return nil, false
	}
	return job.clone(), true
}

func (c *SyncJobCoordinator) GetActivity() RuntimeActivity {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.activity
}

func (c *SyncJobCoordinator) worker() {
	defer close(c.workerDone)

	for {
		select {
		case <-c.stopCh:
			return
		case req := <-c.reqCh:
			c.runJob(req)
		}
	}
}

func (c *SyncJobCoordinator) runJob(req queuedSyncJob) {
	startedAt := time.Now()
	c.mu.Lock()
	job, exists := c.jobs[req.job.ID]
	if !exists {
		c.mu.Unlock()
		return
	}
	job.Status = SyncJobRunning
	job.StartedAt = &startedAt
	if c.queuedCount > 0 {
		c.queuedCount--
	}
	c.activeJobID = job.ID
	activity := c.snapshotActivityLocked(startedAt)
	c.mu.Unlock()

	c.persistActivity(activity)
	if c.events != nil {
		c.events.Dispatch(events.Event{
			Type:   "sync.job.start",
			Source: job.Source,
			Payload: events.GenericPayload{
				Type: "sync.job.start",
				Data: map[string]interface{}{
					"job_id": job.ID,
					"mode":   string(job.Mode),
					"name":   job.Name,
				},
			},
		})
	}

	runErr := req.run(context.Background())
	completedAt := time.Now()

	c.mu.Lock()
	var completedJob *SyncJob
	job, exists = c.jobs[req.job.ID]
	if exists {
		job.CompletedAt = &completedAt
		if runErr != nil {
			job.Status = SyncJobFailed
			job.Error = runErr.Error()
		} else {
			job.Status = SyncJobCompleted
			job.Error = ""
		}
		c.completedOrder = append(c.completedOrder, job.ID)
		completedJob = job.clone()
	}
	c.activeJobID = ""
	c.pruneCompletedJobsLocked()
	activity = c.snapshotActivityLocked(completedAt)
	c.mu.Unlock()

	c.persistActivity(activity)
	if c.events != nil && completedJob != nil {
		payload := map[string]interface{}{
			"job_id": completedJob.ID,
			"mode":   string(completedJob.Mode),
			"name":   completedJob.Name,
		}
		if runErr != nil {
			payload["error"] = runErr.Error()
			c.events.Dispatch(events.Event{
				Type:   "sync.job.error",
				Source: completedJob.Source,
				Payload: events.GenericPayload{
					Type: "sync.job.error",
					Data: payload,
				},
			})
		} else {
			c.events.Dispatch(events.Event{
				Type:   "sync.job.complete",
				Source: completedJob.Source,
				Payload: events.GenericPayload{
					Type: "sync.job.complete",
					Data: payload,
				},
			})
		}
	}
}

func (c *SyncJobCoordinator) heartbeatLoop() {
	defer close(c.heartbeatDone)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case now := <-ticker.C:
			c.mu.Lock()
			activity := c.snapshotActivityLocked(now)
			c.mu.Unlock()
			c.persistActivity(activity)
		}
	}
}

func (c *SyncJobCoordinator) snapshotActivityLocked(now time.Time) RuntimeActivity {
	activity := RuntimeActivity{
		QueueDepth:    c.queuedCount,
		LastHeartbeat: now,
		UpdatedAt:     now,
	}
	if c.activeJobID != "" {
		if activeJob, exists := c.jobs[c.activeJobID]; exists {
			activity.ActiveJobID = activeJob.ID
			activity.ActiveJobName = activeJob.Name
			activity.ActiveJobSource = activeJob.Source
			activity.ActiveJobMode = activeJob.Mode
		}
	}
	c.activity = activity
	return activity
}

func (c *SyncJobCoordinator) persistActivity(activity RuntimeActivity) {
	if err := WriteRuntimeActivity(c.state, activity); err != nil && c.events != nil {
		c.events.Dispatch(events.Warningf("server", "Failed to persist runtime activity: %v", err))
	}
}

func (c *SyncJobCoordinator) pruneCompletedJobsLocked() {
	for len(c.completedOrder) > defaultSyncHistoryLimit {
		oldestID := c.completedOrder[0]
		c.completedOrder = c.completedOrder[1:]

		job, exists := c.jobs[oldestID]
		if !exists || !job.Status.IsTerminal() || c.activeJobID == oldestID {
			continue
		}
		delete(c.jobs, oldestID)
	}
}

func (c *SyncJobCoordinator) nextJobID(now time.Time) string {
	n := c.jobCounterSeed.Add(1)
	return fmt.Sprintf("sync_%d_%d", now.UnixNano(), n)
}
