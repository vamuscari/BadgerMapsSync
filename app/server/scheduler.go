package server

import (
	"badgermaps/api"
	"badgermaps/app/action"
	"badgermaps/app/audit"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/events"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// SyncType is retained only for hard-fail migration checks on legacy scheduled jobs.
type SyncType string

const (
	SyncTypeNone     SyncType = "none"
	SyncTypeAccounts SyncType = "accounts"
	SyncTypeCheckins SyncType = "checkins"
	SyncTypeRoutes   SyncType = "routes"
	SyncTypeFull     SyncType = "full"
	SyncTypePull     SyncType = "pull"
	SyncTypePush     SyncType = "push"
	SyncTypePullPush SyncType = "pull_push"
)

type ScheduledJob struct {
	ID           string         `yaml:"id" json:"id"`
	Name         string         `yaml:"name" json:"name"`
	Schedule     string         `yaml:"schedule" json:"schedule"`
	Steps        []WorkflowStep `yaml:"steps" json:"steps"`
	Enabled      bool           `yaml:"enabled" json:"enabled"`
	LastRun      *time.Time     `yaml:"last_run,omitempty" json:"last_run,omitempty"`
	NextRun      *time.Time     `yaml:"next_run,omitempty" json:"next_run,omitempty"`
	LastSuccess  *time.Time     `yaml:"last_success,omitempty" json:"last_success,omitempty"`
	LastError    string         `yaml:"last_error,omitempty" json:"last_error,omitempty"`
	RunCount     int            `yaml:"run_count" json:"run_count"`
	ErrorCount   int            `yaml:"error_count" json:"error_count"`
	Timezone     string         `yaml:"timezone,omitempty" json:"timezone,omitempty"`
	RetryOnError bool           `yaml:"retry_on_error" json:"retry_on_error"`
	MaxRetries   int            `yaml:"max_retries" json:"max_retries"`

	// Legacy fields are retained solely for hard-fail migration detection.
	LegacySyncType SyncType              `yaml:"sync_type,omitempty" json:"sync_type,omitempty"`
	LegacyActions  []action.ActionConfig `yaml:"actions,omitempty" json:"actions,omitempty"`
	cronID         cron.EntryID
}

type SyncExecutor interface {
	PullAccounts() error
	PullCheckins() error
	PullRoutes() error
	PullProfile() error
	PushAll() error
	PushAccounts() error
	PushCheckins() error
	PullAccount(int) error
	PullCheckin(int) error
	PullRoute(int) error
}

type Scheduler struct {
	cron             *cron.Cron
	jobs             map[string]*ScheduledJob
	mu               sync.RWMutex
	state            *state.State
	db               database.DB
	api              *api.APIClient
	events           *events.EventDispatcher
	auditLogger      *audit.AuditLogger
	syncExecutor     SyncExecutor
	syncQueue        *SyncJobCoordinator
	actionExec       *action.Executor
	workflowProfiles map[string]WorkflowProfile
	globalTZ         string
	running          bool
	stopChan         chan struct{}
}

var schedulerCronParser = cron.NewParser(
	cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

func parseSchedulerSpec(spec string) (cron.Schedule, error) {
	return schedulerCronParser.Parse(spec)
}

func NewScheduler(
	state *state.State,
	db database.DB,
	api *api.APIClient,
	eventBus *events.EventDispatcher,
	auditLogger *audit.AuditLogger,
	syncExecutor SyncExecutor,
	syncQueue *SyncJobCoordinator,
	workflowProfiles map[string]WorkflowProfile,
	globalTimezone string,
) *Scheduler {
	normalizedProfiles := NormalizeWorkflowProfiles(workflowProfiles)
	if len(normalizedProfiles) == 0 {
		normalizedProfiles = DefaultWorkflowProfiles()
	}

	resolvedGlobalTZ := NormalizeTimezone(globalTimezone)
	resolvedGlobalLoc := time.Local
	if resolvedGlobalTZ != "" {
		loc, err := time.LoadLocation(resolvedGlobalTZ)
		if err != nil {
			resolvedGlobalTZ = ""
			if eventBus != nil {
				eventBus.Dispatch(events.Warningf("scheduler", "Invalid global timezone '%s'; falling back to local time", globalTimezone))
			}
		} else {
			resolvedGlobalLoc = loc
		}
	}

	return &Scheduler{
		cron:             cron.New(cron.WithParser(schedulerCronParser), cron.WithLocation(resolvedGlobalLoc)),
		jobs:             make(map[string]*ScheduledJob),
		state:            state,
		db:               db,
		api:              api,
		events:           eventBus,
		auditLogger:      auditLogger,
		syncExecutor:     syncExecutor,
		syncQueue:        syncQueue,
		actionExec:       action.NewExecutor(db, api),
		workflowProfiles: normalizedProfiles,
		globalTZ:         resolvedGlobalTZ,
		stopChan:         make(chan struct{}),
	}
}

// Start begins the scheduler.
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	if err := s.loadJobs(); err != nil {
		return fmt.Errorf("failed to load scheduled jobs: %w", err)
	}

	s.cron.Start()
	s.running = true
	go s.monitor()

	if s.events != nil {
		s.events.Dispatch(events.Infof("scheduler", "Scheduler started with %d jobs", len(s.jobs)))
	}
	if s.auditLogger != nil {
		s.auditLogger.Log(&audit.AuditEntry{
			OperationType: audit.OpScheduledJob,
			Source:        "Scheduler",
			Action:        "START",
			Success:       true,
			Level:         audit.LevelInfo,
		})
	}
	return nil
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopChan)
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false

	if s.events != nil {
		s.events.Dispatch(events.Infof("scheduler", "Scheduler stopped"))
	}
	if s.auditLogger != nil {
		s.auditLogger.Log(&audit.AuditEntry{
			OperationType: audit.OpScheduledJob,
			Source:        "Scheduler",
			Action:        "STOP",
			Success:       true,
			Level:         audit.LevelInfo,
		})
	}
}

