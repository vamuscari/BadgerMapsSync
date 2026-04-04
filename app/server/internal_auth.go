package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const internalAPITokenBytes = 32

func GenerateInternalAPIToken() (string, error) {
	randomBytes := make([]byte, internalAPITokenBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(randomBytes), nil
}
