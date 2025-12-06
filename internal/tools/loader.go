package tools

import (
	"encoding/json"
	"os"

	"github.com/sashabaranov/go-openai"
)

type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

func Load(path string) ([]openai.Tool, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var mcpTools []MCPTool
	if err := json.Unmarshal(data, &mcpTools); err != nil {
		return nil, err
	}

	return toOpenAITools(mcpTools), nil
}

func MustLoad(path string) []openai.Tool {
	tools, err := Load(path)
	if err != nil {
		panic("failed to load tools: " + err.Error())
	}
	return tools
}

func toOpenAITools(mcpTools []MCPTool) []openai.Tool {
	result := make([]openai.Tool, len(mcpTools))
	for i, t := range mcpTools {
		result[i] = toOpenAITool(t)
	}
	return result
}

func toOpenAITool(t MCPTool) openai.Tool {
	return openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
		},
	}
}
