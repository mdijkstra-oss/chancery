package ratelimit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestBackoffDelay(t *testing.T) {
	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{0, 500 * time.Millisecond},
		{1, 1000 * time.Millisecond},
		{2, 2000 * time.Millisecond},
		{3, 4000 * time.Millisecond},
		{4, 8000 * time.Millisecond},
		{5, 16000 * time.Millisecond},
		{6, 30 * time.Second},
		{10, 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			got := backoffDelay(tt.attempt)
			if got != tt.want {
				t.Errorf("backoffDelay(%d) = %v, want %v", tt.attempt, got, tt.want)
			}
		})
	}
}

func TestFullJitter(t *testing.T) {
	t.Run("zero_returns_zero", func(t *testing.T) {
		if got := fullJitter(0); got != 0 {
			t.Fatalf("fullJitter(0) = %v, want 0", got)
		}
	})

	tests := []struct {
		name  string
		input time.Duration
	}{
		{"one_second", time.Second},
		{"thirty_seconds", 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for range 100 {
				got := fullJitter(tt.input)
				if got < 0 || got >= tt.input {
					t.Fatalf("fullJitter(%v) = %v, want [0, %v)", tt.input, got, tt.input)
				}
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"retryable", Retryable(fmt.Errorf("429")), true},
		{"plain", fmt.Errorf("500"), false},
		{"wrapped_retryable", fmt.Errorf("outer: %w", Retryable(fmt.Errorf("429"))), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDo(t *testing.T) {
	tests := []struct {
		name        string
		maxAttempts int
		fn          func(counter *int) (string, error)
		wantResult  string
		wantErr     bool
		wantCalls   int
	}{
		{
			name:        "success_first_try",
			maxAttempts: 3,
			fn:          func(_ *int) (string, error) { return "ok", nil },
			wantResult:  "ok",
			wantCalls:   1,
		},
		{
			name:        "success_after_retry",
			maxAttempts: 3,
			fn: func(c *int) (string, error) {
				if *c < 3 {
					return "", Retryable(fmt.Errorf("429"))
				}
				return "recovered", nil
			},
			wantResult: "recovered",
			wantCalls:  3,
		},
		{
			name:        "non_retryable_stops",
			maxAttempts: 3,
			fn: func(_ *int) (string, error) {
				return "", fmt.Errorf("fatal")
			},
			wantErr:   true,
			wantCalls: 1,
		},
		{
			name:        "max_attempts_exhausted",
			maxAttempts: 2,
			fn: func(_ *int) (string, error) {
				return "", Retryable(fmt.Errorf("429"))
			},
			wantErr:   true,
			wantCalls: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := NewLimiter()
			counter := 0
			result, err := Do(context.Background(), l, "test", tt.maxAttempts, func() (string, error) {
				counter++
				return tt.fn(&counter)
			})
			if (err != nil) != tt.wantErr {
				t.Fatalf("Do() error = %v, wantErr %v", err, tt.wantErr)
			}
			if result != tt.wantResult {
				t.Errorf("Do() result = %q, want %q", result, tt.wantResult)
			}
			if counter != tt.wantCalls {
				t.Errorf("Do() calls = %d, want %d", counter, tt.wantCalls)
			}
		})
	}
}

func TestDoContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	l := NewLimiter()
	counter := 0
	cancel()
	_, err := Do(ctx, l, "test", 3, func() (string, error) {
		counter++
		return "", Retryable(fmt.Errorf("429"))
	})
	if err != context.Canceled {
		t.Fatalf("Do() error = %v, want context.Canceled", err)
	}
}
