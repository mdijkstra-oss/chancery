package responses

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/ratelimit"
)

type recordedRequest struct {
	method string
	path   string
	header http.Header
	body   []byte
}

func recordingBackend(
	t *testing.T,
	status int,
	contentType, body string,
) (*httptest.Server, *recordedRequest) {
	t.Helper()
	recorded := &recordedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		read, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read backend request body: %v", err)
		}
		recorded.method = r.Method
		recorded.path = r.URL.Path
		recorded.header = r.Header.Clone()
		recorded.body = read
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)
	return server, recorded
}

func send(t *testing.T, cfg Config, req Request) (*Response, error) {
	t.Helper()
	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client.Send(context.Background(), req)
}

func TestSendOutboundHeaders(t *testing.T) {
	cases := []struct {
		name      string
		authToken string
		identity  Identity
		want      map[string][]string
	}{{
		name:      "no bearer when the token is unset",
		authToken: "",
		identity:  Identity{RequestID: "abc123"},
		want:      map[string][]string{"Authorization": nil, "X-Request-Id": {"abc123"}},
	}, {
		name:      "bearer when the token is set",
		authToken: "proxy-token",
		identity:  Identity{},
		want:      map[string][]string{"Authorization": {"Bearer proxy-token"}},
	}, {
		name:      "the caller's authorization never travels",
		authToken: "",
		identity: Identity{
			Headers: []Header{{Name: "Authorization", Values: []string{"Bearer callers-jwt"}}},
		},
		want: map[string][]string{"Authorization": nil},
	}, {
		name:      "identity lands on X-* headers",
		authToken: "",
		identity: Identity{
			RequestID: "abc123",
			Agent:     "support/triage",
			Subject:   "user-7",
			Headers:   []Header{{Name: "x-session-id", Values: []string{"s-1", "s-2"}}},
		},
		want: map[string][]string{
			"X-Request-Id": {"abc123"},
			"X-Agent":      {"support/triage"},
			"X-Subject":    {"user-7"},
			"X-Session-Id": {"s-1", "s-2"},
		},
	}, {
		name:      "an empty value is omitted rather than sent blank",
		authToken: "",
		identity:  Identity{RequestID: "abc123", Headers: []Header{{Name: "X-Project-ID"}}},
		want:      map[string][]string{"X-Subject": nil, "X-Agent": nil, "X-Project-Id": nil},
	}, {
		name:      "a header outside the X- prefix is dropped",
		authToken: "",
		identity: Identity{
			Headers: []Header{{Name: "Cookie", Values: []string{"session=1"}}},
		},
		want: map[string][]string{"Cookie": nil},
	}, {
		name:      "a forwarded copy of chancery's own identity is dropped",
		authToken: "",
		identity: Identity{
			RequestID: "abc123",
			Agent:     "support/triage",
			Headers: []Header{
				{Name: "X-Request-ID", Values: []string{"forged"}},
				{Name: "x-agent", Values: []string{"forged"}},
				{Name: "X-Subject", Values: []string{"forged"}},
			},
		},
		want: map[string][]string{
			"X-Request-Id": {"abc123"},
			"X-Agent":      {"support/triage"},
			"X-Subject":    nil,
		},
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, recorded := recordingBackend(t, http.StatusOK, "application/json", "{}")
			resp, err := send(t,
				Config{BaseURL: server.URL, AuthToken: testCase.authToken},
				Request{Body: []byte(`{"input":[]}`), Identity: testCase.identity},
			)
			if err != nil {
				t.Fatalf("Send: %v", err)
			}
			resp.Body.Close()
			for name, want := range testCase.want {
				got := recorded.header.Values(name)
				if len(got) == 0 {
					got = nil
				}
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("header %s: got %v, want %v", name, got, want)
				}
			}
		})
	}
}

func TestSendPostsComposedBody(t *testing.T) {
	cases := []struct {
		name     string
		baseURL  func(string) string
		wantPath string
	}{{
		name:     "base url without a trailing slash",
		baseURL:  func(url string) string { return url },
		wantPath: "/responses",
	}, {
		name:     "base url with a trailing slash",
		baseURL:  func(url string) string { return url + "/" },
		wantPath: "/responses",
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, recorded := recordingBackend(t, http.StatusOK, "application/json", "{}")
			body := `{"input":[{"role":"user","content":"hi"}],"model":"openai/gpt-5.5"}`
			resp, err := send(t, Config{BaseURL: testCase.baseURL(server.URL)},
				Request{Body: []byte(body)})
			if err != nil {
				t.Fatalf("Send: %v", err)
			}
			resp.Body.Close()
			if recorded.method != http.MethodPost {
				t.Fatalf("method: got %s, want POST", recorded.method)
			}
			if recorded.path != testCase.wantPath {
				t.Fatalf("path: got %s, want %s", recorded.path, testCase.wantPath)
			}
			if string(recorded.body) != body {
				t.Fatalf("body: got %s, want %s", recorded.body, body)
			}
			if got := recorded.header.Get("Content-Type"); got != "application/json" {
				t.Fatalf("content type: got %q", got)
			}
		})
	}
}

