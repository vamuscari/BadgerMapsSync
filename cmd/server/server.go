package server

import (
	"badgermaps/api"
	"badgermaps/app"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ServerCmd creates a new server command
func ServerCmd(App *app.App) *cobra.Command {
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run in server mode",
		Long:  `Run the BadgerMaps CLI in server mode, listening for incoming webhooks.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			App.VerifySetupOrExit(cmd)
		},
		Run: func(cmd *cobra.Command, args []string) {
			serverConfig := ServerConfig{
				Host:         App.State.ServerHost,
				Port:         App.State.ServerPort,
				WebhookToken: App.State.WebhookToken,
			}
			runServer(App, &serverConfig)
		},
	}
	return serverCmd
}

func runServer(App *app.App, config *ServerConfig) {
	s := &server{App: App}
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/account/create", s.handleAccountCreateWebhook)
	mux.HandleFunc("/webhook/checkin", s.handleCheckinWebhook)
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	server := &http.Server{Addr: addr, Handler: mux}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		fmt.Printf("Starting server on %s\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-stop
	fmt.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	fmt.Println("Server stopped")
}

type server struct {
	App *app.App
}

func (s *server) handleAccountCreateWebhook(w http.ResponseWriter, r *http.Request) {
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

	logFunc := func(msg string) {
		log.Println(msg)
	}

	// NOTE: Using StoreAccountDetailed as StoreAccountBasic was removed during refactor.
	// The underlying SQL procedure should handle the missing fields gracefully.
	if err := app.StoreAccountDetailed(s.App, &acc, logFunc); err != nil {
		http.Error(w, "failed to store account", http.StatusInternalServerError)
		return
	}
	fmt.Println(color.CyanString("Received and processed account webhook for account: %s", acc.FullName))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Account webhook processed")
}

func (s *server) handleCheckinWebhook(w http.ResponseWriter, r *http.Request) {
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

	logFunc := func(msg string) {
		log.Println(msg)
	}

	if err := app.StoreCheckin(s.App, checkin, logFunc); err != nil {
		http.Error(w, "failed to store checkin", http.StatusInternalServerError)
		return
	}
	fmt.Println(color.CyanString("Received and processed checkin webhook for checkin: %d", checkin.CheckinId))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Checkin webhook processed")
}
