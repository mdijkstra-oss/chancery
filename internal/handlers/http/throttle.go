package http

import (
	"context"
	"log/slog"
)

const defaultMaxConcurrent = 5

var upstream = make(chan struct{}, defaultMaxConcurrent)

func init() {
	for range defaultMaxConcurrent {
		upstream <- struct{}{}
	}
}

func acquireUpstream(ctx context.Context) error {
	select {
	case <-upstream:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseUpstream() {
	select {
	case upstream <- struct{}{}:
	default:
		slog.Error("throttle: release without acquire")
	}
}
