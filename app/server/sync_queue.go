package server

import (
	"badgermaps/app/state"
	"badgermaps/events"
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultSyncQueueBufferSize = 256
	defaultSyncHistoryLimit    = 256
)

type SyncJobKind string

const (
	SyncJobKindWorkflow SyncJobKind = "workflow"
	SyncJobKindSync     SyncJobKind = "sync"
	SyncJobKindAction   SyncJobKind = "action"
)

type SyncJob struct {
	ID            string        `json:"id"`
	Name          string        `json:"name,omitempty"`
	Source        string        `json:"source"`
	Mode          SyncMode      `json:"mode"`
	Kind          SyncJobKind   `json:"job_kind,omitempty"`
	ParentJobID   string        `json:"parent_job_id,omitempty"`
	RootJobID     string        `json:"root_job_id,omitempty"`
	StepID        string        `json:"step_id,omitempty"`
	StepIndex     int           `json:"step_index,omitempty"`
	TotalSteps    int           `json:"total_steps,omitempty"`
	ActionType    string        `json:"action_type,omitempty"`
	CommandText   string        `json:"command_text,omitempty"`
	Status        SyncJobStatus `json:"status"`
	CurrentAction string        `json:"current_action,omitempty"`
	QueuedAt      time.Time     `json:"queued_at"`
	StartedAt     *time.Time    `json:"started_at,omitempty"`
	CompletedAt   *time.Time    `json:"completed_at,omitempty"`
	ErrorCount    int           `json:"error_count,omitempty"`
	Error         string        `json:"error,omitempty"`
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
	Name        string
	Source      string
	Mode        SyncMode
	Kind        SyncJobKind
	ParentJobID string
	RootJobID   string
	StepID      string
	StepIndex   int
	TotalSteps  int
	ActionType  string
	CommandText string
	Run         func(context.Context) error
}

type SyncChildJobRequest struct {
	Name        string
	Source      string
	Mode        SyncMode
	Kind        SyncJobKind
	ParentJobID string
	RootJobID   string
	StepID      string
	StepIndex   int
	TotalSteps  int
	ActionType  string
	CommandText string
	Run         func(context.Context) error
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

	job := c.newJob(req)
	now := job.QueuedAt

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
	c.dispatchJobEvent("sync.job.queued", job, nil)

	return job.clone(), nil
}

func (c *SyncJobCoordinator) RunChildJob(req SyncChildJobRequest) (*SyncJob, error) {
	if req.Run == nil {
		return nil, fmt.Errorf("child sync job run function is required")
	}

	now := time.Now()
	job := &SyncJob{
		ID:          c.nextJobID(now),
		Name:        strings.TrimSpace(req.Name),
		Source:      strings.TrimSpace(req.Source),
		Mode:        req.Mode,
		Kind:        req.Kind,
		ParentJobID: strings.TrimSpace(req.ParentJobID),
		RootJobID:   strings.TrimSpace(req.RootJobID),
		StepID:      strings.TrimSpace(req.StepID),
		StepIndex:   req.StepIndex,
		TotalSteps:  req.TotalSteps,
		ActionType:  strings.TrimSpace(req.ActionType),
		CommandText: strings.TrimSpace(req.CommandText),
		Status:      SyncJobRunning,
		QueuedAt:    now,
		StartedAt:   &now,
	}
	if job.Source == "" {
		job.Source = "manual"
	}
	if job.Kind == "" {
		job.Kind = SyncJobKindSync
	}

	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return nil, fmt.Errorf("sync job coordinator is stopped")
	}
	if c.activeJobID == "" {
		c.mu.Unlock()
		return nil, fmt.Errorf("no active parent job available for child execution")
	}

	parentID := job.ParentJobID
	if parentID == "" {
		parentID = c.activeJobID
		job.ParentJobID = parentID
	}

	if job.RootJobID == "" {
		rootID := parentID
		if parent, ok := c.jobs[parentID]; ok && parent != nil {
			if parent.RootJobID != "" {
				rootID = parent.RootJobID
			}
		}
		job.RootJobID = rootID
	}

	if job.RootJobID == "" {
		job.RootJobID = job.ParentJobID
	}

	c.jobs[job.ID] = job
	previousActive := c.activeJobID
	c.activeJobID = job.ID
	activity := c.snapshotActivityLocked(now)
	c.mu.Unlock()

	c.persistActivity(activity)
	c.dispatchJobEvent("sync.job.start", job, nil)

	runErr := req.Run(context.Background())
	completedAt := time.Now()

	c.mu.Lock()
	live, exists := c.jobs[job.ID]
	if exists {
		live.CompletedAt = &completedAt
		live.CurrentAction = ""
		switch {
		case runErr != nil:
			live.Status = SyncJobFailed
			live.Error = runErr.Error()
		case live.ErrorCount > 0:
			live.Status = SyncJobCompletedWithErrors
			live.Error = ""
		default:
			live.Status = SyncJobCompleted
			live.Error = ""
		}
		c.completedOrder = append(c.completedOrder, live.ID)
	}
	c.activeJobID = previousActive
	c.pruneCompletedJobsLocked()
	activity = c.snapshotActivityLocked(completedAt)
	var result *SyncJob
	if exists {
		result = live.clone()
	}
	c.mu.Unlock()

	c.persistActivity(activity)
	eventJob := result
	if eventJob == nil {
		eventJob = job.clone()
		if runErr != nil {
			eventJob.Status = SyncJobFailed
		}
	}
	if runErr != nil {
		c.dispatchJobEvent("sync.job.error", eventJob, runErr)
	} else {
		c.dispatchJobEvent("sync.job.complete", eventJob, nil)
	}

	if runErr != nil {
		return result, runErr
	}
	return result, nil
}

