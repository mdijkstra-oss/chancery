package prompts

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
	Agents       map[string]CompiledAgent
	Configs      map[string]PromptConfig
	Variants     map[string][]PromptConfig
	Modes        map[string]string
	Guidance     GuidanceRegistry
	ProviderKeys []string
	models       map[string]modelEntry
	providers    map[string]ProviderConfig
}

func (r Registry) ConfigForAgent(name string, modelIndex int) (PromptConfig, error) {
	if modelIndex == 0 {
		cfg, ok := r.Configs[name]
		if !ok {
			return PromptConfig{}, fmt.Errorf("no config for agent: %s", name)
		}
		return cfg, nil
	}
	variants, ok := r.Variants[name]
	if !ok {
		return PromptConfig{}, fmt.Errorf("agent %q has no model variants", name)
	}
	if modelIndex >= len(variants) {
		return PromptConfig{}, fmt.Errorf("agent %q model index %d out of range (have %d)", name, modelIndex, len(variants))
	}
	return variants[modelIndex], nil
}

type ModelOverride struct {
	Name     string
	Provider ProviderConfig
	Pricing  Pricing
}

func (r Registry) LookupModel(key string) (ModelOverride, bool) {
	model, ok := r.models[key]
	if !ok {
		return ModelOverride{}, false
	}
	provider, ok := r.providers[model.Provider]
	if !ok {
		return ModelOverride{}, false
	}
	name := model.Name
	if name == "" {
		name = key
	}
	return ModelOverride{Name: name, Provider: provider, Pricing: model.Pricing}, true
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

	cr := loadConfigDir(promptsDir)
	registry := Registry{
		Agents:       make(map[string]CompiledAgent),
		Configs:      cr.configs,
		Variants:     cr.variants,
		Modes:        compileModes(promptsDir),
		Guidance:     compileGuidance(promptsDir),
		ProviderKeys: cr.providerKeys,
		models:       cr.models,
		providers:    cr.providers,
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

func loadProviderFile(path, providerKey string) (ProviderEntry, map[string]modelEntry) {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read %s: %v", filepath.Base(path), err))
	}
	var pf providerFile
	if err := json.Unmarshal(data, &pf); err != nil {
		panic(fmt.Sprintf("parse %s: %v", filepath.Base(path), err))
	}
	entry := ProviderEntry{Protocol: pf.Protocol, BaseURL: pf.BaseURL, APIKeyEnv: pf.APIKeyEnv, Strict: pf.Strict}
	for key, m := range pf.Models {
		if m.Provider == "" {
			m.Provider = providerKey
			pf.Models[key] = m
		}
	}
	return entry, pf.Models
}

func loadAgentsConfig(path string) map[string]agentConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read agents.json: %v", err))
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		panic(fmt.Sprintf("parse agents.json: %v", err))
	}
	agents := make(map[string]agentConfig, len(raw))
	for key, v := range raw {
		var arr []agentEntry
		if json.Unmarshal(v, &arr) == nil {
			if len(arr) == 0 {
				panic(fmt.Sprintf("agent %q has empty model array", key))
			}
			agents[key] = agentConfig{Variants: arr}
			continue
		}
		var single agentEntry
		if err := json.Unmarshal(v, &single); err != nil {
			panic(fmt.Sprintf("parse agent %q: %v", key, err))
		}
		agents[key] = agentConfig{Variants: []agentEntry{single}}
	}
	return agents
}

type configResult struct {
	configs      map[string]PromptConfig
	variants     map[string][]PromptConfig
	providerKeys []string
	models       map[string]modelEntry
	providers    map[string]ProviderConfig
}

func loadConfigDir(promptsDir string) configResult {
	configDir := filepath.Join(promptsDir, "config")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		panic(fmt.Sprintf("read config dir: %v", err))
	}

	providerEntries := make(map[string]ProviderEntry)
	allModels := make(map[string]modelEntry)
	var agents map[string]agentConfig

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(configDir, entry.Name())
		key := strings.TrimSuffix(entry.Name(), ".json")

		if key == "agents" {
			agents = loadAgentsConfig(path)
			continue
		}

		pe, models := loadProviderFile(path, key)
		providerEntries[key] = pe
		for mk, mv := range models {
			allModels[mk] = mv
		}
	}

	if len(providerEntries) == 0 {
		panic("config: no provider files found")
	}
	if agents == nil {
		panic("config: agents.json not found")
	}

	providers := resolveProviders(providerEntries)
	providerKeys := make([]string, 0, len(providers))
	for key := range providers {
		providerKeys = append(providerKeys, key)
	}

	resolved := resolveModels(allModels)
	configs := make(map[string]PromptConfig, len(agents))
	allVariants := make(map[string][]PromptConfig)
	for key, ac := range agents {
		var built []PromptConfig
		for _, agent := range ac.Variants {
			model, ok := resolved[agent.Model]
			if !ok {
				panic(fmt.Sprintf("agent %q references unknown model %q", key, agent.Model))
			}
			if model.Provider == "" {
				panic(fmt.Sprintf("model %q has no provider", agent.Model))
			}
			provider, ok := providers[model.Provider]
			if !ok {
				panic(fmt.Sprintf("model %q references unknown provider %q", agent.Model, model.Provider))
			}
			built = append(built, mergeConfig(model, agent, provider))
		}
		configs[key] = built[0]
		if len(built) > 1 {
			allVariants[key] = built
		}
	}
	resolvePromptPaths(configs, filepath.Join(promptsDir, "shared"))
	validateSeedProtocols(configs)
	return configResult{configs: configs, variants: allVariants, providerKeys: providerKeys, models: resolved, providers: providers}
}

