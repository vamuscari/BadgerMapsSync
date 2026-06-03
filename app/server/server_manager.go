package server

import (
	"badgermaps/app/action"
	"badgermaps/app/state"
	"fmt"

	"github.com/robfig/cron/v3"
)

const (
	ServerRuntimeModeProcess = "process"
	ServerRuntimeModeService = "service"

	ServerStatusUnknown = "unknown"
	ServerStatusStopped = "stopped"
	ServerStatusPending = "pending"
	ServerStatusRunning = "running"
)

type ServerStatus struct {
	RuntimeMode string
	Installed   bool
	Running     bool
	PID         int
	State       string
	Message     string
}

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

func (sm *ServerManager) GetDetailedStatus() ServerStatus {
	if sm == nil {
		return ServerStatus{State: ServerStatusUnknown}
	}
	return sm.platformStatus()
}

// GetServerStatus checks if the server process is running.
// It returns the PID and a boolean indicating if it's running.
func (sm *ServerManager) GetServerStatus() (int, bool) {
	status := sm.GetDetailedStatus()
	return status.PID, status.Running
}

// StopServer stops the running server process.
func (sm *ServerManager) StopServer() error {
	sm.StopCronJobs()
	return sm.platformStopServer()
}

// RestartServer restarts the server if it is running, or starts it when stopped.
func (sm *ServerManager) RestartServer() error {
	if sm == nil {
		return fmt.Errorf("server manager is unavailable")
	}
	sm.StopCronJobs()
	return sm.platformRestartServer()
}
