package http

import (
	"encoding/json"
	"regexp"
)

var modeMarker = regexp.MustCompile(`^<!--\s*prompt:\s*(\w+)\s*-->$`)

func ExpandMessages(messages []json.RawMessage, modes map[string]string) []json.RawMessage {
	result := make([]json.RawMessage, len(messages))
	for i, raw := range messages {
		result[i] = expandMessage(raw, modes)
	}
	return result
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
