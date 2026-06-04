package messages

import "encoding/json"

type emptyPeek struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

func isPlainMessageType(t string) bool {
	return t == "" || t == "message"
}

func DropEmptyContent(msgs []json.RawMessage) []json.RawMessage {
	result := make([]json.RawMessage, 0, len(msgs))
	for _, m := range msgs {
		var p emptyPeek
		if json.Unmarshal(m, &p) != nil {
			result = append(result, m)
			continue
		}
		if isPlainMessageType(p.Type) && p.Content == "" {
			continue
		}
		result = append(result, m)
	}
	return result
}
