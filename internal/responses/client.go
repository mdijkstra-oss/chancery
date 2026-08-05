package responses

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mdijkstra-oss/chancery/internal/ratelimit"
)

const (
	// A backend reaching a slow provider answers late; a backend that never sends
	// headers is one chancery is waiting on for nothing.
	responseHeaderTimeout = 45 * time.Second
	dialTimeout           = 30 * time.Second
	keepAlive             = 30 * time.Second
	maxErrorBodyBytes     = 8 << 10
)

const (
	requestIDHeader = "X-Request-ID"
	agentHeader     = "X-Agent"
	subjectHeader   = "X-Subject"
)

type Config struct {
	BaseURL   string
	AuthToken string
}

type Client struct {
	url       string
	authToken string
	http      *http.Client
}

// Header is one caller header an operator named for the backend's record.
type Header struct {
	Name   string
	Values []string
}

// Identity is what chancery knows about a request. Every field of it is a log label
// downstream and none of it is a credential.
type Identity struct {
	RequestID string
	Agent     string
	Subject   string
	Headers   []Header
}

// PromptCacheBreakpoints is the alias's answer to whether its model accepts explicit
// cache breakpoints. It is not a body field, so it rides on the URL: a directive to
// whatever translates the request, not a value the model is asked to read.
type Request struct {
	Body                   []byte
	Identity               Identity
	PromptCacheBreakpoints *bool
}

// Only an explicit false is sent. Absent means the backend keeps its own default, and
// a backend that has never heard of the parameter ignores it and answers as before.
func (r Request) url(base string) string {
	if r.PromptCacheBreakpoints == nil || *r.PromptCacheBreakpoints {
		return base
	}
	return base + "?prompt_cache_breakpoints=false"
}

// Response is a backend answer whose body has not been read. The caller closes it.
type Response struct {
	StatusCode  int
	ContentType string
	Body        io.ReadCloser
}

// StatusError is a backend answer chancery does not relay, carrying the status the
// caller receives in its place.
type StatusError struct {
	Status int
	Body   string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("backend status %d: %s", e.Status, e.Body)
}

// StatusErrorFrom turns an answer that will not be relayed into the error that stands
// for it, reading enough of the body to carry the backend's reason and no more.
func StatusErrorFrom(resp *Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	if err != nil {
		return fmt.Errorf("read backend status %d body: %w", resp.StatusCode, err)
	}
	return &StatusError{Status: resp.StatusCode, Body: string(body)}
}

// UnreachableError is a backend that never answered. Nothing reached the caller, so
// there is no half-written stream to reconcile.
type UnreachableError struct {
	Err error
}

func (e *UnreachableError) Error() string { return "backend unreachable: " + e.Err.Error() }
func (e *UnreachableError) Unwrap() error { return e.Err }

// StatusFor is the status a caller receives when Send fails.
func StatusFor(err error) int {
	var unreachable *UnreachableError
	if errors.As(err, &unreachable) {
		return http.StatusServiceUnavailable
	}
	var status *StatusError
	if errors.As(err, &status) {
		return status.Status
	}
	return http.StatusBadGateway
}

// NewClient fixes the one URL every request goes to. A base URL that cannot hold a
// path is a boot-time mistake, so it is rejected here rather than per request.
func NewClient(cfg Config) (*Client, error) {
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("RESPONSES_BASE_URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("RESPONSES_BASE_URL %q needs a scheme and a host", cfg.BaseURL)
	}
	return &Client{
		url:       strings.TrimSuffix(cfg.BaseURL, "/") + "/responses",
		authToken: cfg.AuthToken,
		http: &http.Client{
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				ResponseHeaderTimeout: responseHeaderTimeout,
				DialContext: (&net.Dialer{
					Timeout:   dialTimeout,
					KeepAlive: keepAlive,
				}).DialContext,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
	}, nil
}

// Send returns before anything is written to the caller, so a retryable error can be
// retried with the response still unstarted.
func (c *Client) Send(ctx context.Context, req Request) (*Response, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, req.url(c.url), bytes.NewReader(req.Body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.authToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	setIdentity(httpReq.Header, req.Identity)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, unreachable(err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, rateLimited(resp)
	}
	return &Response{
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
		Body:        resp.Body,
	}, nil
}

// The outbound request is built fresh, so the only headers on it are the ones named
// here; requiring the X- prefix keeps that true for a name that arrives configured.
// The three chancery writes are chancery's: forwarding a caller's copy of one would let it
// append to the backend's accounting record, and the request ID is the value that
// makes chancery's log line and the backend's one story.
func setIdentity(header http.Header, identity Identity) {
	setIfPresent(header, requestIDHeader, identity.RequestID)
	setIfPresent(header, agentHeader, identity.Agent)
	setIfPresent(header, subjectHeader, identity.Subject)
	for _, forwarded := range identity.Headers {
		name := http.CanonicalHeaderKey(forwarded.Name)
		if !strings.HasPrefix(name, "X-") || isIdentityHeader(name) {
			continue
		}
		for _, value := range forwarded.Values {
			if value != "" {
				header.Add(name, value)
			}
		}
	}
}

func isIdentityHeader(canonical string) bool {
	switch canonical {
	case http.CanonicalHeaderKey(requestIDHeader),
		http.CanonicalHeaderKey(agentHeader),
		http.CanonicalHeaderKey(subjectHeader):
		return true
	default:
		return false
	}
}

func setIfPresent(header http.Header, name, value string) {
	if value != "" {
		header.Set(name, value)
	}
}

// A backend that did not answer in time may answer the next attempt; a refused
// connection will not.
func unreachable(err error) error {
	wrapped := &UnreachableError{Err: err}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ratelimit.Retryable(wrapped)
	}
	return wrapped
}

func rateLimited(resp *http.Response) error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
	err := &StatusError{Status: resp.StatusCode, Body: string(body)}
	if delay := ratelimit.ParseRetryAfterHeader(resp.Header.Get("Retry-After")); delay > 0 {
		return ratelimit.RetryableWithDelay(err, delay)
	}
	return ratelimit.Retryable(err)
}
