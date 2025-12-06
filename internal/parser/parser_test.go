package parser

import (
	"strings"
	"testing"

	th "hermes-logos/internal/lib/test-helpers"
)

func testMeta() RequestMeta {
	return RequestMeta{
		Model:     "test-model",
		RequestID: "req123",
	}
}

func TestProcessFullMessage(t *testing.T) {
	tests := []struct {
		Name           string
		Chunks         []string
		ExpectContains []string
		ExpectReply    bool
	}{
		{
			Name:           "plain text passthrough",
			Chunks:         []string{"Hello world"},
			ExpectContains: []string{"Hello world"},
		},
		{
			Name:           "single LLMCMD in one chunk",
			Chunks:         []string{"Hello <LLMCMD><payload>{\"action\":\"test\"}</payload></LLMCMD> world"},
			ExpectContains: []string{"Hello ", "<LLMCMD>", "<llm>test-model</llm>", "<req>req123</req>", "{\"action\":\"test\"}", "<sig>TODO</sig>", "</LLMCMD>", " world"},
		},
		{
			Name:           "LLMCMD split across chunks",
			Chunks:         []string{"Hello <LLM", "CMD><pay", "load>{\"a\":1}</payl", "oad></LLMCMD> done"},
			ExpectContains: []string{"Hello ", "<LLMCMD>", "<llm>test-model</llm>", "{\"a\":1}", "<sig>TODO</sig>", "</LLMCMD>", " done"},
		},
		{
			Name:           "reply attribute detected",
			Chunks:         []string{"<LLMCMD reply><payload>{}</payload></LLMCMD>"},
			ExpectContains: []string{"<LLMCMD reply>", "<llm>test-model</llm>"},
			ExpectReply:    true,
		},
		{
			Name:           "multiple commands",
			Chunks:         []string{"A<LLMCMD><payload>1</payload></LLMCMD>B<LLMCMD><payload>2</payload></LLMCMD>C"},
			ExpectContains: []string{"A", "<LLMCMD>", "1", "<sig>TODO</sig>", "B", "2", "C"},
		},
		{
			Name:           "angle bracket not a tag",
			Chunks:         []string{"1 < 2 and 3 > 1"},
			ExpectContains: []string{"1 < 2 and 3 > 1"},
		},
		{
			Name:           "arbitrary tags before payload pass through",
			Chunks:         []string{"<LLMCMD><message>Opening doors</message><payload>{}</payload></LLMCMD>"},
			ExpectContains: []string{"<LLMCMD>", "<message>Opening doors</message>", "<payload>", "{}", "</payload>", "</LLMCMD>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			output, sawReply := processChunks(tt.Chunks)

			for _, want := range tt.ExpectContains {
				if !strings.Contains(output, want) {
					t.Errorf("output missing %q\nfull output: %s", want, output)
				}
			}

			if tt.ExpectReply && !sawReply {
				t.Error("expected SawReply to be true")
			}
		})
	}
}

func TestPartialChunking(t *testing.T) {
	input := "prefix<LLMCMD><payload>{\"key\":\"value\"}</payload></LLMCMD>suffix"
	mustContain := []string{"prefix", "<LLMCMD>", "<llm>test-model</llm>", "{\"key\":\"value\"}", "<sig>TODO</sig>", "</LLMCMD>", "suffix"}

	tests := []struct {
		Name     string
		Input    int
		Expected bool
	}{
		{"chunk size 1", 1, true},
		{"chunk size 2", 2, true},
		{"chunk size 3", 3, true},
		{"chunk size 5", 5, true},
		{"chunk size 7", 7, true},
		{"chunk size 11", 11, true},
		{"chunk size 13", 13, true},
		{"chunk size 17", 17, true},
		{"chunk size full", len(input), true},
	}

	th.RunFunctionTests(t, tests, func(size int) bool {
		chunks := splitIntoChunks(input, size)
		output, _ := processChunks(chunks)

		for _, want := range mustContain {
			if !strings.Contains(output, want) {
				return false
			}
		}
		return true
	})
}

func processChunks(chunks []string) (string, bool) {
	state := NewState()
	meta := testMeta()
	var output string
	sawReply := false

	for _, chunk := range chunks {
		result := Process(state, chunk, meta)
		state = result.State
		output += result.Output
		if result.SawReply {
			sawReply = true
		}
	}

	return output, sawReply
}

func splitIntoChunks(s string, size int) []string {
	var chunks []string
	for len(s) > 0 {
		if len(s) <= size {
			chunks = append(chunks, s)
			break
		}
		chunks = append(chunks, s[:size])
		s = s[size:]
	}
	return chunks
}
