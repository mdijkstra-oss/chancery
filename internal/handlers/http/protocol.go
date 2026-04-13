package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"hermes-logos/internal/prompts"
)

type RequestParams struct {
	Model            string
	SystemPrompt     string
	ReasoningEffort  string
	ReasoningSummary string
	Verbosity        string
	ServiceTier      string
	ToolChoice       string
	Temperature      *float64
	Tools            []json.RawMessage
	Messages         []json.RawMessage
	ResponseFormat   json.RawMessage
}

type ProtocolFuncs struct {
	BuildRequest  func(RequestParams) any
	ForwardStream func(ctx context.Context, src io.Reader, dst io.Writer, flusher http.Flusher, inspect bool, endpoint string) *UsageResponse
	URLPath       string
}

func MustProtocol(p prompts.Protocol) ProtocolFuncs {
	switch p {
	case prompts.ProtocolResponses:
		return ProtocolFuncs{
			BuildRequest:  buildResponsesRequestFromParams,
			ForwardStream: forwardResponsesStream,
			URLPath:       "/responses",
		}
	case prompts.ProtocolChat:
		return ProtocolFuncs{
			BuildRequest:  buildChatRequestFromParams,
			ForwardStream: forwardChatStream,
			URLPath:       "/chat/completions",
		}
	default:
		panic("unknown protocol: " + string(p))
	}
}

func buildResponsesRequestFromParams(p RequestParams) any {
	return buildResponsesRequest(
		p.Model, p.SystemPrompt,
		p.ReasoningEffort, p.ReasoningSummary,
		p.Verbosity, p.ServiceTier,
		p.Tools, p.ToolChoice, p.Temperature,
		p.Messages, p.ResponseFormat,
	)
}

func forwardResponsesStream(ctx context.Context, src io.Reader, dst io.Writer, flusher http.Flusher, inspect bool, endpoint string) *UsageResponse {
	return forwardStream(ctx, src, dst, flusher, inspect, endpoint)
}
