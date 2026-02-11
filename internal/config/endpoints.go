package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func ResolveFolder(urlPath string) (string, error) {
	folder := "nabu"
	if urlPath != "" {
		folder = filepath.Join("nabu", urlPath)
	}
	path := filepath.Join(PromptsDir, folder)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("unknown path: %s", urlPath)
	}
	return folder, nil
}
