package responses

import "encoding/json"

// A caller placing an explicit cache breakpoint is usually too far upstream to know
// which model answers, and support for them is gated on the model rather than on the
// format: within one provider a current model takes them and last year's rejects them.
// An agent whose model refuses them says so, and the markers are dropped here, so one
// body serves every route the caller might address.
//
// prompt_cache_options goes with them: it carries their TTL and means nothing without a
// breakpoint to apply it to. prompt_cache_key stays, because implicit caching is a
// separate mechanism that the models refusing breakpoints still have.
func stripCacheBreakpoints(fields map[string]json.RawMessage) {
	if input, stripped := strippedInput(fields["input"]); stripped {
		fields["input"] = input
	}
	delete(fields, "prompt_cache_options")
}

func strippedInput(raw json.RawMessage) (json.RawMessage, bool) {
	var items []json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &items) != nil {
		return nil, false
	}
	stripped := false
	for index, item := range items {
		rebuilt, itemStripped := strippedItem(item)
		if itemStripped {
			items[index] = rebuilt
			stripped = true
		}
	}
	if !stripped {
		return nil, false
	}
	rebuilt, err := json.Marshal(items)
	if err != nil {
		return nil, false
	}
	return rebuilt, true
}

func strippedItem(raw json.RawMessage) (json.RawMessage, bool) {
	var item map[string]json.RawMessage
	if json.Unmarshal(raw, &item) != nil {
		return raw, false
	}
	content, stripped := strippedContent(item["content"])
	if !stripped {
		return raw, false
	}
	item["content"] = content
	rebuilt, err := json.Marshal(item)
	if err != nil {
		return raw, false
	}
	return rebuilt, true
}

// Content arrives as a bare string as often as a parts array, and a string carries no
// breakpoint, so failing to decode parts is the ordinary case and not an error.
func strippedContent(raw json.RawMessage) (json.RawMessage, bool) {
	var parts []map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &parts) != nil {
		return raw, false
	}
	stripped := false
	for _, part := range parts {
		if _, present := part["prompt_cache_breakpoint"]; present {
			delete(part, "prompt_cache_breakpoint")
			stripped = true
		}
	}
	if !stripped {
		return raw, false
	}
	rebuilt, err := json.Marshal(parts)
	if err != nil {
		return raw, false
	}
	return rebuilt, true
}
