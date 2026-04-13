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
	Agents     map[string]CompiledAgent
	Configs    map[string]PromptConfig
	Modes      map[string]string
	Approaches ApproachRegistry
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

func resolveInclude(include string, readFile func(string) (string, error), localDir, sharedDir string) (string, error) {
	if localDir != "" {
		if content, err := readFile(filepath.Join(localDir, include)); err == nil {
			return content, nil
		}
	}
	return readFile(filepath.Join(sharedDir, include))
}

func ResolveManifest(lines []Line, readFile func(string) (string, error), sharedDir, localDir string, skip func(string) bool) (CompiledAgent, error) {
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
			content, err := resolveInclude(line.Include, readFile, localDir, sharedDir)
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

func CompileMode(promptsDir, modeKey string) (CompiledAgent, error) {
	modesDir := filepath.Join(promptsDir, "modes")

	data, err := os.ReadFile(filepath.Join(modesDir, modeKey+".md"))
	if err != nil {
		return CompiledAgent{}, fmt.Errorf("read mode %s: %w", modeKey, err)
	}
	lines := ParseManifest(string(data))
	return ResolveManifest(lines, osReadFile, "", modesDir, nil)
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
	return ResolveManifest(lines, osReadFile, sharedDir, filepath.Dir(manifestPath), skip)
}

func CompileRegistry(promptsDir string) Registry {
	agentsDir := filepath.Join(promptsDir, "agents")
	sharedDir := filepath.Join(promptsDir, "shared")

	registry := Registry{
		Agents:     make(map[string]CompiledAgent),
		Configs:    loadAgentsFile(promptsDir),
		Modes:      compileModes(promptsDir),
		Approaches: compileApproaches(promptsDir),
	}

	filepath.Walk(agentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			panic(fmt.Sprintf("walk error at %s: %v", path, err))
		}
		if info.IsDir() {
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
		agent, err := ResolveManifest(lines, osReadFile, sharedDir, filepath.Dir(path), nil)
		if err != nil {
			panic(fmt.Sprintf("compile %s: %v", key, err))
		}
		registry.Agents[key] = agent
		return nil
	})

	resolveAgentConfigs(registry)
	return registry
}

func compileModes(promptsDir string) map[string]string {
	modesDir := filepath.Join(promptsDir, "modes")
	modes := make(map[string]string)

	entries, err := os.ReadDir(modesDir)
	if os.IsNotExist(err) {
		return modes
	}
	if err != nil {
		panic(fmt.Sprintf("read modes dir: %v", err))
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		path := filepath.Join(modesDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			panic(fmt.Sprintf("read mode %s: %v", entry.Name(), err))
		}
		lines := ParseManifest(string(data))
		compiled, err := ResolveManifest(lines, osReadFile, "", modesDir, nil)
		if err != nil {
			panic(fmt.Sprintf("compile mode %s: %v", entry.Name(), err))
		}
		key := strings.TrimSuffix(entry.Name(), ".md")
		modes[key] = compiled.Prompt
	}

	return modes
}

func osReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func loadAgentsFile(promptsDir string) map[string]PromptConfig {
	path := filepath.Join(promptsDir, "agents.json")
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read agents.json: %v", err))
	}
	var file agentsFile
	if err := json.Unmarshal(data, &file); err != nil {
		panic(fmt.Sprintf("parse agents.json: %v", err))
	}
	resolved := resolveModels(file.Models)
	configs := make(map[string]PromptConfig, len(file.Agents))
	for key, agent := range file.Agents {
		model, ok := resolved[agent.Model]
		if !ok {
			panic(fmt.Sprintf("agent %q references unknown model %q", key, agent.Model))
		}
		configs[key] = mergeConfig(model, agent)
	}
	return configs
}

const maxModelDepth = 5

func resolveModels(models map[string]modelEntry) map[string]modelEntry {
	resolved := make(map[string]modelEntry, len(models))
	for key := range models {
		resolved[key] = resolveModel(key, models)
	}
	return resolved
}

func resolveModel(key string, models map[string]modelEntry) modelEntry {
	current := key
	var chain []modelEntry
	for range maxModelDepth + 1 {
		entry, ok := models[current]
		if !ok {
			panic(fmt.Sprintf("model %q references unknown model %q", key, current))
		}
		chain = append(chain, entry)
		if entry.Extends == "" {
			break
		}
		current = entry.Extends
	}
	if chain[len(chain)-1].Extends != "" {
		panic(fmt.Sprintf("model %q extends chain exceeds %d steps", key, maxModelDepth))
	}
	result := chain[len(chain)-1]
	for i := len(chain) - 2; i >= 0; i-- {
		result = overlayModel(result, chain[i])
	}
	return result
}

func overlayModel(base, child modelEntry) modelEntry {
	result := base
	result.Extends = ""
	if child.Name != "" {
		result.Name = child.Name
	}
	if child.Type != "" {
		result.Type = child.Type
	}
	if child.Dimensions != 0 {
		result.Dimensions = child.Dimensions
	}
	if child.ReasoningEffort != "" {
		result.ReasoningEffort = child.ReasoningEffort
	}
	if child.ReasoningSummary != "" {
		result.ReasoningSummary = child.ReasoningSummary
	}
	if child.Verbosity != "" {
		result.Verbosity = child.Verbosity
	}
	if child.ServiceTier != "" {
		result.ServiceTier = child.ServiceTier
	}
	if child.CompactAt != 0 {
		result.CompactAt = child.CompactAt
	}
	if hasPricing(child.Pricing) {
		result.Pricing = child.Pricing
	}
	return result
}

func hasPricing(p Pricing) bool {
	return p.Input != 0 || p.Output != 0 || p.CachedInput != 0
}

func mergeConfig(model modelEntry, agent agentEntry) PromptConfig {
	cfg := PromptConfig{
		Model:            model.Name,
		Dimensions:       model.Dimensions,
		ReasoningEffort:  model.ReasoningEffort,
		ReasoningSummary: model.ReasoningSummary,
		Verbosity:        model.Verbosity,
		ServiceTier:      model.ServiceTier,
		CompactAt:        model.CompactAt,
		Pricing:          model.Pricing,
	}
	if agent.ReasoningEffort != "" {
		cfg.ReasoningEffort = agent.ReasoningEffort
	}
	if agent.ReasoningSummary != "" {
		cfg.ReasoningSummary = agent.ReasoningSummary
	}
	if agent.Verbosity != "" {
		cfg.Verbosity = agent.Verbosity
	}
	if agent.ServiceTier != "" {
		cfg.ServiceTier = agent.ServiceTier
	}
	if agent.Temperature != nil {
		cfg.Temperature = agent.Temperature
	}
	if agent.CompactAt != 0 {
		cfg.CompactAt = agent.CompactAt
	}
	if agent.Dimensions != 0 {
		cfg.Dimensions = agent.Dimensions
	}
	return cfg
}

func resolveAgentConfigs(registry Registry) {
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
