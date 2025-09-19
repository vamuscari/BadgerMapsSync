//go:build windows
// +build windows

package server

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// StartServer starts the server as a detached background process on Windows.
func (sm *ServerManager) StartServer() error {
	if _, running := sm.GetServerStatus(); running {
		return fmt.Errorf("server is already running")
	}

	executable, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(executable, "server")
	// This is the key for Windows: create a new console process.
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000, // CREATE_NEW_CONSOLE
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	pid := cmd.Process.Pid
	if err := ioutil.WriteFile(sm.state.PIDFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return cmd.Process.Release()
}
