package messages

import (
	"encoding/json"

	"github.com/mdijkstra-oss/chancery/internal/fn"
)

type emptyPeek struct {
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content string `json:"content"`
}

func isPlainMessageType(t string) bool {
	return t == "" || t == "message"
}

func DropEmptyContent(msgs []json.RawMessage) []json.RawMessage {
	return fn.Filter(msgs, hasContent)
}

func hasContent(raw json.RawMessage) bool {
	var p emptyPeek
	if json.Unmarshal(raw, &p) != nil {
		return true
	}
	if isPlainMessageType(p.Type) && p.Content == "" {
		return false
	}
	return true
}
