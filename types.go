package confy

type loader[T any] interface {
	apply(current *T) error
}

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

const (
	confyTag            = "confy"
	confyDescriptionTag = "confy_description"
)
