package messages

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestDropEmptyContent(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{
			name: "drops empty system",
			in: []string{
				`{"type":"message","role":"system","content":""}`,
				`{"type":"message","role":"system","content":"keep"}`,
			},
			want: []string{
				`{"type":"message","role":"system","content":"keep"}`,
			},
		},
		{
			name: "drops empty user and assistant",
			in: []string{
				`{"role":"user","content":""}`,
				`{"role":"assistant","content":""}`,
				`{"role":"user","content":"ok"}`,
			},
			want: []string{
				`{"role":"user","content":"ok"}`,
			},
		},
		{
			name: "keeps function_call without content",
			in: []string{
				`{"type":"function_call","name":"foo","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"x","output":"y"}`,
				`{"type":"reasoning","content":""}`,
			},
			want: []string{
				`{"type":"function_call","name":"foo","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"x","output":"y"}`,
				`{"type":"reasoning","content":""}`,
			},
		},
		{
			name: "keeps malformed json untouched",
			in: []string{
				`not json`,
				`{"role":"user","content":"keep"}`,
			},
			want: []string{
				`not json`,
				`{"role":"user","content":"keep"}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DropEmptyContent(toRawMessages(tt.in))
			gotStrs := rawSliceToStrings(got)
			if !reflect.DeepEqual(gotStrs, tt.want) {
				t.Errorf("got %v\nwant %v", gotStrs, tt.want)
			}
		})
	}
}

func toRawMessages(strs []string) []json.RawMessage {
	out := make([]json.RawMessage, len(strs))
	for i, s := range strs {
		out[i] = json.RawMessage(s)
	}
	return out
}
