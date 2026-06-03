//go:build linux || darwin
// +build linux darwin

package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// StartServer starts the server as a detached background process.
func (sm *ServerManager) StartServer() error {
	if _, running := sm.GetServerStatus(); running {
		return fmt.Errorf("server is already running")
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(executable, serverCommandArgs(sm.state)...)
	// This is the key for Unix: creating a new session detaches the child
	// from the parent, so it won't be killed when the parent exits.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// Write the PID to the file
	pid := cmd.Process.Pid
	pidFile := pidFilePath(sm.state)
	if err := os.MkdirAll(filepath.Dir(pidFile), 0755); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to create PID directory: %w", err)
	}
	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		// Try to kill the process we just started if we can't save the PID
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Disown the process
	return cmd.Process.Release()
}

func (sm *ServerManager) platformStatus() ServerStatus {
	pidFile := pidFilePath(sm.state)
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return ServerStatus{
			RuntimeMode: ServerRuntimeModeProcess,
			State:       ServerStatusStopped,
		}
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return ServerStatus{
			RuntimeMode: ServerRuntimeModeProcess,
			State:       ServerStatusStopped,
			Message:     "invalid PID file",
		}
	}

	running := processRunning(pid)
	state := ServerStatusStopped
	if running {
		state = ServerStatusRunning
	}
	return ServerStatus{
		RuntimeMode: ServerRuntimeModeProcess,
		Installed:   true,
		Running:     running,
		PID:         pid,
		State:       state,
	}
}

func (sm *ServerManager) platformStopServer() error {
	pidFile := pidFilePath(sm.state)
	pid, running := sm.GetServerStatus()
	if !running {
		if pid > 0 {
			os.Remove(pidFile)
		}
		return fmt.Errorf("server is not running")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("could not find process: %w", err)
	}

	if err := terminateProcess(process); err != nil {
		return fmt.Errorf("failed to terminate process: %w", err)
	}

	return os.Remove(pidFile)
}

func (sm *ServerManager) platformRestartServer() error {
	if _, running := sm.GetServerStatus(); running {
		if err := sm.platformStopServer(); err != nil {
			return err
		}
	}
	return sm.StartServer()
}
