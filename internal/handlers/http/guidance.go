package http

import (
	"encoding/json"
	"net/http"

	"hermes-logos/internal/prompts"
)

func buildGuidanceDict(keys []string, descriptions map[string]string) map[string]string {
	dict := make(map[string]string, len(keys))
	for _, k := range keys {
		dict[k] = descriptions[k]
	}
	return dict
}

func NewGuidanceHandler(registry prompts.GuidanceRegistry) http.HandlerFunc {
	data, err := json.Marshal(buildGuidanceDict(registry.Keys, registry.Descriptions))
	if err != nil {
		panic("marshal guidance: " + err.Error())
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
