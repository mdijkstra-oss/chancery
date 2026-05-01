package messages

import (
	"encoding/json"
	"testing"
)

func TestReorderToolMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		want     []string
	}{
		{
			name:     "no reordering needed",
			messages: []string{`{"role":"user","content":"hi"}`, `{"role":"assistant","content":"hello"}`},
			want:     []string{`{"role":"user","content":"hi"}`, `{"role":"assistant","content":"hello"}`},
		},
		{
			name: "system between call and output moved after",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"role":"system","content":"extra context"}`,
				`{"type":"function_call_output","call_id":"c1","output":"found"}`,
			},
			want: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"found"}`,
				`{"role":"system","content":"extra context"}`,
			},
		},
		{
			name: "multiple system messages deferred",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"role":"system","content":"ctx1"}`,
				`{"role":"system","content":"ctx2"}`,
				`{"type":"function_call_output","call_id":"c1","output":"result"}`,
			},
			want: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"result"}`,
				`{"role":"system","content":"ctx1"}`,
				`{"role":"system","content":"ctx2"}`,
			},
		},
		{
			name: "multiple calls with interleaved system",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call","call_id":"c2","name":"read","arguments":"{}"}`,
				`{"role":"system","content":"injected"}`,
				`{"type":"function_call_output","call_id":"c1","output":"r1"}`,
				`{"type":"function_call_output","call_id":"c2","output":"r2"}`,
			},
			want: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call","call_id":"c2","name":"read","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"r1"}`,
				`{"type":"function_call_output","call_id":"c2","output":"r2"}`,
				`{"role":"system","content":"injected"}`,
			},
		},
		{
			name: "no system between calls and outputs unchanged",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"found"}`,
				`{"role":"system","content":"after output"}`,
			},
			want: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"found"}`,
				`{"role":"system","content":"after output"}`,
			},
		},
		{
			name: "system messages mixed with outputs",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"a","arguments":"{}"}`,
				`{"type":"function_call","call_id":"c2","name":"b","arguments":"{}"}`,
				`{"role":"system","content":"ctx"}`,
				`{"type":"function_call_output","call_id":"c1","output":"r1"}`,
				`{"role":"system","content":"ctx2"}`,
				`{"type":"function_call_output","call_id":"c2","output":"r2"}`,
			},
			want: []string{
				`{"type":"function_call","call_id":"c1","name":"a","arguments":"{}"}`,
				`{"type":"function_call","call_id":"c2","name":"b","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"r1"}`,
				`{"type":"function_call_output","call_id":"c2","output":"r2"}`,
				`{"role":"system","content":"ctx"}`,
				`{"role":"system","content":"ctx2"}`,
			},
		},
		{
			name: "user message between call and output deferred",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"role":"user","content":"interruption"}`,
				`{"type":"function_call_output","call_id":"c1","output":"found"}`,
			},
			want: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"found"}`,
				`{"role":"user","content":"interruption"}`,
			},
		},
		{
			name: "mixed message types between calls and outputs deferred",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"a","arguments":"{}"}`,
				`{"type":"function_call","call_id":"c2","name":"b","arguments":"{}"}`,
				`{"role":"user","content":"u1"}`,
				`{"role":"system","content":"s1"}`,
				`{"type":"function_call_output","call_id":"c1","output":"r1"}`,
				`{"role":"assistant","content":"a1"}`,
				`{"type":"function_call_output","call_id":"c2","output":"r2"}`,
			},
			want: []string{
				`{"type":"function_call","call_id":"c1","name":"a","arguments":"{}"}`,
				`{"type":"function_call","call_id":"c2","name":"b","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"r1"}`,
				`{"type":"function_call_output","call_id":"c2","output":"r2"}`,
				`{"role":"user","content":"u1"}`,
				`{"role":"system","content":"s1"}`,
				`{"role":"assistant","content":"a1"}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReorderToolMessages(toRaw(tt.messages))
			want := toRaw(tt.want)
			if len(got) != len(want) {
				t.Fatalf("len mismatch: got %d, want %d", len(got), len(want))
			}
			for i := range want {
				assertJSONEqual(t, string(want[i]), string(got[i]))
			}
		})
	}
}

func toRaw(strs []string) []json.RawMessage {
	msgs := make([]json.RawMessage, len(strs))
	for i, s := range strs {
		msgs[i] = json.RawMessage(s)
	}
	return msgs
}

func assertJSONEqual(t *testing.T, want, got string) {
	t.Helper()
	var wantObj, gotObj any
	if json.Unmarshal([]byte(want), &wantObj) != nil {
		t.Fatalf("invalid want json: %s", want)
	}
	if json.Unmarshal([]byte(got), &gotObj) != nil {
		t.Fatalf("invalid got json: %s", got)
	}
	wantNorm, _ := json.Marshal(wantObj)
	gotNorm, _ := json.Marshal(gotObj)
	if string(wantNorm) != string(gotNorm) {
		t.Errorf("mismatch:\nwant: %s\ngot:  %s", string(wantNorm), string(gotNorm))
	}
}
