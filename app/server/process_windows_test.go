//go:build windows

package server

import "testing"

func TestProcessRunningInvalidPID(t *testing.T) {
	if processRunning(0) {
		t.Fatalf("expected processRunning(0) to be false")
	}
	if processRunning(-1) {
		t.Fatalf("expected processRunning(-1) to be false")
	}
}
