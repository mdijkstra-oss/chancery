package sse

import (
	"encoding/json"

	"hermes-logos/internal/protocol"
)

type Event struct {
	Type string
	Data string
}

func BuildCompletedEvent(usage *protocol.UsageResponse) Event {
	data, _ := json.Marshal(map[string]any{
		"response": map[string]any{
			"status": "completed",
			"usage":  usage,
		},
	})
	return Event{
		Type: "response.completed",
		Data: string(data),
	}
}

func BuildFailedEvent(errorType, message string) Event {
	data, _ := json.Marshal(map[string]any{
		"response": map[string]any{
			"status": "failed",
			"error": map[string]any{
				"type":    errorType,
				"message": message,
			},
		},
	})
	return Event{
		Type: "response.failed",
		Data: string(data),
	}
}
