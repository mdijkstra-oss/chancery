package responses

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

const (
	// StallTimeout bounds the gap between reads on a started stream. Response headers
	// are already bounded by the client's own timeout; after them nothing else does.
	StallTimeout    = 90 * time.Second
	relayChunkBytes = 32 << 10
)

var ErrStreamStall = errors.New("backend stream stalled: no data within timeout")

// Relay copies an answer through without reading it. The event sequence the backend
// emits is the one the caller expects, so nothing between them is decoded.
func Relay(w http.ResponseWriter, resp *Response) error {
	if isEventStream(resp.ContentType) {
		return relayStream(w, resp)
	}
	return relayBody(w, resp)
}

// Each read is written and flushed on its own, so an event reaches the caller as soon
// as the backend emits it rather than waiting on the next one.
func relayStream(w http.ResponseWriter, resp *Response) error {
	w.Header().Set("Content-Type", resp.ContentType)
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(resp.StatusCode)
	flush(w)

	body := WithStallTimeout(resp.Body, StallTimeout)
	defer body.Close()

	buffer := make([]byte, relayChunkBytes)
	for {
		read, err := body.Read(buffer)
		if read > 0 {
			if _, writeErr := w.Write(buffer[:read]); writeErr != nil {
				return fmt.Errorf("write stream: %w", writeErr)
			}
			flush(w)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read stream: %w", err)
		}
	}
}

func relayBody(w http.ResponseWriter, resp *Response) error {
	if resp.ContentType != "" {
		w.Header().Set("Content-Type", resp.ContentType)
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		return fmt.Errorf("write body: %w", err)
	}
	return nil
}

func isEventStream(contentType string) bool {
	return strings.HasPrefix(contentType, "text/event-stream")
}

func flush(w http.ResponseWriter) {
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// A backend that accepts a connection and then goes silent would otherwise hold the
// caller open forever.
type stallReader struct {
	inner   io.ReadCloser
	timer   *time.Timer
	timeout time.Duration
	fired   atomic.Bool
}

// WithStallTimeout ends a stream that goes quiet, on whichever path is reading it:
// a caller held open by a silent backend and a terminal held open by one are the same
// failure.
func WithStallTimeout(inner io.ReadCloser, timeout time.Duration) io.ReadCloser {
	reader := &stallReader{inner: inner, timeout: timeout}
	reader.timer = time.AfterFunc(timeout, func() {
		reader.fired.Store(true)
		_ = inner.Close()
	})
	return reader
}

func (s *stallReader) Read(p []byte) (int, error) {
	s.timer.Reset(s.timeout)
	read, err := s.inner.Read(p)
	s.timer.Stop()
	if s.fired.Load() {
		return read, ErrStreamStall
	}
	return read, err
}

func (s *stallReader) Close() error {
	s.timer.Stop()
	return s.inner.Close()
}
