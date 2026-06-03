//go:build windows
// +build windows

package server

import (
	"errors"
	"fmt"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceControlTimeout = 30 * time.Second

const (
	windowsServiceStatusAccess = windows.SERVICE_QUERY_STATUS
	windowsServiceStartAccess  = windows.SERVICE_QUERY_STATUS | windows.SERVICE_START
	windowsServiceStopAccess   = windows.SERVICE_QUERY_STATUS | windows.SERVICE_STOP
)

// StartServer starts the installed Windows service.
func (sm *ServerManager) StartServer() error {
	status := sm.GetDetailedStatus()
	if !status.Installed {
		return fmt.Errorf("Windows service %q is not installed; run 'badgermaps server install --config <global-config-path>' from an elevated shell", WindowsServiceName)
	}
	if status.Running {
		return fmt.Errorf("server is already running")
	}

	s, err := openWindowsService(windowsServiceStartAccess)
	if err != nil {
		return err
	}
	defer s.Close()

	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start Windows service %q: %w", WindowsServiceName, err)
	}
	return waitForServiceState(s, svc.Running, serviceControlTimeout)
}

func (sm *ServerManager) platformStatus() ServerStatus {
	s, err := openWindowsService(windowsServiceStatusAccess)
	if err != nil {
		installed := true
		state := ServerStatusUnknown
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			installed = false
			state = ServerStatusStopped
		}
		return ServerStatus{
			RuntimeMode: ServerRuntimeModeService,
			Installed:   installed,
			State:       state,
			Message:     err.Error(),
		}
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return ServerStatus{
			RuntimeMode: ServerRuntimeModeService,
			Installed:   true,
			State:       ServerStatusUnknown,
			Message:     err.Error(),
		}
	}
	return windowsServiceStatus(status)
}

func (sm *ServerManager) platformStopServer() error {
	s, err := openWindowsService(windowsServiceStopAccess)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return fmt.Errorf("Windows service %q is not installed", WindowsServiceName)
		}
		return err
	}
	defer s.Close()

	status, err := s.Query()
	if err != nil {
		return fmt.Errorf("failed to query Windows service %q: %w", WindowsServiceName, err)
	}
	if status.State == svc.Stopped {
		return fmt.Errorf("server is not running")
	}
	if status.State != svc.StopPending {
		if _, err := s.Control(svc.Stop); err != nil {
			return fmt.Errorf("failed to stop Windows service %q: %w", WindowsServiceName, err)
		}
	}
	return waitForServiceState(s, svc.Stopped, serviceControlTimeout)
}

func (sm *ServerManager) platformRestartServer() error {
	status := sm.GetDetailedStatus()
	if !status.Installed {
		return fmt.Errorf("Windows service %q is not installed; run 'badgermaps server install --config <global-config-path>' from an elevated shell", WindowsServiceName)
	}
	if status.Running || status.State == ServerStatusPending {
		if err := sm.platformStopServer(); err != nil {
			return err
		}
	}
	return sm.StartServer()
}

func openWindowsService(access uint32) (*mgr.Service, error) {
	if access == 0 {
		access = windowsServiceStatusAccess
	}

	scm, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Windows service manager: %w", err)
	}
	defer windows.CloseServiceHandle(scm)

	name, err := windows.UTF16PtrFromString(WindowsServiceName)
	if err != nil {
		return nil, fmt.Errorf("failed to encode Windows service name %q: %w", WindowsServiceName, err)
	}
	handle, err := windows.OpenService(scm, name, access)
	if err != nil {
		return nil, err
	}
	return &mgr.Service{Name: WindowsServiceName, Handle: handle}, nil
}

func waitForServiceState(s *mgr.Service, want svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := s.Query()
		if err != nil {
			return fmt.Errorf("failed to query Windows service %q: %w", WindowsServiceName, err)
		}
		if status.State == want {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for Windows service %q to reach %s", WindowsServiceName, serviceStateLabel(want))
		}
		time.Sleep(300 * time.Millisecond)
	}
}

func windowsServiceStatus(status svc.Status) ServerStatus {
	state := serviceStateLabel(status.State)
	running := status.State == svc.Running
	return ServerStatus{
		RuntimeMode: ServerRuntimeModeService,
		Installed:   true,
		Running:     running,
		PID:         int(status.ProcessId),
		State:       state,
	}
}

func serviceStateLabel(state svc.State) string {
	switch state {
	case svc.Stopped:
		return ServerStatusStopped
	case svc.StartPending, svc.StopPending, svc.ContinuePending, svc.PausePending:
		return ServerStatusPending
	case svc.Running:
		return ServerStatusRunning
	default:
		return ServerStatusUnknown
	}
}
