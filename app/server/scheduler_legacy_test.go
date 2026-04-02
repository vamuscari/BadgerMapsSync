package server

import (
	"badgermaps/app/action"
	"strings"
	"testing"
)

func TestLegacySchedulerJobIDDeterministic(t *testing.T) {
	job := CronJob{
		Name:     "nightly-push",
		Schedule: "0 * * * *",
		Action: action.ActionConfig{
			Type: "exec",
			Args: map[string]interface{}{
				"command": "echo hello",
			},
		},
	}

	id1 := legacySchedulerJobID(job)
	id2 := legacySchedulerJobID(job)
	if id1 != id2 {
		t.Fatalf("expected deterministic legacy job id, got %q and %q", id1, id2)
	}
	if !strings.HasPrefix(id1, "legacy_") {
		t.Fatalf("expected id prefix legacy_, got %q", id1)
	}
}
