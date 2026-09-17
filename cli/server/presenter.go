package server

import (
	"badgermaps/api/models"
	"badgermaps/app"
	"badgermaps/app/action"
	"badgermaps/app/pull"
	"badgermaps/app/push"
	appserver "badgermaps/app/server"
	"badgermaps/database"
	"badgermaps/events"
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// CliPresenter handles the presentation logic for the server command.
type CliPresenter struct {
	App       *app.App
	syncQueue *appserver.SyncJobCoordinator
	scheduler *appserver.Scheduler
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
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

// HandleServerRestart restarts the running server.
func (p *CliPresenter) HandleServerRestart() {
	if err := p.App.Server.RestartServer(); err != nil {
		p.App.Events.Dispatch(events.Errorf("server", "Failed to restart server: %v", err))
		os.Exit(1)
	}
	pid, _ := p.App.Server.GetServerStatus()
	if pid > 0 {
		p.App.Events.Dispatch(events.Infof("server", "Server restarted successfully with PID %d.", pid))
		return
	}
	p.App.Events.Dispatch(events.Infof("server", "Server restarted successfully."))
}

// HandleServerStatus checks and prints the server's status.
func (p *CliPresenter) HandleServerStatus() {
	status := p.App.Server.GetDetailedStatus()
	switch {
	case status.Running && status.PID > 0:
		p.App.Events.Dispatch(events.Infof("server", "Server is running with PID %d.", status.PID))
	case status.Running:
		p.App.Events.Dispatch(events.Infof("server", "Server is running."))
	case status.State == appserver.ServerStatusPending:
		p.App.Events.Dispatch(events.Infof("server", "Server service is pending."))
	case status.RuntimeMode == appserver.ServerRuntimeModeService && !status.Installed:
		p.App.Events.Dispatch(events.Warningf("server", "Windows service is not installed."))
	default:
		p.App.Events.Dispatch(events.Warningf("server", "Server is not running."))
	}
	if status.Message != "" && !status.Running {
		p.App.Events.Dispatch(events.Debugf("server", "Server status detail: %s", status.Message))
	}
}

// RunServer runs the server in the foreground.
func (p *CliPresenter) RunServer(config *ServerConfig) {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := p.RunServerWithContext(ctx, config); err != nil {
		p.App.Events.Dispatch(events.Errorf("server", "Failed to run server: %v", err))
		os.Exit(1)
	}
}

// RunServerWithContext runs the server until context cancellation or a fatal server error.
func (p *CliPresenter) RunServerWithContext(ctx context.Context, config *ServerConfig) error {
	if config != nil {
		if token := strings.TrimSpace(config.InternalAPIToken); token != "" {
			p.App.State.ServerInternalAPIToken = token
			if p.App.Config != nil {
				p.App.Config.Server.InternalAPIToken = token
			}
		}
		if secret := strings.TrimSpace(config.WebhookSecret); secret != "" {
			p.App.State.ServerWebhookSecret = secret
			if p.App.Config != nil {
				p.App.Config.Server.WebhookSecret = secret
			}
		}
	}

	syncQueue := p.App.EnsureSyncCoordinator()
	p.syncQueue = syncQueue
	defer func() {
		p.syncQueue = nil
		p.App.StopSyncCoordinator()
	}()

	scheduler := appserver.NewScheduler(
		p.App.State,
		p.App.DB,
		p.App.API,
		p.App.Events,
		nil,
		&schedulerSyncExecutor{presenter: p},
		syncQueue,
		p.App.Config.WorkflowProfiles,
		p.App.Config.Server.Timezone,
	)
	if err := scheduler.Start(); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	p.scheduler = scheduler
	defer func() {
		p.scheduler = nil
	}()
	defer scheduler.Stop()

	mux := http.NewServeMux()

	logRequests := config.LogRequests
	wrapWithLogging := func(handler http.Handler) http.Handler {
		if logRequests {
			return WebhookLoggingMiddleware(handler, p.App)
		}
		return handler
	}

	enabledWebhooks := p.App.Config.Server.Webhooks
	if len(enabledWebhooks) == 0 {
		enabledWebhooks = map[string]bool{
			app.WebhookAccountCreate: true,
			app.WebhookCheckin:       true,
		}
	}
	webhooksEnabled := enabledWebhooks[app.WebhookAccountCreate] || enabledWebhooks[app.WebhookCheckin]
	webhookSecret := strings.TrimSpace(config.WebhookSecret)
	if webhookSecret == "" && p.App.Config != nil {
		webhookSecret = strings.TrimSpace(p.App.Config.Server.WebhookSecret)
	}
	if err := validateWebhookSecurityConfig(webhooksEnabled, webhookSecret); err != nil {
		return err
	}
	webhookSecurity := appserver.NewWebhookSecurity(webhookSecret, webhooksEnabled)
	wrapWebhookHandler := func(handler http.Handler) http.Handler {
		if logRequests {
			handler = WebhookLoggingMiddleware(handler, p.App)
		}
		return appserver.WebhookSecurityMiddleware(webhookSecurity)(handler)
	}

	accountCreateHandler := wrapWebhookHandler(http.HandlerFunc(p.HandleAccountCreateWebhook))
	checkinHandler := wrapWebhookHandler(http.HandlerFunc(p.HandleCheckinWebhook))

	if enabledWebhooks[app.WebhookAccountCreate] {
		mux.Handle("/webhook/account/create", accountCreateHandler)
	} else {
		p.App.Events.Dispatch(events.Infof("server", "Account create webhook disabled by configuration"))
	}

	if enabledWebhooks[app.WebhookCheckin] {
		mux.Handle("/webhook/checkin", checkinHandler)
	} else {
		p.App.Events.Dispatch(events.Infof("server", "Checkin webhook disabled by configuration"))
	}

	if !enabledWebhooks[app.WebhookAccountCreate] && !enabledWebhooks[app.WebhookCheckin] {
		p.App.Events.Dispatch(events.Warningf("server", "All webhooks are disabled; server will only serve /health"))
	}

	mux.Handle("/internal/jobs/sync", p.withInternalAuth(http.HandlerFunc(p.HandleInternalSyncJob)))
	mux.Handle("/internal/jobs", p.withInternalAuth(http.HandlerFunc(p.HandleInternalSyncJobs)))
	mux.Handle("/internal/jobs/", p.withInternalAuth(http.HandlerFunc(p.HandleInternalSyncJobStatus)))
	mux.Handle("/internal/scheduled-jobs/run", p.withInternalAuth(http.HandlerFunc(p.HandleInternalScheduledJobRun)))
	mux.Handle("/internal/activity", p.withInternalAuth(http.HandlerFunc(p.HandleInternalActivity)))

	if p.App.Config.WebhookCatchAll {
		catchAllHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p.App.Events.Dispatch(events.Warningf("server", "Received request for unhandled path: %s", r.RequestURI))
			http.NotFound(w, r)
		})
		mux.Handle("/", wrapWithLogging(catchAllHandler))
	}

	mux.HandleFunc("/health", p.HandleHealthCheck)
	addr := net.JoinHostPort(normalizeServerHost(config.Host), strconv.Itoa(config.Port))
	server := newHTTPServer(addr, mux)

	serverErr := make(chan error, 1)
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
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case <-ctx.Done():
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}

	p.App.Events.Dispatch(events.Infof("server", "Shutting down server..."))
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	if err := <-serverErr; err != nil {
		return fmt.Errorf("server error: %w", err)
	}
	p.App.Events.Dispatch(events.Infof("server", "Server stopped"))
	return nil
}

