package parser

import (
	"testing"

	th "hermes-logos/internal/lib/test-helpers"
)

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		Name     string
		Input    string
		Expected string
	}{
		{"already normalized", "<LLMCMD>", "<LLMCMD>"},
		{"lowercase", "<llmcmd>", "<LLMCMD>"},
		{"mixed case", "<LlmCmd>", "<LLMCMD>"},
		{"space after <", "< LLMCMD>", "<LLMCMD>"},
		{"space after /", "</ LLMCMD>", "</LLMCMD>"},
		{"multiple spaces", "<LLMCMD  reply>", "<LLMCMD REPLY>"},
		{"tabs", "<LLMCMD\treply>", "<LLMCMD REPLY>"},
	}

	th.RunFunctionTests(t, tests, NormalizeTag)
}

func TestMatchTagWithArgs(t *testing.T) {
	tests := []struct {
		Name     string
		Input    string
		Expected TagMatch
	}{
		{"empty", "", NoMatch},
		{"just <", "<", PartialMatch},
		{"<L", "<L", PartialMatch},
		{"<LLMCMD", "<LLMCMD", PartialMatch},
		{"<LLMCMD ", "<LLMCMD ", PartialMatch},
		{"<LLMCMD>", "<LLMCMD>", FullMatch},
		{"<LLMCMD reply>", "<LLMCMD reply>", FullMatch},
		{"<LLMCMD foo bar>", "<LLMCMD foo bar>", FullMatch},
		{"<X", "<X", NoMatch},
		{"<OTHER>", "<OTHER>", NoMatch},
		{"lowercase", "<llmcmd>", FullMatch},
		{"mixed case", "<LlmCmd ReP>", FullMatch},
		{"space after <", "< LLMCMD>", FullMatch},
		{"space after < partial", "< LLM", PartialMatch},
	}

	th.RunFunctionTests(t, tests, func(in string) TagMatch {
		return MatchTagWithArgs(in, TagLLMCMDPrefix)
	})
}

func TestParseCmdAttrs(t *testing.T) {
	tests := []struct {
		Name     string
		Input    string
		Expected bool
	}{
		{"no attrs", "<LLMCMD>", false},
		{"reply attr", "<LLMCMD reply>", true},
		{"reply with space", "<LLMCMD  reply >", true},
		{"other attr", "<LLMCMD foo>", false},
		{"reply among others", "<LLMCMD foo reply bar>", true},
		{"uppercase REPLY", "<LLMCMD REPLY>", true},
		{"mixed case rEpLy", "<LLMCMD rEpLy>", true},
	}

	th.RunFunctionTests(t, tests, ParseCmdAttrs)
}

func TestMatchTag(t *testing.T) {
	type tagInput struct {
		Buffer string
		Tag    string
	}

	tests := []struct {
		Name     string
		Input    tagInput
		Expected TagMatch
	}{
		{"full match", tagInput{"</payload>", "</PAYLOAD>"}, FullMatch},
		{"full match uppercase", tagInput{"</PAYLOAD>", "</PAYLOAD>"}, FullMatch},
		{"partial match", tagInput{"</pay", "</PAYLOAD>"}, PartialMatch},
		{"partial match mixed", tagInput{"</PaY", "</PAYLOAD>"}, PartialMatch},
		{"no match", tagInput{"<X", "</PAYLOAD>"}, NoMatch},
	}

	th.RunFunctionTests(t, tests, func(in tagInput) TagMatch {
		return MatchTag(in.Buffer, in.Tag)
	})
}
