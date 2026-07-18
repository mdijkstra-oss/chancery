package prompts

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func LoadToolSegments(configRoot string, tools []string) ([]Segment, error) {
	toolsDir := filepath.Join(configRoot, "tools")
	return loadToolDir(configRoot, toolsDir, toolsDir, tools)
}

func LoadToolPrompts(configRoot string, tools []string) (string, []string, error) {
	segments, err := LoadToolSegments(configRoot, tools)
	if err != nil {
		return "", nil, err
	}
	texts := make([]string, len(segments))
	sources := make([]string, len(segments))
	for i, s := range segments {
		texts[i] = s.Content
		sources[i] = s.Source
	}
	return strings.Join(texts, "\n\n"), sources, nil
}

func loadToolDir(configRoot, baseDir, dir string, available []string) ([]Segment, error) {
	resolvedDir, err := resolveConfigPath(configRoot, dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(resolvedDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	names := filterToolEntryNames(entries)
	slices.Sort(names)

	toolSet := toSet(available)
	var segments []Segment
	for _, name := range names {
		path := filepath.Join(dir, name)
		resolvedPath, err := resolveConfigPath(configRoot, path)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(resolvedPath)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			sub, err := loadToolDir(configRoot, baseDir, path, available)
			if err != nil {
				return nil, err
			}
			segments = append(segments, sub...)
			continue
		}
		if req := parseToolRequirement(name); req != "" && !toolSet[req] {
			continue
		}
		data, err := readConfigFile(configRoot, path)
		if err != nil {
			return nil, err
		}
		rel, _ := filepath.Rel(baseDir, path)
		segments = append(segments, Segment{
			Source:  filepath.ToSlash(rel),
			Content: string(data),
		})
	}
	return segments, nil
}

func filterToolEntryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if isMarkdownFile(e) || isPromptDir(e) {
			names = append(names, e.Name())
		}
	}
	return names
}

func parseToolRequirement(name string) string {
	base := strings.TrimSuffix(name, ".md")
	dot := strings.LastIndex(base, ".")
	if dot == -1 {
		return ""
	}
	return base[dot+1:]
}

func toSet(items []string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, item := range items {
		m[item] = true
	}
	return m
}

func isMarkdownFile(e os.DirEntry) bool {
	return !e.IsDir() && strings.HasSuffix(e.Name(), ".md") && !isTemp(e.Name())
}

func isPromptDir(e os.DirEntry) bool {
	return e.IsDir() && !strings.HasPrefix(e.Name(), ".")
}

func isTemp(name string) bool {
	return strings.Contains(name, ".temp")
}
