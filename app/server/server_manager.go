package server

import (
	"badgermaps/app/state"
	"badgermaps/cli/action"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"

	"github.com/robfig/cron/v3"
)

type CronJob struct {
	Name     string                `yaml:"name"`
	Schedule string                `yaml:"schedule"`
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

func (sm *ServerManager) Start(cronJobs []CronJob, actionExecutor ActionExecutor) {
	sm.cron = cron.New()
	for _, job := range cronJobs {
		sm.cron.AddFunc(job.Schedule, func() {
			actionExecutor.ExecuteAction(job.Action)
		})
	}
	sm.cron.Start()
}

// GetServerStatus checks if the server process is running.
// It returns the PID and a boolean indicating if it's running.
func (sm *ServerManager) GetServerStatus() (int, bool) {
	pidData, err := ioutil.ReadFile(sm.state.PIDFile)
	if err != nil {
		return 0, false // PID file doesn't exist
	}

	pid, err := strconv.Atoi(string(pidData))
	if err != nil {
		return 0, false // Invalid PID file content
	}

	// Check if the process actually exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return pid, false // Process not found
	}

	// On UNIX-like systems, sending signal 0 is a standard way to check for existence
	err = process.Signal(syscall.Signal(0))
	return pid, err == nil
}

// StopServer stops the running server process.
func (sm *ServerManager) StopServer() error {
	if sm.cron != nil {
		sm.cron.Stop()
	}
	pid, running := sm.GetServerStatus()
	if !running {
		// If we have a PID but the process isn't running, clean up the stale PID file.
		if pid > 0 {
			os.Remove(sm.state.PIDFile)
		}
		return fmt.Errorf("server is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("could not find process: %w", err)
	}

	// Ask the process to terminate gracefully
	if err := process.Signal(syscall.SIGTERM); err != nil {
		// If graceful shutdown fails, force kill it
		if err := process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
	}

	// Clean up the PID file
	return os.Remove(sm.state.PIDFile)
}
