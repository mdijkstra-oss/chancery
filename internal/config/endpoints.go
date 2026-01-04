package config

type EndpointConfig struct {
	Folders      []string
	IncludeTools bool
}

var Endpoints = map[string]EndpointConfig{
	"converse": {
		Folders:      []string{"nabu/base", "nabu/base-boundaries-intend"},
		IncludeTools: true,
	},
	"execute": {
		Folders:      []string{"nabu/base", "nabu/base-boundaries-intend", "nabu/execute"},
		IncludeTools: true,
	},
}

func GetEndpoint(name string) (EndpointConfig, bool) {
	cfg, ok := Endpoints[name]
	return cfg, ok
}
