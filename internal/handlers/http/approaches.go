package http

import (
	"encoding/json"
	"net/http"

	"hermes-logos/internal/prompts"
)

func buildApproachDict(keys []string, descriptions map[string]string) map[string]string {
	dict := make(map[string]string, len(keys))
	for _, k := range keys {
		dict[k] = descriptions[k]
	}
	return dict
}

func NewApproachesHandler(registry prompts.ApproachRegistry) http.HandlerFunc {
	data, err := json.Marshal(buildApproachDict(registry.Keys, registry.Descriptions))
	if err != nil {
		panic("marshal approaches: " + err.Error())
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
