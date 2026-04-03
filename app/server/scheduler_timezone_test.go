package server

import (
	"badgermaps/app/state"
	"badgermaps/events"
	"path/filepath"
	"strings"
	"testing"
)

func TestSchedulerEffectiveScheduleSpecUsesGlobalTimezoneByDefault(t *testing.T) {
	s := NewScheduler(state.NewState(), nil, nil, nil, nil, nil, nil, nil, "America/Los_Angeles")

	spec, loc, source, err := s.effectiveScheduleSpec(&ScheduledJob{
		Name:     "daily",
		Schedule: "0 0 20 * * *",
	})
	if err != nil {
		t.Fatalf("expected no error resolving schedule spec: %v", err)
	}
	if spec != "0 0 20 * * *" {
		t.Fatalf("expected raw schedule spec, got %q", spec)
	}
	if source != TimezoneSourceGlobal {
		t.Fatalf("expected source %q, got %q", TimezoneSourceGlobal, source)
	}
	if got, want := loc.String(), "America/Los_Angeles"; got != want {
		t.Fatalf("expected location %q, got %q", want, got)
	}
}

func TestSchedulerEffectiveScheduleSpecUsesOverrideAndCronPrefix(t *testing.T) {
	s := NewScheduler(state.NewState(), nil, nil, nil, nil, nil, nil, nil, "America/Los_Angeles")

	spec, loc, source, err := s.effectiveScheduleSpec(&ScheduledJob{
		Name:     "daily",
		Schedule: "0 0 20 * * *",
		Timezone: "America/New_York",
	})
	if err != nil {
		t.Fatalf("expected no error resolving schedule spec: %v", err)
	}
	if !strings.HasPrefix(spec, "CRON_TZ=America/New_York ") {
		t.Fatalf("expected CRON_TZ prefix for override, got %q", spec)
	}
	if source != TimezoneSourceOverride {
		t.Fatalf("expected source %q, got %q", TimezoneSourceOverride, source)
	}
	if got, want := loc.String(), "America/New_York"; got != want {
		t.Fatalf("expected location %q, got %q", want, got)
	}
}

func TestSchedulerEffectiveScheduleSpecFallsBackToLocalWhenGlobalUnset(t *testing.T) {
	s := NewScheduler(state.NewState(), nil, nil, nil, nil, nil, nil, nil, "")

	_, loc, source, err := s.effectiveScheduleSpec(&ScheduledJob{
		Name:     "daily",
		Schedule: "0 0 20 * * *",
	})
	if err != nil {
		t.Fatalf("expected no error resolving schedule spec: %v", err)
	}
	if source != TimezoneSourceLocal {
		t.Fatalf("expected source %q, got %q", TimezoneSourceLocal, source)
	}
	if loc == nil {
		t.Fatalf("expected non-nil location")
	}
}

func TestSchedulerEffectiveScheduleSpecRejectsInvalidOverride(t *testing.T) {
	s := NewScheduler(state.NewState(), nil, nil, nil, nil, nil, nil, nil, "America/Los_Angeles")

	_, _, _, err := s.effectiveScheduleSpec(&ScheduledJob{
		Name:     "daily",
		Schedule: "0 0 20 * * *",
		Timezone: "Mars/Olympus",
	})
	if err == nil {
		t.Fatalf("expected error for invalid override timezone")
	}
}

func TestNextRunForSpecUsesProvidedLocation(t *testing.T) {
	loc, err := ResolveTimezoneLocation("America/New_York")
	if err != nil {
		t.Fatalf("failed to resolve timezone: %v", err)
	}

	next, err := nextRunForSpec("0 0 20 * * *", loc)
	if err != nil {
		t.Fatalf("failed to compute next run: %v", err)
	}
	if next.IsZero() {
		t.Fatalf("expected non-zero next run")
	}
	if got, want := next.Location().String(), "America/New_York"; got != want {
		t.Fatalf("expected next run location %q, got %q", want, got)
	}
}

func TestTestCronExpressionAcceptsFiveFieldSpec(t *testing.T) {
	if err := TestCronExpression("0 20 * * *"); err != nil {
		t.Fatalf("expected five-field cron expression to be accepted: %v", err)
	}
}

func TestTestCronExpressionAcceptsSixFieldSpec(t *testing.T) {
	if err := TestCronExpression("0 0 20 * * *"); err != nil {
		t.Fatalf("expected six-field cron expression to be accepted: %v", err)
	}
}

func TestUpdateJobKeepsTimezoneWhenUpdateTimezoneIsEmpty(t *testing.T) {
	st := state.NewState()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	*st.ConfigFile = configPath

	s := NewScheduler(st, nil, nil, events.NewEventDispatcher(), nil, nil, nil, nil, "")
	s.jobs["job-1"] = &ScheduledJob{
		ID:       "job-1",
		Name:     "daily",
		Schedule: "0 0 20 * * *",
		Steps: []WorkflowStep{
			{ID: "pull_accounts", Type: WorkflowStepTypeSync, SyncMode: SyncModePullAccounts},
		},
		Enabled:  false,
		Timezone: "America/New_York",
	}

	if err := s.UpdateJob("job-1", &ScheduledJob{Name: "updated", Enabled: false}); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	if got, want := s.jobs["job-1"].Timezone, "America/New_York"; got != want {
		t.Fatalf("expected timezone %q to be preserved, got %q", want, got)
	}
}
