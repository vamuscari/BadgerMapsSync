package gui

import "testing"

func TestResolveServerControlVisualState(t *testing.T) {
	running := resolveServerControlVisualState(true)
	if running.StatusLabel != "Running" {
		t.Fatalf("expected running status label %q, got %q", "Running", running.StatusLabel)
	}
	if running.ActionLabel != "Stop" {
		t.Fatalf("expected running action label %q, got %q", "Stop", running.ActionLabel)
	}
	if running.StatusColorName != StatusPositiveColorName {
		t.Fatalf("expected running status color %q, got %q", StatusPositiveColorName, running.StatusColorName)
	}
	if running.ActionColorName != StatusNegativeColorName {
		t.Fatalf("expected running action color %q, got %q", StatusNegativeColorName, running.ActionColorName)
	}

	stopped := resolveServerControlVisualState(false)
	if stopped.StatusLabel != "Stopped" {
		t.Fatalf("expected stopped status label %q, got %q", "Stopped", stopped.StatusLabel)
	}
	if stopped.ActionLabel != "Start" {
		t.Fatalf("expected stopped action label %q, got %q", "Start", stopped.ActionLabel)
	}
	if stopped.StatusColorName != StatusNegativeColorName {
		t.Fatalf("expected stopped status color %q, got %q", StatusNegativeColorName, stopped.StatusColorName)
	}
	if stopped.ActionColorName != StatusPositiveColorName {
		t.Fatalf("expected stopped action color %q, got %q", StatusPositiveColorName, stopped.ActionColorName)
	}
}
