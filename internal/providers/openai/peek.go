package openai

import (
	"encoding/json"

	"github.com/matthijn/hermes-logos/internal/protocol"
)

type completedEnvelope struct {
	Response struct {
		Usage *protocol.UsageResponse `json:"usage"`
	} `json:"response"`
}

func ExtractUsageFromCompleted(data []byte) *protocol.UsageResponse {
	var env completedEnvelope
	if json.Unmarshal(data, &env) != nil {
		return nil
	}
	return env.Response.Usage
}
