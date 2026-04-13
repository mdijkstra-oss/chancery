package protocol

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestStripExtraContent(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		hasField bool
	}{
		{
			name:     "function_call with extra_content",
			input:    `{"type":"function_call","call_id":"c1","name":"get","arguments":"{}","extra_content":{"google":{"thought_signature":"abc"}}}`,
			hasField: false,
		},
		{
			name:     "function_call without extra_content",
			input:    `{"type":"function_call","call_id":"c1","name":"get","arguments":"{}"}`,
			hasField: false,
		},
		{
			name:     "message untouched",
			input:    `{"type":"message","role":"user","content":"hello"}`,
			hasField: false,
		},
		{
			name:     "function_call_output untouched",
			input:    `{"type":"function_call_output","call_id":"c1","output":"ok"}`,
			hasField: false,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			msgs := []json.RawMessage{json.RawMessage(tt.input)}
			result := StripExtraContent(msgs)
			if len(result) != 1 {
				t.Fatalf("expected 1 message, got %d", len(result))
			}

			var obj map[string]json.RawMessage
			if err := json.Unmarshal(result[0], &obj); err != nil {
				t.Fatalf("unmarshal result: %v", err)
			}
			_, hasEC := obj["extra_content"]
			if hasEC != tt.hasField {
				t.Errorf("extra_content present=%v, want %v; result=%s", hasEC, tt.hasField, result[0])
			}
		})
	}
}

func TestStripExtraContentPreservesOtherFields(t *testing.T) {
	input := `{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{\"city\":\"NYC\"}","extra_content":{"google":{"thought_signature":"sig123"}}}`
	msgs := []json.RawMessage{json.RawMessage(input)}
	result := StripExtraContent(msgs)

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(result[0], &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	wantFields := []string{"type", "call_id", "name", "arguments"}
	for _, f := range wantFields {
		if _, ok := obj[f]; !ok {
			t.Errorf("missing field %q in stripped result", f)
		}
	}
	if _, ok := obj["extra_content"]; ok {
		t.Error("extra_content should have been stripped")
	}
}

func TestStripExtraContentDoesNotMutateInput(t *testing.T) {
	input := json.RawMessage(`{"type":"function_call","call_id":"c1","name":"f","arguments":"{}","extra_content":{"x":1}}`)
	original := make(json.RawMessage, len(input))
	copy(original, input)

	msgs := []json.RawMessage{input}
	StripExtraContent(msgs)

	if string(msgs[0]) != string(original) {
		t.Error("input slice was mutated")
	}
}

func TestPrependSystemMessage(t *testing.T) {
	cases := []struct {
		name     string
		prompt   string
		messages []json.RawMessage
		wantLen  int
	}{
		{
			name:     "prepends to empty",
			prompt:   "You are helpful.",
			messages: nil,
			wantLen:  1,
		},
		{
			name:     "prepends to existing",
			prompt:   "System prompt",
			messages: []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi"}`)},
			wantLen:  2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := prependSystemMessage(tc.prompt, tc.messages)
			if len(got) != tc.wantLen {
				t.Fatalf("len = %d, want %d", len(got), tc.wantLen)
			}
			var first InputMessage
			if err := json.Unmarshal(got[0], &first); err != nil {
				t.Fatalf("unmarshal first: %v", err)
			}
			if first.Role != "system" {
				t.Errorf("role = %q, want system", first.Role)
			}
			if first.Content != tc.prompt {
				t.Errorf("content = %q, want %q", first.Content, tc.prompt)
			}
		})
	}
}

func TestExtractJSONSchemaInner(t *testing.T) {
	cases := []struct {
		name     string
		input    json.RawMessage
		wantNil  bool
		wantName string
	}{
		{
			name:    "nil input",
			input:   nil,
			wantNil: true,
		},
		{
			name:    "wrong type",
			input:   json.RawMessage(`{"type":"text"}`),
			wantNil: true,
		},
		{
			name:    "no json_schema field",
			input:   json.RawMessage(`{"type":"json_schema"}`),
			wantNil: true,
		},
		{
			name:     "valid json_schema",
			input:    json.RawMessage(`{"type":"json_schema","json_schema":{"name":"my_schema","schema":{"type":"object"},"strict":true}}`),
			wantNil:  false,
			wantName: "my_schema",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractJSONSchemaInner(tc.input)
			if tc.wantNil && got != nil {
				t.Errorf("expected nil, got %+v", got)
			}
			if !tc.wantNil {
				if got == nil {
					t.Fatal("expected non-nil")
				}
				if got.Name != tc.wantName {
					t.Errorf("name = %q, want %q", got.Name, tc.wantName)
				}
			}
		})
	}
}

