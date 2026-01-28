package http

import "log/slog"

func logOutgoingRequest(req ResponsesRequest, verbose bool) {
	if !verbose {
		return
	}
	slog.Info("outgoing_request",
		"model", req.Model,
		"input_count", len(req.Input),
		"tool_count", len(req.Tools),
		"stream", req.Stream,
	)
	slog.Info("raw_outgoing_request", "data", req)
}