// AddJob adds a new scheduled job.
func (s *Scheduler) AddJob(job *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.prepareJobForWrite(job); err != nil {
		return err
	}
	if job.ID == "" {
		generatedID, err := s.nextAvailableScheduledJobID()
		if err != nil {
			return err
		}
		job.ID = generatedID
	}

	if err := s.attachJobToCron(job); err != nil {
		return err
	}
	previousJob, hadPrevious := s.jobs[job.ID]
	s.jobs[job.ID] = job
	if err := s.saveJobs(); err != nil {
		if job.cronID > 0 {
			s.cron.Remove(job.cronID)
			job.cronID = 0
		}
		if hadPrevious {
			s.jobs[job.ID] = previousJob
		} else {
			delete(s.jobs, job.ID)
		}
		return err
	}

	if s.events != nil {
		s.events.Dispatch(events.Infof("scheduler", "Added scheduled job: %s (%s)", job.Name, job.Schedule))
	}
	return nil
}

// RemoveJob removes a scheduled job.
func (s *Scheduler) RemoveJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	previous := cloneScheduledJob(job)
	if job.cronID > 0 {
		s.cron.Remove(job.cronID)
	}
	delete(s.jobs, jobID)
	if err := s.saveJobs(); err != nil {
		if rollbackErr := s.restoreJobSnapshot(jobID, previous); rollbackErr != nil {
			return fmt.Errorf("failed to remove job: %w (rollback failed: %v)", err, rollbackErr)
		}
		return err
	}

	if s.events != nil {
		s.events.Dispatch(events.Infof("scheduler", "Removed scheduled job: %s", job.Name))
	}
	return nil
}

// UpdateJob updates an existing scheduled job.
func (s *Scheduler) UpdateJob(jobID string, updates *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	previous := cloneScheduledJob(job)

	rollback := func(cause error, removeCurrentCron bool) error {
		if removeCurrentCron && job.cronID > 0 {
			s.cron.Remove(job.cronID)
			job.cronID = 0
		}
		if rollbackErr := s.restoreJobSnapshot(jobID, previous); rollbackErr != nil {
			return fmt.Errorf("%w (rollback failed: %v)", cause, rollbackErr)
		}
		return cause
	}

	if job.cronID > 0 {
		s.cron.Remove(job.cronID)
		job.cronID = 0
	}

	if strings.TrimSpace(updates.Name) != "" {
		job.Name = strings.TrimSpace(updates.Name)
	}
	if strings.TrimSpace(updates.Schedule) != "" {
		job.Schedule = strings.TrimSpace(updates.Schedule)
	}
	if normalizedTimezone := NormalizeTimezone(updates.Timezone); normalizedTimezone != "" {
		job.Timezone = normalizedTimezone
	}
	if updates.Steps != nil {
		job.Steps = updates.Steps
	}
	job.Enabled = updates.Enabled
	job.RetryOnError = updates.RetryOnError
	job.MaxRetries = updates.MaxRetries

	if err := s.prepareJobForWrite(job); err != nil {
		return rollback(err, false)
	}
	if err := s.attachJobToCron(job); err != nil {
		return rollback(err, false)
	}
	if err := s.saveJobs(); err != nil {
		return rollback(err, true)
	}

	if s.events != nil {
		s.events.Dispatch(events.Infof("scheduler", "Updated scheduled job: %s", job.Name))
	}
	return nil
}

