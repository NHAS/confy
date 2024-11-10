package confy

import "math"

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
	env        preference = "env"
	configFile preference = "file"
)

const (
	confyTag            = "confy"
	confyDescriptionTag = "confy_description"
)

const (
	DefaultENVDelimiter = "_"
	DefaultCliDelimiter = "."

	LoggingDisabled = math.MaxInt
)

type Transform func(generated string) string
