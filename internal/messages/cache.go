package messages

import (
	"encoding/json"
	"regexp"
)

var cacheMarker = regexp.MustCompile(`^<!--\s*(cache)\s*-->$`)

func isCacheMarker(raw json.RawMessage) bool {
	_, ok := matchDirective(cacheMarker, raw)
	return ok
}

func ExtractCacheBreakpoints(messages []json.RawMessage) ([]json.RawMessage, map[int]bool) {
	breakpoints := make(map[int]bool)
	cleaned := make([]json.RawMessage, 0, len(messages))
	for _, raw := range messages {
		if isCacheMarker(raw) {
			if len(cleaned) > 0 {
				breakpoints[len(cleaned)-1] = true
			}
			continue
		}
		cleaned = append(cleaned, raw)
	}
	if len(breakpoints) == 0 {
		return cleaned, nil
	}
	return cleaned, breakpoints
}
