package syncproxy

import (
	"badgermaps/app"
	"badgermaps/app/action"
	"badgermaps/app/pull"
	"badgermaps/app/push"
	appserver "badgermaps/app/server"
	"badgermaps/events"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	defaultClientTimeout       = 3 * time.Second
	defaultJobPollInterval     = 1 * time.Second
	defaultActivityRetryCount  = 3
	defaultActivityRetryDelay  = 1 * time.Second
	defaultActivityStaleWindow = 20 * time.Second
	defaultInternalRequestPath = "/internal/jobs/sync"
	defaultScheduledJobRunPath = "/internal/scheduled-jobs/run"
	authorizationHeaderName    = "Authorization"
	bearerAuthPrefix           = "Bearer "
)

type healthStatusError struct {
	statusCode int
}

func (e healthStatusError) Error() string {
	return fmt.Sprintf("health endpoint returned %d", e.statusCode)
}

func RunWithServerRouting(
	a *app.App,
	mode appserver.SyncMode,
	source string,
	resourceID int,
	localRun func() error,
) error {
	if localRun == nil {
		return fmt.Errorf("local sync runner is required")
	}
	if a == nil {
		return localRun()
	}
	if strings.TrimSpace(source) == "" {
		source = "manual"
	}
	if a.Server == nil {
		return runAsLocalWorkflowJob(a, mode, source, resourceID, localRun)
	}

	_, running := a.Server.GetServerStatus()
	if !running {
		return runAsLocalWorkflowJob(a, mode, source, resourceID, localRun)
	}

	baseURL := serverBaseURL(a)
	client := newInternalHTTPClient(baseURL)

	healthy, healthErr := serverHealthy(client, baseURL)
	if healthy {
		return runAsRemoteJob(a, client, baseURL, mode, source, resourceID)
	}

	a.Events.Dispatch(events.Warningf("server.proxy", "Server process is running but unhealthy: %v", healthErr))
	if isHealthStatusError(healthErr) {
		return runAsRemoteJob(a, client, baseURL, mode, source, resourceID)
	}

	activity, activityErr := resolveActivity(client, baseURL, a)
	if activityErr != nil {
		return fmt.Errorf("server is unhealthy and activity could not be determined: %w", activityErr)
	}

	if activity.IsActive() {
		for i := 0; i < defaultActivityRetryCount; i++ {
			time.Sleep(defaultActivityRetryDelay)
			nextActivity, err := resolveActivity(client, baseURL, a)
			if err != nil {
				continue
			}
			activity = nextActivity
			if !activity.IsActive() {
				break
			}
		}
		if activity.IsActive() {
			return fmt.Errorf(
				"server is unhealthy but still has an active job (%s); refusing local execution",
				activity.ActiveJobID,
			)
		}
	}

	if err := a.Server.StopServer(); err != nil {
		if !strings.Contains(strings.ToLower(err.Error()), "not running") {
			return fmt.Errorf("failed to stop stale server: %w", err)
		}
	}

	return runAsLocalWorkflowJob(a, mode, source, resourceID, localRun)
}

func isHealthStatusError(err error) bool {
	var target healthStatusError
	return errors.As(err, &target)
}

func newInternalHTTPClient(baseURL string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(baseURL)), "https://") {
		transport.TLSClientConfig = &tls.Config{
			// These calls target local internal endpoints that require both bearer auth and local-origin checks.
			InsecureSkipVerify: true,
		}
	}

	return &http.Client{
		Timeout:   defaultClientTimeout,
		Transport: transport,
	}
}

func resolveInternalAPIToken(a *app.App) string {
	if a == nil {
		return ""
	}
	if a.Config != nil {
		if token := strings.TrimSpace(a.Config.Server.InternalAPIToken); token != "" {
			return token
		}
	}
	if a.State != nil {
		return strings.TrimSpace(a.State.ServerInternalAPIToken)
	}
	return ""
}

func applyInternalAuthHeader(req *http.Request, a *app.App) error {
	if req == nil {
		return fmt.Errorf("internal request is required")
	}
	token := resolveInternalAPIToken(a)
	if token == "" {
		return fmt.Errorf("internal API token is not configured; set server.internal_api_token in config")
	}
	req.Header.Set(authorizationHeaderName, bearerAuthPrefix+token)
	return nil
}