func resolvePromptPaths(configs map[string]PromptConfig, sharedDir string) {
	cache := make(map[string]string)
	for key, cfg := range configs {
		if cfg.Prompt == "" {
			continue
		}
		content, ok := cache[cfg.Prompt]
		if !ok {
			raw, err := osReadFile(filepath.Join(sharedDir, cfg.Prompt))
			if err != nil {
				panic(fmt.Sprintf("agent %q prompt %q: %v", key, cfg.Prompt, err))
			}
			content = strings.TrimRight(raw, "\n")
			cache[cfg.Prompt] = content
		}
		cfg.Prompt = content
		configs[key] = cfg
	}
}

func validateSeedProtocols(configs map[string]PromptConfig) {
	for key, cfg := range configs {
		if cfg.Seed && cfg.Provider.Protocol != ProtocolGemini {
			panic(fmt.Sprintf("agent %q has seed enabled but provider %q (protocol %q) does not support it", key, cfg.Provider.Key, cfg.Provider.Protocol))
		}
	}
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
	if child.Provider != "" {
		result.Provider = child.Provider
	}
	if child.Name != "" {
		result.Name = child.Name
	}
	if child.Type != "" {
		result.Type = child.Type
	}
	if child.Dimensions != 0 {
		result.Dimensions = child.Dimensions
	}
	if child.MaxTokens != 0 {
		result.MaxTokens = child.MaxTokens
	}
	if child.Prompt != "" {
		result.Prompt = child.Prompt
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
	if child.LegacyThinking {
		result.LegacyThinking = child.LegacyThinking
	}
	if child.AutoCache {
		result.AutoCache = child.AutoCache
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
	return p.Input != 0 || p.Output != 0 || p.CachedInput != 0 || p.CacheWriteInput != 0
}

func mergeConfig(model modelEntry, agent agentEntry, provider ProviderConfig) PromptConfig {
	cfg := PromptConfig{
		Model:            model.Name,
		Prompt:           model.Prompt,
		Dimensions:       model.Dimensions,
		MaxTokens:        model.MaxTokens,
		ReasoningEffort:  model.ReasoningEffort,
		ReasoningSummary: model.ReasoningSummary,
		Verbosity:        model.Verbosity,
		ServiceTier:      model.ServiceTier,
		LegacyThinking:  model.LegacyThinking,
		AutoCache:        model.AutoCache,
		CompactAt:        model.CompactAt,
		Pricing:          model.Pricing,
		Provider:         provider,
	}
	if agent.Prompt != nil {
		cfg.Prompt = *agent.Prompt
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
	if agent.Seed {
		cfg.Seed = true
	}
	if agent.AutoCache != nil {
		cfg.AutoCache = *agent.AutoCache
	}
	if agent.CompactAt != 0 {
		cfg.CompactAt = agent.CompactAt
	}
	if agent.Dimensions != 0 {
		cfg.Dimensions = agent.Dimensions
	}
	return cfg
}

func validateProtocol(p Protocol) {
	switch p {
	case ProtocolResponses, ProtocolGemini, ProtocolCompletions, ProtocolAnthropic:
	default:
		panic(fmt.Sprintf("unknown protocol: %q", p))
	}
}

func resolveProviders(entries map[string]ProviderEntry) map[string]ProviderConfig {
	providers := make(map[string]ProviderConfig, len(entries))
	for key, entry := range entries {
		validateProtocol(entry.Protocol)
		if entry.APIKeyEnv == "" {
			panic(fmt.Sprintf("provider %q has no api_key_env", key))
		}
		apiKey := os.Getenv(entry.APIKeyEnv)
		if apiKey == "" {
			slog.Warn("provider skipped: API key not set", "provider", key, "env", entry.APIKeyEnv)
			continue
		}
		providers[key] = ProviderConfig{
			Key:      key,
			Protocol: entry.Protocol,
			BaseURL:  entry.BaseURL,
			APIKey:   apiKey,
			Strict:   entry.Strict,
		}
	}
	return providers
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