func TestSendBackendStatus(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		retryAfter  string
		wantErr     bool
		wantRetry   bool
		wantDelay   time.Duration
		wantStatus  int
		wantRelayed int
	}{{
		name:        "a server error relays with its status",
		status:      http.StatusInternalServerError,
		wantRelayed: http.StatusInternalServerError,
	}, {
		name:        "a rejected field relays with its status",
		status:      http.StatusBadRequest,
		wantRelayed: http.StatusBadRequest,
	}, {
		name:       "rate limiting is retryable",
		status:     http.StatusTooManyRequests,
		wantErr:    true,
		wantRetry:  true,
		wantStatus: http.StatusTooManyRequests,
	}, {
		name:       "rate limiting carries the server's delay",
		status:     http.StatusTooManyRequests,
		retryAfter: "12",
		wantErr:    true,
		wantRetry:  true,
		wantDelay:  12 * time.Second,
		wantStatus: http.StatusTooManyRequests,
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if testCase.retryAfter != "" {
					w.Header().Set("Retry-After", testCase.retryAfter)
				}
				w.WriteHeader(testCase.status)
				_, _ = io.WriteString(w, `{"error":{"message":"nope"}}`)
			}))
			defer server.Close()

			resp, err := send(t, Config{BaseURL: server.URL}, Request{Body: []byte(`{"input":[]}`)})
			if !testCase.wantErr {
				if err != nil {
					t.Fatalf("Send: %v", err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != testCase.wantRelayed {
					t.Fatalf("status: got %d, want %d", resp.StatusCode, testCase.wantRelayed)
				}
				return
			}
			if err == nil {
				t.Fatal("want an error, got none")
			}
			if ratelimit.IsRetryable(err) != testCase.wantRetry {
				t.Fatalf("retryable: got %v, want %v", ratelimit.IsRetryable(err), testCase.wantRetry)
			}
			if got := ratelimit.ExtractDelay(err); got != testCase.wantDelay {
				t.Fatalf("delay: got %v, want %v", got, testCase.wantDelay)
			}
			if got := StatusFor(err); got != testCase.wantStatus {
				t.Fatalf("status: got %d, want %d", got, testCase.wantStatus)
			}
		})
	}
}

func TestSendUnreachableBackend(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	address := server.URL
	server.Close()

	start := time.Now()
	_, err := send(t, Config{BaseURL: address}, Request{Body: []byte(`{"input":[]}`)})
	if err == nil {
		t.Fatal("want an error, got none")
	}
	var unreachable *UnreachableError
	if !errors.As(err, &unreachable) {
		t.Fatalf("error %v is not an UnreachableError", err)
	}
	if got := StatusFor(err); got != http.StatusServiceUnavailable {
		t.Fatalf("status: got %d, want 503", got)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("a refused connection took %v to report", elapsed)
	}
}

// A failure chancery cannot attribute to the backend's answer is still the backend's,
// so it is a gateway error rather than the caller's fault.
func TestStatusFor(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{name: "unreachable", err: &UnreachableError{Err: errors.New("refused")}, want: 503},
		{name: "wrapped unreachable", err: ratelimit.Retryable(
			&UnreachableError{Err: errors.New("timeout")}), want: 503},
		{name: "status the backend gave", err: &StatusError{Status: 429, Body: "slow down"}, want: 429},
		{name: "wrapped status", err: ratelimit.Retryable(
			&StatusError{Status: 503, Body: "draining"}), want: 503},
		{name: "anything else", err: errors.New("create request: bad method"), want: 502},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := StatusFor(testCase.err); got != testCase.want {
				t.Fatalf("StatusFor(%v) = %d, want %d", testCase.err, got, testCase.want)
			}
		})
	}
}

func TestNewClientRejectsUnusableBaseURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
	}{
		{name: "empty", baseURL: ""},
		{name: "host and port with no scheme", baseURL: "backend:8080"},
		{name: "no host", baseURL: "http://"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := NewClient(Config{BaseURL: testCase.baseURL})
			if err == nil {
				t.Fatal("want an error, got none")
			}
			if !strings.Contains(err.Error(), "RESPONSES_BASE_URL") {
				t.Fatalf("error %q does not name the variable", err)
			}
		})
	}
}