func normalizeServerHost(host string) string {
	host = strings.TrimSpace(host)
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		candidate := strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
		if ip := net.ParseIP(candidate); ip != nil {
			return candidate
		}
	}
	return host
}

type schedulerSyncExecutor struct {
	presenter *CliPresenter
}

func (e *schedulerSyncExecutor) PullAccounts() error {
	return e.presenter.runSyncMode(appserver.SyncModePullAccounts, 0)
}

func (e *schedulerSyncExecutor) PullCheckins() error {
	return e.presenter.runSyncMode(appserver.SyncModePullCheckins, 0)
}

func (e *schedulerSyncExecutor) PullRoutes() error {
	return e.presenter.runSyncMode(appserver.SyncModePullRoutes, 0)
}

func (e *schedulerSyncExecutor) PullProfile() error {
	return e.presenter.runSyncMode(appserver.SyncModePullProfile, 0)
}

func (e *schedulerSyncExecutor) PushAll() error {
	return e.presenter.runSyncMode(appserver.SyncModePush, 0)
}

func (e *schedulerSyncExecutor) PushAccounts() error {
	return e.presenter.runSyncMode(appserver.SyncModePushAccounts, 0)
}

func (e *schedulerSyncExecutor) PushCheckins() error {
	return e.presenter.runSyncMode(appserver.SyncModePushCheckins, 0)
}

