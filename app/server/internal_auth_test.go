package server

import "testing"

func TestGenerateInternalAPIToken(t *testing.T) {
	token, err := GenerateInternalAPIToken()
	if err != nil {
		t.Fatalf("expected token generation to succeed, got %v", err)
	}
	if token == "" {
		t.Fatal("expected generated token to be non-empty")
	}
}
