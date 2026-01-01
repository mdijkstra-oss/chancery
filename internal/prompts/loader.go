package prompts

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func Load(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return loadDir(path)
	}
	return loadFile(path)
}

func LoadFolders(baseDir string, folders []string) (string, error) {
	parts := make([]string, 0, len(folders))
	for _, folder := range folders {
		content, err := loadDir(filepath.Join(baseDir, folder))
		if err != nil {
			return "", err
		}
		parts = append(parts, content)
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

func MustLoad(path string) string {
	prompt, err := Load(path)
	if err != nil {
		panic("failed to load system prompt: " + err.Error())
	}
	return prompt
}

func loadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func loadDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	names := filterFiles(entries)
	if len(names) == 0 {
		return "", errors.New("no system prompt found for path")
	}

	slices.Sort(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		content, err := loadFile(filepath.Join(dir, name))
		if err != nil {
			return "", err
		}
		parts = append(parts, content)
	}

	return strings.Join(parts, "\n\n"), nil
}

func isMarkdownFile(e os.DirEntry) bool {
	return !e.IsDir() && strings.HasSuffix(e.Name(), ".md")
}

func filterFiles(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if isMarkdownFile(e) {
			names = append(names, e.Name())
		}
	}
	return names
}

func ListDirectories(baseDir string) ([]string, error) {
	absPath, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	return filterDirectories(entries), nil
}

func filterDirectories(entries []os.DirEntry) []string {
	dirs := make([]string, 0)
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	return dirs
}