func (e *schedulerSyncExecutor) PullAccount(id int) error {
	return e.presenter.runSyncMode(appserver.SyncModePullAccount, id)
}

func (e *schedulerSyncExecutor) PullCheckin(id int) error {
	return e.presenter.runSyncMode(appserver.SyncModePullCheckin, id)
}

func (e *schedulerSyncExecutor) PullRoute(id int) error {
	return e.presenter.runSyncMode(appserver.SyncModePullRoute, id)
}

func (p *CliPresenter) setActiveJobAction(action string) {
	if p.syncQueue == nil {
		return
	}
	p.syncQueue.SetActiveJobAction(action)
}

func (p *CliPresenter) runSyncMode(mode appserver.SyncMode, resourceID int) error {
	switch mode {
	case appserver.SyncModeNone:
		return nil
	case appserver.SyncModePull:
		p.setActiveJobAction("Pulling accounts")
		if err := pull.PullGroupAccounts(p.App, resourceID, nil); err != nil {
			return err
		}
		p.setActiveJobAction("Pulling check-ins")
		if err := pull.PullGroupCheckins(p.App, nil); err != nil {
			return err
		}
		p.setActiveJobAction("Pulling routes")
		if err := pull.PullGroupRoutes(p.App, nil); err != nil {
			return err
		}
		p.setActiveJobAction("Pulling profile")
		_, err := pull.PullProfile(p.App, nil)
		return err
	case appserver.SyncModePullPush:
		if err := p.runSyncMode(appserver.SyncModePull, 0); err != nil {
			return err
		}
		return p.runSyncMode(appserver.SyncModePush, 0)
	case appserver.SyncModePullAccounts:
		p.setActiveJobAction("Pulling accounts")
		return pull.PullGroupAccounts(p.App, 0, nil)
	case appserver.SyncModePullCheckins:
		p.setActiveJobAction("Pulling check-ins")
		return pull.PullGroupCheckins(p.App, nil)
	case appserver.SyncModePullRoutes:
		p.setActiveJobAction("Pulling routes")
		return pull.PullGroupRoutes(p.App, nil)
	case appserver.SyncModePullProfile:
		p.setActiveJobAction("Pulling profile")
		_, err := pull.PullProfile(p.App, nil)
		return err
	case appserver.SyncModePush:
		p.setActiveJobAction("Pushing accounts")
		if err := push.RunPushAccounts(p.App); err != nil {
			return err
		}
		p.setActiveJobAction("Pushing check-ins")
		return push.RunPushCheckins(p.App)
	case appserver.SyncModePushAccounts:
		p.setActiveJobAction("Pushing accounts")
		return push.RunPushAccounts(p.App)
	case appserver.SyncModePushCheckins:
		p.setActiveJobAction("Pushing check-ins")
		return push.RunPushCheckins(p.App)
	case appserver.SyncModePullAccount:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		p.setActiveJobAction(fmt.Sprintf("Pulling account %d", resourceID))
		_, err := pull.PullAccount(p.App, resourceID)
		return err
	case appserver.SyncModePullCheckin:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		p.setActiveJobAction(fmt.Sprintf("Pulling check-in %d", resourceID))
		_, err := pull.PullCheckin(p.App, resourceID)
		return err
	case appserver.SyncModePullRoute:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		p.setActiveJobAction(fmt.Sprintf("Pulling route %d", resourceID))
		_, err := pull.PullRoute(p.App, resourceID)
		return err
	default:
		return fmt.Errorf("unsupported sync mode: %s", mode)
	}
}

