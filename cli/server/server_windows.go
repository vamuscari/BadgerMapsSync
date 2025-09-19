//go:build windows
// +build windows

package server

import (
	"badgermaps/app"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"golang.org/x/sys/windows/svc"
)

func init() {
	// This function is called when the package is initialized.
	// We add the Windows-specific commands here.
	ServerCmdFunc = func(a *app.App, serverCmd *cobra.Command) {
		serverCmd.AddCommand(newServerInstallCmd(a))
		serverCmd.AddCommand(newServerUninstallCmd(a))
	}

	// This is our hook to check if we are running as a service
	IsWindowsService = func() bool {
		isInteractive, err := svc.IsAnInteractiveSession()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to determine if running in an interactive session: %v\n", err)
			os.Exit(1)
		}
		return !isInteractive
	}

	// The function to run the service
	RunWindowsService = runService
}

func newServerInstallCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "install",
		Short: "Install the server as a Windows service",
		Run: func(cmd *cobra.Command, args []string) {
			if err := installService(); err != nil {
				a.Events.Dispatch(events.Errorf("server", "Failed to install service: %v", err))
				os.Exit(1)
			}
			a.Events.Dispatch(events.Infof("server", "Service '%s' installed successfully.", serviceName))
		},
	}
}

func newServerUninstallCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall the Windows service",
		Run: func(cmd *cobra.Command, args []string) {
			if err := uninstallService(); err != nil {
				a.Events.Dispatch(events.Errorf("server", "Failed to uninstall service: %v", err))
				os.Exit(1)
			}
			a.Events.Dispatch(events.Infof("server", "Service '%s' uninstalled successfully.", serviceName))
		},
	}
}
