package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func isToolFile(name string) bool {
	return strings.HasPrefix(name, "tools.") && strings.HasSuffix(name, ".json")
}

func loadDir(dir string) ([]openai.Tool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && isToolFile(e.Name()) {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)

	var result []openai.Tool
	for _, f := range files {
		tools, err := loadFile(f)
		if err != nil {
			return nil, err
		}
		result = append(result, tools...)
	}
	return result, nil
}

func loadFile(path string) ([]openai.Tool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tools []openai.Tool
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, err
	}

	return tools, nil
}

func LoadFolder(baseDir, folder string) ([]openai.Tool, error) {
	return loadDir(filepath.Join(baseDir, folder))
}
