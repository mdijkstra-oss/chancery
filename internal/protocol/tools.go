package protocol

import "encoding/json"

func ExtractToolNames(raw []json.RawMessage) []string {
	names := make([]string, 0, len(raw))
	for _, r := range raw {
		var tool struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(r, &tool) == nil && tool.Name != "" {
			names = append(names, tool.Name)
		}
	}
	return names
}
