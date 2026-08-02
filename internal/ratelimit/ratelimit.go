package ratelimit

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"math/rand/v2"
	"strconv"
	"sync"
	"time"
)

type RetryableError struct {
	Err   error
	Delay time.Duration
}

func (e *RetryableError) Error() string { return e.Err.Error() }
func (e *RetryableError) Unwrap() error { return e.Err }

type Limiter struct {
	mu       sync.Mutex
	cooldown map[string]time.Time
}

func Retryable(err error) error {
	return &RetryableError{Err: err}
}

func RetryableWithDelay(err error, delay time.Duration) error {
	return &RetryableError{Err: err, Delay: delay}
}

func IsRetryable(err error) bool {
	var re *RetryableError
	return errors.As(err, &re)
}

func ExtractDelay(err error) time.Duration {
	var re *RetryableError
	if errors.As(err, &re) {
		return re.Delay
	}
	return 0
}

func ParseRetryAfterHeader(header string) time.Duration {
	if header == "" {
		return 0
	}
	secs, err := strconv.ParseFloat(header, 64)
	if err != nil {
		return 0
	}
	return time.Duration(secs * float64(time.Second))
}

func NewLimiter() *Limiter {
	return &Limiter{cooldown: make(map[string]time.Time)}
}

func Do[T any](
	ctx context.Context,
	l *Limiter,
	key string,
	maxAttempts int,
	fn func(context.Context) (T, error),
) (T, error) {
	var lastErr error
	for attempt := range maxAttempts {
		if err := l.waitIfNeeded(ctx, key); err != nil {
			var zero T
			return zero, err
		}
		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !IsRetryable(err) {
			var zero T
			return zero, err
		}
		serverDelay := ExtractDelay(err)
		delay := chooseDelay(serverDelay, attempt)
		l.recordCooldown(key, delay)
		if serverDelay > maxRetryWait {
			slog.InfoContext(ctx, "rate limited, delay exceeds the retry window",
				"component", "ratelimit",
				slog.Group("data",
					slog.String("key", key),
					slog.Duration("server_delay", serverDelay),
					slog.Duration("cooldown", delay),
				),
			)
			var zero T
			return zero, err
		}
		slog.InfoContext(ctx, "rate limited, backing off",
			"component", "ratelimit",
			slog.Group("data",
				slog.String("key", key),
				slog.Int("attempt", attempt+1),
				slog.Duration("delay", delay),
			),
		)
		if err := sleepCtx(ctx, delay); err != nil {
			var zero T
			return zero, err
		}
	}
	var zero T
	return zero, lastErr
}

const maxBackoff = 30 * time.Second
const maxRetryWait = 2 * time.Minute

func backoffDelay(attempt int) time.Duration {
	d := time.Duration(500*math.Pow(2, float64(attempt))) * time.Millisecond
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}

func chooseDelay(serverDelay time.Duration, attempt int) time.Duration {
	if serverDelay > 0 {
		return herdJitter(serverDelay)
	}
	return fullJitter(backoffDelay(attempt))
}

func herdJitter(d time.Duration) time.Duration {
	spread := d / 10
	if spread <= 0 {
		return d
	}
	return d + time.Duration(rand.Int64N(int64(spread)))
}

func fullJitter(d time.Duration) time.Duration {
	if d <= 0 {
		return 0
	}
	return time.Duration(rand.Int64N(int64(d)))
}

func (l *Limiter) waitIfNeeded(ctx context.Context, key string) error {
	l.mu.Lock()
	deadline, ok := l.cooldown[key]
	l.mu.Unlock()
	if !ok {
		return nil
	}
	wait := time.Until(deadline)
	if wait <= 0 {
		return nil
	}
	return sleepCtx(ctx, wait)
}

func (l *Limiter) recordCooldown(key string, delay time.Duration) {
	newDeadline := time.Now().Add(delay)
	l.mu.Lock()
	defer l.mu.Unlock()
	if existing, ok := l.cooldown[key]; ok && existing.After(newDeadline) {
		return
	}
	l.cooldown[key] = newDeadline
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
