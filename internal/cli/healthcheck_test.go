package cli

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A container runtime runs this against an image holding no configuration directory,
// so needing one would make the check impossible to run where it is needed.
func TestHealthcheckNeedsNoConfig(t *testing.T) {
	server := healthBackend(t, http.StatusOK)
	code, _, stderr := executeCLI([]string{"healthcheck", "--addr", server})
	if code != exitSuccess {
		t.Fatalf("exit code = %d, want %d: %s", code, exitSuccess, stderr)
	}
}

func TestHealthcheckReportsWhatItFound(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		stopFirst     bool
		wantCode      int
		wantErrorText string
	}{
		{name: "a serving process passes", status: http.StatusOK, wantCode: exitSuccess},
		{
			name:          "a process answering anything else fails",
			status:        http.StatusServiceUnavailable,
			wantCode:      exitFailure,
			wantErrorText: "503",
		},
		{
			name:          "a process that is not listening fails",
			stopFirst:     true,
			wantCode:      exitFailure,
			wantErrorText: "reach",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addr := healthBackend(t, test.status)
			if test.stopFirst {
				addr = closedAddress(t)
			}
			code, _, stderr := executeCLI([]string{"healthcheck", "--addr", addr})
			if code != test.wantCode {
				t.Fatalf("exit code = %d, want %d: %s", code, test.wantCode, stderr)
			}
			if test.wantErrorText != "" && !strings.Contains(stderr, test.wantErrorText) {
				t.Errorf("stderr = %q, want it to mention %q", stderr, test.wantErrorText)
			}
		})
	}
}

// healthBackend answers /health with status and returns its host:port.
func healthBackend(t *testing.T, status int) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("path = %q, want /health", r.URL.Path)
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return strings.TrimPrefix(server.URL, "http://")
}

// closedAddress is an address nothing listens on: a server started for its port and
// stopped before it is named.
func closedAddress(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.NotFoundHandler())
	address := strings.TrimPrefix(server.URL, "http://")
	server.Close()
	return address
}