// executeJob queues a scheduled job for serialized execution.
func (s *Scheduler) executeJob(jobID string) {
	s.mu.RLock()
	job, exists := s.jobs[jobID]
	s.mu.RUnlock()
	if !exists {
		return
	}

	if s.events != nil {
		s.events.Dispatch(events.Event{
			Type:   "scheduler.job.queued",
			Source: "scheduler",
			Payload: events.GenericPayload{
				Type: "scheduler.job.queued",
				Data: map[string]interface{}{
					"job_id":   job.ID,
					"job_name": job.Name,
					"steps":    len(job.Steps),
				},
			},
		})
	}

	if s.syncQueue == nil {
		go func() {
			_ = s.runScheduledJob(jobID)
		}()
		return
	}

	_, err := s.syncQueue.Submit(SyncJobRequest{
		Name:   job.Name,
		Source: "scheduler",
		Mode:   SyncModeWorkflow,
		Kind:   SyncJobKindWorkflow,
		Run: func(_ context.Context) error {
			return s.runScheduledJob(jobID)
		},
	})
	if err != nil {
		if s.events != nil {
			s.events.Dispatch(events.Errorf("scheduler", "Failed to queue scheduled job %s: %v", job.Name, err))
		}
		s.mu.Lock()
		if target, ok := s.jobs[jobID]; ok {
			target.LastError = fmt.Sprintf("queue failure: %v", err)
			target.ErrorCount++
			_ = s.saveJobs()
		}
		s.mu.Unlock()
	}
}

