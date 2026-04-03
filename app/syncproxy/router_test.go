package syncproxy

import (
	"badgermaps/app"
	"badgermaps/app/state"
	"testing"
)

func TestServerBaseURLNormalizesWildcardHostsToLoopback(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "empty host", host: "", want: "http://127.0.0.1:8080"},
		{name: "ipv4 wildcard", host: "0.0.0.0", want: "http://127.0.0.1:8080"},
		{name: "ipv6 wildcard", host: "::", want: "http://127.0.0.1:8080"},
		{name: "asterisk wildcard", host: "*", want: "http://127.0.0.1:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &app.App{State: state.NewState()}
			a.State.ServerHost = tt.host
			a.State.ServerPort = 8080

			if got := serverBaseURL(a); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestServerBaseURLFormatsIPv6LiteralAndTLS(t *testing.T) {
	a := &app.App{State: state.NewState()}
	a.State.ServerHost = "::1"
	a.State.ServerPort = 9443
	a.State.TLSEnabled = true

	if got, want := serverBaseURL(a), "https://[::1]:9443"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
