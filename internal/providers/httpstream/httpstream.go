package httpstream

import (
	"bufio"
	"fmt"
	"io"
	"net/http"

	"github.com/matthijn/hermes-logos/internal/providers/httpx"
	"github.com/matthijn/hermes-logos/internal/providers/sse"
	"github.com/matthijn/hermes-logos/internal/ratelimit"
)

func Open(w io.Writer, req *http.Request, name string) (*bufio.Scanner, io.Closer, error) {
	resp, err := httpx.Client.Do(req)
	if err != nil {
		if httpx.IsConnectTimeout(err) {
			return nil, nil, ratelimit.Retryable(fmt.Errorf("%s: connect timeout: %w", name, err))
		}
		return nil, nil, fmt.Errorf("%s: send request: %w", name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil, statusError(resp, name)
	}

	sse.SetHeaders(w)
	sse.Flush(w)

	body := httpx.WithStallTimeout(resp.Body, httpx.StallTimeout)
	return sse.NewScanner(body), body, nil
}

func statusError(resp *http.Response, name string) error {
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusTooManyRequests {
		err := fmt.Errorf("%s: rate limited (%d): %s", name, resp.StatusCode, body)
		if delay := ratelimit.ParseRetryAfterHeader(resp.Header.Get("Retry-After")); delay > 0 {
			return ratelimit.RetryableWithDelay(err, delay)
		}
		return ratelimit.Retryable(err)
	}
	return fmt.Errorf("%s: unexpected status %d: %s", name, resp.StatusCode, body)
}
