package confy

import (
	"errors"
	"flag"
	"log/slog"
	"math"
	"os"
	"reflect"
)

type option func(*options) error

type configDataOptions struct {
	strictParsing bool
	path          string
	rawData       []byte
	fileType      ConfigType
}

type options struct {
	config configDataOptions

	cli struct {
		delimiter string
	}

	env struct {
		delimiter string
	}

	level  *slog.LevelVar
	logger *slog.Logger

	order []preference
}

func initLogger(o *options, initialLevel slog.Level) {

	o.level = new(slog.LevelVar)

	o.level.Set(initialLevel)

	o.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: o.level,
	}))
}

// Config[T any] takes a structure and populates the exported fields from multiple configurable sources.
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
func Config[T any](config T, suppliedOptions ...option) (result T, warnings []error, err error) {
	if reflect.TypeOf(config).Kind() != reflect.Struct {
		panic("Config(...) only supports configs of Struct type")
	}

	o := options{}
	// disable logging by default
	initLogger(&o, math.MaxInt)

	for _, optFunc := range suppliedOptions {
		err := optFunc(&o)
		if err != nil {
			return result, nil, err
		}
	}

	if len(o.order) == 0 {
		if err := Defaults("config.json")(&o); err != nil {
			return result, nil, err
		}
	}

	orderLoadOpts := map[preference]loader[T]{
		cli:        newCliLoader[T](&o),
		env:        newEnvLoader[T](&o),
		configFile: newConfigLoader[T](&o),
	}

	o.logger.Info("Populating configuration in this order: ", slog.Any("order", o.order))

	for _, p := range o.order {

		f, ok := orderLoadOpts[p]
		if !ok {
			panic("unknown preference option: " + p)
		}

		err := f.apply(&result)
		if err != nil {
			if len(o.order) > 1 && !errors.Is(err, flag.ErrHelp) {
				o.logger.Warn("parser issued warning", "parser", p, "err", err.Error())

				warnings = append(warnings, err)
			} else {
				o.logger.Error("parser issued error", "parser", p, "err", err.Error())
				return result, nil, err
			}
		}
	}

	return
}

// WithLogLevel sets the current slog output level
// Defaulty logging is disabled
// Very useful for debugging
func WithLogLevel(level slog.Level) option {
	return func(c *options) error {
		c.level.Set(level)
		return nil
	}
}

// Defaults adds all three configuration determination options as on.
// The configs struct will be configured config file -> envs -> cli, so that cli takes precedence over more static options, for ease of user configuration.
// The config file will be parsed in a non-strict way (unknown fields will just be ignored) and the config file type is automatically determined from extension (supports yaml, toml and json), if you want to change this, add the FromConfigFile(...) option after Defaults(...)
// path string : config file path
func Defaults(path string) option {
	return func(c *options) error {

		// Process in config file -> env -> cli order
		FromConfigFile(path, false, Auto)(c)
		FromEnvs("_")(c)
		FromCli(".")(c)

		WithLogLevel(slog.LevelError)

		return nil
	}
}

// FromConfigFile tells confy to load a config file from path
// path: string config file path
// strictParsing: bool allow unknown fields to exist in config file
// configType: ConfigType, what type the config file is expected to be, use `Auto` if you dont care and just want it to choose for you. Supports yaml, toml and json
func FromConfigFile(path string, strictParsing bool, configType ConfigType) option {
	return func(c *options) error {

		// Unset bytes, ConfigBytes conflicts with ConfigFile
		c.config.rawData = nil
		c.config.path = path

		c.config.fileType = configType
		c.config.strictParsing = strictParsing

		c.order = append(c.order, configFile)

		return nil
	}
}

// FromConfigBytes tells confy to load a config file from raw bytes
// data: []byte config file raw bytes
// strictParsing: bool allow unknown fields to exist in config file
// configType: ConfigType, what type the config bytes are supports yaml, toml and json, the Auto configuration will return an error
func FromConfigBytes(data []byte, strictParsing bool, configType ConfigType) option {
	return func(c *options) error {
		if configType == Auto {
			return errors.New("you cannot use automatic configuration type determination from bytes")
		}

		// Unset path, ConfigBytes conflicts with ConfigFile
		c.config.path = ""
		c.config.rawData = data

		c.config.fileType = configType
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
func FromEnvs(delimiter string) option {
	return func(c *options) error {
		c.cli.delimiter = delimiter
		c.order = append(c.order, env)
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
func FromCli(delimiter string) option {
	return func(c *options) error {
		c.cli.delimiter = delimiter
		c.order = append(c.order, cli)
		return nil
	}
}