func (p *CliPresenter) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	if p.App.DB != nil && p.App.DB.IsConnected() {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "OK")
	} else {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
}

func (p *CliPresenter) HandleInternalSyncJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if p.syncQueue == nil {
		http.Error(w, "Sync queue unavailable", http.StatusServiceUnavailable)
		return
	}

	var req appserver.SyncJobSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Mode) == "" && len(req.Steps) == 0 && strings.TrimSpace(req.WorkflowProfile) == "" {
		http.Error(w, "mode is required when workflow_profile/steps are not provided", http.StatusBadRequest)
		return
	}

	mode := appserver.SyncModeWorkflow
	if strings.TrimSpace(req.Mode) != "" {
		parsedMode, err := appserver.ParseSyncMode(req.Mode)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mode = parsedMode
	}

	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "manual"
	}

	steps, profileName, err := p.resolveWorkflowSubmitSteps(req, mode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	job, err := p.syncQueue.Submit(appserver.SyncJobRequest{
		Name:   req.Name,
		Source: source,
		Mode:   mode,
		Kind:   appserver.SyncJobKindWorkflow,
		Run: func(_ context.Context) error {
			_, runErr := appserver.ExecuteWorkflowSteps(appserver.WorkflowExecutionOptions{
				Queue:      p.syncQueue,
				Source:     source,
				ParentMode: mode,
				ResourceID: req.ResourceID,
				Steps:      steps,
				RunSync: func(stepMode appserver.SyncMode, stepResourceID int) error {
					return p.runSyncMode(stepMode, stepResourceID)
				},
				RunAction: func(cfg action.ActionConfig, step appserver.WorkflowStep) error {
					execCtx := &action.ExecutionContext{
						EventType: "workflow.step.action",
						Source:    source,
						Payload: map[string]interface{}{
							"workflow_profile": profileName,
							"step_id":          step.ID,
							"step_name":        step.EffectiveName(),
						},
					}
					return p.App.ExecuteActionSyncWithContext(cfg, execCtx)
				},
			})
			return runErr
		},
	})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to queue sync job: %v", err), http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(job)
}

func (p *CliPresenter) HandleInternalScheduledJobRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if p.scheduler == nil {
		http.Error(w, "Scheduler unavailable", http.StatusServiceUnavailable)
		return
	}

	var req appserver.ScheduledJobRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request payload: %v", err), http.StatusBadRequest)
		return
	}

	jobID := strings.TrimSpace(req.JobID)
	if jobID == "" {
		http.Error(w, "job_id is required", http.StatusBadRequest)
		return
	}

	if err := p.scheduler.RunJobNow(jobID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "job not found") {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("failed to queue scheduled job: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(appserver.ScheduledJobRunResponse{
		JobID:  jobID,
		Status: "queued",
	})
}

