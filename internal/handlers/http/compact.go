package http

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type compactedArgs struct {
	Summary string `json:"summary"`
}

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

func stripForCompaction(messages []json.RawMessage) []json.RawMessage {
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

func estimateTokens(input []json.RawMessage) int {
	total := 0
	for _, raw := range input {
		total += len(raw)
	}
	return total / 4
}

func shouldCompact(compactAt int, tokens int) bool {
	return compactAt > 0 && tokens > compactAt
}

func appendCompactTrigger(messages []json.RawMessage) []json.RawMessage {
	trigger := InputMessage{Type: "message", Role: "user", Content: "Summarize the entire conversation above now."}
	triggerJSON, _ := json.Marshal(trigger)
	return append(messages, triggerJSON)
}

func buildCompactRequest(model, systemPrompt string, messages []json.RawMessage) ResponsesRequest {
	return ResponsesRequest{
		Model:  model,
		Input:  prependSystemMessage(systemPrompt, appendCompactTrigger(messages)),
		Stream: true,
		Store:  false,
	}
}

func writeSSEEvent(w io.Writer, flusher http.Flusher, event, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	flusher.Flush()
}

func buildCompletedData(usage *UsageResponse) string {
	event := ResponseCompletedEvent{
		Type:     "response.completed",
		Response: ResponseObject{Usage: usage},
	}
	data, _ := json.Marshal(event)
	return string(data)
}

func buildCompactedDoneData(summary string) string {
	argsJSON, _ := json.Marshal(compactedArgs{Summary: summary})
	item := map[string]interface{}{
		"item": map[string]interface{}{
			"type":      "function_call",
			"call_id":   "compact_0",
			"name":      "compacted",
			"arguments": string(argsJSON),
		},
	}
	data, _ := json.Marshal(item)
	return string(data)
}

func streamCompaction(src io.Reader, w io.Writer, flusher http.Flusher) (*UsageResponse, error) {
	writeSSEEvent(w, flusher, "response.output_item.added",
		`{"item":{"type":"function_call","name":"compacted"}}`)

	scanner := bufio.NewScanner(src)
	var summary strings.Builder
	var currentEvent string
	var usage *UsageResponse

	for scanner.Scan() {
		line := scanner.Text()

		if eventType, ok := strings.CutPrefix(line, "event: "); ok {
			currentEvent = eventType
			continue
		}

		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}

		if currentEvent == "response.output_text.delta" {
			delta := extractTextDelta(data)
			if delta == "" {
				continue
			}
			summary.WriteString(delta)
			escapedDelta, _ := json.Marshal(delta)
			writeSSEEvent(w, flusher, "response.function_call_arguments.delta",
				fmt.Sprintf(`{"delta":%s}`, string(escapedDelta)))
		}

		if currentEvent == "response.completed" {
			usage = extractCompletedUsage(data)
		}
	}

	writeSSEEvent(w, flusher, "response.output_item.done", buildCompactedDoneData(summary.String()))
	writeSSEEvent(w, flusher, "response.completed", buildCompletedData(usage))

	return usage, scanner.Err()
}