func FetchServerJobsSnapshot(a *app.App) (*appserver.SyncJobListResponse, error) {
	if a == nil {
		return nil, fmt.Errorf("server context unavailable")
	}
	running := false
	if a.Server != nil {
		_, running = a.Server.GetServerStatus()
	}
	if !running {
		if queue := a.GetSyncCoordinator(); queue != nil {
			return &appserver.SyncJobListResponse{
				Activity: queue.GetActivity(),
				Jobs:     queue.ListJobs(),
			}, nil
		}
		return nil, fmt.Errorf("server is not running")
	}

	baseURL := serverBaseURL(a)
	client := newInternalHTTPClient(baseURL)

	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/jobs", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create jobs request: %w", err)
	}
	if err := applyInternalAuthHeader(req, a); err != nil {
		return nil, fmt.Errorf("failed to authorize jobs request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		activity, activityErr := appserver.ReadRuntimeActivity(a.State)
		if activityErr != nil {
			return nil, fmt.Errorf("failed to fetch jobs: %w", err)
		}
		return &appserver.SyncJobListResponse{
			Activity: activity,
			Jobs:     []*appserver.SyncJob{},
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch jobs (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var snapshot appserver.SyncJobListResponse
	if err := json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return nil, fmt.Errorf("failed to decode jobs response: %w", err)
	}

	if snapshot.Jobs == nil {
		snapshot.Jobs = []*appserver.SyncJob{}
	}
	return &snapshot, nil
}

func RunScheduledJobNow(a *app.App, jobID string) error {
	trimmedID := strings.TrimSpace(jobID)
	if trimmedID == "" {
		return fmt.Errorf("job id is required")
	}
	if a == nil {
		return fmt.Errorf("server context unavailable")
	}

	running := false
	if a.Server != nil {
		_, running = a.Server.GetServerStatus()
	}
	if running {
		return runScheduledJobNowRemote(a, trimmedID)
	}
	return runScheduledJobNowLocal(a, trimmedID)
}

func runScheduledJobNowRemote(a *app.App, jobID string) error {
	baseURL := serverBaseURL(a)
	client := newInternalHTTPClient(baseURL)

	payload := appserver.ScheduledJobRunRequest{JobID: jobID}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal scheduled job request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+defaultScheduledJobRunPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create scheduled job request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := applyInternalAuthHeader(req, a); err != nil {
		return fmt.Errorf("failed to authorize scheduled job request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit scheduled job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to submit scheduled job (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	a.Events.Dispatch(events.Infof("server.proxy", "Scheduled job %s submitted to active server", jobID))
	return nil
}

func runScheduledJobNowLocal(a *app.App, jobID string) error {
	queue := a.EnsureSyncCoordinator()
	scheduler := appserver.NewScheduler(
		a.State,
		a.DB,
		a.API,
		a.Events,
		nil,
		schedulerLocalSyncExecutor{app: a},
		queue,
		a.Config.WorkflowProfiles,
		a.Config.Server.Timezone,
	)
	if err := scheduler.QueueStoredJobNow(jobID); err != nil {
		return err
	}

	a.Events.Dispatch(events.Infof("server.proxy", "Scheduled job %s queued locally", jobID))
	return nil
}

func runAsRemoteJob(
	a *app.App,
	client *http.Client,
	baseURL string,
	mode appserver.SyncMode,
	source string,
	resourceID int,
) error {
	profileName, steps, err := resolveWorkflowForMode(a, mode)
	if err != nil {
		return err
	}

	payload := appserver.SyncJobSubmitRequest{
		Mode:            string(mode),
		Source:          source,
		ResourceID:      resourceID,
		WorkflowProfile: profileName,
		Steps:           steps,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal sync request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, baseURL+defaultInternalRequestPath, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create sync request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if err := applyInternalAuthHeader(req, a); err != nil {
		return fmt.Errorf("failed to authorize sync request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to submit sync job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to submit sync job (%d): %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var job appserver.SyncJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return fmt.Errorf("failed to decode sync job response: %w", err)
	}
	if strings.TrimSpace(job.ID) == "" {
		return fmt.Errorf("sync job response missing job id")
	}

	a.Events.Dispatch(events.Infof("server.proxy", "Sync job %s submitted to active server", job.ID))

	for {
		current, err := fetchJobStatus(client, baseURL, job.ID, a)
		if err != nil {
			return err
		}
		switch current.Status {
		case appserver.SyncJobCompleted:
			return nil
		case appserver.SyncJobCompletedWithErrors:
			a.Events.Dispatch(events.Warningf("server.proxy", "Workflow job %s completed with %d error(s)", current.ID, current.ErrorCount))
			return nil
		case appserver.SyncJobFailed:
			if current.Error == "" {
				return fmt.Errorf("server sync job %s failed", current.ID)
			}
			return fmt.Errorf("server sync job %s failed: %s", current.ID, current.Error)
		default:
			time.Sleep(defaultJobPollInterval)
		}
	}
}

func runAsLocalWorkflowJob(
	a *app.App,
	mode appserver.SyncMode,
	source string,
	resourceID int,
	localRun func() error,
) error {
	profileName, steps, err := resolveWorkflowForMode(a, mode)
	if err != nil {
		return err
	}

	queue := a.EnsureSyncCoordinator()
	jobName := strings.TrimSpace(source)
	if jobName == "" {
		jobName = fmt.Sprintf("workflow:%s", mode)
	}

	job, err := queue.Submit(appserver.SyncJobRequest{
		Name:   jobName,
		Source: source,
		Mode:   mode,
		Kind:   appserver.SyncJobKindWorkflow,
		Run: func(_ context.Context) error {
			_, execErr := appserver.ExecuteWorkflowSteps(appserver.WorkflowExecutionOptions{
				Queue:      queue,
				Source:     source,
				ParentMode: mode,
				ResourceID: resourceID,
				Steps:      steps,
				RunSync: func(stepMode appserver.SyncMode, stepResourceID int) error {
					if stepMode == mode && localRun != nil {
						return localRun()
					}
					return runLocalSyncMode(a, stepMode, stepResourceID)
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
					return a.ExecuteActionSyncWithContext(cfg, execCtx)
				},
			})
			return execErr
		},
	})
	if err != nil {
		return fmt.Errorf("failed to submit local workflow job: %w", err)
	}

	for {
		current, exists := queue.GetJob(job.ID)
		if !exists {
			return fmt.Errorf("workflow job %s disappeared from local queue", job.ID)
		}
		switch current.Status {
		case appserver.SyncJobCompleted:
			return nil
		case appserver.SyncJobCompletedWithErrors:
			a.Events.Dispatch(events.Warningf("server.proxy", "Workflow job %s completed with %d error(s)", current.ID, current.ErrorCount))
			return nil
		case appserver.SyncJobFailed:
			if strings.TrimSpace(current.Error) == "" {
				return fmt.Errorf("workflow job %s failed", current.ID)
			}
			return fmt.Errorf("workflow job %s failed: %s", current.ID, current.Error)
		default:
			time.Sleep(defaultJobPollInterval)
		}
	}
}

func resolveWorkflowForMode(a *app.App, mode appserver.SyncMode) (string, []appserver.WorkflowStep, error) {
	profileName := appserver.DefaultWorkflowProfileForMode(mode)
	profiles := appserver.NormalizeWorkflowProfiles(nil)
	if a != nil && a.Config != nil {
		profiles = appserver.NormalizeWorkflowProfiles(a.Config.WorkflowProfiles)
	}
	if len(profiles) == 0 {
		profiles = appserver.DefaultWorkflowProfiles()
	}
	if profile, ok := profiles[profileName]; ok {
		if err := appserver.ValidateWorkflowProfile(profile); err != nil {
			return "", nil, err
		}
		return profileName, profile.Steps, nil
	}

	fallbackID := strings.TrimSpace(string(mode))
	if fallbackID == "" {
		fallbackID = "step_1"
	}
	steps := []appserver.WorkflowStep{
		{
			ID:       fallbackID,
			Name:     fallbackID,
			Type:     appserver.WorkflowStepTypeSync,
			SyncMode: mode,
		},
	}
	if err := appserver.ValidateWorkflowSteps(steps); err != nil {
		return "", nil, err
	}
	return "", steps, nil
}

type schedulerLocalSyncExecutor struct {
	app *app.App
}

func (e schedulerLocalSyncExecutor) PullAccounts() error {
	return runLocalSyncMode(e.app, appserver.SyncModePullAccounts, 0)
}

func (e schedulerLocalSyncExecutor) PullCheckins() error {
	return runLocalSyncMode(e.app, appserver.SyncModePullCheckins, 0)
}

func (e schedulerLocalSyncExecutor) PullRoutes() error {
	return runLocalSyncMode(e.app, appserver.SyncModePullRoutes, 0)
}

func (e schedulerLocalSyncExecutor) PullProfile() error {
	return runLocalSyncMode(e.app, appserver.SyncModePullProfile, 0)
}

func (e schedulerLocalSyncExecutor) PushAll() error {
	return runLocalSyncMode(e.app, appserver.SyncModePush, 0)
}

func (e schedulerLocalSyncExecutor) PushAccounts() error {
	return runLocalSyncMode(e.app, appserver.SyncModePushAccounts, 0)
}

func (e schedulerLocalSyncExecutor) PushCheckins() error {
	return runLocalSyncMode(e.app, appserver.SyncModePushCheckins, 0)
}

func (e schedulerLocalSyncExecutor) PullAccount(resourceID int) error {
	return runLocalSyncMode(e.app, appserver.SyncModePullAccount, resourceID)
}

func (e schedulerLocalSyncExecutor) PullCheckin(resourceID int) error {
	return runLocalSyncMode(e.app, appserver.SyncModePullCheckin, resourceID)
}

func (e schedulerLocalSyncExecutor) PullRoute(resourceID int) error {
	return runLocalSyncMode(e.app, appserver.SyncModePullRoute, resourceID)
}

func runLocalSyncMode(a *app.App, mode appserver.SyncMode, resourceID int) error {
	switch mode {
	case appserver.SyncModeNone:
		return nil
	case appserver.SyncModePull:
		if err := pull.PullGroupAccounts(a, 0, nil); err != nil {
			return err
		}
		if err := pull.PullGroupCheckins(a, nil); err != nil {
			return err
		}
		if err := pull.PullGroupRoutes(a, nil); err != nil {
			return err
		}
		_, err := pull.PullProfile(a, nil)
		return err
	case appserver.SyncModePullPush:
		if err := runLocalSyncMode(a, appserver.SyncModePull, 0); err != nil {
			return err
		}
		return runLocalSyncMode(a, appserver.SyncModePush, 0)
	case appserver.SyncModePullAccounts:
		return pull.PullGroupAccounts(a, 0, nil)
	case appserver.SyncModePullCheckins:
		return pull.PullGroupCheckins(a, nil)
	case appserver.SyncModePullRoutes:
		return pull.PullGroupRoutes(a, nil)
	case appserver.SyncModePullProfile:
		_, err := pull.PullProfile(a, nil)
		return err
	case appserver.SyncModePush:
		if err := push.RunPushAccounts(a); err != nil {
			return err
		}
		return push.RunPushCheckins(a)
	case appserver.SyncModePushAccounts:
		return push.RunPushAccounts(a)
	case appserver.SyncModePushCheckins:
		return push.RunPushCheckins(a)
	case appserver.SyncModePullAccount:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		_, err := pull.PullAccount(a, resourceID)
		return err
	case appserver.SyncModePullCheckin:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		_, err := pull.PullCheckin(a, resourceID)
		return err
	case appserver.SyncModePullRoute:
		if resourceID <= 0 {
			return fmt.Errorf("resource id is required for mode %s", mode)
		}
		_, err := pull.PullRoute(a, resourceID)
		return err
	default:
		return fmt.Errorf("unsupported sync mode: %s", mode)
	}
}

func fetchJobStatus(client *http.Client, baseURL string, jobID string, a *app.App) (*appserver.SyncJob, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/jobs/"+jobID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
	}
	if err := applyInternalAuthHeader(req, a); err != nil {
		return nil, fmt.Errorf("failed to authorize status request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch job status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch job status (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var job appserver.SyncJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return nil, fmt.Errorf("failed to decode job status response: %w", err)
	}
	return &job, nil
}

func resolveActivity(client *http.Client, baseURL string, a *app.App) (appserver.RuntimeActivity, error) {
	activity, err := fetchActivity(client, baseURL, a)
	if err == nil {
		return activity, nil
	}

	fileActivity, fileErr := appserver.ReadRuntimeActivity(a.State)
	if fileErr != nil {
		return appserver.RuntimeActivity{}, err
	}
	if fileActivity.LastHeartbeat.IsZero() || time.Since(fileActivity.LastHeartbeat) > defaultActivityStaleWindow {
		return appserver.RuntimeActivity{}, fmt.Errorf("runtime activity heartbeat is stale")
	}
	return fileActivity, nil
}

func fetchActivity(client *http.Client, baseURL string, a *app.App) (appserver.RuntimeActivity, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/activity", nil)
	if err != nil {
		return appserver.RuntimeActivity{}, fmt.Errorf("failed to create activity request: %w", err)
	}
	if err := applyInternalAuthHeader(req, a); err != nil {
		return appserver.RuntimeActivity{}, fmt.Errorf("failed to authorize activity request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return appserver.RuntimeActivity{}, fmt.Errorf("failed to fetch activity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return appserver.RuntimeActivity{}, fmt.Errorf("failed to fetch activity (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var activity appserver.RuntimeActivity
	if err := json.NewDecoder(resp.Body).Decode(&activity); err != nil {
		return appserver.RuntimeActivity{}, fmt.Errorf("failed to decode activity response: %w", err)
	}
	return activity, nil
}

func serverHealthy(client *http.Client, baseURL string) (bool, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		return false, fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, healthStatusError{statusCode: resp.StatusCode}
	}
	return true, nil
}

func serverBaseURL(a *app.App) string {
	host := resolveInternalHost(a.State.ServerHost)
	if ip := net.ParseIP(host); ip != nil && ip.To4() == nil {
		host = "[" + host + "]"
	}

	port := a.State.ServerPort
	if port <= 0 {
		port = 8080
	}

	scheme := "http"
	if a.State.TLSEnabled {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s:%d", scheme, host, port)
}

func resolveInternalHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")
	switch strings.ToLower(host) {
	case "", "0.0.0.0", "::", "*":
		return "127.0.0.1"
	default:
		return host
	}
}
