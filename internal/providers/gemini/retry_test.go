package gemini

import (
	"fmt"
	"testing"
	"time"

	"google.golang.org/genai"
)

func TestExtractRetryDelay(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want time.Duration
	}{
		{
			"integer_seconds",
			&genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "4422s"},
				},
			},
			4422 * time.Second,
		},
		{
			"fractional_seconds",
			&genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "1.5s"},
				},
			},
			1500 * time.Millisecond,
		},
		{
			"float64_value",
			&genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": float64(30)},
				},
			},
			30 * time.Second,
		},
		{
			"multiple_details_picks_retry_info",
			&genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"@type": "type.googleapis.com/google.rpc.Help"},
					{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "60s"},
				},
			},
			60 * time.Second,
		},
		{
			"no_retry_info",
			&genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"@type": "type.googleapis.com/google.rpc.Help"},
				},
			},
			0,
		},
		{
			"not_api_error",
			fmt.Errorf("some other error"),
			0,
		},
		{
			"empty_details",
			&genai.APIError{Code: 429},
			0,
		},
		{
			"missing_suffix",
			&genai.APIError{
				Code: 429,
				Details: []map[string]any{
					{"@type": "type.googleapis.com/google.rpc.RetryInfo", "retryDelay": "4422"},
				},
			},
			0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractRetryDelay(tt.err)
			if got != tt.want {
				t.Errorf("ExtractRetryDelay() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormatRetryDelay(t *testing.T) {
	tests := []struct {
		name  string
		input time.Duration
		want  string
	}{
		{"zero", 0, ""},
		{"seconds_only", 45 * time.Second, "45s"},
		{"minutes_and_seconds", 90 * time.Second, "1m30s"},
		{"hours", 4422 * time.Second, "1h13m42s"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRetryDelay(tt.input)
			if got != tt.want {
				t.Errorf("FormatRetryDelay(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
