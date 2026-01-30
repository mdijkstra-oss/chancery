package config

import (
	"fmt"
	"os"
	"path/filepath"
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
}

func GetEndpoint(name string) (EndpointConfig, bool) {
	cfg, ok := Endpoints[name]
	return cfg, ok
}
