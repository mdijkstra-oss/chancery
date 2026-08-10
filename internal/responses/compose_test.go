package responses

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestCompose(t *testing.T) {
	cases := []struct {
		name  string
		body  string
		agent Agent
		want  string
	}{{
		name:  "no overrides differs only by model and instructions",
		body:  `{"input":[{"role":"user","content":"hi"}],"stream":true,"store":false}`,
		agent: Agent{Model: "openai/gpt-5.5", Instructions: "You are a hound."},
		want: `{"input":[{"role":"user","content":"hi"}],"stream":true,"store":false,
			"model":"openai/gpt-5.5","instructions":"You are a hound."}`,
	}, {
		name:  "instructions prepended in front of the caller's",
		body:  `{"input":[],"instructions":"Answer in French."}`,
		agent: Agent{Instructions: "You are a hound."},
		want:  `{"input":[],"instructions":"You are a hound.\n\nAnswer in French."}`,
	}, {
		name: "every frontmatter field lands in its body position",
		body: `{"input":[]}`,
		agent: Agent{
			Model:            "deepseek/deepseek-v4-pro",
			Instructions:     "Be brief.",
			ReasoningEffort:  "high",
			ReasoningSummary: "auto",
			Verbosity:        "low",
			ServiceTier:      "priority",
			MaxOutputTokens:  4096,
		},
		want: `{"input":[],"model":"deepseek/deepseek-v4-pro","instructions":"Be brief.",
			"reasoning":{"effort":"high","summary":"auto"},"text":{"verbosity":"low"},
			"service_tier":"priority","max_output_tokens":4096}`,
	}, {
		name:  "unknown top-level key survives",
		body:  `{"input":[],"prompt_cache_key":"abc","truncation":"auto","metadata":{"run":"7"}}`,
		agent: Agent{Model: "openai/gpt-5.5"},
		want: `{"input":[],"prompt_cache_key":"abc","truncation":"auto","metadata":{"run":"7"},
			"model":"openai/gpt-5.5"}`,
	}, {
		name: "nested value survives untouched",
		body: `{"input":[{"role":"user","content":[{"type":"input_text","text":"hi",
			"prompt_cache_breakpoint":true}]}],"tools":[{"type":"function","name":"lookup",
			"parameters":{"type":"object","properties":{"q":{"type":"string"}}}}]}`,
		agent: Agent{Model: "openai/gpt-5.5"},
		want: `{"input":[{"role":"user","content":[{"type":"input_text","text":"hi",
			"prompt_cache_breakpoint":true}]}],"tools":[{"type":"function","name":"lookup",
			"parameters":{"type":"object","properties":{"q":{"type":"string"}}}}],
			"model":"openai/gpt-5.5"}`,
	}, {
		name:  "caller's text format keeps its place beside verbosity",
		body:  `{"input":[],"text":{"format":{"type":"json_schema","name":"reply"}}}`,
		agent: Agent{Verbosity: "high"},
		want: `{"input":[],"text":{"format":{"type":"json_schema","name":"reply"},
			"verbosity":"high"}}`,
	}, {
		name:  "caller's reasoning summary survives an effort override",
		body:  `{"input":[],"reasoning":{"summary":"detailed"}}`,
		agent: Agent{ReasoningEffort: "minimal"},
		want:  `{"input":[],"reasoning":{"summary":"detailed","effort":"minimal"}}`,
	}, {
		name:  "agent value wins over the caller's",
		body:  `{"input":[],"model":"caller/model","service_tier":"default"}`,
		agent: Agent{Model: "openai/gpt-5.5", ServiceTier: "priority"},
		want:  `{"input":[],"model":"openai/gpt-5.5","service_tier":"priority"}`,
	}, {
		name:  "unset agent fields are not written",
		body:  `{"input":[]}`,
		agent: Agent{Model: "openai/gpt-5.5"},
		want:  `{"input":[],"model":"openai/gpt-5.5"}`,
	}, {
		name:  "a per-request tool choice overrides the caller's",
		body:  `{"input":[],"tool_choice":"auto"}`,
		agent: Agent{ToolChoice: "required"},
		want:  `{"input":[],"tool_choice":"required"}`,
	}, {
		name:  "null instructions is replaced rather than joined",
		body:  `{"input":[],"instructions":null}`,
		agent: Agent{Instructions: "You are a hound."},
		want:  `{"input":[],"instructions":"You are a hound."}`,
	}, {
		// The caller marks its prompt once and addresses whichever route it likes;
		// the agent behind an older model is where the markers stop.
		name: "a model refusing breakpoints loses the markers and their options",
		body: `{"input":[{"role":"user","content":[{"type":"input_text","text":"hi",
			"prompt_cache_breakpoint":true}]}],"prompt_cache_options":{"ttl":"30m"},
			"prompt_cache_key":"tenant-7"}`,
		agent: Agent{Model: "openai/gpt-5.5", StripCacheBreakpoints: true},
		want: `{"input":[{"role":"user","content":[{"type":"input_text","text":"hi"}]}],
			"prompt_cache_key":"tenant-7","model":"openai/gpt-5.5"}`,
	}, {
		// Implicit caching is a separate mechanism, and a model that refuses
		// breakpoints still has it.
		name:  "the cache key survives a model that refuses breakpoints",
		body:  `{"input":[],"prompt_cache_key":"tenant-7"}`,
		agent: Agent{StripCacheBreakpoints: true},
		want:  `{"input":[],"prompt_cache_key":"tenant-7"}`,
	}, {
		name:  "content that is a bare string carries no marker to strip",
		body:  `{"input":[{"role":"user","content":"hi"}]}`,
		agent: Agent{StripCacheBreakpoints: true},
		want:  `{"input":[{"role":"user","content":"hi"}]}`,
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			composed, err := Compose([]byte(testCase.body), testCase.agent)
			if err != nil {
				t.Fatalf("Compose: %v", err)
			}
			assertSameJSON(t, composed, []byte(testCase.want))
		})
	}
}