func (s *Scheduler) runScheduledJob(jobID string) error {
	s.mu.Lock()
	job, exists := s.jobs[jobID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("job not found: %s", jobID)
	}
	startTime := time.Now()
	job.LastRun = &startTime
	job.RunCount++
	previousLastSuccess := cloneTimePtr(job.LastSuccess)
	jobName := job.Name
	jobRetryOnError := job.RetryOnError
	jobMaxRetries := job.MaxRetries
	if jobMaxRetries <= 0 {
		jobMaxRetries = 1
	}
	steps := append([]WorkflowStep(nil), job.Steps...)
	s.mu.Unlock()

	if s.events != nil {
		s.events.Dispatch(events.Event{
			Type:   "scheduler.job.start",
			Source: "scheduler",
			Payload: events.GenericPayload{
				Type: "scheduler.job.start",
				Data: map[string]interface{}{
					"job_id":   jobID,
					"job_name": jobName,
					"steps":    len(steps),
				},
			},
		})
		s.events.Dispatch(events.Infof("scheduler", "Executing scheduled workflow job: %s", jobName))
	}
	if s.auditLogger != nil {
		s.auditLogger.Log(&audit.AuditEntry{
			OperationType: audit.OpScheduledJob,
			Source:        "Scheduler",
			Action:        "EXECUTE",
			Resource:      jobName,
			ResourceID:    jobID,
			Success:       false,
			Level:         audit.LevelInfo,
			Metadata: map[string]interface{}{
				"steps": len(steps),
			},
		})
	}

	var runErr error
	actionFailures := 0
	retries := 0
	for retries < jobMaxRetries {
		actionFailures, runErr = ExecuteWorkflowSteps(WorkflowExecutionOptions{
			Queue:      s.syncQueue,
			Source:     "scheduler",
			ParentMode: SyncModeWorkflow,
			Steps:      steps,
			RunSync: func(mode SyncMode, resourceID int) error {
				return s.executeSyncMode(mode, resourceID)
			},
			RunAction: func(cfg action.ActionConfig, step WorkflowStep) error {
				return s.executeActionStep(cfg, step, "scheduler")
			},
		})
		if runErr == nil {
			break
		}
		retries++
		if jobRetryOnError && retries < jobMaxRetries {
			waitTime := time.Duration(retries*retries) * time.Second
			if s.events != nil {
				s.events.Dispatch(events.Warningf("scheduler", "Job %s failed, retrying in %v (attempt %d/%d)", jobName, waitTime, retries+1, jobMaxRetries))
			}
			time.Sleep(waitTime)
		}
	}

	duration := time.Since(startTime)
	status := SyncJobCompleted
	lastError := ""
	errorDelta := 0
	lastSuccess := previousLastSuccess
	if runErr != nil {
		status = SyncJobFailed
		lastError = runErr.Error()
		errorDelta = 1
	} else if actionFailures > 0 {
		status = SyncJobCompletedWithErrors
		lastError = fmt.Sprintf("workflow completed with %d action step error(s)", actionFailures)
		errorDelta = actionFailures
		now := time.Now()
		lastSuccess = &now
	} else {
		now := time.Now()
		lastSuccess = &now
	}

	s.mu.Lock()
	if target, ok := s.jobs[jobID]; ok {
		target.LastError = lastError
		target.ErrorCount += errorDelta
		target.LastSuccess = lastSuccess
		if target.Enabled {
			switch {
			case target.cronID > 0:
				entry := s.cron.Entry(target.cronID)
				if !entry.Next.IsZero() {
					nextRun := entry.Next
					target.NextRun = &nextRun
				} else if estimatedNext, err := s.estimateNextRun(target); err == nil {
					target.NextRun = estimatedNext
				}
			default:
				if estimatedNext, err := s.estimateNextRun(target); err == nil {
					target.NextRun = estimatedNext
				}
			}
		}
		_ = s.saveJobs()
	}
	s.mu.Unlock()

	if s.events != nil {
		eventType := "scheduler.job.complete"
		if runErr != nil {
			eventType = "scheduler.job.error"
		}
		s.events.Dispatch(events.Event{
			Type:   events.EventType(eventType),
			Source: "scheduler",
			Payload: events.GenericPayload{
				Type: events.EventType(eventType),
				Data: map[string]interface{}{
					"job_id":      jobID,
					"job_name":    jobName,
					"status":      string(status),
					"error_count": errorDelta,
					"error":       lastError,
				},
			},
		})
		if runErr != nil {
			s.events.Dispatch(events.Errorf("scheduler", "Scheduled job failed: %s - %v", jobName, runErr))
		} else if status == SyncJobCompletedWithErrors {
			s.events.Dispatch(events.Warningf("scheduler", "Scheduled workflow completed with errors: %s (duration: %v)", jobName, duration))
		} else {
			s.events.Dispatch(events.Infof("scheduler", "✓ Scheduled workflow completed: %s (duration: %v)", jobName, duration))
		}
	}

	if s.auditLogger != nil {
		s.auditLogger.LogSync("workflow", 0, status == SyncJobCompleted, duration, runErr)
	}

	if runErr != nil {
		return runErr
	}
	return nil
}

func (s *Scheduler) executeActionStep(cfg action.ActionConfig, step WorkflowStep, source string) error {
	executor := s.actionExec
	if executor == nil {
		executor = action.NewExecutor(s.db, s.api)
	}
	actionInstance, err := action.NewActionFromConfig(cfg)
	if err != nil {
		return err
	}
	if err := actionInstance.Validate(); err != nil {
		return err
	}

	stepName := step.EffectiveName()
	execCtx := &action.ExecutionContext{
		EventType: "workflow.step.action",
		Source:    source,
		Payload: map[string]interface{}{
			"step_id":   step.ID,
			"step_name": stepName,
		},
	}
	return actionInstance.Execute(executor.WithContext(execCtx))
}

