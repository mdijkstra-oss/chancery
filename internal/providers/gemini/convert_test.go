package gemini

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mdijkstra-oss/chancery/internal/protocol"
	"google.golang.org/genai"
)

func TestBuildCallIDToName(t *testing.T) {
	tests := []struct {
		name     string
		messages []string
		want     map[string]string
	}{
		{
			name:     "empty",
			messages: nil,
			want:     map[string]string{},
		},
		{
			name: "single function call",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search"}`,
			},
			want: map[string]string{"c1": "search"},
		},
		{
			name: "multiple function calls mixed with other messages",
			messages: []string{
				`{"role":"user","content":"hi"}`,
				`{"type":"function_call","call_id":"c1","name":"search"}`,
				`{"type":"function_call_output","call_id":"c1","output":"result"}`,
				`{"type":"function_call","call_id":"c2","name":"read_file"}`,
			},
			want: map[string]string{"c1": "search", "c2": "read_file"},
		},
		{
			name: "ignores non-function-call types",
			messages: []string{
				`{"role":"user","content":"hi"}`,
				`{"role":"assistant","content":"hello"}`,
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := toRawMessages(tt.messages)
			got := BuildCallIDToName(msgs)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractLeadingSystem(t *testing.T) {
	tests := []struct {
		name        string
		messages    []string
		wantLeading []string
		wantRest    int
	}{
		{
			name:        "empty",
			messages:    nil,
			wantLeading: nil,
			wantRest:    0,
		},
		{
			name: "no leading system",
			messages: []string{
				`{"role":"user","content":"hello"}`,
			},
			wantLeading: nil,
			wantRest:    1,
		},
		{
			name: "single leading system",
			messages: []string{
				`{"role":"system","content":"db schema"}`,
				`{"role":"user","content":"hello"}`,
			},
			wantLeading: []string{"db schema"},
			wantRest:    1,
		},
		{
			name: "multiple leading system",
			messages: []string{
				`{"role":"system","content":"tools info"}`,
				`{"role":"system","content":"db schema"}`,
				`{"role":"user","content":"hello"}`,
				`{"role":"system","content":"cursor context"}`,
			},
			wantLeading: []string{"tools info", "db schema"},
			wantRest:    2,
		},
		{
			name: "all system falls through to rest",
			messages: []string{
				`{"role":"system","content":"a"}`,
				`{"role":"system","content":"b"}`,
			},
			wantLeading: nil,
			wantRest:    2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := toRawMessages(tt.messages)
			leading, rest := ExtractLeadingSystem(msgs)
			if diff := cmp.Diff(tt.wantLeading, leading); diff != "" {
				t.Errorf("leading mismatch (-want +got):\n%s", diff)
			}
			if len(rest) != tt.wantRest {
				t.Errorf("rest len = %d, want %d", len(rest), tt.wantRest)
			}
		})
	}
}

func TestMessagesToContents(t *testing.T) {
	tests := []struct {
		name      string
		messages  []string
		callIDMap map[string]string
		check     func(t *testing.T, got []*genai.Content)
	}{
		{
			name: "user message",
			messages: []string{
				`{"role":"user","content":"hello"}`,
			},
			callIDMap: nil,
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if got[0].Role != "user" {
					t.Errorf("role = %q, want user", got[0].Role)
				}
				assertTextPart(t, got[0], "hello")
			},
		},
		{
			name: "assistant message",
			messages: []string{
				`{"role":"assistant","content":"hi there"}`,
			},
			callIDMap: nil,
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if got[0].Role != "model" {
					t.Errorf("role = %q, want model", got[0].Role)
				}
				assertTextPart(t, got[0], "hi there")
			},
		},
		{
			name: "system message wrapped as user",
			messages: []string{
				`{"role":"system","content":"you are helpful"}`,
				`{"role":"user","content":"hello"}`,
			},
			callIDMap: nil,
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 2 {
					t.Fatalf("len = %d, want 2", len(got))
				}
				if got[0].Role != "user" {
					t.Errorf("role = %q, want user", got[0].Role)
				}
				assertTextPart(t, got[0], "<system_message>\nyou are helpful\n</system_message>")
				if got[1].Role != "user" {
					t.Errorf("role = %q, want user", got[1].Role)
				}
				assertTextPart(t, got[1], "hello")
			},
		},
		{
			name: "function call",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{\"q\":\"test\"}"}`,
			},
			callIDMap: map[string]string{"c1": "search"},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if got[0].Role != "model" {
					t.Errorf("role = %q, want model", got[0].Role)
				}
				if got[0].Parts[0].FunctionCall == nil {
					t.Fatal("expected FunctionCall part")
				}
				if got[0].Parts[0].FunctionCall.Name != "search" {
					t.Errorf("name = %q, want search", got[0].Parts[0].FunctionCall.Name)
				}
			},
		},
		{
			name: "function call with thought signature",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{\"q\":\"test\"}","extra_content":{"google":{"thought_signature":"dGVzdHNpZw=="}}}`,
			},
			callIDMap: map[string]string{"c1": "search"},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				part := got[0].Parts[0]
				if part.FunctionCall == nil {
					t.Fatal("expected FunctionCall part")
				}
				if string(part.ThoughtSignature) != "testsig" {
					t.Errorf("ThoughtSignature = %q, want testsig", string(part.ThoughtSignature))
				}
			},
		},
		{
			name: "function call without extra_content has nil signature",
			messages: []string{
				`{"type":"function_call","call_id":"c1","name":"search","arguments":"{}"}`,
			},
			callIDMap: map[string]string{"c1": "search"},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if got[0].Parts[0].ThoughtSignature != nil {
					t.Errorf("expected nil ThoughtSignature, got %v", got[0].Parts[0].ThoughtSignature)
				}
			},
		},
		{
			name: "function call output",
			messages: []string{
				`{"type":"function_call_output","call_id":"c1","output":"{\"result\":\"found\"}"}`,
			},
			callIDMap: map[string]string{"c1": "search"},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if got[0].Role != "user" {
					t.Errorf("role = %q, want user", got[0].Role)
				}
				if got[0].Parts[0].FunctionResponse == nil {
					t.Fatal("expected FunctionResponse part")
				}
				if got[0].Parts[0].FunctionResponse.Name != "search" {
					t.Errorf("name = %q, want search", got[0].Parts[0].FunctionResponse.Name)
				}
			},
		},
		{
			name: "function call output with plain text",
			messages: []string{
				`{"type":"function_call_output","call_id":"c1","output":"plain text result"}`,
			},
			callIDMap: map[string]string{"c1": "search"},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				resp := got[0].Parts[0].FunctionResponse.Response
				if resp["output"] != "plain text result" {
					t.Errorf("output = %v, want plain text result", resp["output"])
				}
			},
		},
		{
			name: "reasoning with thought signature",
			messages: []string{
				`{"type":"reasoning","id":"r1","extra_content":{"google":{"thought_signature":"dGVzdHNpZw=="}}}`,
			},
			callIDMap: nil,
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if !got[0].Parts[0].Thought {
					t.Error("expected Thought=true")
				}
				if string(got[0].Parts[0].ThoughtSignature) != "testsig" {
					t.Errorf("signature = %q, want testsig", string(got[0].Parts[0].ThoughtSignature))
				}
			},
		},
		{
			name: "reasoning without extra_content skipped",
			messages: []string{
				`{"type":"reasoning","id":"r1"}`,
			},
			callIDMap: nil,
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 0 {
					t.Errorf("len = %d, want 0", len(got))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msgs := toRawMessages(tt.messages)
			got := MessagesToContents(msgs, tt.callIDMap)
			tt.check(t, got)
		})
	}
}

