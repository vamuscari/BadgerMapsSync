package server

import (
	"badgermaps/app/state"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"syscall"
)

type ServerManager struct {
	state *state.State
}

func NewServerManager(state *state.State) *ServerManager {
	return &ServerManager{state: state}
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
