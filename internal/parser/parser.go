package parser

import (
	"fmt"
	"time"

	"hermes-logos/internal/lib/utils"
)

func Process(state State, chunk string, meta RequestMeta) Result {
	switch state.Phase {
	case Outside:
		return processOutside(state, chunk, meta)
	case InCmd:
		return processInCmd(state, chunk, meta)
	case InPayload:
		return processInPayload(state, chunk, meta)
	default:
		panic(fmt.Sprintf("unknown phase: %d", state.Phase))
	}
}

func processOutside(state State, chunk string, meta RequestMeta) Result {
	return scanLoop(state, chunk, meta, false, matchOutside)
}

func matchOutside(buffer string, _ State, meta RequestMeta) MatchOutcome {
	match := MatchTagWithArgs(buffer, TagLLMCMDPrefix)
	if match != FullMatch {
		return MatchOutcome{Match: match}
	}
	isReply := ParseCmdAttrs(buffer)
	return MatchOutcome{
		Match:    FullMatch,
		Output:   injectMetadata(buffer, meta),
		NewState: TransitionToCmd(isReply),
		SawReply: isReply,
	}
}

func processInCmd(state State, chunk string, meta RequestMeta) Result {
	return scanLoop(state, chunk, meta, state.IsReply, matchInCmd)
}

func matchInCmd(buffer string, _ State, _ RequestMeta) MatchOutcome {
	payloadMatch := MatchPayloadOpen(buffer)
	closeMatch := MatchCmdClose(buffer)

	if payloadMatch == FullMatch {
		return MatchOutcome{
			Match:    FullMatch,
			Output:   TagPayloadOut,
			NewState: TransitionToPayload(),
		}
	}

	if closeMatch == FullMatch {
		return MatchOutcome{
			Match:    FullMatch,
			Output:   TagLLMCMDCloseOut,
			NewState: TransitionToOutside(),
		}
	}

	if payloadMatch == PartialMatch || closeMatch == PartialMatch {
		return MatchOutcome{Match: PartialMatch}
	}

	return MatchOutcome{Match: NoMatch}
}

func processInPayload(state State, chunk string, meta RequestMeta) Result {
	combined := state.PayloadBuffer + chunk
	sawReply := state.IsReply

	closeIdx := findClosePayload(combined)

	if closeIdx == -1 {
		return Result{State: withPayload(state, combined), Output: "", SawReply: sawReply}
	}

	payloadContent := combined[:closeIdx]
	closeTagLen := findClosePayloadTagLen(combined[closeIdx:])
	output := payloadContent + TagPayloadCloseOut + formatSignature()

	newState := TransitionToCmd(false)
	remaining := combined[closeIdx+closeTagLen:]

	if len(remaining) > 0 {
		return continueProcess(newState, remaining, output, meta, sawReply)
	}

	return Result{State: newState, Output: output, SawReply: sawReply}
}

func continueProcess(state State, remaining string, outputSoFar string, meta RequestMeta, sawReply bool) Result {
	result := Process(state, remaining, meta)
	return Result{State: result.State, Output: outputSoFar + result.Output, SawReply: sawReply || result.SawReply}
}

type AccumStep struct {
	Input     string
	Buffer    string
	Output    string
	Exhausted bool
}

type MatchOutcome struct {
	Match    TagMatch
	Output   string
	NewState State
	SawReply bool
}

type Matcher func(buffer string, state State, meta RequestMeta) MatchOutcome

func scanLoop(state State, chunk string, meta RequestMeta, sawReply bool, match Matcher) Result {
	acc := AccumStep{Input: chunk, Buffer: state.Buffer, Output: ""}

	for len(acc.Input) > 0 || len(acc.Buffer) > 0 {
		acc = accumulate(acc.Input, acc.Buffer, acc.Output)
		if acc.Exhausted {
			return Result{State: clearBuffer(state), Output: acc.Output, SawReply: sawReply}
		}

		outcome := match(acc.Buffer, state, meta)
		if outcome.SawReply {
			sawReply = true
		}

		switch outcome.Match {
		case FullMatch:
			output := acc.Output + outcome.Output
			if len(acc.Input) > 0 {
				return continueProcess(outcome.NewState, acc.Input, output, meta, sawReply)
			}
			return Result{State: outcome.NewState, Output: output, SawReply: sawReply}

		case PartialMatch:
			if len(acc.Input) == 0 {
				return Result{State: withBuffer(state, acc.Buffer), Output: acc.Output, SawReply: sawReply}
			}

		case NoMatch:
			var rejected string
			acc.Buffer, acc.Input, rejected = rejectBufferHead(acc.Buffer, acc.Input)
			acc.Output += rejected
		}
	}

	return Result{State: clearBuffer(state), Output: acc.Output, SawReply: sawReply}
}

func accumulate(input, buffer, output string) AccumStep {
	if len(buffer) == 0 {
		tagIdx := FindTagStart(input)
		if tagIdx == -1 {
			return AccumStep{Input: "", Buffer: "", Output: output + input, Exhausted: true}
		}
		if tagIdx > 0 {
			output += input[:tagIdx]
			input = input[tagIdx:]
		}
		return AccumStep{Input: input[1:], Buffer: string(input[0]), Output: output, Exhausted: false}
	}
	if len(input) > 0 {
		return AccumStep{Input: input[1:], Buffer: buffer + string(input[0]), Output: output, Exhausted: false}
	}
	return AccumStep{Input: "", Buffer: buffer, Output: output, Exhausted: false}
}

func rejectBufferHead(buffer, input string) (newBuffer, newInput, rejected string) {
	return "", buffer[1:] + input, string(buffer[0])
}

func findClosePayload(s string) int {
	norm := NormalizeTag(s)
	idx := 0
	normIdx := 0
	for normIdx <= len(norm)-len(TagPayloadClose) {
		if norm[normIdx:normIdx+len(TagPayloadClose)] == TagPayloadClose {
			return idx
		}
		if idx < len(s) {
			idx++
			normIdx = len(NormalizeTag(s[:idx]))
		} else {
			break
		}
	}
	return -1
}

func findClosePayloadTagLen(s string) int {
	for i := 1; i <= len(s); i++ {
		if NormalizeTag(s[:i]) == TagPayloadClose {
			return i
		}
	}
	return len(TagPayloadClose)
}

func injectMetadata(tag string, meta RequestMeta) string {
	ts := time.Now().UTC().Format(time.RFC3339)
	cmdID := utils.GenerateID()
	return fmt.Sprintf("%s<llm>%s</llm><req>%s</req><ts>%s</ts><cmd_id>%s</cmd_id>",
		tag, meta.Model, meta.RequestID, ts, cmdID)
}

func formatSignature() string {
	return "<sig>TODO</sig>"
}
