package confy

type ConfigType string

const (
	Yaml ConfigType = "yaml"
	Json ConfigType = "json"
	Toml ConfigType = "toml"
	Auto ConfigType = "auto"
)