func TestToTextFormat(t *testing.T) {
	cases := []struct {
		name    string
		input   json.RawMessage
		wantNil bool
		check   func(t *testing.T, got json.RawMessage)
	}{
		{
			name:    "nil returns nil",
			input:   nil,
			wantNil: true,
		},
		{
			name:  "non-json_schema passes through",
			input: json.RawMessage(`{"type":"text"}`),
			check: func(t *testing.T, got json.RawMessage) {
				if diff := cmp.Diff(`{"type":"text"}`, string(got)); diff != "" {
					t.Errorf("passthrough mismatch:\n%s", diff)
				}
			},
		},
		{
			name:  "json_schema converted to text format",
			input: json.RawMessage(`{"type":"json_schema","json_schema":{"name":"output","schema":{"type":"object","properties":{"x":{"type":"string"}}},"strict":true}}`),
			check: func(t *testing.T, got json.RawMessage) {
				var tf textFormat
				if err := json.Unmarshal(got, &tf); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if tf.Type != "json_schema" {
					t.Errorf("type = %q, want json_schema", tf.Type)
				}
				if tf.Name != "output" {
					t.Errorf("name = %q, want output", tf.Name)
				}
				if !tf.Strict {
					t.Error("strict = false, want true")
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := toTextFormat(tc.input)
			if tc.wantNil && got != nil {
				t.Errorf("expected nil, got %s", got)
			}
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestBuildTextConfig(t *testing.T) {
	cases := []struct {
		name           string
		verbosity      string
		responseFormat json.RawMessage
		wantNil        bool
	}{
		{
			name:    "both empty returns nil",
			wantNil: true,
		},
		{
			name:      "verbosity only",
			verbosity: "concise",
			wantNil:   false,
		},
		{
			name:           "format only",
			responseFormat: json.RawMessage(`{"type":"text"}`),
			wantNil:        false,
		},
		{
			name:           "both set",
			verbosity:      "verbose",
			responseFormat: json.RawMessage(`{"type":"text"}`),
			wantNil:        false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildTextConfig(tc.verbosity, tc.responseFormat)
			if tc.wantNil && got != nil {
				t.Errorf("expected nil, got %+v", got)
			}
			if !tc.wantNil && got == nil {
				t.Error("expected non-nil")
			}
		})
	}
}

func ptr(f float64) *float64 { return &f }

func TestBuildResponsesRequest(t *testing.T) {
	tools := []json.RawMessage{json.RawMessage(`{"name":"shell","type":"function"}`)}
	msgs := []json.RawMessage{json.RawMessage(`{"role":"user","content":"hi"}`)}

	cases := []struct {
		name  string
		build func() ResponsesRequest
		check func(t *testing.T, r ResponsesRequest)
	}{
		{
			name: "basic fields",
			build: func() ResponsesRequest {
				return buildResponsesRequest("gpt-5", "Be helpful.", "", "", "", "", nil, "", nil, msgs, nil)
			},
			check: func(t *testing.T, r ResponsesRequest) {
				if r.Model != "gpt-5" {
					t.Errorf("model = %q", r.Model)
				}
				if !r.Stream {
					t.Error("stream should be true")
				}
				if r.Store {
					t.Error("store should be false")
				}
				if len(r.Input) != 2 {
					t.Errorf("input len = %d, want 2 (system + user)", len(r.Input))
				}
				if r.Reasoning != nil {
					t.Error("reasoning should be nil")
				}
			},
		},
		{
			name: "reasoning effort set",
			build: func() ResponsesRequest {
				return buildResponsesRequest("gpt-5", "prompt", "high", "auto", "", "", nil, "", nil, msgs, nil)
			},
			check: func(t *testing.T, r ResponsesRequest) {
				if r.Reasoning == nil {
					t.Fatal("reasoning should not be nil")
				}
				if r.Reasoning.Effort != "high" {
					t.Errorf("effort = %q", r.Reasoning.Effort)
				}
				if r.Reasoning.Summary != "auto" {
					t.Errorf("summary = %q", r.Reasoning.Summary)
				}
				if len(r.Include) == 0 {
					t.Error("include should have reasoning.encrypted_content")
				}
			},
		},
		{
			name: "reasoning off skipped",
			build: func() ResponsesRequest {
				return buildResponsesRequest("gpt-5", "prompt", "off", "", "", "", nil, "", nil, msgs, nil)
			},
			check: func(t *testing.T, r ResponsesRequest) {
				if r.Reasoning != nil {
					t.Error("reasoning should be nil when effort is off")
				}
			},
		},
		{
			name: "temperature set",
			build: func() ResponsesRequest {
				return buildResponsesRequest("gpt-5", "prompt", "", "", "", "", nil, "", ptr(0.7), msgs, nil)
			},
			check: func(t *testing.T, r ResponsesRequest) {
				if r.Temperature == nil || *r.Temperature != 0.7 {
					t.Errorf("temperature = %v, want 0.7", r.Temperature)
				}
			},
		},
		{
			name: "tool choice with tools",
			build: func() ResponsesRequest {
				return buildResponsesRequest("gpt-5", "prompt", "", "", "", "", tools, "required", nil, msgs, nil)
			},
			check: func(t *testing.T, r ResponsesRequest) {
				if r.ToolChoice == nil || *r.ToolChoice != "required" {
					t.Errorf("tool_choice = %v, want required", r.ToolChoice)
				}
			},
		},
		{
			name: "tool choice without tools ignored",
			build: func() ResponsesRequest {
				return buildResponsesRequest("gpt-5", "prompt", "", "", "", "", nil, "required", nil, msgs, nil)
			},
			check: func(t *testing.T, r ResponsesRequest) {
				if r.ToolChoice != nil {
					t.Error("tool_choice should be nil without tools")
				}
			},
		},
		{
			name: "service tier passed",
			build: func() ResponsesRequest {
				return buildResponsesRequest("gpt-5", "prompt", "", "", "", "flex", nil, "", nil, msgs, nil)
			},
			check: func(t *testing.T, r ResponsesRequest) {
				if r.ServiceTier != "flex" {
					t.Errorf("service_tier = %q", r.ServiceTier)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.check(t, tc.build())
		})
	}
}
