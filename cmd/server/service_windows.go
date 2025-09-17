//go:build windows
// +build windows

package server

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	serviceName        = "BadgerMapsSync"
	serviceDisplayName = "BadgerMaps Sync Service"
	serviceDescription = "Handles webhooks and background tasks for BadgerMaps synchronization."
)

var elog debug.Log

// badgerMapsService is the struct that will implement the svc.Handler interface.
type badgerMapsService struct{}

// Execute is the main entry point for the service.
func (s *badgerMapsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	// This is where we would start our actual server logic.
	// For this example, we'll just run a ticker.
	// In the real implementation, this would start the runServer() http server.
	elog.Info(1, fmt.Sprintf("Service '%s' started successfully.", serviceName))
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, fmt.Sprintf("Service '%s' stopping.", serviceName))
				break loop
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return
}

// runService is called by the main function if the program is not running interactively.
func runService() {
	var err error
	elog, err = eventlog.Open(serviceName)
	if err != nil {
		return
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("Starting service '%s'.", serviceName))
	if err = svc.Run(serviceName, &badgerMapsService{}); err != nil {
		elog.Error(1, fmt.Sprintf("Service '%s' failed: %v", serviceName, err))
		return
	}
	elog.Info(1, fmt.Sprintf("Service '%s' stopped.", serviceName))
}

// installService registers the program as a Windows service.
func installService() error {
	exepath, err := os.Executable()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if err == nil {
		s.Close()
		return fmt.Errorf("service '%s' already exists", serviceName)
	}
	s, err = m.CreateService(serviceName, exepath, mgr.Config{
		DisplayName: serviceDisplayName,
		Description: serviceDescription,
		StartType:   mgr.StartAutomatic,
	}, "server") // The last argument "server" is passed to the executable on start
	if err != nil {
		return err
	}
	defer s.Close()

	// Set up event logging
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("failed to install event log source: %s", err)
	}
	return nil
}

// uninstallService removes the Windows service registration.
func uninstallService() error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service '%s' is not installed", serviceName)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(serviceName)
	if err != nil {
		return fmt.Errorf("failed to remove event log source: %s", err)
	}
	return nil
}
