package prompts

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

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
	Root         string
	Agents       map[string]CompiledAgent
	Configs      map[string]PromptConfig
	NamedConfigs map[string]map[string]PromptConfig
	Defaults     map[string]string
	Descriptions map[string]string
	Modes        map[string]string
	ProviderKeys []string
	models       map[string]modelEntry
	providers    map[string]ProviderConfig
}

type ResolvedAgent struct {
	Path   string
	Name   string
	Prompt CompiledAgent
	Config PromptConfig
}

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Diagnostic struct {
	Severity Severity `json:"severity"`
	Path     string   `json:"path"`
	Message  string   `json:"message"`
}

type Report struct {
	Diagnostics []Diagnostic `json:"diagnostics"`
}

func (r Registry) ResolveAgent(reference string) (ResolvedAgent, error) {
	if agent, ok := r.Agents[reference]; ok {
		cfg, found := r.Configs[reference]
		if !found {
			return ResolvedAgent{}, fmt.Errorf("no config for agent: %s", reference)
		}
		return ResolvedAgent{Path: reference, Name: r.Defaults[reference], Prompt: agent, Config: cfg}, nil
	}

	dot := strings.LastIndex(reference, ".")
	if dot == -1 {
		return ResolvedAgent{}, fmt.Errorf("unknown agent: %s", reference)
	}
	path := reference[:dot]
	name := reference[dot+1:]
	agent, ok := r.Agents[path]
	if !ok {
		return ResolvedAgent{}, fmt.Errorf("unknown agent: %s", path)
	}
	configs, ok := r.NamedConfigs[path]
	if !ok {
		return ResolvedAgent{}, fmt.Errorf("agent %q has no named models", path)
	}
	cfg, ok := configs[name]
	if !ok {
		return ResolvedAgent{}, fmt.Errorf("agent %q has no model named %q", path, name)
	}
	return ResolvedAgent{Path: path, Name: name, Prompt: agent, Config: cfg}, nil
}

func (r Registry) AgentPaths() []string {
	paths := make([]string, 0, len(r.Agents))
	for path := range r.Agents {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}

func (r Registry) ModelCount() int {
	count := 0
	for path := range r.Agents {
		if named := r.NamedConfigs[path]; len(named) > 0 {
			count += len(named)
			continue
		}
		count++
	}
	return count
}

func (r Registry) ProviderCount() int {
	return len(r.ProviderKeys)
}

func (r Registry) WithAPIKeys(lookup func(string) string) (Registry, error) {
	result := r
	result.Agents = cloneMap(r.Agents)
	result.Configs = make(map[string]PromptConfig, len(r.Configs))
	for key, cfg := range r.Configs {
		resolved, err := ResolveAPIKey(cfg, lookup)
		if err != nil {
			return Registry{}, fmt.Errorf("agent %q: %w", key, err)
		}
		result.Configs[key] = resolved
	}
	result.NamedConfigs = make(map[string]map[string]PromptConfig, len(r.NamedConfigs))
	for path, configs := range r.NamedConfigs {
		resolvedConfigs := make(map[string]PromptConfig, len(configs))
		for name, cfg := range configs {
			resolved, err := ResolveAPIKey(cfg, lookup)
			if err != nil {
				return Registry{}, fmt.Errorf("agent %q model %q: %w", path, name, err)
			}
			resolvedConfigs[name] = resolved
		}
		result.NamedConfigs[path] = resolvedConfigs
	}
	result.Defaults = cloneMap(r.Defaults)
	result.Descriptions = cloneMap(r.Descriptions)
	result.Modes = cloneMap(r.Modes)
	result.ProviderKeys = slices.Clone(r.ProviderKeys)
	result.models = cloneMap(r.models)
	result.providers = make(map[string]ProviderConfig, len(r.providers))
	for key, provider := range r.providers {
		apiKey := lookup(provider.APIKeyEnv)
		if apiKey == "" {
			return Registry{}, fmt.Errorf("provider %q: environment variable %s is not set", key, provider.APIKeyEnv)
		}
		provider.APIKey = apiKey
		result.providers[key] = provider
	}
	return result, nil
}

func ResolveAPIKey(cfg PromptConfig, lookup func(string) string) (PromptConfig, error) {
	env := cfg.Provider.APIKeyEnv
	if env == "" {
		return PromptConfig{}, fmt.Errorf("provider %q has no api_key_env", cfg.Provider.Key)
	}
	apiKey := lookup(env)
	if apiKey == "" {
		return PromptConfig{}, fmt.Errorf("environment variable %s is not set", env)
	}
	result := cfg
	result.Provider.APIKey = apiKey
	return result, nil
}

func (r Report) HasErrors() bool {
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (r Report) ErrorCount() int {
	count := 0
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == SeverityError {
			count++
		}
	}
	return count
}

func (r Report) WarningCount() int {
	count := 0
	for _, diagnostic := range r.Diagnostics {
		if diagnostic.Severity == SeverityWarning {
			count++
		}
	}
	return count
}

var includePattern = regexp.MustCompile(`^\[([^\]]+\.md)\]$`)

func ParseManifest(content string) []Line {
	raw := strings.Split(content, "\n")
	var lines []Line
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if match := includePattern.FindStringSubmatch(trimmed); match != nil {
			lines = append(lines, Line{Include: match[1]})
			continue
		}
		lines = append(lines, Line{Literal: line})
	}
	return lines
}

