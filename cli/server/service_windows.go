//go:build windows
// +build windows

package server

import (
	"badgermaps/app"
	appserver "badgermaps/app/server"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const (
	serviceName        = appserver.WindowsServiceName
	serviceDisplayName = appserver.WindowsServiceDisplayName
	serviceDescription = appserver.WindowsServiceDescription
)

var elog debug.Log

// badgerMapsService is the struct that will implement the svc.Handler interface.
type badgerMapsService struct {
	app       *app.App
	presenter *CliPresenter
}

// Execute is the main entry point for the service.
func (s *badgerMapsService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}

	if s.app == nil || s.presenter == nil {
		elog.Error(1, "Service initialization failed: app context is unavailable")
		changes <- svc.Status{State: svc.StopPending}
		return false, 1
	}

	config := &ServerConfig{
		Host:             s.app.State.ServerHost,
		Port:             s.app.State.ServerPort,
		TLSEnabled:       s.app.State.TLSEnabled,
		TLSCert:          s.app.State.TLSCert,
		TLSKey:           s.app.State.TLSKey,
		WebhookSecret:    s.app.State.ServerWebhookSecret,
		InternalAPIToken: s.app.State.ServerInternalAPIToken,
		LogRequests:      s.app.State.ServerLogRequests,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- s.presenter.RunServerWithContext(ctx, config)
	}()

	elog.Info(1, fmt.Sprintf("Service '%s' started successfully.", serviceName))
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	var runErr error
	serverExited := false

loop:
	for {
		select {
		case err := <-serverDone:
			serverExited = true
			runErr = err
			break loop
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, fmt.Sprintf("Service '%s' stopping.", serviceName))
				cancel()
				break loop
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}

	if !serverExited {
		select {
		case err := <-serverDone:
			runErr = err
		case <-time.After(10 * time.Second):
			runErr = fmt.Errorf("timed out waiting for server shutdown")
		}
	}

	if runErr != nil && !errors.Is(runErr, context.Canceled) {
		elog.Error(1, fmt.Sprintf("Service '%s' stopped with error: %v", serviceName, runErr))
		return false, 1
	}

	return
}

// runService is called by the main function if the program is not running interactively.
func runService(a *app.App) {
	var err error
	elog, err = eventlog.Open(serviceName)
	if err != nil {
		elog = debug.New(serviceName)
	}
	defer elog.Close()

	if a == nil {
		elog.Error(1, fmt.Sprintf("Service '%s' failed to start: app context is nil", serviceName))
		return
	}

	elog.Info(1, fmt.Sprintf("Starting service '%s'.", serviceName))
	handler := &badgerMapsService{
		app:       a,
		presenter: NewCliPresenter(a),
	}
	if err = svc.Run(serviceName, handler); err != nil {
		elog.Error(1, fmt.Sprintf("Service '%s' failed: %v", serviceName, err))
		return
	}
	elog.Info(1, fmt.Sprintf("Service '%s' stopped.", serviceName))
}

// installService registers the program as a Windows service.
func installService(a *app.App, explicitConfigPath string) error {
	exepath, err := os.Executable()
	if err != nil {
		return err
	}

	configPath, err := resolveWindowsServiceConfigPath(a, explicitConfigPath, exepath)
	if err != nil {
		return err
	}
	serviceArgs := []string{"server", "--config", configPath}

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
		DisplayName:      serviceDisplayName,
		Description:      serviceDescription,
		StartType:        mgr.StartAutomatic,
		ServiceStartName: appserver.WindowsServiceAccount,
	}, serviceArgs...) // Subcommand and optional config path passed to executable on service start.
	if err != nil {
		return err
	}
	defer s.Close()

	// Set up event logging
	err = eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		s.Delete()
		return fmt.Errorf("failed to install event log source: %s", err)
	}
	return nil
}

func resolveWindowsServiceConfigPath(a *app.App, explicitConfigPath string, executablePath string) (string, error) {
	configPath := strings.TrimSpace(explicitConfigPath)
	if configPath == "" {
		configPath = filepath.Join(filepath.Dir(executablePath), "config.yaml")
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve service config path %q: %w", configPath, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("service config %q does not exist; create a global config or pass --config", absPath)
		}
		return "", fmt.Errorf("failed to inspect service config %q: %w", absPath, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("service config %q is a directory", absPath)
	}
	f, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("service config %q is not readable: %w", absPath, err)
	}
	f.Close()

	if a != nil {
		a.SetConfigFilePath(absPath)
	}
	return absPath, nil
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

	status, err := s.Query()
	if err == nil && status.State != svc.Stopped {
		if status.State != svc.StopPending {
			if _, stopErr := s.Control(svc.Stop); stopErr != nil {
				return fmt.Errorf("failed to stop service '%s' before uninstall: %w", serviceName, stopErr)
			}
		}
		if waitErr := waitForWindowsServiceState(s, svc.Stopped, 30*time.Second); waitErr != nil {
			return waitErr
		}
	} else if err != nil {
		return fmt.Errorf("failed to query service '%s': %w", serviceName, err)
	}

	if err := s.Delete(); err != nil {
		return err
	}
	err = eventlog.Remove(serviceName)
	if err != nil {
		return fmt.Errorf("failed to remove event log source: %s", err)
	}
	return nil
}

func waitForWindowsServiceState(s *mgr.Service, want svc.State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		status, err := s.Query()
		if err != nil {
			return fmt.Errorf("failed to query service '%s': %w", serviceName, err)
		}
		if status.State == want {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for service '%s' to stop", serviceName)
		}
		time.Sleep(300 * time.Millisecond)
	}
}
