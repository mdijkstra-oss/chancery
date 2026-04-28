package completions

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"hermes-logos/internal/protocol"
)

func TestMessagesToCompletions(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt string
		messages     []string
		want         []CompletionsMessage
	}{
		{
			name:         "simple conversation",
			systemPrompt: "be helpful",
			messages: []string{
				`{"role":"user","content":"hello"}`,
				`{"role":"assistant","content":"hi there"}`,
			},
			want: []CompletionsMessage{
				{Role: "system", Content: "be helpful"},
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi there"},
			},
		},
		{
			name:         "no system prompt",
			systemPrompt: "",
			messages: []string{
				`{"role":"user","content":"hello"}`,
			},
			want: []CompletionsMessage{
				{Role: "user", Content: "hello"},
			},
		},
		{
			name:         "function call grouping",
			systemPrompt: "",
			messages: []string{
				`{"role":"assistant","content":"let me search"}`,
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{\"q\":\"test\"}"}`,
				`{"type":"function_call","call_id":"c2","name":"read","arguments":"{\"f\":\"a.txt\"}"}`,
			},
			want: []CompletionsMessage{
				{
					Role:    "assistant",
					Content: "let me search",
					ToolCalls: []ToolCallEntry{
						{ID: "c1", Type: "function", Function: ToolCallFunction{Name: "search", Arguments: `{"q":"test"}`}},
						{ID: "c2", Type: "function", Function: ToolCallFunction{Name: "read", Arguments: `{"f":"a.txt"}`}},
					},
				},
			},
		},
		{
			name:         "function calls without preceding assistant",
			systemPrompt: "",
			messages: []string{
				`{"role":"user","content":"do it"}`,
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
			},
			want: []CompletionsMessage{
				{Role: "user", Content: "do it"},
				{
					Role: "assistant",
					ToolCalls: []ToolCallEntry{
						{ID: "c1", Type: "function", Function: ToolCallFunction{Name: "search", Arguments: "{}"}},
					},
				},
			},
		},
		{
			name:         "function call output",
			systemPrompt: "",
			messages: []string{
				`{"type":"function_call_output","call_id":"c1","output":"found it"}`,
			},
			want: []CompletionsMessage{
				{Role: "tool", Content: "found it", ToolCallID: "c1"},
			},
		},
		{
			name:         "reasoning attached to assistant",
			systemPrompt: "",
			messages: []string{
				`{"type":"reasoning","id":"r1","extra_content":{"deepseek":{"reasoning_content":"thinking hard"}}}`,
				`{"role":"assistant","content":"answer"}`,
			},
			want: []CompletionsMessage{
				{Role: "assistant", Content: "answer", ReasoningContent: "thinking hard"},
			},
		},
		{
			name:         "reasoning without extra_content ignored",
			systemPrompt: "",
			messages: []string{
				`{"type":"reasoning","id":"r1"}`,
				`{"role":"assistant","content":"answer"}`,
			},
			want: []CompletionsMessage{
				{Role: "assistant", Content: "answer"},
			},
		},
		{
			name:         "reasoning attached to function calls",
			systemPrompt: "",
			messages: []string{
				`{"type":"reasoning","id":"r1","extra_content":{"deepseek":{"reasoning_content":"let me think"}}}`,
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
			},
			want: []CompletionsMessage{
				{
					Role:             "assistant",
					ReasoningContent: "let me think",
					ToolCalls: []ToolCallEntry{
						{ID: "c1", Type: "function", Function: ToolCallFunction{Name: "search", Arguments: "{}"}},
					},
				},
			},
		},
		{
			name:         "reasoning attached to assistant with function calls",
			systemPrompt: "",
			messages: []string{
				`{"type":"reasoning","id":"r1","extra_content":{"deepseek":{"reasoning_content":"deep thought"}}}`,
				`{"role":"assistant","content":"searching"}`,
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"found"}`,
			},
			want: []CompletionsMessage{
				{
					Role:             "assistant",
					Content:          "searching",
					ReasoningContent: "deep thought",
					ToolCalls: []ToolCallEntry{
						{ID: "c1", Type: "function", Function: ToolCallFunction{Name: "search", Arguments: "{}"}},
					},
				},
				{Role: "tool", Content: "found", ToolCallID: "c1"},
			},
		},
		{
			name:         "system messages in conversation",
			systemPrompt: "base",
			messages: []string{
				`{"role":"system","content":"extra context"}`,
				`{"role":"user","content":"hello"}`,
			},
			want: []CompletionsMessage{
				{Role: "system", Content: "base"},
				{Role: "system", Content: "extra context"},
				{Role: "user", Content: "hello"},
			},
		},
		{
			name:         "assistant already has content then function calls attach",
			systemPrompt: "",
			messages: []string{
				`{"role":"assistant","content":"thinking"}`,
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
				`{"type":"function_call_output","call_id":"c1","output":"result"}`,
			},
			want: []CompletionsMessage{
				{
					Role:    "assistant",
					Content: "thinking",
					ToolCalls: []ToolCallEntry{
						{ID: "c1", Type: "function", Function: ToolCallFunction{Name: "search", Arguments: "{}"}},
					},
				},
				{Role: "tool", Content: "result", ToolCallID: "c1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := toRawMessages(tt.messages)
			got := MessagesToCompletions(tt.systemPrompt, msgs)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestToolsToCompletions(t *testing.T) {
	tests := []struct {
		name   string
		tools  []string
		strict bool
		want   []CompletionsTool
	}{
		{
			name:  "empty",
			tools: nil,
			want:  nil,
		},
		{
			name: "single tool no strict",
			tools: []string{
				`{"name":"search","description":"Search the web","parameters":{"type":"object","properties":{"q":{"type":"string"}}}}`,
			},
			strict: false,
			want: []CompletionsTool{
				{
					Type: "function",
					Function: CompletionsToolDef{
						Name:        "search",
						Description: "Search the web",
						Parameters:  json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`),
					},
				},
			},
		},
		{
			name: "single tool strict empty params omitted",
			tools: []string{
				`{"name":"search","description":"Search","parameters":{"type":"object"}}`,
			},
			strict: true,
			want: []CompletionsTool{
				{
					Type: "function",
					Function: CompletionsToolDef{
						Name:        "search",
						Description: "Search",
						Parameters:  nil,
						Strict:      boolPtr(true),
					},
				},
			},
		},
		{
			name: "multiple tools",
			tools: []string{
				`{"name":"search","description":"Search"}`,
				`{"name":"read","description":"Read file"}`,
			},
			strict: false,
			want: []CompletionsTool{
				{Type: "function", Function: CompletionsToolDef{Name: "search", Description: "Search"}},
				{Type: "function", Function: CompletionsToolDef{Name: "read", Description: "Read file"}},
			},
		},
		{
			name: "invalid tool skipped",
			tools: []string{
				`{"name":"","description":"no name"}`,
				`{"name":"valid","description":"ok"}`,
			},
			strict: false,
			want: []CompletionsTool{
				{Type: "function", Function: CompletionsToolDef{Name: "valid", Description: "ok"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToolsToCompletions(toRawMessages(tt.tools), tt.strict)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestBuildRequest(t *testing.T) {
	temp := 0.7
	tests := []struct {
		name   string
		params protocol.RequestParams
		strict bool
		check  func(t *testing.T, req CompletionsRequest)
	}{
		{
			name: "full request assembly",
			params: protocol.RequestParams{
				Model:        "deepseek-v4-flash",
				SystemPrompt: "be helpful",
				Temperature:  &temp,
				ToolChoice:   "auto",
				Tools: []json.RawMessage{
					json.RawMessage(`{"name":"search","description":"Search"}`),
				},
				Messages: []json.RawMessage{
					json.RawMessage(`{"role":"user","content":"hello"}`),
				},
			},
			strict: true,
			check: func(t *testing.T, req CompletionsRequest) {
				if req.Model != "deepseek-v4-flash" {
					t.Errorf("model = %q", req.Model)
				}
				if !req.Stream {
					t.Error("expected stream=true")
				}
				if req.StreamOptions == nil || !req.StreamOptions.IncludeUsage {
					t.Error("expected stream_options.include_usage=true")
				}
				if req.Temperature == nil || *req.Temperature != 0.7 {
					t.Errorf("temperature = %v", req.Temperature)
				}
				if req.ToolChoice != "auto" {
					t.Errorf("tool_choice = %q", req.ToolChoice)
				}
				if len(req.Messages) != 2 {
					t.Fatalf("messages len = %d, want 2", len(req.Messages))
				}
				if req.Messages[0].Role != "system" {
					t.Errorf("messages[0].role = %q", req.Messages[0].Role)
				}
				if len(req.Tools) != 1 {
					t.Fatalf("tools len = %d", len(req.Tools))
				}
				if req.Tools[0].Function.Strict == nil || !*req.Tools[0].Function.Strict {
					t.Error("expected strict=true on tool")
				}
				if req.ParallelToolCalls == nil || !*req.ParallelToolCalls {
					t.Error("expected parallel_tool_calls=true")
				}
			},
		},
		{
			name: "no tools means no tool_choice",
			params: protocol.RequestParams{
				Model:      "deepseek-v4-flash",
				ToolChoice: "auto",
				Messages: []json.RawMessage{
					json.RawMessage(`{"role":"user","content":"hello"}`),
				},
			},
			strict: false,
			check: func(t *testing.T, req CompletionsRequest) {
				if req.ToolChoice != "" {
					t.Errorf("tool_choice = %q, want empty", req.ToolChoice)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := BuildRequest(tt.params, tt.strict)
			tt.check(t, req)
		})
	}
}

func toRawMessages(strs []string) []json.RawMessage {
	msgs := make([]json.RawMessage, len(strs))
	for i, s := range strs {
		msgs[i] = json.RawMessage(s)
	}
	return msgs
}

func boolPtr(b bool) *bool { return &b }

func TestSanitizeSchema(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "no format unchanged",
			in:   `{"type":"object","properties":{"name":{"type":"string"}}}`,
			want: `{"type":"object","properties":{"name":{"type":"string"}}}`,
		},
		{
			name: "supported format preserved",
			in:   `{"type":"string","format":"email"}`,
			want: `{"type":"string","format":"email"}`,
		},
		{
			name: "unsupported format stripped",
			in:   `{"type":"string","format":"date"}`,
			want: `{"type":"string"}`,
		},
		{
			name: "nested property format stripped",
			in:   `{"type":"object","properties":{"d":{"type":"string","format":"date"},"e":{"type":"string","format":"email"}}}`,
			want: `{"type":"object","properties":{"d":{"type":"string"},"e":{"type":"string","format":"email"}}}`,
		},
		{
			name: "anyOf variant format stripped",
			in:   `{"anyOf":[{"type":"string","format":"date"},{"type":"null"}]}`,
			want: `{"anyOf":[{"type":"string"},{"type":"null"}]}`,
		},
		{
			name: "deeply nested format stripped",
			in:   `{"type":"object","properties":{"nested":{"type":"object","properties":{"deep":{"type":"string","format":"date-time"}}}}}`,
			want: `{"type":"object","properties":{"nested":{"type":"object","properties":{"deep":{"type":"string"}}}}}`,
		},
		{
			name: "items format stripped",
			in:   `{"type":"array","items":{"type":"string","format":"date"}}`,
			want: `{"type":"array","items":{"type":"string"}}`,
		},
		{
			name: "nil returns nil",
			in:   "",
			want: "",
		},
		{
			name: "uuid format preserved",
			in:   `{"type":"string","format":"uuid"}`,
			want: `{"type":"string","format":"uuid"}`,
		},
		{
			name: "uri format stripped",
			in:   `{"type":"string","format":"uri"}`,
			want: `{"type":"string"}`,
		},
		{
			name: "empty object returns nil",
			in:   `{"type":"object","properties":{},"required":[],"additionalProperties":false}`,
			want: "",
		},
		{
			name: "object with no properties key returns nil",
			in:   `{"type":"object","additionalProperties":false}`,
			want: "",
		},
		{
			name: "bare object type returns nil",
			in:   `{"type":"object"}`,
			want: "",
		},
		{
			name: "object with properties preserved",
			in:   `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}`,
			want: `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}`,
		},
		{
			name: "nested empty object property removed",
			in:   `{"type":"object","properties":{"meta":{"type":"object","properties":{},"additionalProperties":false},"name":{"type":"string"}}}`,
			want: `{"type":"object","properties":{"name":{"type":"string"}}}`,
		},
		{
			name: "nested empty object removed from required",
			in:   `{"type":"object","properties":{"meta":{"type":"object","properties":{}},"name":{"type":"string"}},"required":["meta","name"],"additionalProperties":false}`,
			want: `{"type":"object","properties":{"name":{"type":"string"}},"required":["name"],"additionalProperties":false}`,
		},
		{
			name: "empty object filtered from anyOf",
			in:   `{"anyOf":[{"type":"object","properties":{}},{"type":"null"}]}`,
			want: `{"anyOf":[{"type":"null"}]}`,
		},
		{
			name: "non-object type unchanged",
			in:   `{"type":"string"}`,
			want: `{"type":"string"}`,
		},
		{
			name: "string minLength stripped",
			in:   `{"type":"string","minLength":1}`,
			want: `{"type":"string"}`,
		},
		{
			name: "string maxLength stripped",
			in:   `{"type":"string","maxLength":100}`,
			want: `{"type":"string"}`,
		},
		{
			name: "array minItems stripped",
			in:   `{"type":"array","items":{"type":"string"},"minItems":1}`,
			want: `{"type":"array","items":{"type":"string"}}`,
		},
		{
			name: "array maxItems stripped",
			in:   `{"type":"array","items":{"type":"string"},"maxItems":10}`,
			want: `{"type":"array","items":{"type":"string"}}`,
		},
		{
			name: "nested string constraints stripped",
			in:   `{"type":"object","properties":{"name":{"type":"string","minLength":1,"maxLength":50}}}`,
			want: `{"type":"object","properties":{"name":{"type":"string"}}}`,
		},
		{
			name: "nested anyOf flattened and deduped",
			in:   `{"anyOf":[{"anyOf":[{"type":"array","items":{"type":"string"}},{"type":"null"}]},{"type":"null"}]}`,
			want: `{"anyOf":[{"type":"array","items":{"type":"string"}},{"type":"null"}]}`,
		},
		{
			name: "described anyOf wrapper flattened",
			in:   `{"anyOf":[{"anyOf":[{"type":"string"},{"type":"object","properties":{"f":{"type":"string"}},"required":["f"],"additionalProperties":false}],"description":"tagged"},{"type":"null"}]}`,
			want: `{"anyOf":[{"type":"string"},{"type":"object","properties":{"f":{"type":"string"}},"required":["f"],"additionalProperties":false},{"type":"null"}]}`,
		},
		{
			name: "variant with type and anyOf preserved",
			in:   `{"anyOf":[{"type":"string","anyOf":[{"type":"string"}]},{"type":"null"}]}`,
			want: `{"anyOf":[{"type":"string","anyOf":[{"type":"string"}]},{"type":"null"}]}`,
		},
		{
			name: "real chart series pattern flattened",
			in:   `{"anyOf":[{"description":"Optional series field","anyOf":[{"type":"string","minLength":1},{"type":"object","properties":{"field":{"type":"string"}},"required":["field"],"additionalProperties":false}]},{"type":"null"}]}`,
			want: `{"anyOf":[{"type":"string"},{"type":"object","properties":{"field":{"type":"string"}},"required":["field"],"additionalProperties":false},{"type":"null"}]}`,
		},
		{
			name: "real ask options pattern flattened",
			in:   `{"anyOf":[{"description":"Concrete choices","anyOf":[{"minItems":2,"type":"array","items":{"type":"string"}},{"type":"null"}]},{"type":"null"}]}`,
			want: `{"anyOf":[{"type":"array","items":{"type":"string"}},{"type":"null"}]}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var in json.RawMessage
			if tt.in != "" {
				in = json.RawMessage(tt.in)
			}
			got := sanitizeSchema(in)
			if tt.want == "" {
				if got != nil {
					t.Errorf("want nil, got %s", string(got))
				}
				return
			}
			assertJSONEqual(t, tt.want, string(got))
		})
	}
}

func TestToolsToCompletionsStripsFormat(t *testing.T) {
	tools := []string{
		`{"name":"create_doc","description":"Create document","parameters":{"type":"object","properties":{"date":{"anyOf":[{"type":"string","format":"date"},{"type":"null"}]},"title":{"type":"string"}}}}`,
	}
	got := ToolsToCompletions(toRawMessages(tools), false)
	if len(got) != 1 {
		t.Fatalf("want 1 tool, got %d", len(got))
	}
	params := string(got[0].Function.Parameters)
	if strings.Contains(params, `"format"`) {
		t.Errorf("expected format stripped, got %s", params)
	}
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
