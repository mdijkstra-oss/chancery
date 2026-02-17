package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type ResolvedPath struct {
	Folder string
	Extra  string
}

var extraNames = map[string]bool{"plan": true, "exec": true, "merge": true, "memory": true}

func ResolveFolder(urlPath string) (ResolvedPath, error) {
	folder := "nabu"
	extra := ""

	if urlPath != "" {
		dir, last := filepath.Split(urlPath)
		if extraNames[last] {
			extra = last
			urlPath = filepath.Clean(dir)
		}
		if urlPath != "" && urlPath != "." {
			folder = filepath.Join("nabu", urlPath)
		}
	}

	path := filepath.Join(PromptsDir, folder)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ResolvedPath{}, fmt.Errorf("unknown path: %s", urlPath)
	}
	return ResolvedPath{Folder: folder, Extra: extra}, nil
}