func TestComposeErrors(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		agent   Agent
		wantErr string
	}{{
		name:    "body is not JSON",
		body:    `not json`,
		agent:   Agent{Model: "openai/gpt-5.5"},
		wantErr: "request body is not valid JSON",
	}, {
		name:    "empty body",
		body:    ``,
		agent:   Agent{Model: "openai/gpt-5.5"},
		wantErr: "request body is not valid JSON",
	}, {
		name:    "body is JSON but not an object",
		body:    `["input"]`,
		agent:   Agent{Model: "openai/gpt-5.5"},
		wantErr: "request body must be a JSON object",
	}, {
		name:    "text is not an object",
		body:    `{"input":[],"text":"nope"}`,
		agent:   Agent{Verbosity: "low"},
		wantErr: "text must be an object",
	}, {
		name:    "instructions is not a string",
		body:    `{"input":[],"instructions":42}`,
		agent:   Agent{Instructions: "You are a hound."},
		wantErr: "instructions must be a string",
	}, {
		name:    "reasoning is not an object",
		body:    `{"input":[],"reasoning":"high"}`,
		agent:   Agent{ReasoningEffort: "low"},
		wantErr: "reasoning must be an object",
	}}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := Compose([]byte(testCase.body), testCase.agent)
			if err == nil {
				t.Fatal("want an error, got none")
			}
			if !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("error %q does not mention %q", err, testCase.wantErr)
			}
			// The message is the 400 the caller reads, and a Go type in it tells
			// them about chancery rather than about what they sent.
			for _, leak := range []string{"json.RawMessage", "Go value", "Go struct"} {
				if strings.Contains(err.Error(), leak) {
					t.Errorf("error %q names %q", err, leak)
				}
			}
		})
	}
}

func assertSameJSON(t *testing.T, got, want []byte) {
	t.Helper()
	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("decode composed body: %v", err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("decode wanted body: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("got %s\nwant %s", got, want)
	}
}
