package protocol

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestExtractToolNames(t *testing.T) {
	cases := []struct {
		name string
		raw  []json.RawMessage
		want []string
	}{
		{
			name: "extracts names",
			raw: []json.RawMessage{
				json.RawMessage(`{"name":"orientate","type":"function"}`),
				json.RawMessage(`{"name":"run_local_shell","type":"function"}`),
			},
			want: []string{"orientate", "run_local_shell"},
		},
		{
			name: "skips invalid json",
			raw: []json.RawMessage{
				json.RawMessage(`{"name":"foo"}`),
				json.RawMessage(`not json`),
			},
			want: []string{"foo"},
		},
		{
			name: "empty input",
			raw:  nil,
			want: []string{},
		},
		{
			name: "skips missing name",
			raw: []json.RawMessage{
				json.RawMessage(`{"type":"function"}`),
			},
			want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractToolNames(tc.raw)
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
