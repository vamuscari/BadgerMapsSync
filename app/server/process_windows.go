//go:build windows

package server

import (
	"os"

	"golang.org/x/sys/windows"
)

const windowsProcessStillActiveExitCode uint32 = 259

func processRunning(pid int) bool {
	if pid <= 0 {
		return false
	}

	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer windows.CloseHandle(handle)

	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false
	}
	return exitCode == windowsProcessStillActiveExitCode
}

func terminateProcess(process *os.Process) error {
	return process.Kill()
}
