package server

import (
	"badgermaps/app"
	"badgermaps/events"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	App               *app.App
	ServerCmdFunc     func(a *app.App, serverCmd *cobra.Command)
	IsWindowsService  func() bool
	RunWindowsService func()
)

// ServerCmd creates the parent 'server' command
func ServerCmd(a *app.App) *cobra.Command {
	App = a // Store app instance for service handler
	presenter := NewCliPresenter(a)

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Manage the BadgerMaps webhook server",
		Long:  `Start, stop, and configure the webhook server. When run without subcommands, it starts the server in the foreground.`,
		Run: func(cmd *cobra.Command, args []string) {
			if IsWindowsService() {
				RunWindowsService()
				return
			}

			serverConfig := ServerConfig{
				Host:       App.State.ServerHost,
				Port:       App.State.ServerPort,
				TLSEnabled: App.State.TLSEnabled,
				TLSCert:    App.State.TLSCert,
				TLSKey:     App.State.TLSKey,
			}
			presenter.RunServer(&serverConfig)
		},
	}

	// Add platform-specific commands (like install/uninstall on Windows)
	ServerCmdFunc(a, serverCmd)

	// Add platform-agnostic commands
	serverCmd.AddCommand(newServerStartCmd(presenter))
	serverCmd.AddCommand(newServerStopCmd(presenter))
	serverCmd.AddCommand(newServerStatusCmd(presenter))
	serverCmd.AddCommand(newServerSetupCmd(a))
	serverCmd.AddCommand(newServerReplayWebhookCmd(presenter))

	return serverCmd
}

func newServerReplayWebhookCmd(presenter *CliPresenter) *cobra.Command {
	return &cobra.Command{
		Use:   "replay-webhook [id]",
		Short: "Replay a webhook from the log",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			id, err := strconv.Atoi(args[0])
			if err != nil {
				presenter.App.Events.Dispatch(events.Errorf("server", "Invalid webhook ID: %v", err))
				return
			}
			presenter.HandleReplayWebhook(id)
		},
	}
}

func newServerStartCmd(presenter *CliPresenter) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the webhook server in the background",
		Run: func(cmd *cobra.Command, args []string) {
			presenter.HandleServerStart()
		},
	}
}

func newServerStopCmd(presenter *CliPresenter) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the webhook server",
		Run: func(cmd *cobra.Command, args []string) {
			presenter.HandleServerStop()
		},
	}
}

func newServerStatusCmd(presenter *CliPresenter) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check the status of the webhook server",
		Run: func(cmd *cobra.Command, args []string) {
			presenter.HandleServerStatus()
		},
	}
}
