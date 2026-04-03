package server

import (
	"badgermaps/api"
	"badgermaps/app/action"
	"badgermaps/app/audit"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/events"
	"context"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

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
	ID           string                `yaml:"id" json:"id"`
	Name         string                `yaml:"name" json:"name"`
	Schedule     string                `yaml:"schedule" json:"schedule"`
	SyncType     SyncType              `yaml:"sync_type" json:"sync_type"`
	Enabled      bool                  `yaml:"enabled" json:"enabled"`
	LastRun      *time.Time            `yaml:"last_run,omitempty" json:"last_run,omitempty"`
	NextRun      *time.Time            `yaml:"next_run,omitempty" json:"next_run,omitempty"`
	LastSuccess  *time.Time            `yaml:"last_success,omitempty" json:"last_success,omitempty"`
	LastError    string                `yaml:"last_error,omitempty" json:"last_error,omitempty"`
	RunCount     int                   `yaml:"run_count" json:"run_count"`
	ErrorCount   int                   `yaml:"error_count" json:"error_count"`
	Actions      []action.ActionConfig `yaml:"actions,omitempty" json:"actions,omitempty"`
	Timezone     string                `yaml:"timezone,omitempty" json:"timezone,omitempty"`
	RetryOnError bool                  `yaml:"retry_on_error" json:"retry_on_error"`
	MaxRetries   int                   `yaml:"max_retries" json:"max_retries"`
	cronID       cron.EntryID
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
	cron         *cron.Cron
	jobs         map[string]*ScheduledJob
	mu           sync.RWMutex
	state        *state.State
	db           database.DB
	api          *api.APIClient
	events       *events.EventDispatcher
	auditLogger  *audit.AuditLogger
	syncExecutor SyncExecutor
	syncQueue    *SyncJobCoordinator
	actionExec   *action.Executor
	legacyCron   []CronJob
	globalTZ     string
	running      bool
	stopChan     chan struct{}
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
	legacyCronJobs []CronJob,
	globalTimezone string,
) *Scheduler {
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
		cron:         cron.New(cron.WithParser(schedulerCronParser), cron.WithLocation(resolvedGlobalLoc)),
		jobs:         make(map[string]*ScheduledJob),
		state:        state,
		db:           db,
		api:          api,
		events:       eventBus,
		auditLogger:  auditLogger,
		syncExecutor: syncExecutor,
		syncQueue:    syncQueue,
		actionExec:   action.NewExecutor(db, api),
		legacyCron:   append([]CronJob(nil), legacyCronJobs...),
		globalTZ:     resolvedGlobalTZ,
		stopChan:     make(chan struct{}),
	}
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler already running")
	}

	// Load jobs from configuration
	if err := s.loadJobs(); err != nil {
		return fmt.Errorf("failed to load scheduled jobs: %w", err)
	}

	if err := s.importLegacyCronJobs(); err != nil {
		return fmt.Errorf("failed to import legacy cron jobs: %w", err)
	}

	// Start cron scheduler
	s.cron.Start()
	s.running = true

	// Start monitoring goroutine
	go s.monitor()

	s.events.Dispatch(events.Infof("scheduler", "Scheduler started with %d jobs", len(s.jobs)))

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

// Stop halts the scheduler
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

	s.events.Dispatch(events.Infof("scheduler", "Scheduler stopped"))

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

// AddJob adds a new scheduled job
func (s *Scheduler) AddJob(job *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if job.ID == "" {
		job.ID = fmt.Sprintf("job_%d", time.Now().UnixNano())
	}

	// Add to cron if enabled
	if job.Enabled {
		spec, nextRun, err := s.scheduleSpecAndNextRun(job)
		if err != nil {
			return err
		}
		entryID, err := s.cron.AddFunc(spec, func() {
			s.executeJob(job.ID)
		})
		if err != nil {
			return fmt.Errorf("failed to add job to cron: %w", err)
		}
		job.cronID = entryID
		job.NextRun = nextRun
	} else {
		nextRun, err := s.estimateNextRun(job)
		if err != nil {
			return err
		}
		job.NextRun = nextRun
	}

	s.jobs[job.ID] = job
	s.saveJobs()

	s.events.Dispatch(events.Infof("scheduler", "Added scheduled job: %s (%s)", job.Name, job.Schedule))

	return nil
}