func (p *CliPresenter) resolveWorkflowSubmitSteps(req appserver.SyncJobSubmitRequest, mode appserver.SyncMode) ([]appserver.WorkflowStep, string, error) {
	if len(req.Steps) > 0 || strings.TrimSpace(req.WorkflowProfile) != "" {
		steps, err := appserver.ResolveWorkflowSteps(req.WorkflowProfile, req.Steps, p.App.Config.WorkflowProfiles)
		if err != nil {
			return nil, "", err
		}
		return steps, strings.TrimSpace(req.WorkflowProfile), nil
	}

	profileName := appserver.DefaultWorkflowProfileForMode(mode)
	steps, err := appserver.ResolveWorkflowSteps(profileName, nil, p.App.Config.WorkflowProfiles)
	if err == nil {
		return steps, profileName, nil
	}

	fallbackID := strings.TrimSpace(string(mode))
	if fallbackID == "" {
		fallbackID = "step_1"
	}
	fallbackSteps := []appserver.WorkflowStep{
		{
			ID:       fallbackID,
			Name:     fallbackID,
			Type:     appserver.WorkflowStepTypeSync,
			SyncMode: mode,
		},
	}
	if validateErr := appserver.ValidateWorkflowSteps(fallbackSteps); validateErr != nil {
		return nil, "", validateErr
	}
	return fallbackSteps, "", nil
}

func (p *CliPresenter) HandleInternalSyncJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if p.syncQueue == nil {
		http.Error(w, "Sync queue unavailable", http.StatusServiceUnavailable)
		return
	}

	jobID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/internal/jobs/"))
	if jobID == "" {
		http.Error(w, "job id is required", http.StatusBadRequest)
		return
	}

	job, exists := p.syncQueue.GetJob(jobID)
	if !exists {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(job)
}

func (p *CliPresenter) HandleInternalSyncJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if p.syncQueue == nil {
		http.Error(w, "Sync queue unavailable", http.StatusServiceUnavailable)
		return
	}

	response := appserver.SyncJobListResponse{
		Activity: p.syncQueue.GetActivity(),
		Jobs:     p.syncQueue.ListJobs(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (p *CliPresenter) HandleInternalActivity(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var activity appserver.RuntimeActivity
	if p.syncQueue != nil {
		activity = p.syncQueue.GetActivity()
	} else {
		var err error
		activity, err = appserver.ReadRuntimeActivity(p.App.State)
		if err != nil {
			http.Error(w, fmt.Sprintf("activity unavailable: %v", err), http.StatusServiceUnavailable)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(activity)
}

func (p *CliPresenter) withInternalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !hasValidInternalAuthHeader(r, p.configuredInternalAPIToken()) {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if !isLocalRequest(r) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (p *CliPresenter) configuredInternalAPIToken() string {
	if p == nil || p.App == nil {
		return ""
	}
	if p.App.Config != nil {
		if token := strings.TrimSpace(p.App.Config.Server.InternalAPIToken); token != "" {
			return token
		}
	}
	if p.App.State != nil {
		return strings.TrimSpace(p.App.State.ServerInternalAPIToken)
	}
	return ""
}

func hasValidInternalAuthHeader(r *http.Request, expectedToken string) bool {
	if r == nil {
		return false
	}
	expectedToken = strings.TrimSpace(expectedToken)
	if expectedToken == "" {
		return false
	}

	token, ok := extractBearerToken(r.Header.Get("Authorization"))
	if !ok {
		return false
	}
	if len(token) != len(expectedToken) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) == 1
}

func extractBearerToken(authorizationHeader string) (string, bool) {
	authorizationHeader = strings.TrimSpace(authorizationHeader)
	if !strings.HasPrefix(authorizationHeader, "Bearer ") {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authorizationHeader, "Bearer "))
	if token == "" {
		return "", false
	}
	return token, true
}

func validateWebhookSecurityConfig(webhooksEnabled bool, webhookSecret string) error {
	if webhooksEnabled && strings.TrimSpace(webhookSecret) == "" {
		return fmt.Errorf("server.webhook_secret is required when webhooks are enabled")
	}
	return nil
}

func isLocalRequest(r *http.Request) bool {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}

	host = strings.Trim(host, "[]")
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}

	addresses, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	for _, addr := range addresses {
		switch value := addr.(type) {
		case *net.IPNet:
			if value.IP.Equal(ip) {
				return true
			}
		case *net.IPAddr:
			if value.IP.Equal(ip) {
				return true
			}
		}
	}
	return false
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
	var acc models.Account
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
	var checkin models.Checkin
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
