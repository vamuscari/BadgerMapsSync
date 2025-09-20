package server

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWebhookSecurity_VerifySignature(t *testing.T) {
	secretKey := "test-secret-key"
	ws := NewWebhookSecurity(secretKey, true)

	tests := []struct {
		name          string
		body          []byte
		signature     string
		timestamp     string
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid signature with timestamp",
			body:        []byte(`{"test": "data"}`),
			signature:   "", // Will be calculated
			timestamp:   fmt.Sprintf("%d", time.Now().Unix()),
			expectError: false,
		},
		{
			name:          "Invalid signature",
			body:          []byte(`{"test": "data"}`),
			signature:     "sha256=invalid",
			timestamp:     fmt.Sprintf("%d", time.Now().Unix()),
			expectError:   true,
			errorContains: "invalid webhook signature",
		},
		{
			name:          "Missing signature",
			body:          []byte(`{"test": "data"}`),
			signature:     "",
			timestamp:     "",
			expectError:   true,
			errorContains: "webhook signature header missing",
		},
		{
			name:          "Expired timestamp",
			body:          []byte(`{"test": "data"}`),
			signature:     "calculated", // Will be calculated
			timestamp:     fmt.Sprintf("%d", time.Now().Add(-10*time.Minute).Unix()),
			expectError:   true,
			errorContains: "webhook timestamp outside acceptable window",
		},
		{
			name:          "Invalid timestamp format",
			body:          []byte(`{"test": "data"}`),
			signature:     "calculated", // Will be calculated
			timestamp:     "invalid-timestamp",
			expectError:   true,
			errorContains: "invalid webhook timestamp format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(tt.body))

			// Calculate valid signature if marked as "calculated"
			if tt.signature == "" && !tt.expectError {
				mac := hmac.New(sha256.New, []byte(secretKey))
				if tt.timestamp != "" {
					mac.Write([]byte(tt.timestamp))
					mac.Write([]byte("."))
				}
				mac.Write(tt.body)
				tt.signature = "sha256=" + hex.EncodeToString(mac.Sum(nil))
			} else if tt.signature == "calculated" {
				// Calculate signature even for test cases that will fail for other reasons
				mac := hmac.New(sha256.New, []byte(secretKey))
				if tt.timestamp != "" {
					mac.Write([]byte(tt.timestamp))
					mac.Write([]byte("."))
				}
				mac.Write(tt.body)
				tt.signature = "sha256=" + hex.EncodeToString(mac.Sum(nil))
			}

			if tt.signature != "" {
				req.Header.Set("X-Webhook-Signature", tt.signature)
			}
			if tt.timestamp != "" {
				req.Header.Set("X-Webhook-Timestamp", tt.timestamp)
			}

			err := ws.VerifySignature(req, tt.body)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %s", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestWebhookSecurity_Disabled(t *testing.T) {
	ws := NewWebhookSecurity("secret", false)

	req := httptest.NewRequest("POST", "/webhook", nil)
	err := ws.VerifySignature(req, []byte(`{"test": "data"}`))

	if err != nil {
		t.Errorf("Expected no error when security is disabled, got: %v", err)
	}
}

func TestWebhookSecurityMiddleware(t *testing.T) {
	secretKey := "test-secret"
	ws := NewWebhookSecurity(secretKey, true)

	// Create test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with security middleware
	handler := WebhookSecurityMiddleware(ws)(testHandler)

	tests := []struct {
		name           string
		path           string
		body           []byte
		signature      string
		expectedStatus int
	}{
		{
			name:           "Non-webhook path bypasses verification",
			path:           "/api/test",
			body:           []byte(`{"test": "data"}`),
			signature:      "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Webhook path with valid signature",
			path:           "/webhook",
			body:           []byte(`{"test": "data"}`),
			signature:      "", // Will be calculated
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Webhook path with invalid signature",
			path:           "/webhook",
			body:           []byte(`{"test": "data"}`),
			signature:      "invalid",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", tt.path, bytes.NewReader(tt.body))

			// Calculate valid signature if needed
			if tt.signature == "" && strings.HasPrefix(tt.path, "/webhook") && tt.expectedStatus == http.StatusOK {
				timestamp := fmt.Sprintf("%d", time.Now().Unix())
				mac := hmac.New(sha256.New, []byte(secretKey))
				mac.Write([]byte(timestamp))
				mac.Write([]byte("."))
				mac.Write(tt.body)
				signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
				req.Header.Set("X-Webhook-Signature", signature)
				req.Header.Set("X-Webhook-Timestamp", timestamp)
			} else if tt.signature != "" {
				req.Header.Set("X-Webhook-Signature", tt.signature)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestGenerateWebhookSignature(t *testing.T) {
	ws := NewWebhookSecurity("test-secret", true)
	payload := []byte(`{"test": "data"}`)

	signature, timestamp := ws.GenerateWebhookSignature(payload)

	// Verify signature format
	if !strings.HasPrefix(signature, "sha256=") {
		t.Errorf("Signature should have 'sha256=' prefix, got: %s", signature)
	}

	// Verify timestamp is valid Unix timestamp
	if _, err := fmt.Sscanf(timestamp, "%d", new(int64)); err != nil {
		t.Errorf("Timestamp should be valid Unix timestamp, got: %s", timestamp)
	}

	// Verify the generated signature can be verified
	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(payload))
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", timestamp)

	if err := ws.VerifySignature(req, payload); err != nil {
		t.Errorf("Generated signature should be verifiable, got error: %v", err)
	}
}
