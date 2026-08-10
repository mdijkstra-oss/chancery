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
	"slices"
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

// Identity is what chancery knows about a request: its own three labels, and the
// caller's headers to pass along beneath them.
type Identity struct {
	RequestID string
	Agent     string
	Subject   string
	Headers   http.Header
}

type Request struct {
	Body     []byte
	Identity Identity
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
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(req.Body))
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

// Headers that describe this request rather than the caller's, and so cannot be
// carried over from it. The body is recomposed, the connection is this client's, and
// Authorization holds RESPONSES_AUTH_TOKEN. Accept-Encoding is here because a caller
// asking for gzip would get a compressed body relayed on as though it were an event
// stream. The three identity headers are chancery's own: a caller sending its own copy
// would otherwise append to the backend's record.
var reservedHeaders = map[string]struct{}{
	"Authorization":       {},
	"Content-Length":      {},
	"Content-Type":        {},
	"Accept-Encoding":     {},
	"Host":                {},
	"Connection":          {},
	"Keep-Alive":          {},
	"Te":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	requestIDHeader:       {},
	agentHeader:           {},
	subjectHeader:         {},
}

func setIdentity(header http.Header, identity Identity) {
	for name, values := range identity.Headers {
		canonical := http.CanonicalHeaderKey(name)
		if _, reserved := reservedHeaders[canonical]; reserved {
			continue
		}
		header[canonical] = slices.Clone(values)
	}
	setIfPresent(header, requestIDHeader, identity.RequestID)
	setIfPresent(header, agentHeader, identity.Agent)
	setIfPresent(header, subjectHeader, identity.Subject)
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
