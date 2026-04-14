package openai

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"hermes-logos/internal/protocol"
)

func TestExtractUsageFromCompleted(t *testing.T) {
	tests := []struct {
		name string
		data string
		want *protocol.UsageResponse
	}{
		{
			name: "full usage with details",
			data: `{"response":{"usage":{"input_tokens":100,"output_tokens":50,"total_tokens":150,"input_tokens_details":{"cached_tokens":20},"output_tokens_details":{"reasoning_tokens":10}}}}`,
			want: &protocol.UsageResponse{
				InputTokens:         100,
				OutputTokens:        50,
				TotalTokens:         150,
				InputTokensDetails:  &protocol.PromptTokensDetails{CachedTokens: 20},
				OutputTokensDetails: &protocol.OutputTokensDetails{ReasoningTokens: 10},
			},
		},
		{
			name: "usage without details",
			data: `{"response":{"usage":{"input_tokens":200,"output_tokens":80,"total_tokens":280}}}`,
			want: &protocol.UsageResponse{
				InputTokens:  200,
				OutputTokens: 80,
				TotalTokens:  280,
			},
		},
		{
			name: "null usage",
			data: `{"response":{"usage":null}}`,
			want: nil,
		},
		{
			name: "missing response field",
			data: `{"other":"value"}`,
			want: nil,
		},
		{
			name: "invalid json",
			data: `not json`,
			want: nil,
		},
		{
			name: "empty object",
			data: `{}`,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractUsageFromCompleted([]byte(tt.data))
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
