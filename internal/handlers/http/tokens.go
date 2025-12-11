package http

import "encoding/json"

func estimateTokens(text string) int {
	return len(text) / 4
}

func estimateJSONTokens(v interface{}) int {
	data, _ := json.Marshal(v)
	return estimateTokens(string(data))
}

func sumTokens(breakdown TokenBreakdown) int {
	return breakdown.System +
		breakdown.ToolDefs +
		breakdown.UserMsgs +
		breakdown.AssistantMsgs +
		breakdown.ToolCalls +
		breakdown.ToolResponses
}

func unmarshalMessage(raw json.RawMessage) Message {
	var msg Message
	json.Unmarshal(raw, &msg)
	return msg
}

func estimateMessageTokens(msg Message) int {
	tokens := estimateTokens(msg.Content)
	if msg.ToolCalls != nil {
		tokens += estimateJSONTokens(msg.ToolCalls)
	}
	return tokens
}

func calculateTokenBreakdown(req OpenAIRequest, systemPrompt string) TokenBreakdown {
	breakdown := TokenBreakdown{
		System:   estimateTokens(systemPrompt),
		ToolDefs: estimateJSONTokens(req.Tools),
	}

	for _, rawMsg := range req.Messages {
		msg := unmarshalMessage(rawMsg)

		switch msg.Role {
		case "system":
		case "user":
			breakdown.UserMsgs += estimateTokens(msg.Content)
		case "assistant":
			breakdown.AssistantMsgs += estimateTokens(msg.Content)
			if msg.ToolCalls != nil {
				breakdown.ToolCalls += estimateJSONTokens(msg.ToolCalls)
			}
		case "tool":
			breakdown.ToolResponses += estimateTokens(msg.Content)
		default:
			panic("unknown role: " + msg.Role)
		}
	}

	return breakdown
}
