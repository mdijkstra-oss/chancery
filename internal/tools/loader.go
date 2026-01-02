package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/sashabaranov/go-openai"
)

func loadDir(dir string) ([]openai.Tool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
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

func LoadFolders(baseDir string, folders []string) ([]openai.Tool, error) {
	var result []openai.Tool
	for _, folder := range folders {
		tools, err := loadDir(filepath.Join(baseDir, folder))
		if err != nil {
			continue
		}
		result = append(result, tools...)
	}
	return result, nil
}
