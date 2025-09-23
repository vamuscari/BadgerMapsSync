package server

import (
	"badgermaps/api"
	"badgermaps/app"
	"badgermaps/app/pull"
	"badgermaps/database"
	"badgermaps/events"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// CliPresenter handles the presentation logic for the server command.
type CliPresenter struct {
	App *app.App
}

// NewCliPresenter creates a new presenter for the server command.
func NewCliPresenter(a *app.App) *CliPresenter {
	return &CliPresenter{App: a}
}

// HandleServerStart starts the server in the background.
func (p *CliPresenter) HandleServerStart() {
	if err := p.App.Server.StartServer(); err != nil {
		p.App.Events.Dispatch(events.Errorf("server", "Failed to start server: %v", err))
		os.Exit(1)
	}
	pid, _ := p.App.Server.GetServerStatus()
	p.App.Events.Dispatch(events.Infof("server", "Server started successfully with PID %d.", pid))
}

// HandleServerStop stops the running server.
func (p *CliPresenter) HandleServerStop() {
	if err := p.App.Server.StopServer(); err != nil {
		p.App.Events.Dispatch(events.Errorf("server", "Failed to stop server: %v", err))
		os.Exit(1)
	}
	p.App.Events.Dispatch(events.Infof("server", "Server stopped successfully."))
}

// HandleServerStatus checks and prints the server's status.
func (p *CliPresenter) HandleServerStatus() {
	if pid, running := p.App.Server.GetServerStatus(); running {
		p.App.Events.Dispatch(events.Infof("server", "Server is running with PID %d.", pid))
	} else {
		p.App.Events.Dispatch(events.Warningf("server", "Server is not running."))
	}
}

// RunServer runs the server in the foreground.
func (p *CliPresenter) RunServer(config *ServerConfig) {
	p.App.Server.Start(p.App.Config.CronJobs, p.App)
	mux := http.NewServeMux()

	accountCreateHandler := http.HandlerFunc(p.HandleAccountCreateWebhook)
	checkinHandler := http.HandlerFunc(p.HandleCheckinWebhook)

	enabledWebhooks := p.App.Config.Server.Webhooks
	if len(enabledWebhooks) == 0 {
		enabledWebhooks = map[string]bool{
			app.WebhookAccountCreate: true,
			app.WebhookCheckin:       true,
		}
	}

	if enabledWebhooks[app.WebhookAccountCreate] {
		mux.Handle("/webhook/account/create", WebhookLoggingMiddleware(accountCreateHandler, p.App))
	} else {
		p.App.Events.Dispatch(events.Infof("server", "Account create webhook disabled by configuration"))
	}

	if enabledWebhooks[app.WebhookCheckin] {
		mux.Handle("/webhook/checkin", WebhookLoggingMiddleware(checkinHandler, p.App))
	} else {
		p.App.Events.Dispatch(events.Infof("server", "Checkin webhook disabled by configuration"))
	}

	if !enabledWebhooks[app.WebhookAccountCreate] && !enabledWebhooks[app.WebhookCheckin] {
		p.App.Events.Dispatch(events.Warningf("server", "All webhooks are disabled; server will only serve /health"))
	}

	if p.App.Config.WebhookCatchAll {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			WebhookLoggingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				p.App.Events.Dispatch(events.Warningf("server", "Received request for unhandled path: %s", r.RequestURI))
				http.NotFound(w, r)
			}), p.App).ServeHTTP(w, r)
		})
	}

	mux.HandleFunc("/health", p.HandleHealthCheck)
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	server := &http.Server{Addr: addr, Handler: mux}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		p.App.Events.Dispatch(events.Infof("server", "Starting server on %s", addr))
		var err error
		if config.TLSEnabled {
			p.App.Events.Dispatch(events.Infof("server", "TLS is enabled. Starting HTTPS server."))
			err = server.ListenAndServeTLS(config.TLSCert, config.TLSKey)
		} else {
			err = server.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			p.App.Events.Dispatch(events.Errorf("server", "Server error: %v", err))
		}
	}()

	<-stop
	p.App.Events.Dispatch(events.Infof("server", "Shutting down server..."))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		p.App.Events.Dispatch(events.Errorf("server", "Server shutdown error: %v", err))
	}
	p.App.Events.Dispatch(events.Infof("server", "Server stopped"))
}

func (p *CliPresenter) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if p.App.DB != nil && p.App.DB.IsConnected() {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	} else {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
}

func (p *CliPresenter) HandleReplayWebhook(id int) {
	method, uri, headers, body, err := database.GetWebhookLog(p.App.DB, id)
	if err != nil {
		p.App.Events.Dispatch(events.Errorf("server", "Error getting webhook log: %v", err))
		return
	}

	req, err := http.NewRequest(method, uri, bytes.NewBufferString(body))
	if err != nil {
		p.App.Events.Dispatch(events.Errorf("server", "Error creating request: %v", err))
		return
	}

	var headerMap map[string][]string
	if err := json.Unmarshal([]byte(headers), &headerMap); err != nil {
		p.App.Events.Dispatch(events.Errorf("server", "Error unmarshaling headers: %v", err))
		return
	}
	req.Header = headerMap

	rr := httptest.NewRecorder()

	switch uri {
	case "/webhook/account/create":
		if !p.App.Config.Server.Webhooks[app.WebhookAccountCreate] {
			p.App.Events.Dispatch(events.Warningf("server", "Replay requested for disabled account create webhook"))
			return
		}
		p.HandleAccountCreateWebhook(rr, req)
	case "/webhook/checkin":
		if !p.App.Config.Server.Webhooks[app.WebhookCheckin] {
			p.App.Events.Dispatch(events.Warningf("server", "Replay requested for disabled checkin webhook"))
			return
		}
		p.HandleCheckinWebhook(rr, req)
	default:
		p.App.Events.Dispatch(events.Warningf("server", "No handler for path: %s", uri))
		return
	}

	if rr.Code != http.StatusOK {
		p.App.Events.Dispatch(events.Errorf("server", "Replay failed with status code %d: %s", rr.Code, rr.Body.String()))
	} else {
		p.App.Events.Dispatch(events.Infof("server", "Webhook %d replayed successfully", id))
	}
}

func (p *CliPresenter) HandleAccountCreateWebhook(w http.ResponseWriter, r *http.Request) {
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

	if err := pull.StoreAccountDetailed(p.App, &acc); err != nil {
		http.Error(w, "failed to store account", http.StatusInternalServerError)
		return
	}
	p.App.Events.Dispatch(events.Infof("server", "Received and processed account webhook for account: %s", acc.FullName.String))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Account webhook processed")
}

func (p *CliPresenter) HandleCheckinWebhook(w http.ResponseWriter, r *http.Request) {
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

	if err := pull.StoreCheckin(p.App, checkin); err != nil {
		http.Error(w, "failed to store checkin", http.StatusInternalServerError)
		return
	}
	p.App.Events.Dispatch(events.Infof("server", "Received and processed checkin webhook for checkin: %d", checkin.CheckinId.Int64))
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Checkin webhook processed")
}
