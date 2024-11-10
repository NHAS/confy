package confy

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type optionFunc func(*options) error

type configDataOptions struct {
	strictParsing bool

	dataMethod func() (io.Reader, ConfigType, error)
}

type transientOptions struct {
	delimiter string
	transform Transform
}

type options struct {
	config configDataOptions

	cli transientOptions
	env transientOptions

	order        []preference
	currentlySet map[preference]bool
}

var (
	level  *slog.LevelVar
	logger *slog.Logger
)

func init() {
	level = new(slog.LevelVar)
	level.Set(LoggingDisabled)

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	}))
}

// Config[T any] takes a structure and populates the exported fields from multiple configurable sources.
// parameters:
//
// config T;                   A structure to populate
// suppliedOptions ...option:  Various options that can be used, disabling configuration sources etc
//
// return:
//
// result T:         The populated configuration file
// warnings []error: Non-fatal errors, but should be at least printed
// err error:        Fatal errors while parsing/extracting/populating the configuration file
//
// Sources:
//   - CLI using the "flag" package
//   - Environment Variables using os.Getenv(...)
//   - Configuration File using filepath or raw bytes, this supports yaml, json and toml so your configuration can file can be any of those types
//
// Tags
// Confy defines two flags:
//
//   - confy:"field_name;sensitive"
//     The "confy" tag is used to rename the field, meaning changing what env variables, cli flags and configuration file/bytes fields to look for
//
//   - confy_description:"Field Description here"
//     Sets the description of a field when being added to cli parsing, so when using -confy-help (or entering an invalid flag) it will so a good description
//
// Important Note:
//
//	Configuring from Envs or CLI flags is more difficult for complex types (like structures)
//	As such to unmarshal very complex structs, or structs with private members, the struct must implement encoding.TextUnmarshaler and encoding.TextMarshaler to work properly
//
// CLI
// Confy will automatically register flags from the configuration file structure and parse os.Args when FromCli is supplied as an option (or defaults are used)
// flags will be in the following format:
//
//	 struct {
//	    Thing string
//	       Nested struct {
//			   NestedField string
//		   }
//	 }
//
//	 would look for environment variables:
//	 -Thing
//	 -Nested_NestedField
//
// ENV
//
//	 struct {
//	    Thing string
//	       Nested struct {
//			   NestedField string
//		   }
//	 }
//
//	 would look for environment variables:
//	 Thing
//	 Nested_NestedField
func Config[T any](suppliedOptions ...optionFunc) (result T, warnings []error, err error) {
	if reflect.TypeOf(result).Kind() != reflect.Struct {
		panic("Config(...) only supports configs of Struct type")
	}

	o := options{
		currentlySet: make(map[preference]bool),
	}

	for _, optFunc := range suppliedOptions {
		err := optFunc(&o)
		if err != nil {
			return result, nil, err
		}
	}

	if len(o.order) == 0 {
		if err := Defaults("config.json", false)(&o); err != nil {
			return result, nil, err
		}
	}

	orderLoadOpts := map[preference]loader[T]{
		cli:        newCliLoader[T](&o),
		env:        newEnvLoader[T](&o),
		configFile: newConfigLoader[T](&o),
	}

	logger.Info("Populating configuration in this order: ", slog.Any("order", o.order))

	for _, p := range o.order {

		f, ok := orderLoadOpts[p]
		if !ok {
			panic("unknown preference option: " + p)
		}

		err := f.apply(&result)
		if err != nil {
			if len(o.order) > 1 && !errors.Is(err, flag.ErrHelp) {
				logger.Warn("parser issued warning", "parser", p, "err", err.Error())

				warnings = append(warnings, err)
			} else {
				logger.Error("parser issued error", "parser", p, "err", err.Error())
				return result, nil, err
			}
		}
	}

	return
}

// WithLogLevel sets the current slog output level
// Defaulty logging is disabled
// Very useful for debugging
func WithLogLevel(logLevel slog.Level) optionFunc {
	return func(c *options) error {
		level.Set(logLevel)
		return nil
	}
}

// Defaults adds all three configuration determination options as on.
// The configs struct will be configured config file -> envs -> cli, so that cli takes precedence over more static options, for ease of user configuration.
// The config file will be parsed in a non-strict way (unknown fields will just be ignored) and the config file type is automatically determined from extension (supports yaml, toml and json), if you want to change this, add the FromConfigFile(...) option after Defaults(...)
// path string : config file path
func Defaults(path string, strictConfigFileParsing bool) optionFunc {
	return func(c *options) error {

		// Process in config file -> env -> cli order
		err := FromConfigFile(path, false, Auto)(c)
		if err != nil {
			return err
		}
		err = FromEnvs(DefaultENVDelimiter)(c)
		if err != nil {
			return err
		}
		err = FromCli(DefaultCliDelimiter)(c)
		if err != nil {
			return err
		}
		WithLogLevel(slog.LevelError)

		return nil
	}
}

