package http

import (
	"encoding/json"
	"regexp"
)

var modeMarker = regexp.MustCompile(`^<!--\s*prompt:\s*(\w+)\s*-->$`)
var reasoningMarker = regexp.MustCompile(`^<!--\s*reasoning:\s*(\w+)\s*-->$`)
var modelMarker = regexp.MustCompile(`^<!--\s*model:\s*([\w.\-]+)\s*-->$`)
var verbosityMarker = regexp.MustCompile(`^<!--\s*verbosity:\s*(\w+)\s*-->$`)

func ExpandMessages(messages []json.RawMessage, modes map[string]string) []json.RawMessage {
	result := make([]json.RawMessage, len(messages))
	for i, raw := range messages {
		result[i] = expandMessage(raw, modes)
	}
	return result
}

func matchDirective(re *regexp.Regexp, raw json.RawMessage) (string, bool) {
	var msg InputMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return "", false
	}
	if msg.Role != "system" {
		return "", false
	}
	match := re.FindStringSubmatch(msg.Content)
	if match == nil {
		return "", false
	}
	return match[1], true
}

func extractDirective(re *regexp.Regexp, messages []json.RawMessage) ([]json.RawMessage, string) {
	value := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if v, ok := matchDirective(re, messages[i]); ok && value == "" {
			value = v
		}
	}
	result := make([]json.RawMessage, 0, len(messages))
	for _, raw := range messages {
		if _, ok := matchDirective(re, raw); !ok {
			result = append(result, raw)
		}
	}
	return result, value
}

func ExtractReasoningEffort(messages []json.RawMessage) ([]json.RawMessage, string) {
	return extractDirective(reasoningMarker, messages)
}

func ExtractModel(messages []json.RawMessage) ([]json.RawMessage, string) {
	return extractDirective(modelMarker, messages)
}

func ExtractVerbosity(messages []json.RawMessage) ([]json.RawMessage, string) {
	return extractDirective(verbosityMarker, messages)
}

func expandMessage(raw json.RawMessage, modes map[string]string) json.RawMessage {
	var msg InputMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		return raw
	}
	if msg.Role != "system" {
		return raw
	}
	match := modeMarker.FindStringSubmatch(msg.Content)
	if match == nil {
		return raw
	}
	prompt, ok := modes[match[1]]
	if !ok {
		return raw
	}
	msg.Content = prompt
	expanded, err := json.Marshal(msg)
	if err != nil {
		return raw
	}
	return expanded
}
