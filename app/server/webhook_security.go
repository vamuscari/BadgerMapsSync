package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type WebhookSecurity struct {
	secretKey       string
	enabled         bool
	timestampWindow time.Duration
}

func NewWebhookSecurity(secretKey string, enabled bool) *WebhookSecurity {
	return &WebhookSecurity{
		secretKey:       secretKey,
		enabled:         enabled,
		timestampWindow: 5 * time.Minute, // Webhook must be received within 5 minutes
	}
}

// VerifySignature verifies the HMAC-SHA256 signature of a webhook request
func (ws *WebhookSecurity) VerifySignature(r *http.Request, body []byte) error {
	if !ws.enabled {
		return nil // Signature verification disabled
	}

	if ws.secretKey == "" {
		return fmt.Errorf("webhook secret key not configured")
	}

	// Get signature from header
	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		signature = r.Header.Get("X-BadgerMaps-Signature")
	}
	if signature == "" {
		return fmt.Errorf("webhook signature header missing")
	}

	// Validate signature format (GitHub-style)
	if strings.HasPrefix(signature, "sha256=") {
		signature = signature[7:] // Remove "sha256=" prefix
	}

	// Get timestamp from header for replay attack prevention
	timestamp := r.Header.Get("X-Webhook-Timestamp")
	if timestamp != "" {
		if err := ws.verifyTimestamp(timestamp); err != nil {
			return err
		}
	}

	// Calculate expected signature
	expectedSig := ws.calculateSignature(body, timestamp)

	// Compare signatures (constant time comparison to prevent timing attacks)
	if !hmac.Equal([]byte(signature), []byte(expectedSig)) {
		return fmt.Errorf("invalid webhook signature")
	}

	return nil
}

// calculateSignature generates HMAC-SHA256 signature for webhook payload
func (ws *WebhookSecurity) calculateSignature(body []byte, timestamp string) string {
	mac := hmac.New(sha256.New, []byte(ws.secretKey))

	// If timestamp exists, include it in signature calculation
	if timestamp != "" {
		mac.Write([]byte(timestamp))
		mac.Write([]byte("."))
	}

	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// verifyTimestamp checks if webhook timestamp is within acceptable window
func (ws *WebhookSecurity) verifyTimestamp(timestamp string) error {
	var webhookTime time.Time

	// Try parsing as Unix timestamp first (most common format)
	if unixTime, err := strconv.ParseInt(timestamp, 10, 64); err == nil {
		webhookTime = time.Unix(unixTime, 0)
	} else if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
		// Fallback to RFC3339 format
		webhookTime = t
	} else {
		return fmt.Errorf("invalid webhook timestamp format")
	}

	// Check if timestamp is within acceptable window
	now := time.Now()
	diff := now.Sub(webhookTime)

	if diff < 0 {
		diff = -diff // Handle future timestamps
	}

	if diff > ws.timestampWindow {
		return fmt.Errorf("webhook timestamp outside acceptable window (received: %v, current: %v)", webhookTime, now)
	}

	return nil
}

// WebhookSecurityMiddleware creates HTTP middleware for webhook signature verification
func WebhookSecurityMiddleware(ws *WebhookSecurity) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only verify webhooks for specific paths
			if !strings.HasPrefix(r.URL.Path, "/webhook") && !strings.HasPrefix(r.URL.Path, "/api/webhook") {
				next.ServeHTTP(w, r)
				return
			}

			// Read body for signature verification
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}

			// Verify signature
			if err := ws.VerifySignature(r, body); err != nil {
				http.Error(w, fmt.Sprintf("Webhook verification failed: %v", err), http.StatusUnauthorized)
				return
			}

			// Restore body for downstream handlers
			r.Body = io.NopCloser(strings.NewReader(string(body)))

			next.ServeHTTP(w, r)
		})
	}
}

// GenerateWebhookSignature generates a signature for outgoing webhooks
func (ws *WebhookSecurity) GenerateWebhookSignature(payload []byte) (signature string, timestamp string) {
	timestamp = fmt.Sprintf("%d", time.Now().Unix())
	signature = "sha256=" + ws.calculateSignature(payload, timestamp)
	return signature, timestamp
}
