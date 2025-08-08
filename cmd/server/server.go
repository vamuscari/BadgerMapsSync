package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewServerCmd creates a new server command
func NewServerCmd() *cobra.Command {
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
				fmt.Printf("Server will run on schedule: %s\n", schedule)
				// In a real implementation, we would set up a cron job here
				// For now, just run the server normally
			}

			runServer(host, port, tlsEnabled)
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

	return serverCmd
}

// runServer starts the HTTP server
func runServer(host string, port int, tlsEnabled bool) {
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

	// Set up HTTP server
	mux := http.NewServeMux()

	// Register webhook handlers
	mux.HandleFunc("/webhook/account", handleAccountWebhook)
	mux.HandleFunc("/webhook/checkin", handleCheckinWebhook)
	mux.HandleFunc("/webhook/route", handleRouteWebhook)
	mux.HandleFunc("/webhook/profile", handleProfileWebhook)

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
			// In a real implementation, we would load TLS certificates
			// For now, just print a message
			fmt.Println("TLS is enabled but not implemented in this version")
			err = server.ListenAndServe() // Fallback to HTTP
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

// handleAccountWebhook handles account update webhooks
func handleAccountWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// In a real implementation, we would:
	// 1. Validate the webhook signature
	// 2. Parse the JSON payload
	// 3. Process the account update
	// 4. Store it in the database

	fmt.Println(color.CyanString("Received account webhook"))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Account webhook processed")
}

// handleCheckinWebhook handles checkin update webhooks
func handleCheckinWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println(color.CyanString("Received checkin webhook"))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Checkin webhook processed")
}

// handleRouteWebhook handles route update webhooks
func handleRouteWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println(color.CyanString("Received route webhook"))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Route webhook processed")
}

// handleProfileWebhook handles profile update webhooks
func handleProfileWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println(color.CyanString("Received profile webhook"))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Profile webhook processed")
}

// handleHealthCheck handles health check requests
func handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}
