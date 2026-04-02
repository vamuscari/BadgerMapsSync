package syncproxy

import (
	"badgermaps/app"
	appserver "badgermaps/app/server"
	"badgermaps/events"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
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
	if a == nil || a.Server == nil {
		return localRun()
	}

	_, running := a.Server.GetServerStatus()
	if !running {
		return localRun()
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

	return localRun()
}

func isHealthStatusError(err error) bool {
	var target healthStatusError
	return errors.As(err, &target)
}

func newInternalHTTPClient(baseURL string) *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(baseURL)), "https://") {
		transport.TLSClientConfig = &tls.Config{
			// These calls target only local internal endpoints that are already guarded by local-only checks.
			InsecureSkipVerify: true,
		}
	}

	return &http.Client{
		Timeout:   defaultClientTimeout,
		Transport: transport,
	}
}

func runAsRemoteJob(
	a *app.App,
	client *http.Client,
	baseURL string,
	mode appserver.SyncMode,
	source string,
	resourceID int,
) error {
	if source == "" {
		source = "manual"
	}

	payload := appserver.SyncJobSubmitRequest{
		Mode:       string(mode),
		Source:     source,
		ResourceID: resourceID,
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
		current, err := fetchJobStatus(client, baseURL, job.ID)
		if err != nil {
			return err
		}
		switch current.Status {
		case appserver.SyncJobCompleted:
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

func fetchJobStatus(client *http.Client, baseURL string, jobID string) (*appserver.SyncJob, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/jobs/"+jobID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create status request: %w", err)
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
	activity, err := fetchActivity(client, baseURL)
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

func fetchActivity(client *http.Client, baseURL string) (appserver.RuntimeActivity, error) {
	req, err := http.NewRequest(http.MethodGet, baseURL+"/internal/activity", nil)
	if err != nil {
		return appserver.RuntimeActivity{}, fmt.Errorf("failed to create activity request: %w", err)
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
	host := strings.TrimSpace(a.State.ServerHost)
	if host == "" && a.Config != nil {
		host = strings.TrimSpace(a.Config.Server.Host)
	}
	host = resolveInternalHost(host)

	port := a.State.ServerPort
	if port <= 0 && a.Config != nil {
		port = a.Config.Server.Port
	}
	if port <= 0 {
		port = 8080
	}

	scheme := "http"
	if a.State.TLSEnabled || (a.Config != nil && a.Config.Server.TLSEnabled) {
		scheme = "https"
	}

	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(host, strconv.Itoa(port)))
}

func resolveInternalHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.Trim(host, "[]")
	switch strings.ToLower(host) {
	case "", "0.0.0.0", "::", "*":
		return "127.0.0.1"
	default:
		return host
	}
}
