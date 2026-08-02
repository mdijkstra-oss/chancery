package responses

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

type flushRecorder struct {
	header  http.Header
	status  int
	body    bytes.Buffer
	flushed chan []byte
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{header: http.Header{}, flushed: make(chan []byte, 32)}
}

func (f *flushRecorder) Header() http.Header         { return f.header }
func (f *flushRecorder) Write(p []byte) (int, error) { return f.body.Write(p) }
func (f *flushRecorder) WriteHeader(status int)      { f.status = status }
func (f *flushRecorder) Flush()                      { f.flushed <- slices.Clone(f.body.Bytes()) }

func waitForFlush(t *testing.T, recorder *flushRecorder) string {
	t.Helper()
	select {
	case flushed := <-recorder.flushed:
		return string(flushed)
	case <-time.After(2 * time.Second):
		t.Fatal("no flush within 2s")
		return ""
	}
}

func TestRelayStreamFlushesPerEvent(t *testing.T) {
	events := []string{
		"event: response.created\ndata: {\"type\":\"response.created\"}\n\n",
		"event: response.output_text.delta\ndata: {\"delta\":\"hi\"}\n\n",
		"event: response.completed\ndata: {\"type\":\"response.completed\"}\n\n",
	}
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		for _, event := range events {
			<-release
			_, _ = io.WriteString(w, event)
			w.(http.Flusher).Flush()
		}
	}))
	defer server.Close()

	resp, err := send(t, Config{BaseURL: server.URL}, Request{Body: []byte(`{"input":[]}`)})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	defer resp.Body.Close()

	recorder := newFlushRecorder()
	done := make(chan error, 1)
	go func() { done <- Relay(recorder, resp) }()

	waitForFlush(t, recorder)
	relayed := ""
	for index, event := range events {
		release <- struct{}{}
		relayed += event
		if got := waitForFlush(t, recorder); got != relayed {
			t.Fatalf("event %d: got %q, want %q", index, got, relayed)
		}
	}

	if err := <-done; err != nil {
		t.Fatalf("Relay: %v", err)
	}
	if recorder.status != http.StatusOK {
		t.Fatalf("status: got %d, want 200", recorder.status)
	}
	if got := recorder.header.Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("content type: got %q", got)
	}
	if got := recorder.header.Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("cache control: got %q", got)
	}
}

func TestRelayStreamCutShort(t *testing.T) {
	partial := "event: response.output_text.delta\ndata: {\"delta\":\"hi\"}\n\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, partial)
		w.(http.Flusher).Flush()
		panic(http.ErrAbortHandler)
	}))
	defer server.Close()

	resp, err := send(t, Config{BaseURL: server.URL}, Request{Body: []byte(`{"input":[]}`)})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	defer resp.Body.Close()

	recorder := newFlushRecorder()
	err = Relay(recorder, resp)
	if err == nil {
		t.Fatal("want an error, got none")
	}
	if got := recorder.body.String(); got != partial {
		t.Fatalf("relayed %q, want %q", got, partial)
	}
	if strings.Contains(recorder.body.String(), "response.completed") {
		t.Fatal("a cut stream must not carry a completion")
	}
}

// A backend that accepts the connection and then goes silent would otherwise hold the
// caller open for as long as it stayed quiet.
func TestStallTimeoutEndsASilentStream(t *testing.T) {
	silent := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "event: response.created\ndata: {}\n\n")
		w.(http.Flusher).Flush()
		<-silent
	}))
	defer func() {
		close(silent)
		server.Close()
	}()

	resp, err := send(t, Config{BaseURL: server.URL}, Request{Body: []byte(`{"input":[]}`)})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	body := WithStallTimeout(resp.Body, 100*time.Millisecond)
	defer body.Close()

	buffer := make([]byte, 4096)
	read, err := body.Read(buffer)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if got := string(buffer[:read]); !strings.Contains(got, "response.created") {
		t.Fatalf("first read = %q", got)
	}

	start := time.Now()
	if _, err = body.Read(buffer); !errors.Is(err, ErrStreamStall) {
		t.Fatalf("second read error = %v, want ErrStreamStall", err)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("a silent backend took %v to report", elapsed)
	}
}

func TestRelayBody(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		contentType string
		body        string
	}{{
		name:        "a non-streaming answer",
		status:      http.StatusOK,
		contentType: "application/json",
		body:        `{"id":"resp_1","output":[]}`,
	}, {
		name:        "an error the backend names",
		status:      http.StatusBadRequest,
		contentType: "application/json",
		body:        `{"error":{"message":"unsupported reasoning.effort"}}`,
	}, {
		name:        "an answer with no content type",
		status:      http.StatusBadGateway,
		contentType: "",
		body:        "upstream refused",
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			server, _ := recordingBackend(t, testCase.status, testCase.contentType, testCase.body)
			resp, err := send(t, Config{BaseURL: server.URL}, Request{Body: []byte(`{"input":[]}`)})
			if err != nil {
				t.Fatalf("Send: %v", err)
			}
			defer resp.Body.Close()

			recorder := newFlushRecorder()
			if err := Relay(recorder, resp); err != nil {
				t.Fatalf("Relay: %v", err)
			}
			if recorder.status != testCase.status {
				t.Fatalf("status: got %d, want %d", recorder.status, testCase.status)
			}
			if got := recorder.body.String(); got != testCase.body {
				t.Fatalf("body: got %q, want %q", got, testCase.body)
			}
			if testCase.contentType != "" {
				if got := recorder.header.Get("Content-Type"); got != testCase.contentType {
					t.Fatalf("content type: got %q, want %q", got, testCase.contentType)
				}
			}
		})
	}
}

func TestRelayComposedRequestRoundTrip(t *testing.T) {
	server, recorded := recordingBackend(t, http.StatusOK, "text/event-stream",
		"event: response.completed\ndata: {}\n\n")

	composed, err := Compose([]byte(`{"input":[{"role":"user","content":"hi"}],"stream":true}`),
		Agent{Model: "openai/gpt-5.5", Instructions: "You are a hound."})
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}

	client, err := NewClient(Config{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := client.Send(context.Background(), Request{
		Body:     composed,
		Identity: Identity{RequestID: "abc123", Agent: "support/triage"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	defer resp.Body.Close()

	recorder := newFlushRecorder()
	if err := Relay(recorder, resp); err != nil {
		t.Fatalf("Relay: %v", err)
	}

	assertSameJSON(t, recorded.body, []byte(`{"input":[{"role":"user","content":"hi"}],
		"stream":true,"model":"openai/gpt-5.5","instructions":"You are a hound."}`))
	if got := recorded.header.Get("X-Request-Id"); got != "abc123" {
		t.Fatalf("request id: got %q", got)
	}
	if got := recorder.body.String(); !strings.Contains(got, "response.completed") {
		t.Fatalf("relayed %q", got)
	}
}
