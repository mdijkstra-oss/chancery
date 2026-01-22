package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func isTemp(name string) bool {
	return strings.Contains(name, ".temp")
}

func isToolFile(name string) bool {
	return strings.HasPrefix(name, "tools.") && strings.HasSuffix(name, ".json") && !isTemp(name)
}

func loadDir(dir string) ([]json.RawMessage, error) {
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

	var result []json.RawMessage
	for _, f := range files {
		tools, err := loadFile(f)
		if err != nil {
			return nil, err
		}
		result = append(result, tools...)
	}
	return result, nil
}

func loadFile(path string) ([]json.RawMessage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var tools []json.RawMessage
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, err
	}

	return tools, nil
}

func LoadFolder(baseDir, folder string) ([]json.RawMessage, error) {
	return loadDir(filepath.Join(baseDir, folder))
}
