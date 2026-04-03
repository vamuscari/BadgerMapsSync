package gui

import (
	"badgermaps/app"
	appserver "badgermaps/app/server"
	"strings"
	"testing"
	"time"
)

func TestFormatServerJobLineUsesProvidedTimezone(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	started := time.Date(2026, time.April, 2, 20, 30, 0, 0, time.UTC)
	completed := started.Add(15 * time.Minute)
	job := &appserver.SyncJob{
		ID:          "sync_1",
		Source:      "scheduler",
		Mode:        appserver.SyncModePull,
		Status:      appserver.SyncJobCompleted,
		QueuedAt:    time.Date(2026, time.April, 2, 20, 0, 0, 0, time.UTC),
		StartedAt:   &started,
		CompletedAt: &completed,
	}

	line := formatServerJobLine(job, loc)
	if !strings.Contains(line, "queued=2026-04-02 16:00:00 [America/New_York]") {
		t.Fatalf("expected queued time converted to New York timezone, got %q", line)
	}
	if !strings.Contains(line, "started=2026-04-02 16:30:00 [America/New_York]") {
		t.Fatalf("expected started time converted to New York timezone, got %q", line)
	}
	if !strings.Contains(line, "completed=2026-04-02 16:45:00 [America/New_York]") {
		t.Fatalf("expected completed time converted to New York timezone, got %q", line)
	}
}

func TestFormatServerActivityLineIncludesTimezone(t *testing.T) {
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	activity := appserver.RuntimeActivity{
		QueueDepth:      1,
		LastHeartbeat:   time.Date(2026, time.April, 2, 20, 0, 0, 0, time.UTC),
		ActiveJobID:     "sync_1",
		ActiveJobName:   "manual pull",
		ActiveJobMode:   appserver.SyncModePull,
		ActiveJobAction: "Pulling accounts",
	}
	line := formatServerActivityLine(activity, loc)
	if !strings.Contains(line, "Action: Pulling accounts") {
		t.Fatalf("expected active action in activity line, got %q", line)
	}
	if !strings.Contains(line, "Heartbeat: 2026-04-02 15:00:00 [America/Chicago]") {
		t.Fatalf("expected heartbeat converted to Chicago timezone, got %q", line)
	}
	if !strings.Contains(line, "TZ: America/Chicago") {
		t.Fatalf("expected timezone label in activity line, got %q", line)
	}
}

func TestFormatExplorerCellValueJobLogTimestampStringUsesServerTimezone(t *testing.T) {
	a := app.NewApp()
	a.Config.Server.Timezone = "America/New_York"
	ui := &Gui{app: a}

	got := ui.formatExplorerCellValue("JobLog", "StartedAt", "2026-04-02 20:30:00")
	if want := "2026-04-02 16:30:00 [America/New_York]"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatExplorerCellValueLeavesNonTimestampColumnsUntouched(t *testing.T) {
	a := app.NewApp()
	a.Config.Server.Timezone = "America/New_York"
	ui := &Gui{app: a}

	got := ui.formatExplorerCellValue("JobLog", "Status", "completed")
	if want := "completed"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatTimestampInDisplayTimezoneUsesAppTimezone(t *testing.T) {
	a := app.NewApp()
	a.Config.Server.Timezone = "America/New_York"
	ui := &Gui{app: a}

	ts := time.Date(2026, time.April, 2, 20, 30, 0, 0, time.UTC)
	got := ui.formatTimestampInDisplayTimezone(ts, time.RFC3339)
	if want := a.FormatTimestampInDisplayTimezone(ts, time.RFC3339); got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatServerJobLineIncludesCurrentAction(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	started := time.Date(2026, time.April, 2, 20, 30, 0, 0, time.UTC)
	job := &appserver.SyncJob{
		ID:            "sync_2",
		Source:        "manual",
		Mode:          appserver.SyncModePull,
		Status:        appserver.SyncJobRunning,
		QueuedAt:      time.Date(2026, time.April, 2, 20, 0, 0, 0, time.UTC),
		StartedAt:     &started,
		CurrentAction: "Pulling check-ins",
	}

	line := formatServerJobLine(job, loc)
	if !strings.Contains(line, "action=Pulling check-ins") {
		t.Fatalf("expected current action in job line, got %q", line)
	}
}

func TestFormatServerJobLineLabelsParentAndSubprocess(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to load timezone: %v", err)
	}

	parent := &appserver.SyncJob{
		ID:       "sync_parent",
		Name:     "Nightly Workflow",
		Source:   "scheduler",
		Mode:     appserver.SyncModePull,
		Kind:     appserver.SyncJobKindWorkflow,
		Status:   appserver.SyncJobRunning,
		QueuedAt: time.Date(2026, time.April, 2, 20, 0, 0, 0, time.UTC),
	}
	parentLine := formatServerJobLine(parent, loc)
	if !strings.Contains(parentLine, "Parent: sync_parent (Nightly Workflow)") {
		t.Fatalf("expected parent label with name, got %q", parentLine)
	}
	if strings.Contains(parentLine, "parent=") {
		t.Fatalf("did not expect parent linkage on parent line, got %q", parentLine)
	}

	subprocess := &appserver.SyncJob{
		ID:          "sync_child_1",
		Name:        "Pull Accounts",
		Source:      "scheduler",
		Mode:        appserver.SyncModePullAccounts,
		Kind:        appserver.SyncJobKindSync,
		Status:      appserver.SyncJobRunning,
		ParentJobID: "sync_parent",
		StepID:      "pull_accounts",
		StepIndex:   1,
		TotalSteps:  3,
		QueuedAt:    time.Date(2026, time.April, 2, 20, 1, 0, 0, time.UTC),
	}
	subprocessLine := formatServerJobLine(subprocess, loc)
	if !strings.Contains(subprocessLine, "Subprocess: sync_child_1 (Pull Accounts)") {
		t.Fatalf("expected subprocess label with name, got %q", subprocessLine)
	}
	if !strings.Contains(subprocessLine, "parent=sync_parent") {
		t.Fatalf("expected parent linkage for subprocess, got %q", subprocessLine)
	}
	if !strings.Contains(subprocessLine, "step=1/3 (pull_accounts)") {
		t.Fatalf("expected step metadata for subprocess, got %q", subprocessLine)
	}
}

func TestFilterActiveAndQueuedJobs(t *testing.T) {
	jobs := []*appserver.SyncJob{
		{ID: "running", Status: appserver.SyncJobRunning},
		{ID: "queued", Status: appserver.SyncJobQueued},
		{ID: "failed", Status: appserver.SyncJobFailed},
		{ID: "completed", Status: appserver.SyncJobCompleted},
	}

	filtered := filterActiveAndQueuedJobs(jobs)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 active/queued jobs, got %d", len(filtered))
	}
	if filtered[0].ID != "running" || filtered[1].ID != "queued" {
		t.Fatalf("unexpected filtered order: %+v", filtered)
	}
}
