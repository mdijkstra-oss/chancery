package http

import (
	"encoding/json"
	"log/slog"
)

func logOutgoingRequest(req OpenAIRequest, verbose bool) {
	slog.Info("outgoing_request",
		"model", req.Model,
		"message_count", len(req.Messages),
		"tool_count", len(req.Tools),
		"stream", req.Stream,
	)

	if verbose {
		reqJSON, _ := json.Marshal(req)
		slog.Info("raw_outgoing_request", "data", string(reqJSON))
	}
}