func TestMergeConsecutiveContents(t *testing.T) {
	tests := []struct {
		name  string
		input []*genai.Content
		check func(t *testing.T, got []*genai.Content)
	}{
		{
			name:  "empty",
			input: nil,
			check: func(t *testing.T, got []*genai.Content) {
				if got != nil {
					t.Errorf("want nil, got %v", got)
				}
			},
		},
		{
			name: "single entry unchanged",
			input: []*genai.Content{
				genai.NewContentFromText("hello", "user"),
			},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if len(got[0].Parts) != 1 {
					t.Fatalf("parts = %d, want 1", len(got[0].Parts))
				}
			},
		},
		{
			name: "alternating roles unchanged",
			input: []*genai.Content{
				genai.NewContentFromText("hello", "user"),
				genai.NewContentFromText("hi", "model"),
			},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 2 {
					t.Fatalf("len = %d, want 2", len(got))
				}
			},
		},
		{
			name: "consecutive user merged",
			input: []*genai.Content{
				genai.NewContentFromText("system context", "user"),
				genai.NewContentFromText("actual message", "user"),
			},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if got[0].Role != "user" {
					t.Errorf("role = %q, want user", got[0].Role)
				}
				if len(got[0].Parts) != 2 {
					t.Fatalf("parts = %d, want 2", len(got[0].Parts))
				}
				if got[0].Parts[0].Text != "system context" {
					t.Errorf("part[0] = %q, want system context", got[0].Parts[0].Text)
				}
				if got[0].Parts[1].Text != "actual message" {
					t.Errorf("part[1] = %q, want actual message", got[0].Parts[1].Text)
				}
			},
		},
		{
			name: "three consecutive user then model",
			input: []*genai.Content{
				genai.NewContentFromText("a", "user"),
				genai.NewContentFromText("b", "user"),
				genai.NewContentFromText("c", "user"),
				genai.NewContentFromText("response", "model"),
			},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 2 {
					t.Fatalf("len = %d, want 2", len(got))
				}
				if len(got[0].Parts) != 3 {
					t.Fatalf("user parts = %d, want 3", len(got[0].Parts))
				}
				if got[1].Role != "model" {
					t.Errorf("second role = %q, want model", got[1].Role)
				}
			},
		},
		{
			name: "does not mutate input",
			input: []*genai.Content{
				genai.NewContentFromText("a", "user"),
				genai.NewContentFromText("b", "user"),
			},
			check: func(t *testing.T, got []*genai.Content) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputLen := len(tt.input)
			got := MergeConsecutiveContents(tt.input)
			tt.check(t, got)
			if len(tt.input) != inputLen {
				t.Error("input slice was mutated")
			}
		})
	}
}