// RemoveJob removes a scheduled job
func (s *Scheduler) RemoveJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Remove from cron if scheduled
	if job.cronID > 0 {
		s.cron.Remove(job.cronID)
	}

	delete(s.jobs, jobID)
	s.saveJobs()

	s.events.Dispatch(events.Infof("scheduler", "Removed scheduled job: %s", job.Name))

	return nil
}

// UpdateJob updates an existing scheduled job
func (s *Scheduler) UpdateJob(jobID string, updates *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Remove from cron if currently scheduled
	if job.cronID > 0 {
		s.cron.Remove(job.cronID)
		job.cronID = 0
	}

	// Update fields
	if updates.Name != "" {
		job.Name = updates.Name
	}
	if updates.Schedule != "" {
		job.Schedule = updates.Schedule
	}
	if updates.SyncType != "" {
		job.SyncType = updates.SyncType
	}
	if normalizedTimezone := NormalizeTimezone(updates.Timezone); normalizedTimezone != "" {
		job.Timezone = normalizedTimezone
	}
	job.Enabled = updates.Enabled
	job.RetryOnError = updates.RetryOnError
	job.MaxRetries = updates.MaxRetries

	if updates.Actions != nil {
		job.Actions = updates.Actions
	}

	// Re-add to cron if enabled
	if job.Enabled {
		spec, nextRun, err := s.scheduleSpecAndNextRun(job)
		if err != nil {
			return err
		}

		entryID, err := s.cron.AddFunc(spec, func() {
			s.executeJob(jobID)
		})
		if err != nil {
			return fmt.Errorf("failed to update job in cron: %w", err)
		}
		job.cronID = entryID
		job.NextRun = nextRun
	} else {
		nextRun, err := s.estimateNextRun(job)
		if err != nil {
			return err
		}
		job.NextRun = nextRun
	}

	s.saveJobs()

	s.events.Dispatch(events.Infof("scheduler", "Updated scheduled job: %s", job.Name))

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

	s.events.Dispatch(events.Event{
		Type:   "scheduler.job.queued",
		Source: "scheduler",
		Payload: events.GenericPayload{
			Type: "scheduler.job.queued",
			Data: map[string]interface{}{
				"job_id":    job.ID,
				"job_name":  job.Name,
				"sync_type": string(job.SyncType),
			},
		},
	})

	if s.syncQueue == nil {
		go s.runScheduledJob(jobID)
		return
	}

	_, err := s.syncQueue.Submit(SyncJobRequest{
		Name:   job.Name,
		Source: "scheduler",
		Mode:   s.syncModeForJob(job.SyncType),
		Run: func(_ context.Context) error {
			return s.runScheduledJob(jobID)
		},
	})
	if err != nil {
		s.events.Dispatch(events.Errorf("scheduler", "Failed to queue scheduled job %s: %v", job.Name, err))
		s.mu.Lock()
		if target, ok := s.jobs[jobID]; ok {
			target.LastError = fmt.Sprintf("queue failure: %v", err)
			target.ErrorCount++
			s.saveJobs()
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
	jobName := job.Name
	syncType := job.SyncType
	retryOnError := job.RetryOnError
	maxRetries := job.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}
	s.mu.Unlock()

	s.events.Dispatch(events.Event{
		Type:   "scheduler.job.start",
		Source: "scheduler",
		Payload: events.GenericPayload{
			Type: "scheduler.job.start",
			Data: map[string]interface{}{
				"job_id":    jobID,
				"job_name":  jobName,
				"sync_type": string(syncType),
			},
		},
	})
	s.events.Dispatch(events.Infof("scheduler", "Executing scheduled job: %s", jobName))

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
				"sync_type": string(syncType),
			},
		})
	}

	var runErr error
	retries := 0
	for retries < maxRetries {
		runErr = s.executeSyncType(syncType)
		if runErr == nil {
			break
		}
		retries++
		if retryOnError && retries < maxRetries {
			waitTime := time.Duration(retries*retries) * time.Second
			s.events.Dispatch(events.Warningf("scheduler", "Job %s failed, retrying in %v (attempt %d/%d)",
				jobName, waitTime, retries+1, maxRetries))
			time.Sleep(waitTime)
		}
	}

	duration := time.Since(startTime)
	s.mu.Lock()
	if target, ok := s.jobs[jobID]; ok {
		if runErr != nil {
			target.LastError = runErr.Error()
			target.ErrorCount++
		} else {
			now := time.Now()
			target.LastSuccess = &now
			target.LastError = ""
		}

		if target.Enabled && target.cronID > 0 {
			entry := s.cron.Entry(target.cronID)
			nextRun := entry.Next
			target.NextRun = &nextRun
		}
		s.saveJobs()
	}
	s.mu.Unlock()

	if runErr != nil {
		s.events.Dispatch(events.Event{
			Type:   "scheduler.job.error",
			Source: "scheduler",
			Payload: events.GenericPayload{
				Type: "scheduler.job.error",
				Data: map[string]interface{}{
					"job_id":    jobID,
					"job_name":  jobName,
					"sync_type": string(syncType),
					"error":     runErr.Error(),
				},
			},
		})
		s.events.Dispatch(events.Errorf("scheduler", "Scheduled job failed: %s - %v", jobName, runErr))
		if s.auditLogger != nil {
			s.auditLogger.LogSync(string(syncType), 0, false, duration, runErr)
		}
		return runErr
	}

	s.events.Dispatch(events.Event{
		Type:   "scheduler.job.complete",
		Source: "scheduler",
		Payload: events.GenericPayload{
			Type: "scheduler.job.complete",
			Data: map[string]interface{}{
				"job_id":    jobID,
				"job_name":  jobName,
				"sync_type": string(syncType),
			},
		},
	})
	s.events.Dispatch(events.Infof("scheduler", "✓ Scheduled job completed: %s (duration: %v)", jobName, duration))
	if s.auditLogger != nil {
		s.auditLogger.LogSync(string(syncType), 0, true, duration, nil)
	}

	if len(job.Actions) > 0 {
		for _, actionConfig := range job.Actions {
			if !actionConfig.IsEnabled() {
				continue
			}
			action, err := action.NewActionFromConfig(actionConfig)
			if err != nil {
				s.events.Dispatch(events.Errorf("scheduler", "Failed to create action: %v", err))
				continue
			}
			if err := action.Execute(s.actionExec); err != nil {
				s.events.Dispatch(events.Errorf("scheduler", "Failed to execute action: %v", err))
			}
		}
	}

	return nil
}

