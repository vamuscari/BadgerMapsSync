package server

import (
	"badgermaps/app/action"
	"badgermaps/app/state"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"
)

type CronJob struct {
	Name     string              `yaml:"name"`
	Schedule string              `yaml:"schedule"`
	Action   action.ActionConfig `yaml:"action"`
}

type ActionExecutor interface {
	ExecuteAction(action.ActionConfig) error
}

type ServerManager struct {
	state *state.State
	cron  *cron.Cron
}

func NewServerManager(state *state.State) *ServerManager {
	return &ServerManager{
		state: state,
	}
}

func (sm *ServerManager) Start(cronJobs []CronJob, actionExecutor ActionExecutor) error {
	sm.cron = cron.New()
	for _, job := range cronJobs {
		job := job // capture loop variable for closures
		if _, err := sm.cron.AddFunc(job.Schedule, func() {
			actionExecutor.ExecuteAction(job.Action)
		}); err != nil {
			return fmt.Errorf("failed to schedule cron job '%s': %w", job.Name, err)
		}
	}
	sm.cron.Start()
	return nil
}

// StopCronJobs stops any in-process cron scheduler owned by this manager.
func (sm *ServerManager) StopCronJobs() {
	if sm.cron != nil {
		sm.cron.Stop()
		sm.cron = nil
	}
}

// GetServerStatus checks if the server process is running.
// It returns the PID and a boolean indicating if it's running.
func (sm *ServerManager) GetServerStatus() (int, bool) {
	pidFile := pidFilePath(sm.state)
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, false // PID file doesn't exist
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return 0, false // Invalid PID file content
	}

	return pid, processRunning(pid)
}

// StopServer stops the running server process.
func (sm *ServerManager) StopServer() error {
	sm.StopCronJobs()

	pidFile := pidFilePath(sm.state)
	pid, running := sm.GetServerStatus()
	if !running {
		// If we have a PID but the process isn't running, clean up the stale PID file.
		if pid > 0 {
			os.Remove(pidFile)
		}
		return fmt.Errorf("server is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("could not find process: %w", err)
	}

	// Ask the process to terminate gracefully
	if err := terminateProcess(process); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	// Clean up the PID file
	return os.Remove(pidFile)
}