func TestToolsToGemini(t *testing.T) {
	tests := []struct {
		name  string
		tools []string
		check func(t *testing.T, got []*genai.Tool)
	}{
		{
			name:  "empty",
			tools: nil,
			check: func(t *testing.T, got []*genai.Tool) {
				if got != nil {
					t.Errorf("want nil, got %v", got)
				}
			},
		},
		{
			name: "single tool",
			tools: []string{
				`{"name":"search","description":"Search the web","parameters":{"type":"object","properties":{"q":{"type":"string"}},"additionalProperties":false}}`,
			},
			check: func(t *testing.T, got []*genai.Tool) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if len(got[0].FunctionDeclarations) != 1 {
					t.Fatalf("decl count = %d, want 1", len(got[0].FunctionDeclarations))
				}
				decl := got[0].FunctionDeclarations[0]
				if decl.Name != "search" {
					t.Errorf("name = %q, want search", decl.Name)
				}
				if decl.Description != "Search the web" {
					t.Errorf("description = %q, want Search the web", decl.Description)
				}
				if decl.Parameters != nil {
					t.Error("expected Parameters to be nil (using ParametersJsonSchema)")
				}
				if decl.ParametersJsonSchema == nil {
					t.Fatal("expected ParametersJsonSchema to be set")
				}
				schema := decl.ParametersJsonSchema.(map[string]any)
				if schema["type"] != "object" {
					t.Errorf("type = %v, want object", schema["type"])
				}
				if schema["additionalProperties"] != false {
					t.Errorf("additionalProperties = %v, want false", schema["additionalProperties"])
				}
			},
		},
		{
			name: "multiple tools",
			tools: []string{
				`{"name":"search","description":"Search"}`,
				`{"name":"read","description":"Read file"}`,
			},
			check: func(t *testing.T, got []*genai.Tool) {
				if len(got) != 1 {
					t.Fatalf("len = %d, want 1", len(got))
				}
				if len(got[0].FunctionDeclarations) != 2 {
					t.Fatalf("decl count = %d, want 2", len(got[0].FunctionDeclarations))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ToolsToGemini(toRawMessages(tt.tools))
			tt.check(t, got)
		})
	}
}

