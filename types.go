package confy

type ConfigType string

const (
	Yaml ConfigType = "yaml"
	Json ConfigType = "json"
	Toml ConfigType = "toml"
	Auto ConfigType = "auto"
)

type preference string

const (
	cli        preference = "cli"
	env        preference = "Env"
	configFile preference = "file"
)

type loader[T any] func(o options, current *T) error

const (
	confyTag            = "confy"
	confyDescriptionTag = "confy_description"
)
