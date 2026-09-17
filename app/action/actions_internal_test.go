package action

import "testing"

func TestReplaceEventTokensHandlesOverlappingTokens(t *testing.T) {
	ctx := &ExecutionContext{
		Payload: map[string]interface{}{
			"account": map[string]interface{}{
				"id": "456",
			},
		},
	}
	const want = `{"account":{"id":"456"}}`

	for range 1000 {
		if got := replaceEventTokens("$EVENT_PAYLOAD_JSON", ctx); got != want {
			t.Fatalf("replaceEventTokens() = %q, want %q", got, want)
		}
	}
}