func (c *SyncJobCoordinator) IncrementActiveJobErrorCount(delta int) {
	if delta <= 0 {
		return
	}
	now := time.Now()

	c.mu.Lock()
	if c.activeJobID == "" {
		c.mu.Unlock()
		return
	}
	if job, exists := c.jobs[c.activeJobID]; exists && job != nil {
		job.ErrorCount += delta
	}
	activity := c.snapshotActivityLocked(now)
	c.mu.Unlock()

	c.persistActivity(activity)
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

func (c *SyncJobCoordinator) SetActiveJobAction(action string) {
	now := time.Now()
	trimmed := strings.TrimSpace(action)

	c.mu.Lock()
	if c.activeJobID == "" {
		c.mu.Unlock()
		return
	}

	job, exists := c.jobs[c.activeJobID]
	if !exists {
		c.mu.Unlock()
		return
	}

	job.CurrentAction = trimmed
	activity := c.snapshotActivityLocked(now)
	c.mu.Unlock()

	c.persistActivity(activity)
}

func (c *SyncJobCoordinator) ListJobs() []*SyncJob {
	c.mu.RLock()
	defer c.mu.RUnlock()

	jobs := make([]*SyncJob, 0, len(c.jobs))
	for _, job := range c.jobs {
		jobs = append(jobs, job.clone())
	}

	sort.SliceStable(jobs, func(i, j int) bool {
		left := jobs[i]
		right := jobs[j]

		leftRank := syncJobRank(left.Status)
		rightRank := syncJobRank(right.Status)
		if leftRank != rightRank {
			return leftRank < rightRank
		}

		switch left.Status {
		case SyncJobRunning:
			if left.StartedAt == nil || right.StartedAt == nil {
				return left.QueuedAt.Before(right.QueuedAt)
			}
			return left.StartedAt.Before(*right.StartedAt)
		case SyncJobQueued:
			return left.QueuedAt.Before(right.QueuedAt)
		default:
			if left.CompletedAt == nil || right.CompletedAt == nil {
				return left.QueuedAt.After(right.QueuedAt)
			}
			return left.CompletedAt.After(*right.CompletedAt)
		}
	})

	return jobs
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
	job.CurrentAction = ""
	if c.queuedCount > 0 {
		c.queuedCount--
	}
	if job.RootJobID == "" {
		job.RootJobID = job.ID
	}
	c.activeJobID = job.ID
	activity := c.snapshotActivityLocked(startedAt)
	c.mu.Unlock()

	c.persistActivity(activity)
	c.dispatchJobEvent("sync.job.start", job, nil)

	runErr := req.run(context.Background())
	completedAt := time.Now()

	c.mu.Lock()
	var completedJob *SyncJob
	job, exists = c.jobs[req.job.ID]
	if exists {
		job.CompletedAt = &completedAt
		job.CurrentAction = ""
		switch {
		case runErr != nil:
			job.Status = SyncJobFailed
			job.Error = runErr.Error()
		case job.ErrorCount > 0:
			job.Status = SyncJobCompletedWithErrors
			job.Error = ""
		default:
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
	if runErr != nil {
		c.dispatchJobEvent("sync.job.error", completedJob, runErr)
	} else {
		c.dispatchJobEvent("sync.job.complete", completedJob, nil)
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
			activity.ActiveJobKind = activeJob.Kind
			activity.ActiveJobParentID = activeJob.ParentJobID
			activity.ActiveJobRootID = activeJob.RootJobID
			activity.ActiveJobStepID = activeJob.StepID
			activity.ActiveJobStepIndex = activeJob.StepIndex
			activity.ActiveJobTotalSteps = activeJob.TotalSteps
			activity.ActiveJobAction = activeJob.CurrentAction
			activity.ActiveJobErrorCount = activeJob.ErrorCount
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

func (c *SyncJobCoordinator) newJob(req SyncJobRequest) *SyncJob {
	now := time.Now()
	job := &SyncJob{
		ID:          c.nextJobID(now),
		Name:        strings.TrimSpace(req.Name),
		Source:      strings.TrimSpace(req.Source),
		Mode:        req.Mode,
		Kind:        req.Kind,
		ParentJobID: strings.TrimSpace(req.ParentJobID),
		RootJobID:   strings.TrimSpace(req.RootJobID),
		StepID:      strings.TrimSpace(req.StepID),
		StepIndex:   req.StepIndex,
		TotalSteps:  req.TotalSteps,
		ActionType:  strings.TrimSpace(req.ActionType),
		CommandText: strings.TrimSpace(req.CommandText),
		Status:      SyncJobQueued,
		QueuedAt:    now,
	}
	if job.Source == "" {
		job.Source = "manual"
	}
	if job.Kind == "" {
		if job.ParentJobID == "" {
			job.Kind = SyncJobKindWorkflow
		} else {
			job.Kind = SyncJobKindSync
		}
	}
	if job.RootJobID == "" {
		if job.ParentJobID != "" {
			job.RootJobID = job.ParentJobID
		} else {
			job.RootJobID = job.ID
		}
	}
	return job
}

func (c *SyncJobCoordinator) dispatchJobEvent(eventType string, job *SyncJob, runErr error) {
	if c.events == nil || job == nil {
		return
	}

	payload := map[string]interface{}{
		"job_id":        job.ID,
		"mode":          string(job.Mode),
		"name":          job.Name,
		"job_kind":      string(job.Kind),
		"parent_job_id": job.ParentJobID,
		"root_job_id":   job.RootJobID,
		"step_id":       job.StepID,
		"step_index":    job.StepIndex,
		"total_steps":   job.TotalSteps,
		"action_type":   job.ActionType,
		"command_text":  job.CommandText,
		"status":        string(job.Status),
		"error_count":   job.ErrorCount,
	}
	if runErr != nil {
		payload["error"] = runErr.Error()
	}

	c.events.Dispatch(events.Event{
		Type:   events.EventType(eventType),
		Source: job.Source,
		Payload: events.GenericPayload{
			Type: events.EventType(eventType),
			Data: payload,
		},
	})
}

func syncJobRank(status SyncJobStatus) int {
	switch status {
	case SyncJobRunning:
		return 0
	case SyncJobQueued:
		return 1
	case SyncJobFailed:
		return 2
	case SyncJobCompletedWithErrors:
		return 3
	case SyncJobCompleted:
		return 4
	default:
		return 5
	}
}
