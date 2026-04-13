package messages

import (
	"encoding/json"
	"log/slog"
	"regexp"

	"hermes-logos/internal/prompts"
	"hermes-logos/internal/protocol"
)

var approachMarker = regexp.MustCompile(`^<!--\s*approach:\s*([\w/\-]+)\s*-->$`)
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

func marshalSystemMessage(content string) json.RawMessage {
	b, _ := json.Marshal(protocol.InputMessage{Type: "message", Role: "system", Content: content})
	return b
}

func collectApproachMarkers(messages []json.RawMessage) ([]string, []int) {
	var keys []string
	var positions []int
	for i, raw := range messages {
		if key, ok := matchDirective(approachMarker, raw); ok {
			keys = append(keys, key)
			positions = append(positions, i)
		}
	}
	return keys, positions
}

func extraIndexKeys(resolved, requested []string) []string {
	reqSet := make(map[string]bool, len(requested))
	for _, k := range requested {
		reqSet[k] = true
	}
	var extra []string
	for _, k := range resolved {
		if !reqSet[k] {
			extra = append(extra, k)
		}
	}
	return extra
}

func appendApproachContent(result []json.RawMessage, key string, entries map[string]prompts.Approach) []json.RawMessage {
	a, ok := entries[key]
	if !ok {
		return result
	}
	return append(result, marshalSystemMessage("["+key+"]\n"+a.Content))
}

func ExpandApproaches(messages []json.RawMessage, entries map[string]prompts.Approach) []json.RawMessage {
	requested, positions := collectApproachMarkers(messages)
	if len(requested) == 0 {
		return messages
	}

	resolved := prompts.ResolveApproachKeys(requested)
	extra := extraIndexKeys(resolved, requested)

	markerSet := make(map[int]bool, len(positions))
	for _, pos := range positions {
		markerSet[pos] = true
	}

	firstMarker := positions[0]
	result := make([]json.RawMessage, 0, len(messages)+len(extra))
	for i, raw := range messages {
		if !markerSet[i] {
			result = append(result, raw)
			continue
		}
		if i == firstMarker {
			for _, ek := range extra {
				result = appendApproachContent(result, ek, entries)
			}
		}
		key, _ := matchDirective(approachMarker, raw)
		if _, ok := entries[key]; !ok {
			slog.Warn("approach key not found in registry", "component", "expand", slog.Group("data", slog.String("key", key)))
		}
		result = appendApproachContent(result, key, entries)
	}
	return result
}
