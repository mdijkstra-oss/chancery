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

type ComposeResult struct {
	Prompt  string
	Sources []string
}

type fragment struct {
	path    string
	content string
}

func ComposePrompt(baseDir string, opts ComposeOpts) (ComposeResult, error) {
	var fragments []fragment

	for _, folder := range ancestorFolders(opts.Folder) {
		loaded, err := loadLayer(filepath.Join(baseDir, folder), opts.Tools)
		if err != nil {
			return ComposeResult{}, err
		}
		fragments = append(fragments, loaded...)
	}

	if opts.Chat {
		loaded, err := loadLayer(filepath.Join(baseDir, "tools", "chat"), opts.Tools)
		if err != nil {
			return ComposeResult{}, err
		}
		fragments = append(fragments, loaded...)
	}

	loaded, err := loadToolFiles(filepath.Join(baseDir, "tools"), opts.Tools)
	if err != nil {
		return ComposeResult{}, err
	}
	fragments = append(fragments, loaded...)

	if opts.Extra != "" {
		loaded, err := loadExtraLayer(baseDir, opts.Extra, opts.Tools)
		if err != nil {
			return ComposeResult{}, err
		}
		fragments = append(fragments, loaded...)
	}

	return buildResult(baseDir, fragments), nil
}

func buildResult(baseDir string, fragments []fragment) ComposeResult {
	layers := make([]string, len(fragments))
	sources := make([]string, len(fragments))
	for i, f := range fragments {
		layers[i] = f.content
		sources[i] = relativePath(baseDir, f.path)
	}
	return ComposeResult{
		Prompt:  strings.Join(layers, "\n\n"),
		Sources: sources,
	}
}

func relativePath(baseDir, path string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
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
	result, err := ComposePrompt(baseDir, ComposeOpts{Folder: folder})
	if err != nil {
		return "", err
	}
	return result.Prompt, nil
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

func loadLayer(dir string, tools []string) ([]fragment, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	names := filterMarkdownNames(entries)
	slices.Sort(names)

	toolSet := toSet(tools)
	fragments := make([]fragment, 0, len(names))
	for _, name := range names {
		if req := parseToolRequirement(name); req != "" && !toolSet[req] {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fragments = append(fragments, fragment{path: path, content: string(data)})
	}
	return fragments, nil
}

func loadToolFiles(toolsDir string, available []string) ([]fragment, error) {
	return loadToolFilesRecursive(toolsDir, available, true)
}

func loadToolFilesRecursive(dir string, available []string, skipChat bool) ([]fragment, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	names := filterToolEntryNames(entries, skipChat)
	slices.Sort(names)

	toolSet := toSet(available)
	var fragments []fragment
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
			fragments = append(fragments, sub...)
			continue
		}
		if req := parseToolRequirement(name); req != "" && !toolSet[req] {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		fragments = append(fragments, fragment{path: path, content: string(data)})
	}
	return fragments, nil
}

var validExtras = map[string]bool{"plan": true, "exec": true, "merge": true, "memory": true}

func loadExtraLayer(baseDir, name string, tools []string) ([]fragment, error) {
	if !validExtras[name] {
		return nil, fmt.Errorf("unknown extra prompt: %s", name)
	}
	return loadLayer(filepath.Join(baseDir, "extra", name), tools)
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

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

func parseToolRequirement(name string) string {
	base := strings.TrimSuffix(name, ".md")
	dot := strings.LastIndex(base, ".")
	if dot == -1 {
		return ""
	}
	return base[dot+1:]
}
