package sse

import "testing"

func TestParseSSEOutput(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantText  string
		wantCalls int
	}{
		{
			name:     "text deltas",
			input:    "event: response.output_text.delta\ndata: {\"delta\":\"hello \"}\n\nevent: response.output_text.delta\ndata: {\"delta\":\"world\"}\n\n",
			wantText: "hello world",
		},
		{
			name:      "tool call",
			input:     "event: response.output_item.added\ndata: {\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"search\"}}\n\nevent: response.function_call_arguments.delta\ndata: {\"delta\":\"{\\\"q\\\":\\\"test\\\"}\"}\n\n",
			wantText:  "",
			wantCalls: 1,
		},
		{
			name:      "mixed text and tool call",
			input:     "event: response.output_text.delta\ndata: {\"delta\":\"thinking...\"}\n\nevent: response.output_item.added\ndata: {\"item\":{\"type\":\"function_call\",\"call_id\":\"call_1\",\"name\":\"search\"}}\n\nevent: response.function_call_arguments.delta\ndata: {\"delta\":\"{\\\"q\\\":\\\"test\\\"}\"}\n\n",
			wantText:  "thinking...",
			wantCalls: 1,
		},
		{
			name:     "empty input",
			input:    "",
			wantText: "",
		},
		{
			name:     "reasoning events ignored",
			input:    "event: response.reasoning_summary_text.delta\ndata: {\"delta\":\"internal thought\"}\n\nevent: response.output_text.delta\ndata: {\"delta\":\"visible\"}\n\n",
			wantText: "visible",
		},
		{
			name:      "multiple tool calls",
			input:     "event: response.output_item.added\ndata: {\"item\":{\"type\":\"function_call\",\"call_id\":\"c1\",\"name\":\"search\"}}\n\nevent: response.function_call_arguments.delta\ndata: {\"delta\":\"{\\\"q\\\":\\\"a\\\"}\"}\n\nevent: response.output_item.added\ndata: {\"item\":{\"type\":\"function_call\",\"call_id\":\"c2\",\"name\":\"read\"}}\n\nevent: response.function_call_arguments.delta\ndata: {\"delta\":\"{\\\"path\\\":\\\"x\\\"}\"}\n\n",
			wantText:  "",
			wantCalls: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSSEOutput([]byte(tt.input))
			if got.Text != tt.wantText {
				t.Errorf("Text = %q, want %q", got.Text, tt.wantText)
			}
			if len(got.ToolCalls) != tt.wantCalls {
				t.Errorf("ToolCalls = %d, want %d", len(got.ToolCalls), tt.wantCalls)
			}
		})
	}
}

func TestParseSSEOutput_ToolCallDetails(t *testing.T) {
	input := "event: response.output_item.added\ndata: {\"item\":{\"type\":\"function_call\",\"call_id\":\"call_abc\",\"name\":\"search\"}}\n\nevent: response.function_call_arguments.delta\ndata: {\"delta\":\"{\\\"query\\\":\"}\n\nevent: response.function_call_arguments.delta\ndata: {\"delta\":\"\\\"hello\\\"}\"}\n\n"

	got := ParseSSEOutput([]byte(input))
	if len(got.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(got.ToolCalls))
	}
	tc := got.ToolCalls[0]
	if tc.Name != "search" {
		t.Errorf("Name = %q, want %q", tc.Name, "search")
	}
	if tc.CallID != "call_abc" {
		t.Errorf("CallID = %q, want %q", tc.CallID, "call_abc")
	}
	if tc.Arguments != `{"query":"hello"}` {
		t.Errorf("Arguments = %q, want %q", tc.Arguments, `{"query":"hello"}`)
	}
}

func TestCollector_Write(t *testing.T) {
	c := &Collector{}
	c.Write([]byte("event: response.output_text.delta\n"))
	c.Write([]byte("data: {\"delta\":\"hi\"}\n\n"))

	got := c.Result()
	if got.Text != "hi" {
		t.Errorf("Text = %q, want %q", got.Text, "hi")
	}
}
