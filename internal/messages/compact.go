package messages

import (
	"encoding/json"

	"github.com/matthijn/hermes-logos/internal/protocol"
)

func isCompactableMessage(raw json.RawMessage) bool {
	var peek struct {
		Type string `json:"type"`
		Role string `json:"role"`
	}
	if json.Unmarshal(raw, &peek) != nil {
		return false
	}
	if peek.Role == "system" {
		return false
	}
	if peek.Type == "reasoning" {
		return false
	}
	return true
}

func StripForCompaction(messages []json.RawMessage) []json.RawMessage {
	result := make([]json.RawMessage, 0, len(messages))
	for _, raw := range messages {
		if isCompactableMessage(raw) {
			result = append(result, raw)
		}
	}
	return dropLast(result)
}

func dropLast(messages []json.RawMessage) []json.RawMessage {
	if len(messages) == 0 {
		return messages
	}
	return messages[:len(messages)-1]
}

func ShouldCompact(compactAt int, tokens int) bool {
	return compactAt > 0 && tokens > compactAt
}

func appendCompactTrigger(messages []json.RawMessage) []json.RawMessage {
	trigger := protocol.InputMessage{Type: "message", Role: "user", Content: "Summarize the entire conversation above now."}
	triggerJSON, _ := json.Marshal(trigger)
	return append(messages, triggerJSON)
}

func BuildCompactParams(model, systemPrompt string, messages []json.RawMessage) protocol.RequestParams {
	return protocol.RequestParams{
		Model:        model,
		SystemPrompt: systemPrompt,
		Messages:     appendCompactTrigger(messages),
	}
}