func ResolveManifest(lines []Line, readFile func(string) (string, error), sharedDir, localDir string, skip func(string) bool) (CompiledAgent, error) {
	var parts []string
	var sources []string
	var segments []Segment
	for _, line := range lines {
		if line.Include != "" {
			if skip != nil && skip(line.Include) {
				placeholder := "[" + line.Include + "]"
				parts = append(parts, placeholder)
				segments = append(segments, Segment{Content: placeholder})
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

func ManifestKeyFromPath(path, root string) string {
	rel, err := filepath.Rel(root, path)
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

func Load(root string) (Registry, Report) {
	registry := Registry{
		Root:         root,
		Agents:       make(map[string]CompiledAgent),
		Configs:      make(map[string]PromptConfig),
		NamedConfigs: make(map[string]map[string]PromptConfig),
		Defaults:     make(map[string]string),
		Descriptions: make(map[string]string),
		Modes:        make(map[string]string),
		models:       make(map[string]modelEntry),
		providers:    make(map[string]ProviderConfig),
	}
	var report Report

	info, err := os.Stat(root)
	if err != nil {
		report.addError(root, err.Error())
		return registry, report
	}
	if !info.IsDir() {
		report.addError(root, "config path is not a directory")
		return registry, report
	}

	loadProviders(root, &registry, &report)
	compileModes(root, &registry, &report)
	validateToolPaths(root, &report)
	loadAgents(root, &registry, &report)
	report.sort()
	return registry, report
}

func resolveInclude(include string, readFile func(string) (string, error), localDir, sharedDir string) (string, error) {
	if localDir != "" {
		if content, err := readFile(filepath.Join(localDir, include)); err == nil {
			return content, nil
		}
	}
	return readFile(filepath.Join(sharedDir, include))
}

func loadProviders(root string, registry *Registry, report *Report) {
	path := filepath.Join(root, "providers.yaml")
	data, err := readConfigFile(root, path)
	if err != nil {
		report.addError("providers.yaml", err.Error())
		return
	}
	var file providersFile
	if err := decodeYAML(data, &file); err != nil {
		report.addError("providers.yaml", "malformed YAML: "+err.Error())
		return
	}
	if len(file.Providers) == 0 {
		report.addError("providers.yaml", "no providers configured")
		return
	}

	allModels := make(map[string]modelEntry)
	providerKeys := make([]string, 0, len(file.Providers))
	for key, providerFile := range file.Providers {
		providerKeys = append(providerKeys, key)
		validateProvider(key, providerFile, report)
		registry.providers[key] = ProviderConfig{
			Key:       key,
			Protocol:  providerFile.Protocol,
			BaseURL:   providerFile.BaseURL,
			APIKeyEnv: providerFile.APIKeyEnv,
			Strict:    providerFile.Strict,
		}
		for modelKey, model := range providerFile.Models {
			if _, exists := allModels[modelKey]; exists {
				report.addError("providers.yaml", fmt.Sprintf("model %q is defined more than once", modelKey))
				continue
			}
			if model.Provider == "" {
				model.Provider = key
			}
			allModels[modelKey] = model
		}
	}
	slices.Sort(providerKeys)
	registry.ProviderKeys = providerKeys

	resolved, err := resolveModels(allModels)
	if err != nil {
		report.addError("providers.yaml", err.Error())
		return
	}
	registry.models = resolved
	for key, model := range resolved {
		if _, ok := registry.providers[model.Provider]; !ok {
			report.addError("providers.yaml", fmt.Sprintf("model %q references unknown provider %q", key, model.Provider))
		}
	}
}

func validateProvider(key string, provider providerFile, report *Report) {
	if err := validateProtocol(provider.Protocol); err != nil {
		report.addError("providers.yaml", fmt.Sprintf("provider %q: %v", key, err))
	}
	if provider.BaseURL == "" {
		report.addError("providers.yaml", fmt.Sprintf("provider %q has no base_url", key))
	}
	if provider.APIKeyEnv == "" {
		report.addError("providers.yaml", fmt.Sprintf("provider %q has no api_key_env", key))
	}
	if len(provider.Models) == 0 {
		report.addError("providers.yaml", fmt.Sprintf("provider %q has no models", key))
	}
}

func compileModes(root string, registry *Registry, report *Report) {
	modesDir := filepath.Join(root, "modes")
	entries, err := os.ReadDir(modesDir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		report.addError("modes", err.Error())
		return
	}
	sharedDir := filepath.Join(root, "shared")
	for _, entry := range entries {
		if entry.IsDir() || !isMarkdownName(entry.Name()) {
			continue
		}
		path := filepath.Join(modesDir, entry.Name())
		data, err := readConfigFile(root, path)
		if err != nil {
			report.addError(relativePath(root, path), err.Error())
			continue
		}
		compiled, err := ResolveManifest(ParseManifest(string(data)), configReader(root), sharedDir, modesDir, nil)
		if err != nil {
			report.addError(relativePath(root, path), err.Error())
			continue
		}
		key := strings.TrimSuffix(entry.Name(), ".md")
		registry.Modes[key] = compiled.Prompt
	}
}

func validateToolPaths(root string, report *Report) {
	toolsDir := filepath.Join(root, "tools")
	err := filepath.WalkDir(toolsDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) && path == toolsDir {
				return nil
			}
			report.addError(relativePath(root, path), walkErr.Error())
			return nil
		}
		if _, err := resolveConfigPath(root, path); err != nil {
			report.addError(relativePath(root, path), err.Error())
			if path == toolsDir || entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		report.addError("tools", err.Error())
	}
}

func loadAgents(root string, registry *Registry, report *Report) {
	var fragments []string
	referenced := make(map[string]bool)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			report.addError(relativePath(root, path), walkErr.Error())
			return nil
		}
		if path == root {
			return nil
		}
		rel := relativePath(root, path)
		if entry.IsDir() {
			if isReservedDirectory(rel) || strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !isMarkdownName(entry.Name()) {
			return nil
		}
		data, err := readConfigFile(root, path)
		if err != nil {
			report.addError(rel, err.Error())
			return nil
		}
		frontmatter, body, found, err := parseAgentFile(data)
		if err != nil {
			report.addError(rel, err.Error())
			return nil
		}
		if !found {
			fragments = append(fragments, path)
			return nil
		}
		loadAgent(root, path, body, frontmatter, registry, report, referenced)
		return nil
	})
	if err != nil {
		report.addError(root, err.Error())
	}
	validateNamedRoutes(registry, report)
	for _, path := range fragments {
		if !referenced[filepath.Clean(path)] {
			report.addError(relativePath(root, path), "orphaned Markdown file has no frontmatter and is not included by an agent")
		}
	}
}

func validateNamedRoutes(registry *Registry, report *Report) {
	for path, configs := range registry.NamedConfigs {
		for name := range configs {
			route := path + "." + name
			if _, exists := registry.Agents[route]; exists {
				report.addError(route, fmt.Sprintf("agent path %q collides with named model route %q", route, route))
			}
		}
	}
}

func validateAgentPromptField(path string, prompt *string, report *Report) {
	if prompt == nil {
		return
	}
	if *prompt == "" {
		report.addWarning(path, "prompt frontmatter is ignored; use the Markdown body")
		return
	}
	report.addError(path, "prompt frontmatter is not supported; move the prompt into the Markdown body")
}

func isNamedModelName(name string) bool {
	return name != "" && !strings.ContainsAny(name, ".\\/")
}

func loadAgent(root, path, body string, frontmatter agentFrontmatter, registry *Registry, report *Report, referenced map[string]bool) {
	rel := relativePath(root, path)
	key := ManifestKeyFromPath(path, root)
	if key == "" {
		report.addError(rel, "top-level index.md has no routable path")
		return
	}
	if _, exists := registry.Agents[key]; exists {
		report.addError(rel, fmt.Sprintf("duplicate agent path %q", key))
		return
	}

	lines := ParseManifest(body)
	markIncludes(lines, filepath.Dir(path), filepath.Join(root, "shared"), referenced)
	compiled, err := ResolveManifest(lines, configReader(root), filepath.Join(root, "shared"), filepath.Dir(path), nil)
	if err != nil {
		report.addError(rel, err.Error())
		return
	}
	registry.Agents[key] = compiled
	registry.Descriptions[key] = frontmatter.Description
	validateAgentPromptField(rel, frontmatter.Prompt, report)
	if strings.TrimSpace(compiled.Prompt) == "" {
		report.addWarning(rel, "empty prompt body; the prompt must come from the caller")
	}

	hasSingle := frontmatter.Model != ""
	hasNamed := len(frontmatter.Models) > 0
	if hasSingle == hasNamed {
		report.addError(rel, "frontmatter must define exactly one of model or models")
		return
	}
	if hasSingle {
		if frontmatter.Default != "" {
			report.addError(rel, "default is only valid with models")
		}
		cfg, err := buildConfig(root, frontmatter.agentEntry, registry)
		if err != nil {
			report.addError(rel, err.Error())
			return
		}
		registry.Configs[key] = cfg
		return
	}

	defaultName := frontmatter.Default
	if len(frontmatter.Models) > 1 && defaultName == "" {
		report.addError(rel, "models with more than one entry require default")
	}
	if len(frontmatter.Models) == 1 && defaultName == "" {
		for name := range frontmatter.Models {
			defaultName = name
		}
	}
	if defaultName != "" {
		if _, ok := frontmatter.Models[defaultName]; !ok {
			report.addError(rel, fmt.Sprintf("default %q does not name a models entry", defaultName))
		}
	}

	configs := make(map[string]PromptConfig, len(frontmatter.Models))
	for name, modelSettings := range frontmatter.Models {
		if !isNamedModelName(name) {
			report.addError(rel, fmt.Sprintf("model entry name %q must not contain dots or slashes", name))
			continue
		}
		validateAgentPromptField(rel, modelSettings.Prompt, report)
		if modelSettings.Model == "" {
			report.addError(rel, fmt.Sprintf("model entry %q has no model", name))
			continue
		}
		settings := overlayAgent(frontmatter.agentEntry, modelSettings)
		settings.Model = modelSettings.Model
		cfg, err := buildConfig(root, settings, registry)
		if err != nil {
			report.addError(rel, fmt.Sprintf("model entry %q: %v", name, err))
			continue
		}
		configs[name] = cfg
	}
	registry.NamedConfigs[key] = configs
	registry.Defaults[key] = defaultName
	if cfg, ok := configs[defaultName]; ok {
		registry.Configs[key] = cfg
	}
}

func buildConfig(root string, agent agentEntry, registry *Registry) (PromptConfig, error) {
	model, ok := registry.models[agent.Model]
	if !ok {
		return PromptConfig{}, fmt.Errorf("unknown model %q", agent.Model)
	}
	provider, ok := registry.providers[model.Provider]
	if !ok {
		return PromptConfig{}, fmt.Errorf("model %q references unknown provider %q", agent.Model, model.Provider)
	}
	cfg := mergeConfig(model, agent, provider)
	if cfg.Prompt != "" {
		data, err := readConfigFile(root, filepath.Join(root, "shared", cfg.Prompt))
		if err != nil {
			return PromptConfig{}, fmt.Errorf("model prompt %q: %w", cfg.Prompt, err)
		}
		cfg.Prompt = strings.TrimRight(string(data), "\n")
	}
	if cfg.Seed && cfg.Provider.Protocol != ProtocolGemini {
		return PromptConfig{}, fmt.Errorf("seed is enabled but protocol %q does not support it", cfg.Provider.Protocol)
	}
	return cfg, nil
}

func parseAgentFile(data []byte) (agentFrontmatter, string, bool, error) {
	content := strings.ReplaceAll(string(data), "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return agentFrontmatter{}, content, false, nil
	}
	lines := strings.Split(content[4:], "\n")
	for index, line := range lines {
		if line != "---" {
			continue
		}
		var frontmatter agentFrontmatter
		if err := decodeYAML([]byte(strings.Join(lines[:index], "\n")), &frontmatter); err != nil {
			return agentFrontmatter{}, "", true, fmt.Errorf("malformed YAML frontmatter: %w", err)
		}
		body := strings.Join(lines[index+1:], "\n")
		return frontmatter, body, true, nil
	}
	return agentFrontmatter{}, "", true, fmt.Errorf("frontmatter is missing closing delimiter")
}

func decodeYAML(data []byte, target any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(target); err != nil {
		return sanitizeYAMLError(err)
	}
	return nil
}

var unknownFieldPattern = regexp.MustCompile(`field (\S+) not found in type \S+`)

func sanitizeYAMLError(err error) error {
	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		return err
	}
	messages := make([]string, len(typeErr.Errors))
	for index, message := range typeErr.Errors {
		messages[index] = unknownFieldPattern.ReplaceAllString(message, `unknown field "$1"`)
	}
	return errors.New(strings.Join(messages, "; "))
}

func markIncludes(lines []Line, localDir, sharedDir string, referenced map[string]bool) {
	for _, line := range lines {
		if line.Include == "" {
			continue
		}
		path := filepath.Join(localDir, line.Include)
		if _, err := os.Stat(path); err != nil {
			path = filepath.Join(sharedDir, line.Include)
		}
		if _, err := os.Stat(path); err == nil {
			referenced[filepath.Clean(path)] = true
		}
	}
}

func isReservedDirectory(rel string) bool {
	first, _, _ := strings.Cut(filepath.ToSlash(rel), "/")
	switch first {
	case "shared", "modes", "tools":
		return true
	default:
		return false
	}
}

func isMarkdownName(name string) bool {
	return strings.HasSuffix(name, ".md") && !strings.Contains(name, ".temp")
}

func configReader(root string) func(string) (string, error) {
	return func(path string) (string, error) {
		data, err := readConfigFile(root, path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
}

func readConfigFile(root, path string) ([]byte, error) {
	resolved, err := resolveConfigPath(root, path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(resolved)
}

func resolveConfigPath(root, path string) (string, error) {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	filePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !isWithin(rootPath, filePath) {
		return "", fmt.Errorf("path escapes config directory: %s", path)
	}
	resolvedRoot, err := filepath.EvalSymlinks(rootPath)
	if err != nil {
		return "", err
	}
	resolvedFile, err := filepath.EvalSymlinks(filePath)
	if err != nil {
		return "", err
	}
	if !isWithin(resolvedRoot, resolvedFile) {
		return "", fmt.Errorf("path escapes config directory through symlink: %s", path)
	}
	return resolvedFile, nil
}

func isWithin(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

const maxModelDepth = 5

func resolveModels(models map[string]modelEntry) (map[string]modelEntry, error) {
	resolved := make(map[string]modelEntry, len(models))
	for key := range models {
		model, err := resolveModel(key, models)
		if err != nil {
			return nil, err
		}
		if model.Name == "" {
			model.Name = key
		}
		resolved[key] = model
	}
	return resolved, nil
}

func resolveModel(key string, models map[string]modelEntry) (modelEntry, error) {
	current := key
	var chain []modelEntry
	visited := make(map[string]bool)
	for len(chain) <= maxModelDepth {
		if visited[current] {
			return modelEntry{}, fmt.Errorf("model %q has an extends cycle at %q", key, current)
		}
		visited[current] = true
		entry, ok := models[current]
		if !ok {
			return modelEntry{}, fmt.Errorf("model %q references unknown model %q", key, current)
		}
		chain = append(chain, entry)
		if entry.Extends == "" {
			break
		}
		current = entry.Extends
	}
	if chain[len(chain)-1].Extends != "" {
		return modelEntry{}, fmt.Errorf("model %q extends chain exceeds %d steps", key, maxModelDepth)
	}
	result := chain[len(chain)-1]
	for index := len(chain) - 2; index >= 0; index-- {
		result = overlayModel(result, chain[index])
	}
	return result, nil
}

func overlayModel(base, child modelEntry) modelEntry {
	result := base
	result.Extends = ""
	result.Provider = coalesce(base.Provider, child.Provider)
	result.Name = coalesce(base.Name, child.Name)
	result.Type = coalesce(base.Type, child.Type)
	result.Dimensions = coalesce(base.Dimensions, child.Dimensions)
	result.MaxTokens = coalesce(base.MaxTokens, child.MaxTokens)
	result.Prompt = coalesce(base.Prompt, child.Prompt)
	result.ReasoningEffort = coalesce(base.ReasoningEffort, child.ReasoningEffort)
	result.ReasoningSummary = coalesce(base.ReasoningSummary, child.ReasoningSummary)
	result.Verbosity = coalesce(base.Verbosity, child.Verbosity)
	result.ServiceTier = coalesce(base.ServiceTier, child.ServiceTier)
	result.LegacyThinking = coalesce(base.LegacyThinking, child.LegacyThinking)
	result.AutoCache = coalesce(base.AutoCache, child.AutoCache)
	result.CacheTTL = coalesce(base.CacheTTL, child.CacheTTL)
	return result
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
		LegacyThinking:   model.LegacyThinking,
		AutoCache:        model.AutoCache,
		CacheTTL:         model.CacheTTL,
		Provider:         provider,
	}
	cfg.ReasoningEffort = coalesce(cfg.ReasoningEffort, agent.ReasoningEffort)
	cfg.ReasoningSummary = coalesce(cfg.ReasoningSummary, agent.ReasoningSummary)
	cfg.Verbosity = coalesce(cfg.Verbosity, agent.Verbosity)
	cfg.ServiceTier = coalesce(cfg.ServiceTier, agent.ServiceTier)
	cfg.CacheTTL = coalesce(cfg.CacheTTL, agent.CacheTTL)
	cfg.Dimensions = coalesce(cfg.Dimensions, agent.Dimensions)
	cfg.MaxTokens = coalesce(cfg.MaxTokens, agent.MaxTokens)
	if agent.LegacyThinking != nil {
		cfg.LegacyThinking = *agent.LegacyThinking
	}
	if agent.Temperature != nil {
		cfg.Temperature = agent.Temperature
	}
	if agent.Seed != nil {
		cfg.Seed = *agent.Seed
	}
	if agent.AutoCache != nil {
		cfg.AutoCache = *agent.AutoCache
	}
	return cfg
}

func overlayAgent(base, child agentEntry) agentEntry {
	result := base
	result.Model = coalesce(base.Model, child.Model)
	result.ReasoningEffort = coalesce(base.ReasoningEffort, child.ReasoningEffort)
	result.ReasoningSummary = coalesce(base.ReasoningSummary, child.ReasoningSummary)
	result.Verbosity = coalesce(base.Verbosity, child.Verbosity)
	result.ServiceTier = coalesce(base.ServiceTier, child.ServiceTier)
	result.LegacyThinking = coalesce(base.LegacyThinking, child.LegacyThinking)
	result.Temperature = coalesce(base.Temperature, child.Temperature)
	result.Seed = coalesce(base.Seed, child.Seed)
	result.AutoCache = coalesce(base.AutoCache, child.AutoCache)
	result.CacheTTL = coalesce(base.CacheTTL, child.CacheTTL)
	result.Dimensions = coalesce(base.Dimensions, child.Dimensions)
	result.MaxTokens = coalesce(base.MaxTokens, child.MaxTokens)
	return result
}

func validateProtocol(protocol Protocol) error {
	switch protocol {
	case ProtocolResponses, ProtocolGemini, ProtocolCompletions, ProtocolAnthropic:
		return nil
	default:
		return fmt.Errorf("unknown protocol: %q", protocol)
	}
}

func (r *Report) addError(path, message string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{Severity: SeverityError, Path: filepath.ToSlash(path), Message: message})
}

func (r *Report) addWarning(path, message string) {
	r.Diagnostics = append(r.Diagnostics, Diagnostic{Severity: SeverityWarning, Path: filepath.ToSlash(path), Message: message})
}

func (r *Report) sort() {
	slices.SortFunc(r.Diagnostics, func(left, right Diagnostic) int {
		if left.Severity != right.Severity {
			return strings.Compare(string(left.Severity), string(right.Severity))
		}
		if left.Path != right.Path {
			return strings.Compare(left.Path, right.Path)
		}
		return strings.Compare(left.Message, right.Message)
	})
}

func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func coalesce[T comparable](base, override T) T {
	var zero T
	if override != zero {
		return override
	}
	return base
}

func cloneMap[K comparable, V any](source map[K]V) map[K]V {
	result := make(map[K]V, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}
