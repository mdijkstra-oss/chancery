package prompts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type Pricing struct {
	Input       float64 `json:"input"`
	Output      float64 `json:"output"`
	CachedInput float64 `json:"cached_input"`
}

type PromptConfig struct {
	Model            string  `json:"model"`
	ReasoningEffort  string  `json:"reasoning_effort"`
	ReasoningSummary string  `json:"reasoning_summary"`
	Verbosity        string  `json:"verbosity"`
	Pricing          Pricing `json:"pricing"`
}

type ComposeOpts struct {
	Folder string
	Tools  []string
	Chat   bool
	Extra  string
}

func ComposePrompt(baseDir string, opts ComposeOpts) (string, error) {
	var layers []string

	for _, folder := range ancestorFolders(opts.Folder) {
		agent, err := loadLayer(filepath.Join(baseDir, folder))
		if err != nil {
			return "", err
		}
		layers = append(layers, agent...)
	}

	if opts.Chat {
		chat, err := loadLayer(filepath.Join(baseDir, "tools", "chat"))
		if err != nil {
			return "", err
		}
		layers = append(layers, chat...)
	}

	tools, err := loadToolFiles(filepath.Join(baseDir, "tools"), opts.Tools)
	if err != nil {
		return "", err
	}
	layers = append(layers, tools...)

	if opts.Extra != "" {
		extra, err := loadExtraLayer(baseDir, opts.Extra)
		if err != nil {
			return "", err
		}
		layers = append(layers, extra...)
	}

	return strings.Join(layers, "\n\n"), nil
}

func ResolveConfig(baseDir, folder string) (PromptConfig, error) {
	for _, ancestor := range reverseSlice(ancestorFolders(folder)) {
		path := filepath.Join(baseDir, ancestor, "config.json")
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return PromptConfig{}, err
		}
		var cfg PromptConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			return PromptConfig{}, err
		}
		return cfg, nil
	}
	return PromptConfig{}, os.ErrNotExist
}

func LoadFolder(baseDir, folder string) (string, error) {
	return ComposePrompt(baseDir, ComposeOpts{Folder: folder})
}

func LoadConfig(baseDir, folder string) (PromptConfig, error) {
	return ResolveConfig(baseDir, folder)
}

func MustLoad(path string) string {
	prompt, err := Load(path)
	if err != nil {
		panic("failed to load system prompt: " + err.Error())
	}
	return prompt
}

func Load(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
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

func loadLayer(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	names := filterMarkdownNames(entries)
	slices.Sort(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		parts = append(parts, string(data))
	}
	return parts, nil
}

func loadToolFiles(toolsDir string, available []string) ([]string, error) {
	return loadToolFilesRecursive(toolsDir, available, true)
}

func loadToolFilesRecursive(dir string, available []string, skipChat bool) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	names := filterToolEntryNames(entries, skipChat)
	slices.Sort(names)

	var parts []string
	for _, name := range names {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			sub, err := loadToolFilesRecursive(path, available, false)
			if err != nil {
				return nil, err
			}
			parts = append(parts, sub...)
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fm, body := ParseFrontmatter(string(data))
		if HasRequired(fm.Requires, available) {
			parts = append(parts, body)
		}
	}
	return parts, nil
}

var validExtras = map[string]bool{"plan": true, "exec": true, "merge": true}

func loadExtraLayer(baseDir, name string) ([]string, error) {
	if !validExtras[name] {
		return nil, fmt.Errorf("unknown extra prompt: %s", name)
	}
	return loadLayer(filepath.Join(baseDir, "extra", name))
}

func ancestorFolders(folder string) []string {
	parts := strings.Split(filepath.ToSlash(folder), "/")
	ancestors := make([]string, 0, len(parts))
	for i := range parts {
		ancestors = append(ancestors, filepath.Join(parts[:i+1]...))
	}
	return ancestors
}

func reverseSlice(s []string) []string {
	out := make([]string, len(s))
	for i, v := range s {
		out[len(s)-1-i] = v
	}
	return out
}

func filterMarkdownNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if isMarkdownFile(e) {
			names = append(names, e.Name())
		}
	}
	return names
}

func filterToolEntryNames(entries []os.DirEntry, skipChat bool) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if skipChat && e.IsDir() && e.Name() == "chat" {
			continue
		}
		if isMarkdownFile(e) || isPromptDir(e) {
			names = append(names, e.Name())
		}
	}
	return names
}

func isTemp(name string) bool {
	return strings.Contains(name, ".temp")
}

func isMarkdownFile(e os.DirEntry) bool {
	return !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && !isTemp(e.Name())
}

func isPromptDir(e os.DirEntry) bool {
	return e.IsDir() && !strings.HasPrefix(e.Name(), ".")
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
