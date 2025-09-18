package server

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/pull"
	"badgermaps/app/server"
	"badgermaps/events"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var (
	App             *app.App
	ServerCmdFunc   func(a *app.App, serverCmd *cobra.Command)
	IsWindowsService func() bool
	RunWindowsService func()
)

// ServerCmd creates the parent 'server' command
func ServerCmd(a *app.App) *cobra.Command {
	App = a // Store app instance for service handler
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Manage the BadgerMaps webhook server",
		Long:  `Start, stop, and configure the webhook server. When run without subcommands, it starts the server in the foreground.`,
		Run: func(cmd *cobra.Command, args []string) {
			if IsWindowsService() {
				RunWindowsService()
				return
			}

			// This code runs if 'badgermaps server' is executed directly in a terminal.
			serverConfig := ServerConfig{
				Host:         App.State.ServerHost,
				Port:         App.State.ServerPort,
				TLSEnabled:   App.State.TLSEnabled,
				TLSCert:      App.State.TLSCert,
				TLSKey:       App.State.TLSKey,
			}
			runServer(App, &serverConfig)
		},
	}

	// Add platform-specific commands (like install/uninstall on Windows)
	ServerCmdFunc(a, serverCmd)

	// Add platform-agnostic commands
	serverCmd.AddCommand(newServerStartCmd(a))
	serverCmd.AddCommand(newServerStopCmd(a))
	serverCmd.AddCommand(newServerStatusCmd(a))
	serverCmd.AddCommand(newServerSetupCmd(a))

	return serverCmd
}

func newServerStartCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the webhook server in the background",
		Run: func(cmd *cobra.Command, args []string) {
			if err := server.StartServer(a); err != nil {
				a.Events.Dispatch(events.Errorf("server", "Failed to start server: %v", err))
				os.Exit(1)
			}
			pid, _ := server.GetServerStatus(a)
			a.Events.Dispatch(events.Infof("server", "Server started successfully with PID %d.", pid))
		},
	}
}

func newServerStopCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the webhook server",
		Run: func(cmd *cobra.Command, args []string) {
			if err := server.StopServer(a); err != nil {
				a.Events.Dispatch(events.Errorf("server", "Failed to stop server: %v", err))
				os.Exit(1)
			}
			a.Events.Dispatch(events.Infof("server", "Server stopped successfully."))
		},
	}
}

func newServerStatusCmd(a *app.App) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check the status of the webhook server",
		Run: func(cmd *cobra.Command, args []string) {
			if pid, running := server.GetServerStatus(a); running {
				a.Events.Dispatch(events.Infof("server", "Server is running with PID %d.", pid))
			} else {
				a.Events.Dispatch(events.Warningf("server", "Server is not running."))
			}
		},
	}
}

func runServer(App *app.App, config *ServerConfig) {
	s := &httpServer{App: App}
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/account/create", s.handleAccountCreateWebhook)
	mux.HandleFunc("/webhook/checkin", s.handleCheckinWebhook)
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	server := &http.Server{Addr: addr, Handler: mux}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		App.Events.Dispatch(events.Infof("server", "Starting server on %s", addr))
		var err error
		if config.TLSEnabled {
			App.Events.Dispatch(events.Infof("server", "TLS is enabled. Starting HTTPS server."))
			err = server.ListenAndServeTLS(config.TLSCert, config.TLSKey)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			App.Events.Dispatch(events.Errorf("server", "Server error: %v", err))
		}
	}()

	<-stop
	App.Events.Dispatch(events.Infof("server", "Shutting down server..."))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		App.Events.Dispatch(events.Errorf("server", "Server shutdown error: %v", err))
	}
	App.Events.Dispatch(events.Infof("server", "Server stopped"))
}

type httpServer struct {
	App *app.App
}

func (s *httpServer) handleAccountCreateWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusInternalServerError)
		return
	}
	var acc api.Account
	if err := json.Unmarshal(body, &acc); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := pull.StoreAccountDetailed(s.App, &acc); err != nil {
		http.Error(w, "failed to store account", http.StatusInternalServerError)
		return
	}
	s.App.Events.Dispatch(events.Infof("server", "Received and processed account webhook for account: %s", acc.FullName.String))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Account webhook processed")
}

func (s *httpServer) handleCheckinWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Body == nil {
		http.Error(w, "Please send a request body", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusInternalServerError)
		return
	}
	var checkin api.Checkin
	if err := json.Unmarshal(body, &checkin); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := pull.StoreCheckin(s.App, checkin); err != nil {
		http.Error(w, "failed to store checkin", http.StatusInternalServerError)
		return
	}
	s.App.Events.Dispatch(events.Infof("server", "Received and processed checkin webhook for checkin: %d", checkin.CheckinId.Int64))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Checkin webhook processed")
}
