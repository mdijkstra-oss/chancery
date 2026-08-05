// Package responses composes an openai-responses body from a resolved agent and
// relays the backend's answer. Every field it writes is one the format already has.
package responses

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
)

// Agent is what chancery writes onto the body: the fields the resolved route names.
type Agent struct {
	Model            string
	Instructions     string
	ReasoningEffort  string
	ReasoningSummary string
	Verbosity        string
	ServiceTier      string
	ToolChoice       string
	MaxOutputTokens  int
}

// Compose keeps the received body as raw per-key JSON, so a key the agent does not
// set survives whatever it holds and whether or not chancery has heard of it.
func Compose(body []byte, agent Agent) ([]byte, error) {
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, requestBodyError(err)
	}

	if agent.Model != "" {
		fields["model"] = jsonValue(agent.Model)
	}
	if agent.Instructions != "" {
		instructions, err := prependInstructions(fields["instructions"], agent.Instructions)
		if err != nil {
			return nil, err
		}
		fields["instructions"] = instructions
	}
	if agent.ServiceTier != "" {
		fields["service_tier"] = jsonValue(agent.ServiceTier)
	}
	if agent.ToolChoice != "" {
		fields["tool_choice"] = jsonValue(agent.ToolChoice)
	}
	if agent.MaxOutputTokens > 0 {
		fields["max_output_tokens"] = jsonValue(agent.MaxOutputTokens)
	}

	reasoning := map[string]json.RawMessage{}
	if agent.ReasoningEffort != "" {
		reasoning["effort"] = jsonValue(agent.ReasoningEffort)
	}
	if agent.ReasoningSummary != "" {
		reasoning["summary"] = jsonValue(agent.ReasoningSummary)
	}
	if err := mergeObject(fields, "reasoning", reasoning); err != nil {
		return nil, err
	}

	text := map[string]json.RawMessage{}
	if agent.Verbosity != "" {
		text["verbosity"] = jsonValue(agent.Verbosity)
	}
	if err := mergeObject(fields, "text", text); err != nil {
		return nil, err
	}

	composed, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encode request body: %w", err)
	}
	return composed, nil
}

// The agent's prompt goes in front of the caller's rather than over it: overwriting
// silently discards something the caller wrote, and refusing makes an ordinary
// Responses body an error.
func prependInstructions(existing json.RawMessage, prompt string) (json.RawMessage, error) {
	if len(existing) == 0 || isNull(existing) {
		return jsonValue(prompt), nil
	}
	var callers string
	if err := json.Unmarshal(existing, &callers); err != nil {
		return nil, errors.New("instructions must be a string")
	}
	if callers == "" {
		return jsonValue(prompt), nil
	}
	return jsonValue(prompt + "\n\n" + callers), nil
}

// Only the keys the agent sets are written, so a format the caller put under text or
// a reasoning key chancery does not know travels beside them.
func mergeObject(
	fields map[string]json.RawMessage,
	key string,
	values map[string]json.RawMessage,
) error {
	if len(values) == 0 {
		return nil
	}
	merged := map[string]json.RawMessage{}
	if existing, ok := fields[key]; ok && !isNull(existing) {
		if err := json.Unmarshal(existing, &merged); err != nil {
			return fmt.Errorf("%s must be an object", key)
		}
	}
	maps.Copy(merged, values)
	encoded, err := json.Marshal(merged)
	if err != nil {
		return fmt.Errorf("encode %s: %w", key, err)
	}
	fields[key] = encoded
	return nil
}

// Every message here reaches the caller as a 400, so each names the field and the JSON
// shape the format wants. A decoder error names the Go type it failed to build, which
// is chancery's business; a syntax error names a position in what the caller sent, which
// is theirs.
func requestBodyError(err error) error {
	var syntax *json.SyntaxError
	if errors.As(err, &syntax) {
		return fmt.Errorf("request body is not valid JSON: %w", syntax)
	}
	return errors.New("request body must be a JSON object")
}

func isNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

// A scalar always encodes, so a failure here is a broken runtime rather than a bad
// request.
func jsonValue[T string | int | float64](value T) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("encode %v: %v", value, err))
	}
	return encoded
}
