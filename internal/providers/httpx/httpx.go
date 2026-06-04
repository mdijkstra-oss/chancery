package httpx

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	ConnectTimeout = 45 * time.Second
	StallTimeout   = 90 * time.Second
)

var ErrStreamStall = errors.New("stream stalled: no data within timeout")

var Client = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ResponseHeaderTimeout: ConnectTimeout,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

func IsConnectTimeout(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return false
}

type stallReader struct {
	r       io.ReadCloser
	timer   *time.Timer
	timeout time.Duration
	fired   atomic.Bool
}

func WithStallTimeout(r io.ReadCloser, timeout time.Duration) io.ReadCloser {
	sr := &stallReader{r: r, timeout: timeout}
	sr.timer = time.AfterFunc(timeout, func() {
		sr.fired.Store(true)
		_ = r.Close()
	})
	return sr
}

func (s *stallReader) Read(p []byte) (int, error) {
	s.timer.Reset(s.timeout)
	n, err := s.r.Read(p)
	s.timer.Stop()
	if s.fired.Load() {
		return n, ErrStreamStall
	}
	return n, err
}

func (s *stallReader) Close() error {
	s.timer.Stop()
	return s.r.Close()
}
