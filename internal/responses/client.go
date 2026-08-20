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
	BaseURL       string
	AuthToken     string
	GatewayHeader string
	GatewayToken  string
}

type Client struct {
	url           string
	authToken     string
	gatewayHeader string
	gatewayToken  string
	http          *http.Client
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
	gatewayHeader, err := gatewayHeaderName(cfg.GatewayHeader, cfg.GatewayToken)
	if err != nil {
		return nil, err
	}
	return &Client{
		url:           strings.TrimSuffix(cfg.BaseURL, "/") + "/responses",
		authToken:     cfg.AuthToken,
		gatewayHeader: gatewayHeader,
		gatewayToken:  cfg.GatewayToken,
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
	if c.gatewayHeader != "" {
		httpReq.Header.Set(c.gatewayHeader, c.gatewayToken)
	}
	c.setIdentity(httpReq.Header, req.Identity)

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

func (c *Client) setIdentity(header http.Header, identity Identity) {
	for name, values := range identity.Headers {
		canonical := http.CanonicalHeaderKey(name)
		if _, reserved := reservedHeaders[canonical]; reserved {
			continue
		}
		if c.gatewayHeader != "" && canonical == c.gatewayHeader {
			continue
		}
		header[canonical] = slices.Clone(values)
	}
	setIfPresent(header, requestIDHeader, identity.RequestID)
	setIfPresent(header, agentHeader, identity.Agent)
	setIfPresent(header, subjectHeader, identity.Subject)
}

// Whatever stands in front of the backend names the header its credential arrives
// under, and the names disagree: X-Auth-Token on Scaleway, Authorization on Cloud Run.
// Half a pair is a deployment that would either send an unauthenticated request or
// hold a credential it never sends, so it is refused at boot rather than at the first
// call.
func gatewayHeaderName(name, token string) (string, error) {
	switch {
	case name == "" && token == "":
		return "", nil
	case name == "":
		return "", errors.New("RESPONSES_GATEWAY_TOKEN is set without RESPONSES_GATEWAY_HEADER")
	case token == "":
		return "", errors.New("RESPONSES_GATEWAY_HEADER is set without RESPONSES_GATEWAY_TOKEN")
	}
	if !validHeaderName(name) {
		return "", fmt.Errorf("RESPONSES_GATEWAY_HEADER %q is not a header name", name)
	}
	return http.CanonicalHeaderKey(name), nil
}

// RFC 9110 token characters. Anything else reaches the wire verbatim, where a colon
// or a newline is a second header of the caller's choosing.
func validHeaderName(name string) bool {
	const special = "!#$%&'*+-.^_`|~"
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case strings.IndexByte(special, c) >= 0:
		default:
			return false
		}
	}
	return name != ""
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
