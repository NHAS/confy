package confy

import (
	"errors"
	"math"
)

type loader[T any] interface {
	apply(current *T) (bool, error)
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

var errFatal = errors.New("fatal confy error: ")
