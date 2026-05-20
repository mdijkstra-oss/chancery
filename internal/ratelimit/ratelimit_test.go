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
		{"retryable_with_delay", RetryableWithDelay(fmt.Errorf("429"), 60*time.Second), true},
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

func TestExtractDelay(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want time.Duration
	}{
		{"no_delay", Retryable(fmt.Errorf("429")), 0},
		{"with_delay", RetryableWithDelay(fmt.Errorf("429"), 60*time.Second), 60 * time.Second},
		{"plain_error", fmt.Errorf("500"), 0},
		{"wrapped", fmt.Errorf("outer: %w", RetryableWithDelay(fmt.Errorf("429"), 30*time.Second)), 30 * time.Second},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractDelay(tt.err); got != tt.want {
				t.Errorf("ExtractDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHerdJitter(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
	}{
		{"one_minute", time.Minute},
		{"one_hour", time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spread := tt.d / 10
			for range 100 {
				got := herdJitter(tt.d)
				if got < tt.d || got >= tt.d+spread {
					t.Fatalf("herdJitter(%v) = %v, want [%v, %v)", tt.d, got, tt.d, tt.d+spread)
				}
			}
		})
	}
}

func TestChooseDelay(t *testing.T) {
	t.Run("prefers_server_delay", func(t *testing.T) {
		serverDelay := time.Hour
		got := chooseDelay(serverDelay, 0)
		if got < serverDelay || got >= serverDelay+serverDelay/10 {
			t.Fatalf("chooseDelay with server delay = %v, want [%v, %v)", got, serverDelay, serverDelay+serverDelay/10)
		}
	})

	t.Run("falls_back_to_backoff", func(t *testing.T) {
		got := chooseDelay(0, 0)
		if got >= backoffDelay(0) {
			t.Fatalf("chooseDelay without server delay = %v, want < %v", got, backoffDelay(0))
		}
	})
}

func TestParseRetryAfterHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   time.Duration
	}{
		{"empty", "", 0},
		{"integer_seconds", "60", 60 * time.Second},
		{"fractional", "1.5", 1500 * time.Millisecond},
		{"invalid", "abc", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseRetryAfterHeader(tt.header); got != tt.want {
				t.Errorf("ParseRetryAfterHeader(%q) = %v, want %v", tt.header, got, tt.want)
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
		{
			name:        "quota_exhausted_fails_immediately",
			maxAttempts: 3,
			fn: func(_ *int) (string, error) {
				return "", RetryableWithDelay(fmt.Errorf("quota exceeded"), time.Hour)
			},
			wantErr:   true,
			wantCalls: 1,
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

func TestDoQuotaExhaustedRecordsCooldown(t *testing.T) {
	l := NewLimiter()
	_, _ = Do(context.Background(), l, "model-x", 3, func() (string, error) {
		return "", RetryableWithDelay(fmt.Errorf("quota exceeded"), time.Hour)
	})
	l.mu.Lock()
	deadline, ok := l.cooldown["model-x"]
	l.mu.Unlock()
	if !ok {
		t.Fatal("expected cooldown to be recorded for model-x")
	}
	remaining := time.Until(deadline)
	if remaining < 50*time.Minute {
		t.Fatalf("cooldown remaining = %v, want > 50m", remaining)
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
