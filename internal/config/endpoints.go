package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EndpointConfig struct {
	Folder string
}

func ValidateEndpoints() error {
	for name, cfg := range Endpoints {
		path := filepath.Join(PromptsDir, cfg.Folder)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("endpoint %q: folder %q does not exist", name, path)
		}
	}
	return nil
}

var Endpoints = map[string]EndpointConfig{
	"converse": {
		Folder: "nabu/converse",
	},
	"ask-expert": {
		Folder: "nabu/expert",
	},
}

func GetEndpoint(name string) (EndpointConfig, bool) {
	if cfg, ok := Endpoints[name]; ok {
		return cfg, true
	}

	base, subpath := splitEndpointPath(name)
	if subpath == "" {
		return EndpointConfig{}, false
	}

	baseCfg, ok := Endpoints[base]
	if !ok {
		return EndpointConfig{}, false
	}

	folder := filepath.Join(baseCfg.Folder, subpath)
	path := filepath.Join(PromptsDir, folder)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return EndpointConfig{}, false
	}

	return EndpointConfig{Folder: folder}, true
}

func splitEndpointPath(name string) (base, subpath string) {
	idx := strings.Index(name, "/")
	if idx == -1 {
		return name, ""
	}
	return name[:idx], name[idx+1:]
}
