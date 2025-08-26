package server

import (
	"badgermapscli/api"
	"badgermapscli/app"
	"badgermapscli/cmd/pull"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/robfig/cron/v3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ServerCmd creates a new server command
func ServerCmd(config *app.State) *cobra.Command {

	config.VerifySetupOrExit()

	var (
		host       string
		port       int
		tlsEnabled bool
		schedule   string
	)

	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run in server mode",
		Long:  `Run the BadgerMaps CLI in server mode, listening for incoming webhooks.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If schedule is provided, set up scheduled execution
			if schedule != "" {
				c := cron.New()
				_, err := c.AddFunc(schedule, func() {
					fmt.Println("Running scheduled job: pull all")
					pull.PullAllAccounts(config, 0)
					// pull.PullAllCheckins(config)
					pull.PullAllRoutes(config)
				})
				if err != nil {
					log.Fatalf("Error scheduling job: %v", err)
				}
				go c.Start()
				fmt.Printf("Scheduled job added: pull all on schedule: %s\n", schedule)
			}

			runServer(config, host, port, tlsEnabled)
		},
	}

	// Add flags
	serverCmd.Flags().StringVar(&host, "host", "", "Host address to listen on (default is from env or 0.0.0.0)")
	serverCmd.Flags().IntVar(&port, "port", 0, "Port to listen on (default is from env or 8080)")
	serverCmd.Flags().BoolVar(&tlsEnabled, "tls", false, "Enable TLS/HTTPS (default is from env or false)")
	serverCmd.Flags().StringVar(&schedule, "schedule", "", "Run on schedule using cron syntax (e.g., \"0 */6 * * *\" for every 6 hours)")

	// Bind flags to viper
	viper.BindPFlag("SERVER_HOST", serverCmd.Flags().Lookup("host"))
	viper.BindPFlag("SERVER_PORT", serverCmd.Flags().Lookup("port"))
	viper.BindPFlag("SERVER_TLS_ENABLED", serverCmd.Flags().Lookup("tls"))

	// serverCmd.AddCommand(newServerSetupCmd(config))

	return serverCmd
}

// runServer starts the HTTP server
func runServer(App *app.State, host string, port int, tlsEnabled bool) {
	// Get configuration from viper if not provided via flags
	if host == "" {
		host = viper.GetString("SERVER_HOST")
		if host == "" {
			host = "0.0.0.0" // Default
		}
	}

	if port == 0 {
		port = viper.GetInt("SERVER_PORT")
		if port == 0 {
			port = 8080 // Default
		}
	}

	if !tlsEnabled {
		tlsEnabled = viper.GetBool("SERVER_TLS_ENABLED")
	}

	s := &server{App: App}

	// Set up HTTP server
	mux := http.NewServeMux()

	// Register webhook handlers
	mux.HandleFunc("/webhook/account", s.handleAccountWebhook)
	mux.HandleFunc("/webhook/checkin", s.handleCheckinWebhook)
	mux.HandleFunc("/webhook/route", s.handleRouteWebhook)
	mux.HandleFunc("/webhook/profile", s.handleProfileWebhook)

	// Add health check endpoint
	mux.HandleFunc("/health", handleHealthCheck)

	// Create server
	addr := fmt.Sprintf("%s:%d", host, port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Set up graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		fmt.Printf("Starting server on %s (TLS: %v)\n", addr, tlsEnabled)
		var err error
		if tlsEnabled {
			certFile := viper.GetString("SERVER_TLS_CERT")
			keyFile := viper.GetString("SERVER_TLS_KEY")
			if certFile == "" || keyFile == "" {
				log.Fatal("TLS is enabled, but certificate and key files are not specified. Please use the 'server setup' command to configure them.")
			}
			err = server.ListenAndServeTLS(certFile, keyFile)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	fmt.Println("Shutting down server...")

	// Give the server 5 seconds to finish processing requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}

	fmt.Println("Server stopped")
}

type server struct {
	App *app.State
}

// handleAccountWebhook handles account update webhooks
func (s *server) handleAccountWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
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

	fmt.Println(color.CyanString("Received and processed account webhook for account: %s", acc.FullName))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Account webhook processed")
}

// handleCheckinWebhook handles checkin update webhooks
func (s *server) handleCheckinWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
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

	fmt.Println(color.CyanString("Received and processed checkin webhook for checkin ID: %d", checkin.ID))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Checkin webhook processed")
}

// handleRouteWebhook handles route update webhooks
func (s *server) handleRouteWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusInternalServerError)
		return
	}

	var route api.Route
	if err := json.Unmarshal(body, &route); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := pull.StoreRoute(s.App, route); err != nil {
		http.Error(w, "failed to store route", http.StatusInternalServerError)
		return
	}

	fmt.Println(color.CyanString("Received and processed route webhook for route ID: %d", route.ID))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Route webhook processed")
}

// handleProfileWebhook handles profile update webhooks
func (s *server) handleProfileWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "can't read body", http.StatusInternalServerError)
		return
	}

	var profile api.UserProfile
	if err := json.Unmarshal(body, &profile); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := pull.StoreProfile(s.App, &profile); err != nil {
		http.Error(w, "failed to store profile", http.StatusInternalServerError)
		return
	}

	fmt.Println(color.CyanString("Received and processed profile webhook for profile ID: %d", profile.ID))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Profile webhook processed")
}

// handleHealthCheck handles health check requests
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}