// executeSyncMode executes the specific sync operation.
func (s *Scheduler) executeSyncMode(mode SyncMode, resourceID int) error {
	if s.syncExecutor == nil {
		return fmt.Errorf("sync executor not configured")
	}

	switch mode {
	case SyncModeNone:
		return nil
	case SyncModePullAccounts:
		return s.syncExecutor.PullAccounts()
	case SyncModePullCheckins:
		return s.syncExecutor.PullCheckins()
	case SyncModePullRoutes:
		return s.syncExecutor.PullRoutes()
	case SyncModePullProfile:
		return s.syncExecutor.PullProfile()
	case SyncModePull:
		if err := s.syncExecutor.PullAccounts(); err != nil {
			return fmt.Errorf("failed to pull accounts: %w", err)
		}
		if err := s.syncExecutor.PullCheckins(); err != nil {
			return fmt.Errorf("failed to pull checkins: %w", err)
		}
		if err := s.syncExecutor.PullRoutes(); err != nil {
			return fmt.Errorf("failed to pull routes: %w", err)
		}
		if err := s.syncExecutor.PullProfile(); err != nil {
			return fmt.Errorf("failed to pull profile: %w", err)
		}
		return nil
	case SyncModePush:
		return s.syncExecutor.PushAll()
	case SyncModePushAccounts:
		return s.syncExecutor.PushAccounts()
	case SyncModePushCheckins:
		return s.syncExecutor.PushCheckins()
	case SyncModePullPush:
		if err := s.executeSyncMode(SyncModePull, 0); err != nil {
			return err
		}
		return s.executeSyncMode(SyncModePush, 0)
	case SyncModePullAccount:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		return s.syncExecutor.PullAccount(resourceID)
	case SyncModePullCheckin:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		return s.syncExecutor.PullCheckin(resourceID)
	case SyncModePullRoute:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		return s.syncExecutor.PullRoute(resourceID)
	default:
		return fmt.Errorf("unsupported sync mode: %s", mode)
	}
}

// GetJobs returns all scheduled jobs.
func (s *Scheduler) GetJobs() []*ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetJob returns a specific job.
func (s *Scheduler) GetJob(jobID string) (*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return job, nil
}

// RunJobNow executes a job immediately.
func (s *Scheduler) RunJobNow(jobID string) error {
	s.mu.RLock()
	_, exists := s.jobs[jobID]
	s.mu.RUnlock()
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}
	go s.executeJob(jobID)
	return nil
}

// QueueStoredJobNow loads persisted jobs when needed and queues a specific job immediately.
func (s *Scheduler) QueueStoredJobNow(jobID string) error {
	trimmedID := strings.TrimSpace(jobID)
	if trimmedID == "" {
		return fmt.Errorf("job id is required")
	}

	s.mu.Lock()
	if len(s.jobs) == 0 {
		if err := s.loadJobs(); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to load scheduled jobs: %w", err)
		}
	}
	_, exists := s.jobs[trimmedID]
	s.mu.Unlock()

	if !exists {
		return fmt.Errorf("job not found: %s", trimmedID)
	}
	return s.RunJobNow(trimmedID)
}

// monitor handles background monitoring tasks.
func (s *Scheduler) monitor() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.updateNextRunTimes()
		}
	}
}

// updateNextRunTimes updates the next run times for all jobs.
func (s *Scheduler) updateNextRunTimes() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, job := range s.jobs {
		if job.Enabled && job.cronID > 0 {
			entry := s.cron.Entry(job.cronID)
			if entry.ID > 0 {
				nextRun := entry.Next
				job.NextRun = &nextRun
			}
		}
	}
}

func (s *Scheduler) scheduleSpecAndNextRun(job *ScheduledJob) (string, *time.Time, error) {
	spec, location, _, err := s.effectiveScheduleSpec(job)
	if err != nil {
		return "", nil, err
	}

	nextRun, err := nextRunForSpec(spec, location)
	if err != nil {
		return "", nil, err
	}
	return spec, &nextRun, nil
}

func (s *Scheduler) estimateNextRun(job *ScheduledJob) (*time.Time, error) {
	_, nextRun, err := s.scheduleSpecAndNextRun(job)
	if err != nil {
		return nil, err
	}
	return nextRun, nil
}

func (s *Scheduler) effectiveScheduleSpec(job *ScheduledJob) (string, *time.Location, string, error) {
	if job == nil {
		return "", nil, "", fmt.Errorf("job is required")
	}

	schedule := strings.TrimSpace(job.Schedule)
	if schedule == "" {
		return "", nil, "", fmt.Errorf("job schedule is required")
	}

	timezone, source, location, err := ResolveEffectiveTimezone(job.Timezone, s.globalTZ)
	if err != nil {
		return "", nil, "", err
	}

	spec := schedule
	if source == TimezoneSourceOverride {
		spec = fmt.Sprintf("CRON_TZ=%s %s", timezone, schedule)
	}
	return spec, location, source, nil
}

func nextRunForSpec(spec string, location *time.Location) (time.Time, error) {
	schedule, err := parseSchedulerSpec(spec)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid cron expression: %w", err)
	}
	if location == nil {
		location = time.Local
	}
	return schedule.Next(time.Now().In(location)), nil
}

