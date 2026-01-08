package config

import (
	"fmt"
	"os"
	"path/filepath"
)

type EndpointConfig struct {
	Folders      []string
	IncludeTools bool
}

func ValidateEndpoints() error {
	for name, cfg := range Endpoints {
		for _, folder := range cfg.Folders {
			path := filepath.Join(PromptsDir, folder)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("endpoint %q: folder %q does not exist", name, path)
			}
		}
	}
	return nil
}

var Endpoints = map[string]EndpointConfig{
	"converse": {
		Folders:      []string{"nabu/base"},
		IncludeTools: true,
	},
}

func GetEndpoint(name string) (EndpointConfig, bool) {
	cfg, ok := Endpoints[name]
	return cfg, ok
}
