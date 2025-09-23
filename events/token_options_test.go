package events

import "testing"

func TestEventTokenOptionsIncludesPayloadFields(t *testing.T) {
	options := EventTokenOptions("pull.start", "accounts")

	if len(options) == 0 {
		t.Fatalf("expected options, got none")
	}

	if !tokenPresent(options, "$EVENT_PAYLOAD[ResourceID]") {
		t.Fatalf("expected ResourceID payload token to be present")
	}
}

func TestEventTokenOptionsIncludesCustomOption(t *testing.T) {
	options := EventTokenOptions("pull.start", "")
	if !tokenPresent(options, customPayloadOption.Token) && customPayloadOption.Format == "" {
		t.Fatalf("expected custom payload option to be present")
	}

	foundFormat := false
	for _, opt := range options {
		if opt.RequiresPath && opt.Format == customPayloadOption.Format {
			foundFormat = true
			break
		}
	}
	if !foundFormat {
		t.Fatalf("expected custom payload format option to be present")
	}
}

func TestEventWithoutPayloadFallsBackToBaseTokens(t *testing.T) {
	options := EventTokenOptions("connection.status.changed", "")
	if len(options) != len(baseTokenOptions)+1 {
		t.Fatalf("expected only base tokens and custom option, got %d", len(options))
	}
}

func TestEventTokenOptionsIncludeResourceID(t *testing.T) {
	options := EventTokenOptions("pull.complete", "account")
	if !tokenPresent(options, "$EVENT_PAYLOAD[resource_id]") {
		t.Fatalf("expected resource_id token for pull.complete/account")
	}

	options = EventTokenOptions("pull.error", "route")
	if !tokenPresent(options, "$EVENT_PAYLOAD[resource_id]") {
		t.Fatalf("expected resource_id token for pull.error/route")
	}
}

func tokenPresent(options []EventTokenOption, token string) bool {
	for _, opt := range options {
		if opt.Token == token {
			return true
		}
	}
	return false
}
