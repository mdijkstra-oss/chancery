package http

import (
	"encoding/json"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func shouldEnablePromptCaching(model string) bool {
	return strings.Contains(model, "claude") || strings.Contains(model, "anthropic")
}

func wrapToolsWithCache(tools []openai.Tool, enableCaching bool) []ToolWithCache {
	wrapped := make([]ToolWithCache, len(tools))
	for i, tool := range tools {
		wrapped[i] = ToolWithCache{Tool: tool}
	}
	if enableCaching && len(wrapped) > 0 {
		wrapped[len(wrapped)-1].CacheControl = &CacheControl{Type: "ephemeral"}
	}
	return wrapped
}

func shouldAddBreakpoint(accumulated, nextThreshold, breakpointsUsed, maxBreakpoints int) bool {
	return accumulated >= nextThreshold && breakpointsUsed < maxBreakpoints
}

func buildMessageWithCache(msg Message) MessageWithCache {
	return MessageWithCache{
		Role:         msg.Role,
		Content:      msg.Content,
		ToolCalls:    msg.ToolCalls,
		ToolCallID:   msg.ToolCallID,
		CacheControl: &CacheControl{Type: "ephemeral"},
	}
}

func addCacheBreakpoints(messages []json.RawMessage, enableCaching bool, cacheInterval int) ([]json.RawMessage, []CacheBreakpointInfo) {
	if !enableCaching || cacheInterval <= 0 {
		return messages, nil
	}

	result := make([]json.RawMessage, len(messages))
	accumulated := 0
	nextThreshold := cacheInterval
	breakpointsUsed := 0
	maxBreakpoints := 3
	var breakpoints []CacheBreakpointInfo

	for i, rawMsg := range messages {
		msg := unmarshalMessage(rawMsg)
		if msg.Role == "" {
			result[i] = rawMsg
			continue
		}

		msgTokens := estimateMessageTokens(msg)
		accumulated += msgTokens

		if shouldAddBreakpoint(accumulated, nextThreshold, breakpointsUsed, maxBreakpoints) {
			msgWithCache := buildMessageWithCache(msg)
			msgJSON, _ := json.Marshal(msgWithCache)
			result[i] = msgJSON

			breakpoints = append(breakpoints, CacheBreakpointInfo{
				MessageIndex:  i,
				TokenPos:      accumulated,
				BreakpointNum: breakpointsUsed + 2,
			})

			breakpointsUsed++
			nextThreshold += cacheInterval
		} else {
			result[i] = rawMsg
		}
	}

	return result, breakpoints
}

func calculateCachedTokens(breakdown TokenBreakdown, breakpoints []CacheBreakpointInfo, enableCaching, hasTools bool) int {
	if !enableCaching {
		return 0
	}

	if len(breakpoints) == 0 {
		if hasTools {
			return breakdown.System + breakdown.ToolDefs
		}
		return 0
	}

	maxPos := 0
	for _, bp := range breakpoints {
		if bp.TokenPos > maxPos {
			maxPos = bp.TokenPos
		}
	}
	return maxPos
}
