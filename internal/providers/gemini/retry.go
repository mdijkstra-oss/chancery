package gemini

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"google.golang.org/genai"
)

func ExtractRetryDelay(err error) time.Duration {
	var apiErr *genai.APIError
	if !errors.As(err, &apiErr) {
		return 0
	}
	for _, detail := range apiErr.Details {
		if d := parseRetryDelay(detail); d > 0 {
			return d
		}
	}
	return 0
}

func parseRetryDelay(detail map[string]any) time.Duration {
	typ, _ := detail["@type"].(string)
	if typ != "type.googleapis.com/google.rpc.RetryInfo" {
		return 0
	}
	raw, ok := detail["retryDelay"]
	if !ok {
		return 0
	}
	return parseDurationValue(raw)
}

func parseDurationValue(raw any) time.Duration {
	switch v := raw.(type) {
	case string:
		return parseGoogleDuration(v)
	case float64:
		return time.Duration(v * float64(time.Second))
	default:
		return 0
	}
}

func parseGoogleDuration(s string) time.Duration {
	if !strings.HasSuffix(s, "s") {
		return 0
	}
	numStr := strings.TrimSuffix(s, "s")
	if strings.Contains(numStr, ".") {
		f, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return 0
		}
		return time.Duration(f * float64(time.Second))
	}
	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil {
		return 0
	}
	return time.Duration(n) * time.Second
}
