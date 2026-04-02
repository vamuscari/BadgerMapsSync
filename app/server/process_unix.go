//go:build !windows

package server

import (
	"errors"
	"os"
	"syscall"
)

func processRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil || errors.Is(err, syscall.EPERM)
}

func terminateProcess(process *os.Process) error {
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return process.Kill()
	}
	return nil
}
