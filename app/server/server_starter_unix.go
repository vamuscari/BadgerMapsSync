//go:build linux || darwin
// +build linux darwin

package server

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
