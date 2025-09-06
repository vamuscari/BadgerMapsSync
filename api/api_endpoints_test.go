package api

import (
	"testing"
)

func TestNewEndpoints(t *testing.T) {
	baseURL := "http://test.com"
	endpoints := NewEndpoints(baseURL)

	if endpoints.baseURL != baseURL {
		t.Errorf("Expected baseURL to be %s, but got %s", baseURL, endpoints.baseURL)
	}
}

func TestDefaultEndpoints(t *testing.T) {
	endpoints := DefaultEndpoints()

	if endpoints.baseURL != DefaultApiBaseURL {
		t.Errorf("Expected baseURL to be %s, but got %s", DefaultApiBaseURL, endpoints.baseURL)
	}
}
