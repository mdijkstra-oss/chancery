package messages

import (
	"encoding/json"
	"regexp"

	"github.com/matthijn/hermes-logos/internal/protocol"
)

var modeMarker = regexp.MustCompile(`^<!--\s*prompt:\s*(\w+)\s*-->$`)

func ExpandMessages(messages []json.RawMessage, modes map[string]string) []json.RawMessage {
	lastIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if _, ok := matchDirective(modeMarker, messages[i]); ok {
			lastIdx = i
			break
		}
	}

	result := make([]json.RawMessage, 0, len(messages))
	for i, raw := range messages {
		if _, ok := matchDirective(modeMarker, raw); ok && i != lastIdx {
			continue
		}
		if i == lastIdx {
			result = append(result, expandMessage(raw, modes))
		} else {
			result = append(result, raw)
		}
	}
	return result
}

func matchDirective(re *regexp.Regexp, raw json.RawMessage) (string, bool) {
	var msg protocol.InputMessage
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

func expandMessage(raw json.RawMessage, modes map[string]string) json.RawMessage {
	var msg protocol.InputMessage
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