func TestBuildConfig(t *testing.T) {
	temp := 0.7

	tests := []struct {
		name          string
		params        protocol.RequestParams
		leadingSystem []string
		check         func(t *testing.T, cfg *genai.GenerateContentConfig)
	}{
		{
			name: "system prompt",
			params: protocol.RequestParams{
				SystemPrompt: "be helpful",
			},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.SystemInstruction == nil {
					t.Fatal("expected SystemInstruction")
				}
				assertTextPart(t, cfg.SystemInstruction, "be helpful")
			},
		},
		{
			name: "system prompt with leading system",
			params: protocol.RequestParams{
				SystemPrompt: "be helpful",
			},
			leadingSystem: []string{"tools info", "db schema"},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.SystemInstruction == nil {
					t.Fatal("expected SystemInstruction")
				}
				assertTextPart(t, cfg.SystemInstruction, "be helpful\n\ntools info\n\ndb schema")
			},
		},
		{
			name:          "leading system without backend prompt",
			leadingSystem: []string{"tools info"},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.SystemInstruction == nil {
					t.Fatal("expected SystemInstruction")
				}
				assertTextPart(t, cfg.SystemInstruction, "tools info")
			},
		},
		{
			name: "temperature",
			params: protocol.RequestParams{
				Temperature: &temp,
			},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.Temperature == nil {
					t.Fatal("expected Temperature")
				}
				if *cfg.Temperature != 0.7 {
					t.Errorf("temp = %f, want 0.7", *cfg.Temperature)
				}
			},
		},
		{
			name: "thinking with level (non-legacy)",
			params: protocol.RequestParams{
				ReasoningEffort: "high",
				LegacyThinking:  false,
			},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.ThinkingConfig == nil {
					t.Fatal("expected ThinkingConfig")
				}
				if !cfg.ThinkingConfig.IncludeThoughts {
					t.Error("expected IncludeThoughts=true")
				}
				if cfg.ThinkingConfig.ThinkingLevel != genai.ThinkingLevelHigh {
					t.Errorf("level = %q, want HIGH", cfg.ThinkingConfig.ThinkingLevel)
				}
				if cfg.ThinkingConfig.ThinkingBudget != nil {
					t.Error("expected ThinkingBudget=nil for non-legacy")
				}
			},
		},
		{
			name: "thinking with budget (legacy)",
			params: protocol.RequestParams{
				ReasoningEffort: "low",
				LegacyThinking:  true,
			},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.ThinkingConfig == nil {
					t.Fatal("expected ThinkingConfig")
				}
				if cfg.ThinkingConfig.ThinkingBudget == nil {
					t.Fatal("expected ThinkingBudget for legacy")
				}
				if *cfg.ThinkingConfig.ThinkingBudget != 4096 {
					t.Errorf("budget = %d, want 4096", *cfg.ThinkingConfig.ThinkingBudget)
				}
			},
		},
		{
			name: "reasoning off",
			params: protocol.RequestParams{
				ReasoningEffort: "off",
			},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.ThinkingConfig != nil {
					t.Error("expected ThinkingConfig=nil for off")
				}
			},
		},
		{
			name: "response format json_schema",
			params: protocol.RequestParams{
				ResponseFormat: json.RawMessage(`{"type":"json_schema","json_schema":{"name":"test","schema":{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false}}}`),
			},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.ResponseMIMEType != "application/json" {
					t.Errorf("mime = %q, want application/json", cfg.ResponseMIMEType)
				}
				if cfg.ResponseSchema != nil {
					t.Error("expected ResponseSchema to be nil (using ResponseJsonSchema)")
				}
				if cfg.ResponseJsonSchema == nil {
					t.Fatal("expected ResponseJsonSchema")
				}
				schema := cfg.ResponseJsonSchema.(map[string]any)
				if schema["type"] != "object" {
					t.Errorf("type = %v, want object", schema["type"])
				}
				if schema["additionalProperties"] != false {
					t.Errorf("additionalProperties = %v, want false", schema["additionalProperties"])
				}
			},
		},
		{
			name: "seed enabled",
			params: protocol.RequestParams{
				Seed: true,
			},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.Seed == nil {
					t.Fatal("expected Seed")
				}
				if *cfg.Seed != 12041989 {
					t.Errorf("seed = %d, want 12041989", *cfg.Seed)
				}
			},
		},
		{
			name:   "seed disabled",
			params: protocol.RequestParams{},
			check: func(t *testing.T, cfg *genai.GenerateContentConfig) {
				if cfg.Seed != nil {
					t.Errorf("expected nil Seed, got %d", *cfg.Seed)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BuildConfig(tt.params, tt.leadingSystem)
			tt.check(t, cfg)
		})
	}
}

