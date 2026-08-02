package sse

import (
	"encoding/json"

	"github.com/mdijkstra-oss/chancery/internal/protocol"
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

func TextDeltaEvent(text string) Event {
	data, _ := json.Marshal(map[string]string{"delta": text})
	return Event{Type: "response.output_text.delta", Data: string(data)}
}

func ReasoningDeltaEvent(text string) Event {
	data, _ := json.Marshal(map[string]string{"delta": text})
	return Event{Type: "response.reasoning_summary_text.delta", Data: string(data)}
}

func FinishReasonToEvent[T ~string](errs map[T]string, reason T) *Event {
	message, ok := errs[reason]
	if !ok {
		return nil
	}
	event := BuildFailedEvent(string(reason), message)
	return &event
}
