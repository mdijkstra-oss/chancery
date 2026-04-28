package messages

import "encoding/json"

func ReorderToolMessages(messages []json.RawMessage) []json.RawMessage {
	result := make([]json.RawMessage, 0, len(messages))
	i := 0
	for i < len(messages) {
		if !hasType(messages[i], "function_call") {
			result = append(result, messages[i])
			i++
			continue
		}
		var calls []json.RawMessage
		for i < len(messages) && hasType(messages[i], "function_call") {
			calls = append(calls, messages[i])
			i++
		}
		var outputs, deferred []json.RawMessage
		for i < len(messages) {
			if hasType(messages[i], "function_call_output") {
				outputs = append(outputs, messages[i])
			} else if hasRole(messages[i], "system") {
				deferred = append(deferred, messages[i])
			} else {
				break
			}
			i++
		}
		result = append(result, calls...)
		result = append(result, outputs...)
		result = append(result, deferred...)
	}
	return result
}

func hasType(raw json.RawMessage, typ string) bool {
	var peek struct{ Type string `json:"type"` }
	return json.Unmarshal(raw, &peek) == nil && peek.Type == typ
}

func hasRole(raw json.RawMessage, role string) bool {
	var peek struct{ Role string `json:"role"` }
	return json.Unmarshal(raw, &peek) == nil && peek.Role == role
}