func TestBuildThinkingConfig(t *testing.T) {
	tests := []struct {
		name   string
		effort string
		legacy bool
		check  func(t *testing.T, tc *genai.ThinkingConfig)
	}{
		{
			name:   "non-legacy minimal",
			effort: "minimal",
			legacy: false,
			check: func(t *testing.T, tc *genai.ThinkingConfig) {
				if tc.ThinkingLevel != genai.ThinkingLevelMinimal {
					t.Errorf("level = %q, want MINIMAL", tc.ThinkingLevel)
				}
			},
		},
		{
			name:   "non-legacy medium",
			effort: "medium",
			legacy: false,
			check: func(t *testing.T, tc *genai.ThinkingConfig) {
				if tc.ThinkingLevel != genai.ThinkingLevelMedium {
					t.Errorf("level = %q, want MEDIUM", tc.ThinkingLevel)
				}
			},
		},
		{
			name:   "legacy low",
			effort: "low",
			legacy: true,
			check: func(t *testing.T, tc *genai.ThinkingConfig) {
				if tc.ThinkingBudget == nil || *tc.ThinkingBudget != 4096 {
					t.Errorf("budget = %v, want 4096", tc.ThinkingBudget)
				}
			},
		},
		{
			name:   "legacy high",
			effort: "high",
			legacy: true,
			check: func(t *testing.T, tc *genai.ThinkingConfig) {
				if tc.ThinkingBudget == nil || *tc.ThinkingBudget != 16384 {
					t.Errorf("budget = %v, want 16384", tc.ThinkingBudget)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := buildThinkingConfig(tt.effort, tt.legacy)
			if !tc.IncludeThoughts {
				t.Error("expected IncludeThoughts=true")
			}
			tt.check(t, tc)
		})
	}
}

func TestBuildThinkingConfigUnknownEffortPanics(t *testing.T) {
	tests := []struct {
		name   string
		effort string
		legacy bool
	}{
		{"level path", "unknown", false},
		{"budget path", "unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("buildThinkingConfig(%q, %v) did not panic", tt.effort, tt.legacy)
				}
			}()
			buildThinkingConfig(tt.effort, tt.legacy)
		})
	}
}

func TestBuildConfigReasoningOff(t *testing.T) {
	tests := []struct {
		name            string
		effort          string
		wantThinkingNil bool
	}{
		{"none disables thinking", "none", true},
		{"off disables thinking", "off", true},
		{"empty disables thinking", "", true},
		{"minimal enables thinking", "minimal", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BuildConfig(protocol.RequestParams{ReasoningEffort: tt.effort}, nil)
			if tt.wantThinkingNil {
				if cfg.ThinkingConfig != nil {
					t.Errorf("ThinkingConfig = %+v, want nil", cfg.ThinkingConfig)
				}
				return
			}
			if cfg.ThinkingConfig == nil {
				t.Fatal("ThinkingConfig = nil, want thinking enabled")
			}
		})
	}
}

func TestToolChoiceToMode(t *testing.T) {
	tests := []struct {
		choice string
		want   genai.FunctionCallingConfigMode
	}{
		{"", genai.FunctionCallingConfigModeValidated},
		{"required", genai.FunctionCallingConfigModeAny},
		{"none", genai.FunctionCallingConfigModeNone},
		{"auto", genai.FunctionCallingConfigModeValidated},
		{"unknown", genai.FunctionCallingConfigModeValidated},
	}
	for _, tt := range tests {
		t.Run(tt.choice, func(t *testing.T) {
			got := toolChoiceToMode(tt.choice)
			if got != tt.want {
				t.Errorf("toolChoiceToMode(%q) = %q, want %q", tt.choice, got, tt.want)
			}
		})
	}
}