// executeSyncType executes the specific sync operation
func (s *Scheduler) executeSyncType(syncType SyncType) error {
	if s.syncExecutor == nil {
		return fmt.Errorf("sync executor not configured")
	}

	switch syncType {
	case SyncTypeNone:
		return nil

	case SyncTypeAccounts:
		return s.syncExecutor.PullAccounts()

	case SyncTypeCheckins:
		return s.syncExecutor.PullCheckins()

	case SyncTypeRoutes:
		return s.syncExecutor.PullRoutes()

	case SyncTypeFull, SyncTypePull:
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

	case SyncTypePush:
		return s.syncExecutor.PushAll()

	case SyncTypePullPush:
		if err := s.executeSyncType(SyncTypePull); err != nil {
			return err
		}
		return s.executeSyncType(SyncTypePush)

	default:
		return fmt.Errorf("unknown sync type: %s", syncType)
	}
}

// GetJobs returns all scheduled jobs
func (s *Scheduler) GetJobs() []*ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*ScheduledJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}
	return jobs
}

// GetJob returns a specific job
func (s *Scheduler) GetJob(jobID string) (*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// RunJobNow executes a job immediately
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

// monitor handles background monitoring tasks
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

// updateNextRunTimes updates the next run times for all jobs
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

func (s *Scheduler) syncModeForJob(syncType SyncType) SyncMode {
	switch syncType {
	case SyncTypePull, SyncTypeFull:
		return SyncModePull
	case SyncTypePush:
		return SyncModePush
	case SyncTypePullPush:
		return SyncModePullPush
	case SyncTypeAccounts:
		return SyncModePullAccounts
	case SyncTypeCheckins:
		return SyncModePullCheckins
	case SyncTypeRoutes:
		return SyncModePullRoutes
	case SyncTypeNone:
		return SyncModeNone
	default:
		return SyncModeNone
	}
}

func (s *Scheduler) importLegacyCronJobs() error {
	if len(s.legacyCron) == 0 {
		return nil
	}

	imported := 0
	for _, legacy := range s.legacyCron {
		legacyID := legacySchedulerJobID(legacy)
		if _, exists := s.jobs[legacyID]; exists {
			continue
		}

		legacyJob := &ScheduledJob{
			ID:       legacyID,
			Name:     fmt.Sprintf("legacy:%s", legacy.Name),
			Schedule: legacy.Schedule,
			SyncType: SyncTypeNone,
			Enabled:  true,
			Actions:  []action.ActionConfig{legacy.Action},
		}

		if err := s.addLoadedJob(legacyJob); err != nil {
			return fmt.Errorf("failed importing legacy job %s: %w", legacy.Name, err)
		}
		imported++
	}

	if imported > 0 {
		if err := s.saveJobs(); err != nil {
			return fmt.Errorf("failed saving imported legacy jobs: %w", err)
		}
		s.events.Dispatch(events.Infof("scheduler", "Imported %d legacy cron job(s)", imported))
	}
	return nil
}

func legacySchedulerJobID(job CronJob) string {
	payload, err := json.Marshal(job)
	if err != nil {
		payload = []byte(fmt.Sprintf("%s|%s|%v", job.Name, job.Schedule, job.Action))
	}
	sum := sha1.Sum(payload)
	return fmt.Sprintf("legacy_%x", sum[:8])
}

func (s *Scheduler) addLoadedJob(job *ScheduledJob) error {
	if job.ID == "" {
		job.ID = fmt.Sprintf("job_%d", time.Now().UnixNano())
	}
	job.Timezone = NormalizeTimezone(job.Timezone)
	job.cronID = 0
	s.jobs[job.ID] = job

	spec, nextRun, err := s.scheduleSpecAndNextRun(job)
	if err != nil {
		if job.Timezone != "" {
			invalidTimezone := job.Timezone
			job.Timezone = ""
			spec, nextRun, err = s.scheduleSpecAndNextRun(job)
			if err == nil {
				if s.events != nil {
					s.events.Dispatch(events.Warningf(
						"scheduler",
						"Invalid timezone '%s' for job '%s'; falling back to global/local timezone",
						invalidTimezone,
						job.Name,
					))
				}
			}
		}
	}
	if err != nil {
		return fmt.Errorf("invalid schedule for job %s: %w", job.Name, err)
	}
	job.NextRun = nextRun

	if !job.Enabled {
		return nil
	}

	entryID, err := s.cron.AddFunc(spec, func(jobID string) func() {
		return func() {
			s.executeJob(jobID)
		}
	}(job.ID))
	if err != nil {
		return fmt.Errorf("failed to schedule job %s: %w", job.Name, err)
	}
	job.cronID = entryID

	return nil
}

// loadJobs loads scheduled jobs from configuration
func (s *Scheduler) loadJobs() error {
	configDir := schedulerConfigDir(s.state)
	jobsFile := filepath.Join(configDir, "scheduled_jobs.json")

	// Check if file exists
	if _, err := os.Stat(jobsFile); os.IsNotExist(err) {
		// No jobs file yet, that's ok
		return nil
	}

	// Read jobs file
	data, err := os.ReadFile(jobsFile)
	if err != nil {
		return fmt.Errorf("failed to read jobs file: %w", err)
	}

	// Unmarshal jobs
	var loadedJobs map[string]*ScheduledJob
	if err := json.Unmarshal(data, &loadedJobs); err != nil {
		return fmt.Errorf("failed to unmarshal jobs: %w", err)
	}

	for id, job := range loadedJobs {
		job.ID = id
		if err := s.addLoadedJob(job); err != nil {
			s.events.Dispatch(events.Errorf("scheduler", "Failed to load job %s: %v", job.Name, err))
		}
	}

	s.events.Dispatch(events.Infof("scheduler", "Loaded %d scheduled jobs", len(loadedJobs)))
	return nil
}

// saveJobs saves scheduled jobs to configuration
func (s *Scheduler) saveJobs() error {
	configDir := schedulerConfigDir(s.state)
	jobsFile := filepath.Join(configDir, "scheduled_jobs.json")

	// Create a copy of jobs without runtime fields
	jobsToSave := make(map[string]*ScheduledJob)
	for id, job := range s.jobs {
		// Create a copy without runtime-specific fields
		jobCopy := *job
		jobCopy.cronID = 0 // Don't save internal cron ID
		jobsToSave[id] = &jobCopy
	}

	// Marshal jobs
	data, err := json.MarshalIndent(jobsToSave, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create jobs directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(jobsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	return nil
}

// TestCronExpression tests if a cron expression is valid
func TestCronExpression(expression string) error {
	_, err := parseSchedulerSpec(expression)
	return err
}

// GetNextRunTime calculates the next run time for a cron expression
func GetNextRunTime(expression string) (*time.Time, error) {
	schedule, err := parseSchedulerSpec(expression)
	if err != nil {
		return nil, err
	}

	next := schedule.Next(time.Now())
	return &next, nil
}
