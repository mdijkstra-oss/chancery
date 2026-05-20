package gemini

import (
	"errors"
	"fmt"
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

func FormatRetryDelay(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%dm%ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
