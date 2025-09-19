//go:build linux || darwin
// +build linux darwin

package server

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
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

	cmd := exec.Command(executable, "server")
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
	if err := ioutil.WriteFile(sm.state.PIDFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		// Try to kill the process we just started if we can't save the PID
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	// Disown the process
	return cmd.Process.Release()
}