// FromConfigFile tells confy to load a config file from path
// path: string config file path
// strictParsing: bool allow unknown fields to exist in config file
// configType: ConfigType, what type the config file is expected to be, use `Auto` if you dont care and just want it to choose for you. Supports yaml, toml and json
func FromConfigFile(path string, strictParsing bool, configType ConfigType) optionFunc {
	return func(c *options) error {
		if c.currentlySet[configFile] {
			return errors.New("duplicate configuration information source, " + string(configFile) + " FromConfig* option set twice, mutually exclusive")
		}
		c.currentlySet[configFile] = true

		c.config.dataMethod = func() (io.Reader, ConfigType, error) {
			configData, err := os.Open(path)
			if err != nil {
				return nil, "", fmt.Errorf("failed to open config file %q, err: %s", path, err)
			}

			var fileType ConfigType
			if configType == Auto {
				ext := strings.ToLower(filepath.Ext(path))
				switch ext {
				case ".yml", ".yaml":
					logger.Info("yaml chosen as config type", "file_path", path)

					fileType = Yaml
				case ".json", ".js":
					logger.Info("json chosen as config type", "file_path", path)

					fileType = Json
				case ".toml", ".tml":
					logger.Info("toml chosen as config type", "file_path", path)

					fileType = Toml
				default:
					return nil, "", fmt.Errorf("unsupported file extension %q", strings.ToLower(filepath.Ext(path)))
				}
			}

			return configData, fileType, err
		}

		c.config.strictParsing = strictParsing

		c.order = append(c.order, configFile)

		return nil
	}
}

// FromConfigBytes tells confy to load a config file from raw bytes
// data: []byte config file raw bytes
// strictParsing: bool allow unknown fields to exist in config file
// configType: ConfigType, what type the config bytes are supports yaml, toml and json, the Auto configuration will return an error
func FromConfigBytes(data []byte, strictParsing bool, configType ConfigType) optionFunc {
	return func(c *options) error {
		if c.currentlySet[configFile] {
			return errors.New("duplicate configuration information source, " + string(configFile) + " FromConfig* option set twice, mutually exclusive")
		}
		c.currentlySet[configFile] = true

		if configType == Auto {
			return errors.New("you cannot use automatic configuration type determination from bytes")
		}

		c.config.dataMethod = func() (io.Reader, ConfigType, error) {
			if len(data) == 0 {
				return nil, "", errors.New("no config data supplied")
			}

			if configType == Auto {
				return nil, "", errors.New("cannot use Auto to determine config type from bytes")
			}

			return bytes.NewBuffer(data), configType, nil
		}

		c.config.strictParsing = strictParsing

		c.order = append(c.order, configFile)

		return nil
	}
}

// FromEnvs sets confy to automatically populate the configuration structure from environment variables
// delimiter: string when looking for environment variables this string should be used for denoting nested structures
// e.g
//
//	 delimiter = _
//	 struct {
//	    Thing string
//	       Nested struct {
//			   NestedField string
//		   }
//	 }
//
//	 would look for environment variables:
//	 Thing
//	 Nested_NestedField
//
//	 Configuring from Envs cannot be as comprehensive for complex types (like structures) as using the configuration file.
//	 To unmarshal very complex structs, the struct must implement encoding.TextUnmarshaler
func FromEnvs(delimiter string) optionFunc {
	return func(c *options) error {
		c.env.delimiter = delimiter
		c.order = append(c.order, env)
		return nil
	}
}

// WithEnvTransform runs the auto generated env through function t(generated string)string
// allowing you to change the env variable name if required
//
// E.g
//
//	transformFunc := func(env string)string {
//	   return strings.ToUpper(env)
//	}
//
// WithCliTransform(transformFunc)
// results searching for env variables that are all upper case
func WithEnvTransform(t Transform) optionFunc {
	return func(c *options) error {
		if t == nil {
			logger.Warn("WithEnvTransform was used, but transform was nil")
		}

		c.env.transform = t
		return nil
	}
}

// FromCli will automatically look for configuration variables from CLI flags
// delimiter: string when looking for cli flags this string should be used for denoting nested structures
// e.g
//
//	 delimiter = .
//	 struct {
//	    Thing string
//	       Nested struct {
//			   NestedField string
//		   }
//	 }
//
//	 would look for CLI flags:
//	 -Thing
//	 -Nested_NestedField
//
//	 Configuring from Cli cannot be as comprehensive for complex types (like structures) as using the configuration file.
//	 To unmarshal very complex structs, the struct must implement encoding.TextUnmarshaler and encoding.TextMarshaler
func FromCli(delimiter string) optionFunc {
	return func(c *options) error {
		c.cli.delimiter = delimiter
		c.order = append(c.order, cli)
		return nil
	}
}

// WithCliTransform runs the auto generated cli flag name through function t(generated string)string
// allowing you to change the flag name if required
//
// E.g
//
//	transformFunc := func(flag string)string {
//	   return strings.ToUpper(flag)
//	}
//
// WithCliTransform(transformFunc)
// results in cli flag names that are all upper case
func WithCliTransform(t Transform) optionFunc {
	return func(c *options) error {
		if t == nil {
			logger.Warn("WithCliTranform was used, but transform was nil")
		}

		c.cli.transform = t
		return nil
	}
}
