package responses

import (
	"encoding/json"

	"github.com/mdijkstra-oss/chancery/internal/fn"
)

// tool is the one field a tool prompt is found by. The rest of the tool — its
// parameters, its type, whatever else the caller attached — is never decoded, so it
// travels to the backend exactly as it arrived.
type tool struct {
	Name string `json:"name"`
}

// ToolNames reads the tools a request names. tools is a sibling of input rather than
// part of it, so a name here is not message content. A built-in tool the format
// identifies by type alone carries no name and asks for no prompt.
func ToolNames(body []byte) []string {
	var fields struct {
		Tools []tool `json:"tools"`
	}
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil
	}
	named := fn.Filter(fields.Tools, func(t tool) bool { return t.Name != "" })
	return fn.Map(named, func(t tool) string { return t.Name })
}