func TestBuildConfigToolConfig(t *testing.T) {
	oneTool := []json.RawMessage{json.RawMessage(`{"name":"search","description":"Search"}`)}
	tests := []struct {
		name      string
		params    protocol.RequestParams
		wantMode  genai.FunctionCallingConfigMode
		wantNilTC bool
	}{
		{
			name:      "no tools produces no ToolConfig",
			params:    protocol.RequestParams{},
			wantNilTC: true,
		},
		{
			name:     "tools without toolChoice defaults to VALIDATED",
			params:   protocol.RequestParams{Tools: oneTool},
			wantMode: genai.FunctionCallingConfigModeValidated,
		},
		{
			name:     "tools with required maps to ANY",
			params:   protocol.RequestParams{Tools: oneTool, ToolChoice: "required"},
			wantMode: genai.FunctionCallingConfigModeAny,
		},
		{
			name:     "tools with none maps to NONE",
			params:   protocol.RequestParams{Tools: oneTool, ToolChoice: "none"},
			wantMode: genai.FunctionCallingConfigModeNone,
		},
		{
			name:     "tools with auto still maps to VALIDATED",
			params:   protocol.RequestParams{Tools: oneTool, ToolChoice: "auto"},
			wantMode: genai.FunctionCallingConfigModeValidated,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := BuildConfig(tt.params, nil)
			if tt.wantNilTC {
				if cfg.ToolConfig != nil {
					t.Errorf("expected nil ToolConfig, got %v", cfg.ToolConfig)
				}
				return
			}
			if cfg.ToolConfig == nil {
				t.Fatal("expected ToolConfig")
			}
			if cfg.ToolConfig.FunctionCallingConfig == nil {
				t.Fatal("expected FunctionCallingConfig")
			}
			if cfg.ToolConfig.FunctionCallingConfig.Mode != tt.wantMode {
				t.Errorf("mode = %q, want %q", cfg.ToolConfig.FunctionCallingConfig.Mode, tt.wantMode)
			}
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

func TestEnsureThoughtSignatures(t *testing.T) {
	realSig := []byte("real-signature")

	tests := []struct {
		name     string
		thinking bool
		contents []*genai.Content
		check    func(t *testing.T, got []*genai.Content)
	}{
		{
			name:     "thinking disabled is no-op",
			thinking: false,
			contents: []*genai.Content{{
				Role: "model",
				Parts: []*genai.Part{{
					FunctionCall: &genai.FunctionCall{Name: "search", ID: "c1"},
				}},
			}},
			check: func(t *testing.T, got []*genai.Content) {
				if got[0].Parts[0].ThoughtSignature != nil {
					t.Error("expected nil sig when thinking disabled")
				}
			},
		},
		{
			name:     "single FC with sig unchanged",
			thinking: true,
			contents: []*genai.Content{{
				Role: "model",
				Parts: []*genai.Part{{
					FunctionCall:     &genai.FunctionCall{Name: "search", ID: "c1"},
					ThoughtSignature: realSig,
				}},
			}},
			check: func(t *testing.T, got []*genai.Content) {
				if string(got[0].Parts[0].ThoughtSignature) != "real-signature" {
					t.Errorf("sig = %q, want real-signature", string(got[0].Parts[0].ThoughtSignature))
				}
			},
		},
		{
			name:     "single FC missing sig gets fallback",
			thinking: true,
			contents: []*genai.Content{{
				Role: "model",
				Parts: []*genai.Part{{
					FunctionCall: &genai.FunctionCall{Name: "search", ID: "c1"},
				}},
			}},
			check: func(t *testing.T, got []*genai.Content) {
				if string(got[0].Parts[0].ThoughtSignature) != string(fallbackThoughtSig) {
					t.Errorf("sig = %q, want fallback", string(got[0].Parts[0].ThoughtSignature))
				}
			},
		},
		{
			name:     "parallel FCs first has sig rest untouched",
			thinking: true,
			contents: []*genai.Content{{
				Role: "model",
				Parts: []*genai.Part{
					{FunctionCall: &genai.FunctionCall{Name: "search", ID: "c1"}, ThoughtSignature: realSig},
					{FunctionCall: &genai.FunctionCall{Name: "read", ID: "c2"}},
					{FunctionCall: &genai.FunctionCall{Name: "write", ID: "c3"}},
				},
			}},
			check: func(t *testing.T, got []*genai.Content) {
				if string(got[0].Parts[0].ThoughtSignature) != "real-signature" {
					t.Errorf("first sig = %q, want real-signature", string(got[0].Parts[0].ThoughtSignature))
				}
				if got[0].Parts[1].ThoughtSignature != nil {
					t.Errorf("second sig = %v, want nil", got[0].Parts[1].ThoughtSignature)
				}
				if got[0].Parts[2].ThoughtSignature != nil {
					t.Errorf("third sig = %v, want nil", got[0].Parts[2].ThoughtSignature)
				}
			},
		},
		{
			name:     "parallel FCs first missing sig gets fallback rest untouched",
			thinking: true,
			contents: []*genai.Content{{
				Role: "model",
				Parts: []*genai.Part{
					{FunctionCall: &genai.FunctionCall{Name: "search", ID: "c1"}},
					{FunctionCall: &genai.FunctionCall{Name: "read", ID: "c2"}},
				},
			}},
			check: func(t *testing.T, got []*genai.Content) {
				if string(got[0].Parts[0].ThoughtSignature) != string(fallbackThoughtSig) {
					t.Errorf("first sig = %q, want fallback", string(got[0].Parts[0].ThoughtSignature))
				}
				if got[0].Parts[1].ThoughtSignature != nil {
					t.Errorf("second sig = %v, want nil", got[0].Parts[1].ThoughtSignature)
				}
			},
		},
		{
			name:     "user content skipped",
			thinking: true,
			contents: []*genai.Content{{
				Role:  "user",
				Parts: []*genai.Part{{Text: "hello"}},
			}},
			check: func(t *testing.T, got []*genai.Content) {
				if got[0].Parts[0].ThoughtSignature != nil {
					t.Error("expected nil sig for user content")
				}
			},
		},
		{
			name:     "thought part before FC preserves both",
			thinking: true,
			contents: []*genai.Content{{
				Role: "model",
				Parts: []*genai.Part{
					{Thought: true, ThoughtSignature: realSig},
					{FunctionCall: &genai.FunctionCall{Name: "search", ID: "c1"}, ThoughtSignature: realSig},
				},
			}},
			check: func(t *testing.T, got []*genai.Content) {
				if string(got[0].Parts[0].ThoughtSignature) != "real-signature" {
					t.Errorf("thought sig = %q, want real-signature", string(got[0].Parts[0].ThoughtSignature))
				}
				if string(got[0].Parts[1].ThoughtSignature) != "real-signature" {
					t.Errorf("FC sig = %q, want real-signature", string(got[0].Parts[1].ThoughtSignature))
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EnsureThoughtSignatures(tt.contents, tt.thinking)
			tt.check(t, got)
		})
	}
}

func TestLastBreakpointIndex(t *testing.T) {
	tests := []struct {
		name        string
		breakpoints map[int]bool
		want        int
	}{
		{"nil map", nil, -1},
		{"empty map", map[int]bool{}, -1},
		{"single entry", map[int]bool{3: true}, 3},
		{"multiple entries", map[int]bool{1: true, 5: true, 3: true}, 5},
		{"zero index", map[int]bool{0: true}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LastBreakpointIndex(tt.breakpoints)
			if got != tt.want {
				t.Errorf("LastBreakpointIndex() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSplitAtLastBreakpoint(t *testing.T) {
	msgs := toRawMessages([]string{
		`{"role":"system","content":"sys"}`,
		`{"role":"user","content":"chunk1"}`,
		`{"role":"user","content":"chunk2"}`,
		`{"role":"user","content":"tail"}`,
	})

	tests := []struct {
		name          string
		breakpoints   map[int]bool
		wantPrefixLen int
		wantTailLen   int
	}{
		{"nil breakpoints", nil, 0, 4},
		{"empty breakpoints", map[int]bool{}, 0, 4},
		{"split at index 1", map[int]bool{1: true}, 2, 2},
		{"split at last of multiple", map[int]bool{0: true, 2: true}, 3, 1},
		{"split at end", map[int]bool{3: true}, 4, 0},
		{"index out of range", map[int]bool{10: true}, 0, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, tail := SplitAtLastBreakpoint(msgs, tt.breakpoints)
			if len(prefix) != tt.wantPrefixLen {
				t.Errorf("prefix len = %d, want %d", len(prefix), tt.wantPrefixLen)
			}
			if len(tail) != tt.wantTailLen {
				t.Errorf("tail len = %d, want %d", len(tail), tt.wantTailLen)
			}
			if tt.wantPrefixLen+tt.wantTailLen == len(msgs) && tt.wantPrefixLen > 0 {
				if len(prefix)+len(tail) != len(msgs) {
					t.Error("prefix + tail should equal original length")
				}
			}
		})
	}
}

func TestStripLeadingSystem(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		wantLen int
	}{
		{
			"all system returns nil",
			[]string{
				`{"type":"message","role":"system","content":"a"}`,
				`{"type":"message","role":"system","content":"b"}`,
			},
			0,
		},
		{
			"strips leading system keeps rest",
			[]string{
				`{"type":"message","role":"system","content":"sys1"}`,
				`{"type":"message","role":"system","content":"sys2"}`,
				`{"type":"message","role":"user","content":"hello"}`,
				`{"type":"message","role":"assistant","content":"hi"}`,
			},
			2,
		},
		{
			"no system messages",
			[]string{
				`{"type":"message","role":"user","content":"hello"}`,
				`{"type":"message","role":"assistant","content":"hi"}`,
			},
			2,
		},
		{
			"empty input",
			nil,
			0,
		},
		{
			"single system",
			[]string{`{"type":"message","role":"system","content":"only"}`},
			0,
		},
		{
			"system then user then system",
			[]string{
				`{"type":"message","role":"system","content":"leading"}`,
				`{"type":"message","role":"user","content":"q"}`,
				`{"type":"message","role":"system","content":"trailing"}`,
			},
			2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripLeadingSystem(toRawMessages(tt.input))
			if len(got) != tt.wantLen {
				t.Errorf("len = %d, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestContentHash(t *testing.T) {
	tools := toRawMessages([]string{`{"name":"search"}`})
	msgs := toRawMessages([]string{`{"role":"user","content":"hello"}`})

	h1 := ContentHash("gemini-3.5-flash", "sys", tools, msgs)
	h2 := ContentHash("gemini-3.5-flash", "sys", tools, msgs)
	if h1 != h2 {
		t.Error("same inputs should produce same hash")
	}

	h3 := ContentHash("gemini-2.5-flash", "sys", tools, msgs)
	if h1 == h3 {
		t.Error("different model should produce different hash")
	}

	h4 := ContentHash("gemini-3.5-flash", "different", tools, msgs)
	if h1 == h4 {
		t.Error("different system prompt should produce different hash")
	}

	h5 := ContentHash("gemini-3.5-flash", "sys", nil, msgs)
	if h1 == h5 {
		t.Error("different tools should produce different hash")
	}

	diffMsgs := toRawMessages([]string{`{"role":"user","content":"bye"}`})
	h6 := ContentHash("gemini-3.5-flash", "sys", tools, diffMsgs)
	if h1 == h6 {
		t.Error("different messages should produce different hash")
	}

	if len(h1) != 64 {
		t.Errorf("hash length = %d, want 64 (sha256 hex)", len(h1))
	}
}

func assertTextPart(t *testing.T, content *genai.Content, want string) {
	t.Helper()
	if len(content.Parts) == 0 {
		t.Fatal("no parts")
	}
	if content.Parts[0].Text != want {
		t.Errorf("text = %q, want %q", content.Parts[0].Text, want)
	}
}
