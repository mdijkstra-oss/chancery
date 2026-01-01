package config

type EndpointConfig struct {
	Folders      []string
	IncludeTools bool
}

var Endpoints = map[string]EndpointConfig{
	"converse": {
		Folders:      []string{"base", "base-boundaries-intend"},
		IncludeTools: true,
	},
	"plan": {
		Folders:      []string{"base", "base-boundaries-intend", "plan"},
		IncludeTools: true,
	},
	"execute": {
		Folders:      []string{"base", "base-boundaries-intend", "execute"},
		IncludeTools: true,
	},
}

func GetEndpoint(name string) (EndpointConfig, bool) {
	cfg, ok := Endpoints[name]
	return cfg, ok
}
