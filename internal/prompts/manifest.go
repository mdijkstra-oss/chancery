package prompts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const PromptsDir = "prompts"

type Line struct {
	Include string
	Literal string
}

type CompiledAgent struct {
	Prompt   string
	Sources  []string
	Segments []Segment
}

type Registry struct {
	Agents  map[string]CompiledAgent
	Configs map[string]PromptConfig
}

var includePattern = regexp.MustCompile(`^\[([^\]]+\.md)\]$`)

func ParseManifest(content string) []Line {
	raw := strings.Split(content, "\n")
	var lines []Line
	for _, r := range raw {
		trimmed := strings.TrimSpace(r)
		if m := includePattern.FindStringSubmatch(trimmed); m != nil {
			lines = append(lines, Line{Include: m[1]})
			continue
		}
		lines = append(lines, Line{Literal: r})
	}
	return lines
}

func ResolveManifest(lines []Line, readFile func(string) (string, error), sharedDir string, skip func(string) bool) (CompiledAgent, error) {
	var parts []string
	var sources []string
	var segments []Segment
	for _, line := range lines {
		if line.Include != "" {
			if skip != nil && skip(line.Include) {
				parts = append(parts, "["+line.Include+"]")
				segments = append(segments, Segment{Content: "[" + line.Include + "]"})
				continue
			}
			path := filepath.Join(sharedDir, line.Include)
			content, err := readFile(path)
			if err != nil {
				return CompiledAgent{}, fmt.Errorf("include %q: %w", line.Include, err)
			}
			trimmed := strings.TrimRight(content, "\n")
			source := filepath.ToSlash(line.Include)
			parts = append(parts, trimmed)
			sources = append(sources, source)
			segments = append(segments, Segment{Source: source, Content: trimmed})
			continue
		}
		parts = append(parts, line.Literal)
		segments = append(segments, Segment{Content: line.Literal})
	}
	prompt := strings.TrimRight(strings.Join(parts, "\n"), "\n")
	return CompiledAgent{Prompt: prompt, Sources: sources, Segments: segments}, nil
}

func ManifestKeyFromPath(path, agentsDir string) string {
	rel, err := filepath.Rel(agentsDir, path)
	if err != nil {
		return path
	}
	rel = filepath.ToSlash(rel)
	dir, file := filepath.Split(rel)
	dir = strings.TrimSuffix(dir, "/")
	name := strings.TrimSuffix(file, ".md")
	if name == "index" {
		return dir
	}
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

func CompileAgent(promptsDir, agentKey string, skip func(string) bool) (CompiledAgent, error) {
	agentsDir := filepath.Join(promptsDir, "agents")
	sharedDir := filepath.Join(promptsDir, "shared")

	manifestPath := filepath.Join(agentsDir, agentKey+".md")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		manifestPath = filepath.Join(agentsDir, agentKey, "index.md")
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return CompiledAgent{}, fmt.Errorf("read manifest %s: %w", agentKey, err)
	}
	lines := ParseManifest(string(data))
	return ResolveManifest(lines, osReadFile, sharedDir, skip)
}

func CompileRegistry(promptsDir string) Registry {
	agentsDir := filepath.Join(promptsDir, "agents")
	sharedDir := filepath.Join(promptsDir, "shared")

	registry := Registry{
		Agents:  make(map[string]CompiledAgent),
		Configs: make(map[string]PromptConfig),
	}

	filepath.Walk(agentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(fmt.Sprintf("walk error at %s: %v", path, err))
		}
		if info.IsDir() {
			loadConfigInto(registry.Configs, path, agentsDir)
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}
		key := ManifestKeyFromPath(path, agentsDir)
		data, err := os.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("read manifest %s: %v", path, err))
		}
		lines := ParseManifest(string(data))
		agent, err := ResolveManifest(lines, osReadFile, sharedDir, nil)
		if err != nil {
			panic(fmt.Sprintf("compile %s: %v", key, err))
		}
		registry.Agents[key] = agent
		return nil
	})

	resolveAgentConfigs(registry, agentsDir)
	return registry
}

func osReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func loadConfigInto(configs map[string]PromptConfig, dir, agentsDir string) {
	path := filepath.Join(dir, "config.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		panic(fmt.Sprintf("read config %s: %v", path, err))
	}
	var cfg PromptConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		panic(fmt.Sprintf("parse config %s: %v", path, err))
	}
	rel, err := filepath.Rel(agentsDir, dir)
	if err != nil {
		panic(fmt.Sprintf("rel path %s: %v", dir, err))
	}
	key := filepath.ToSlash(rel)
	if key == "." {
		key = ""
	}
	configs[key] = cfg
}

func resolveAgentConfigs(registry Registry, agentsDir string) {
	for agentKey := range registry.Agents {
		if _, found := registry.Configs[agentKey]; found {
			continue
		}
		cfg, found := walkUpConfig(agentKey, registry.Configs)
		if !found {
			panic(fmt.Sprintf("no config found for agent %q", agentKey))
		}
		registry.Configs[agentKey] = cfg
	}
}

func walkUpConfig(key string, configs map[string]PromptConfig) (PromptConfig, bool) {
	for {
		if cfg, ok := configs[key]; ok {
			return cfg, true
		}
		slash := strings.LastIndex(key, "/")
		if slash == -1 {
			if cfg, ok := configs[""]; ok {
				return cfg, true
			}
			return PromptConfig{}, false
		}
		key = key[:slash]
	}
}