func (s *Scheduler) prepareJobForWrite(job *ScheduledJob) error {
	if job == nil {
		return fmt.Errorf("job is required")
	}
	job.Name = strings.TrimSpace(job.Name)
	job.Schedule = strings.TrimSpace(job.Schedule)
	job.Timezone = NormalizeTimezone(job.Timezone)
	if job.MaxRetries <= 0 {
		job.MaxRetries = 1
	}

	if err := ValidateScheduledJobDefinition(job, s.workflowProfiles); err != nil {
		return err
	}
	if err := TestCronExpression(job.Schedule); err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}
	if err := ValidateTimezone(job.Timezone); err != nil {
		return err
	}
	return nil
}

func (s *Scheduler) attachJobToCron(job *ScheduledJob) error {
	job.cronID = 0
	if job.Enabled {
		spec, nextRun, err := s.scheduleSpecAndNextRun(job)
		if err != nil {
			return err
		}
		entryID, err := s.cron.AddFunc(spec, func(jobID string) func() {
			return func() {
				s.executeJob(jobID)
			}
		}(job.ID))
		if err != nil {
			return fmt.Errorf("failed to add job to cron: %w", err)
		}
		job.cronID = entryID
		job.NextRun = nextRun
		return nil
	}

	nextRun, err := s.estimateNextRun(job)
	if err != nil {
		return err
	}
	job.NextRun = nextRun
	return nil
}

// loadJobs loads scheduled jobs from configuration.
func (s *Scheduler) loadJobs() error {
	configDir := schedulerConfigDir(s.state)
	jobsFile := filepath.Join(configDir, "scheduled_jobs.json")

	if _, err := os.Stat(jobsFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(jobsFile)
	if err != nil {
		return fmt.Errorf("failed to read jobs file: %w", err)
	}

	var loadedJobs map[string]*ScheduledJob
	if err := json.Unmarshal(data, &loadedJobs); err != nil {
		return fmt.Errorf("failed to unmarshal jobs: %w", err)
	}

	ids := make([]string, 0, len(loadedJobs))
	for id := range loadedJobs {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		job := loadedJobs[id]
		if job == nil {
			continue
		}
		job.ID = id
		if err := s.prepareJobForWrite(job); err != nil {
			return fmt.Errorf("scheduled job %q is invalid: %w", id, err)
		}
		if err := s.attachJobToCron(job); err != nil {
			return fmt.Errorf("failed loading scheduled job %q: %w", id, err)
		}
		s.jobs[id] = job
	}

	if s.events != nil {
		s.events.Dispatch(events.Infof("scheduler", "Loaded %d scheduled jobs", len(s.jobs)))
	}
	return nil
}

// saveJobs saves scheduled jobs to configuration.
func (s *Scheduler) saveJobs() error {
	configDir := schedulerConfigDir(s.state)
	jobsFile := filepath.Join(configDir, "scheduled_jobs.json")

	jobsToSave := make(map[string]*ScheduledJob, len(s.jobs))
	for id, job := range s.jobs {
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

func cloneScheduledJob(job *ScheduledJob) *ScheduledJob {
	if job == nil {
		return nil
	}
	clone := *job
	clone.Steps = append([]WorkflowStep(nil), job.Steps...)
	clone.LegacyActions = append([]action.ActionConfig(nil), job.LegacyActions...)
	clone.LastRun = cloneTimePtr(job.LastRun)
	clone.NextRun = cloneTimePtr(job.NextRun)
	clone.LastSuccess = cloneTimePtr(job.LastSuccess)
	return &clone
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (s *Scheduler) restoreJobSnapshot(jobID string, snapshot *ScheduledJob) error {
	if snapshot == nil {
		delete(s.jobs, jobID)
		return nil
	}
	restored := cloneScheduledJob(snapshot)
	s.jobs[jobID] = restored
	return s.attachJobToCron(restored)
}

func (s *Scheduler) nextAvailableScheduledJobID() (string, error) {
	for attempt := 0; attempt < 128; attempt++ {
		candidate := GenerateScheduledJobID()
		if _, exists := s.jobs[candidate]; !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique scheduled job id")
}

// TestCronExpression tests if a cron expression is valid.
func TestCronExpression(expression string) error {
	_, err := parseSchedulerSpec(expression)
	return err
}

// GetNextRunTime calculates the next run time for a cron expression.
func GetNextRunTime(expression string) (*time.Time, error) {
	schedule, err := parseSchedulerSpec(expression)
	if err != nil {
		return nil, err
	}

	next := schedule.Next(time.Now())
	return &next, nil
}
