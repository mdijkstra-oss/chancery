package cli

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// callBackend answers with a stream and records the body it was sent, so what call
// composes is read off a real backend rather than off the command's internals.
func callBackend(t *testing.T, status int, stream string) (string, *[]byte) {
	t.Helper()
	var sent []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read forwarded body: %v", err)
		}
		sent = body
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, stream)
	}))
	t.Cleanup(server.Close)
	return server.URL, &sent
}

func TestRunCallRendersStream(t *testing.T) {
	stream := "event: response.created\ndata: {\"type\":\"response.created\"}\n\n" +
		"event: response.output_text.delta\ndata: {\"delta\":\"hello \"}\n\n" +
		"event: response.reasoning_summary_text.delta\ndata: {\"delta\":\"working\"}\n\n" +
		"event: response.output_text.delta\ndata: {\"delta\":\"world\"}\n\n" +
		"event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n"
	backendURL, sent := callBackend(t, http.StatusOK, stream)
	t.Setenv("RESPONSES_BASE_URL", backendURL)

	code, stdout, stderr := executeCLI([]string{
		"--config", validConfig(t), "call", "plain", "--input", "say hi",
	})
	if code != exitSuccess {
		t.Fatalf("exit code = %d: %q", code, stderr)
	}
	if stdout != "hello world" {
		t.Errorf("stdout = %q, want %q", stdout, "hello world")
	}

	var body map[string]any
	if err := json.Unmarshal(*sent, &body); err != nil {
		t.Fatalf("decode forwarded body %s: %v", *sent, err)
	}
	want := map[string]any{
		"model":        "openai/upstream-fast",
		"instructions": "You are plain.",
		"reasoning":    map[string]any{"effort": "low"},
		"stream":       true,
		"input": []any{map[string]any{
			"type": "message", "role": "user", "content": "say hi",
		}},
	}
	if !reflect.DeepEqual(body, want) {
		t.Fatalf("forwarded body = %s\nwant %#v", *sent, want)
	}
}

func TestRunCallReportsStreamFailure(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		stream  string
		wantErr string
	}{
		{
			name:   "the backend failed the response",
			status: http.StatusOK,
			stream: "event: response.output_text.delta\ndata: {\"delta\":\"partial\"}\n\n" +
				"event: response.failed\ndata: {\"response\":{\"status\":\"failed\"," +
				"\"error\":{\"type\":\"limit\",\"message\":\"output truncated\"}}}\n\n",
			wantErr: "backend stream failed: limit: output truncated",
		},
		{
			name:    "the stream stopped without completing",
			status:  http.StatusOK,
			stream:  "event: response.output_text.delta\ndata: {\"delta\":\"partial\"}\n\n",
			wantErr: "ended before response.completed",
		},
		{
			name:    "the backend rejected the request",
			status:  http.StatusBadRequest,
			stream:  `{"error":{"message":"unsupported reasoning.effort"}}`,
			wantErr: "backend status 400",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backendURL, _ := callBackend(t, test.status, test.stream)
			t.Setenv("RESPONSES_BASE_URL", backendURL)
			code, _, stderr := executeCLI([]string{
				"--config", validConfig(t), "call", "plain", "--input", "say hi",
			})
			if code != exitFailure {
				t.Fatalf("exit code = %d, want %d", code, exitFailure)
			}
			if !strings.Contains(stderr, test.wantErr) {
				t.Fatalf("stderr = %q, want it to mention %q", stderr, test.wantErr)
			}
		})
	}
}

func TestRunCallRejectsUnknownAgent(t *testing.T) {
	backendURL, sent := callBackend(t, http.StatusOK, "")
	t.Setenv("RESPONSES_BASE_URL", backendURL)

	code, _, stderr := executeCLI([]string{
		"--config", validConfig(t), "call", "missing", "--input", "say hi",
	})
	if code != exitFailure {
		t.Fatalf("exit code = %d, want %d", code, exitFailure)
	}
	if !strings.Contains(stderr, "unknown agent: missing") {
		t.Errorf("stderr = %q", stderr)
	}
	if *sent != nil {
		t.Errorf("an unresolved agent reached the backend: %s", *sent)
	}
}

// A gateway that assumes a backend fails as a refused connection deep in a request
// rather than where the mistake is, so both commands that reach one refuse at boot.
func TestCommandsRefuseWithoutBackendURL(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "serve", args: []string{"serve"}},
		{name: "call", args: []string{"call", "plain", "--input", "say hi"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("RESPONSES_BASE_URL", "")
			args := append([]string{"--config", validConfig(t)}, test.args...)
			code, _, stderr := executeCLI(args)
			if code != exitFailure {
				t.Fatalf("exit code = %d, want %d", code, exitFailure)
			}
			if !strings.Contains(stderr, "RESPONSES_BASE_URL is required") {
				t.Fatalf("stderr = %q", stderr)
			}
		})
	}
}
