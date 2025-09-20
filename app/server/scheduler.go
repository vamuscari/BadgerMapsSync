package server

import (
	"badgermaps/api"
	"badgermaps/app/action"
	"badgermaps/app/audit"
	"badgermaps/app/state"
	"badgermaps/database"
	"badgermaps/events"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type SyncType string

const (
	SyncTypeAccounts SyncType = "accounts"
	SyncTypeCheckins SyncType = "checkins"
	SyncTypeRoutes   SyncType = "routes"
	SyncTypeFull     SyncType = "full"
	SyncTypePush     SyncType = "push"
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
	PushAll() error
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
	actionExec   *action.Executor
	running      bool
	stopChan     chan struct{}
}

func NewScheduler(
	state *state.State,
	db database.DB,
	api *api.APIClient,
	events *events.EventDispatcher,
	auditLogger *audit.AuditLogger,
	syncExecutor SyncExecutor,
) *Scheduler {
	// Create cron with seconds field support
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

	return &Scheduler{
		cron:         cron.New(cron.WithParser(parser), cron.WithLocation(time.Local)),
		jobs:         make(map[string]*ScheduledJob),
		state:        state,
		db:           db,
		api:          api,
		events:       events,
		auditLogger:  auditLogger,
		syncExecutor: syncExecutor,
		actionExec:   action.NewExecutor(db, api),
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

	// Validate cron expression
	schedule, err := cron.ParseStandard(job.Schedule)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// Calculate next run time with timezone support
	var nextRun time.Time
	if job.Timezone != "" {
		loc, err := time.LoadLocation(job.Timezone)
		if err != nil {
			return fmt.Errorf("invalid timezone: %w", err)
		}
		// Calculate next run in the job's timezone
		nextRun = schedule.Next(time.Now().In(loc))
	} else {
		nextRun = schedule.Next(time.Now())
	}
	job.NextRun = &nextRun

	// Add to cron if enabled
	if job.Enabled {
		entryID, err := s.cron.AddFunc(job.Schedule, func() {
			s.executeJob(job.ID)
		})
		if err != nil {
			return fmt.Errorf("failed to add job to cron: %w", err)
		}
		job.cronID = entryID
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
	job.Enabled = updates.Enabled
	job.RetryOnError = updates.RetryOnError
	job.MaxRetries = updates.MaxRetries

	if updates.Actions != nil {
		job.Actions = updates.Actions
	}

	// Re-add to cron if enabled
	if job.Enabled {
		schedule, err := cron.ParseStandard(job.Schedule)
		if err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}

		// Calculate next run time with timezone support
		var nextRun time.Time
		if job.Timezone != "" {
			loc, err := time.LoadLocation(job.Timezone)
			if err != nil {
				return fmt.Errorf("invalid timezone: %w", err)
			}
			nextRun = schedule.Next(time.Now().In(loc))
		} else {
			nextRun = schedule.Next(time.Now())
		}
		job.NextRun = &nextRun

		entryID, err := s.cron.AddFunc(job.Schedule, func() {
			s.executeJob(jobID)
		})
		if err != nil {
			return fmt.Errorf("failed to update job in cron: %w", err)
		}
		job.cronID = entryID
	}

	s.saveJobs()

	s.events.Dispatch(events.Infof("scheduler", "Updated scheduled job: %s", job.Name))

	return nil
}

// executeJob executes a scheduled job
func (s *Scheduler) executeJob(jobID string) {
	s.mu.RLock()
	job, exists := s.jobs[jobID]
	s.mu.RUnlock()

	if !exists {
		return
	}

	startTime := time.Now()
	job.LastRun = &startTime
	job.RunCount++

	s.events.Dispatch(events.Infof("scheduler", "Executing scheduled job: %s", job.Name))

	if s.auditLogger != nil {
		s.auditLogger.Log(&audit.AuditEntry{
			OperationType: audit.OpScheduledJob,
			Source:        "Scheduler",
			Action:        "EXECUTE",
			Resource:      job.Name,
			ResourceID:    job.ID,
			Success:       false, // Will update after execution
			Level:         audit.LevelInfo,
			Metadata: map[string]interface{}{
				"sync_type": string(job.SyncType),
			},
		})
	}

	// Execute based on sync type
	var err error
	retries := 0
	maxRetries := job.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 1
	}

	for retries < maxRetries {
		err = s.executeSyncType(job)

		if err == nil {
			break
		}

		retries++
		if job.RetryOnError && retries < maxRetries {
			waitTime := time.Duration(retries*retries) * time.Second // Exponential backoff
			s.events.Dispatch(events.Warningf("scheduler", "Job %s failed, retrying in %v (attempt %d/%d)",
				job.Name, waitTime, retries+1, maxRetries))
			time.Sleep(waitTime)
		}
	}

	// Update job status
	duration := time.Since(startTime)

	s.mu.Lock()
	if err != nil {
		job.LastError = err.Error()
		job.ErrorCount++
		s.events.Dispatch(events.Errorf("scheduler", "Scheduled job failed: %s - %v", job.Name, err))

		if s.auditLogger != nil {
			s.auditLogger.LogSync(string(job.SyncType), 0, false, duration, err)
		}
	} else {
		now := time.Now()
		job.LastSuccess = &now
		job.LastError = ""
		s.events.Dispatch(events.Infof("scheduler", "✓ Scheduled job completed: %s (duration: %v)", job.Name, duration))

		if s.auditLogger != nil {
			s.auditLogger.LogSync(string(job.SyncType), 0, true, duration, nil)
		}
	}

	// Calculate next run time
	if job.Enabled && job.cronID > 0 {
		entry := s.cron.Entry(job.cronID)
		nextRun := entry.Next
		job.NextRun = &nextRun
	}

	s.saveJobs()
	s.mu.Unlock()

	// Execute post-sync actions
	if err == nil && len(job.Actions) > 0 {
		for _, actionConfig := range job.Actions {
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
}

// executeSyncType executes the specific sync operation
func (s *Scheduler) executeSyncType(job *ScheduledJob) error {
	if s.syncExecutor == nil {
		return fmt.Errorf("sync executor not configured")
	}

	switch job.SyncType {
	case SyncTypeAccounts:
		return s.syncExecutor.PullAccounts()

	case SyncTypeCheckins:
		return s.syncExecutor.PullCheckins()

	case SyncTypeRoutes:
		return s.syncExecutor.PullRoutes()

	case SyncTypeFull:
		if err := s.syncExecutor.PullAccounts(); err != nil {
			return fmt.Errorf("failed to pull accounts: %w", err)
		}
		if err := s.syncExecutor.PullCheckins(); err != nil {
			return fmt.Errorf("failed to pull checkins: %w", err)
		}
		if err := s.syncExecutor.PullRoutes(); err != nil {
			return fmt.Errorf("failed to pull routes: %w", err)
		}
		return nil

	case SyncTypePush:
		return s.syncExecutor.PushAll()

	default:
		return fmt.Errorf("unknown sync type: %s", job.SyncType)
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

	// Execute in background
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

// loadJobs loads scheduled jobs from configuration
func (s *Scheduler) loadJobs() error {
	// Get jobs file path from state
	configDir := "."
	if s.state.ConfigFile != nil && *s.state.ConfigFile != "" {
		configDir = filepath.Dir(*s.state.ConfigFile)
	}
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

	// Add each job to scheduler
	for id, job := range loadedJobs {
		job.ID = id // Ensure ID is set
		// Reset runtime fields
		job.cronID = 0

		// Add job without saving (to avoid rewriting file)
		s.jobs[id] = job

		// Schedule if enabled
		if job.Enabled {
			schedule, err := cron.ParseStandard(job.Schedule)
			if err != nil {
				s.events.Dispatch(events.Errorf("scheduler", "Failed to parse schedule for job %s: %v", job.Name, err))
				continue
			}

			// Calculate next run time
			var nextRun time.Time
			if job.Timezone != "" {
				loc, err := time.LoadLocation(job.Timezone)
				if err == nil {
					nextRun = schedule.Next(time.Now().In(loc))
				} else {
					nextRun = schedule.Next(time.Now())
				}
			} else {
				nextRun = schedule.Next(time.Now())
			}
			job.NextRun = &nextRun

			// Add to cron
			entryID, err := s.cron.AddFunc(job.Schedule, func(jobID string) func() {
				return func() {
					s.executeJob(jobID)
				}
			}(job.ID))

			if err != nil {
				s.events.Dispatch(events.Errorf("scheduler", "Failed to schedule job %s: %v", job.Name, err))
				continue
			}
			job.cronID = entryID
		}
	}

	s.events.Dispatch(events.Infof("scheduler", "Loaded %d scheduled jobs", len(loadedJobs)))
	return nil
}

// saveJobs saves scheduled jobs to configuration
func (s *Scheduler) saveJobs() error {
	// Get jobs file path from state
	configDir := "."
	if s.state.ConfigFile != nil && *s.state.ConfigFile != "" {
		configDir = filepath.Dir(*s.state.ConfigFile)
	}
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

	// Write to file
	if err := os.WriteFile(jobsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	return nil
}

// TestCronExpression tests if a cron expression is valid
func TestCronExpression(expression string) error {
	_, err := cron.ParseStandard(expression)
	return err
}

// GetNextRunTime calculates the next run time for a cron expression
func GetNextRunTime(expression string) (*time.Time, error) {
	schedule, err := cron.ParseStandard(expression)
	if err != nil {
		return nil, err
	}

	next := schedule.Next(time.Now())
	return &next, nil
}
