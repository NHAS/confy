package confy

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"time"
)

type OptionFunc func(*options) error

type configDataOptions struct {
	strictParsing bool
	required      bool

	dataMethod func() (io.Reader, ConfigType, error)
}

type envOptions struct {
	delimiter string
	transform Transform
}

type cliOptions struct {
	envOptions
	commandLine *flag.FlagSet
}

type options struct {
	config configDataOptions

	cli cliOptions
	env envOptions

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
func Config[T any](suppliedOptions ...OptionFunc) (result T, warnings []error, err error) {
	if reflect.TypeOf(result).Kind() != reflect.Struct {
		panic("Config(...) only supports configs of Struct type")
	}

	o := options{
		currentlySet: make(map[preference]bool),
	}

	orderLoadOpts := map[preference]loader[T]{
		cli:        newCliLoader[T](&o),
		env:        newEnvLoader[T](&o),
		configFile: newConfigLoader[T](&o),
	}

	var errs []error
	for _, optFunc := range suppliedOptions {
		err := optFunc(&o)
		if err != nil {
			errs = append(errs, err)
		}
	}

	cErr := errors.Join(errs...)
	if cErr != nil {
		// special case, if cli is enabled print out the help from that too
		if errors.Is(cErr, flag.ErrHelp) && slices.Contains(o.order, cli) {
			orderLoadOpts[cli].apply(&result)
		}
		return result, nil, cErr
	}

	if len(o.order) == 0 {
		if err := Defaults("config", "config.json")(&o); err != nil {
			if errors.Is(err, flag.ErrHelp) && slices.Contains(o.order, cli) {
				orderLoadOpts[cli].apply(&result)
			}

			return result, nil, err
		}
	}

	logger.Info("Populating configuration in this order: ", slog.Any("order", o.order))

	anythingWasSet := false
	for _, p := range o.order {

		f, ok := orderLoadOpts[p]
		if !ok {
			panic("unknown preference option: " + p)
		}

		somethingWasSet, err := f.apply(&result)
		if err != nil {

			if errors.Is(err, errFatal) {
				return result, nil, err
			}

			if len(o.order) > 1 && !errors.Is(err, flag.ErrHelp) {
				logger.Warn("parser issued warning", "parser", p, "err", err.Error())

				warnings = append(warnings, err)
			} else {
				logger.Error("parser issued error", "parser", p, "err", err.Error())
				return result, nil, err
			}
		}

		if somethingWasSet {
			anythingWasSet = true
		}

	}

	if !anythingWasSet {
		return result, warnings, fmt.Errorf("nothing was set in configuration from sources: %s, warnings: %v", o.order, errors.Join(warnings...))
	}

	return
}

// WithLogLevel sets the current slog output level
// Defaulty logging is disabled
// Very useful for debugging
func WithLogLevel(logLevel slog.Level) OptionFunc {
	return func(c *options) error {
		level.Set(logLevel)
		return nil
	}
}

// Defaults adds all three configuration determination options as on.
// The configs struct will be configured config file -> envs -> cli, so that cli takes precedence over more static options, for ease of user configuration.
// The config file will be parsed in a non-strict way (unknown fields will just be ignored) and the config file type is automatically determined from extension (supports yaml, toml and json), if you want to change this, add the FromConfigFile(...) option after Defaults(...)
// cliFlag string : cli flag that holds the config path
// defaultPath string: if the cli flag is not defined, used this as the default path to load config from
func Defaults(cliFlag, defaultPath string) OptionFunc {
	return func(c *options) error {

		// Process in config file -> env -> cli order
		errs := []error{}
		err := FromConfigFileFlagPath(cliFlag, defaultPath, "config file path", Auto)(c)
		if err != nil {
			errs = append(errs, err)
		}

		err = FromEnvs(ENVDelimiter)(c)
		if err != nil {
			errs = append(errs, err)

		}
		err = FromCli(CLIDelimiter)(c)
		if err != nil {
			errs = append(errs, err)

		}
		WithLogLevel(slog.LevelError)

		return errors.Join(errs...)
	}
}

// DefaultsFromPath adds all three configuration determination sources.
// The configs struct will be configured config file -> envs -> cli, so that cli takes precedence over more static options, for ease of user configuration.
// The config file will be parsed in a non-strict way (unknown fields will just be ignored) and the config file type is automatically determined from extension (supports yaml, toml and json), if you want to change this, add the FromConfigFile(...) option after Defaults(...)
// path string : config file path
func DefaultsFromPath(path string) OptionFunc {
	return func(c *options) error {

		// Process in config file -> env -> cli order
		errs := []error{}

		err := FromConfigFile(path, Auto)(c)
		if err != nil {
			errs = append(errs, err)
		}
		err = FromEnvs(ENVDelimiter)(c)
		if err != nil {
			errs = append(errs, err)
		}
		err = FromCli(CLIDelimiter)(c)
		if err != nil {
			errs = append(errs, err)
		}
		WithLogLevel(slog.LevelError)

		return errors.Join(errs...)
	}
}

// FromConfigFile tells confy to load a config file from path
// path: string config file path
// configType: ConfigType, what type the config file is expected to be, use `Auto` if you dont care and just want it to choose for you. Supports yaml, toml and json
func FromConfigFile(path string, configType ConfigType) OptionFunc {
	return func(c *options) error {
		if c.currentlySet[configFile] {
			return errors.New("duplicate configuration information source, " + string(configFile) + " FromConfig* option set twice, mutually exclusive")
		}
		c.currentlySet[configFile] = true

		c.config.dataMethod = func() (io.Reader, ConfigType, error) {
			configData, err := os.Open(path)
			if err != nil {
				return nil, "", err
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

		c.order = append(c.order, configFile)

		return nil
	}
}

// FromConfigBytes tells confy to load a config file from raw bytes
// data: []byte config file raw bytes
// configType: ConfigType, what type the config bytes are supports yaml, toml and json, the Auto configuration will return an error
func FromConfigBytes(data []byte, configType ConfigType) OptionFunc {
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

		c.order = append(c.order, configFile)

		return nil
	}
}

// FromConfigURL tells confy to load file from a url (http/https)
// url: string url of configuration file
// configType: ConfigType, what type the config file is expected to be, use `Auto` if you dont care and just want it to choose for you. Supports yaml, toml and json
func FromConfigURL(urlOpt string, configType ConfigType) OptionFunc {
	return func(c *options) error {
		if c.currentlySet[configFile] {
			return errors.New("duplicate configuration information source, " + string(configFile) + " FromConfig* option set twice, mutually exclusive")
		}
		c.currentlySet[configFile] = true

		c.config.dataMethod = func() (io.Reader, ConfigType, error) {

			u, err := url.Parse(urlOpt)
			if err != nil {
				return nil, configType, err
			}

			client := http.Client{
				Timeout: 20 * time.Second,
			}

			resp, err := client.Get(urlOpt)
			if err != nil {
				return nil, configType, fmt.Errorf("failed to get config from url: %s, err: %s", urlOpt, err)
			}

			if resp.StatusCode < 200 || resp.StatusCode > 299 {
				resp.Body.Close()
				return nil, configType, fmt.Errorf("status code was not okay: %s", resp.Status)
			}

			var fileType ConfigType
			if configType == Auto {
				ext := strings.ToLower(filepath.Ext(u.Path))
				switch ext {
				case ".yml", ".yaml":
					logger.Info("yaml chosen as config type from extension", "url_path", u.Path)

					fileType = Yaml
				case ".json", ".js":
					logger.Info("json chosen as config type from extension", "url_path", u.Path)

					fileType = Json
				case ".toml", ".tml":
					logger.Info("toml chosen as config type from extension", "url_path", u.Path)

					fileType = Toml
				default:
					logger.Info("no extension in url, using content type instead")
					contentType := resp.Header.Get("content-type")
					switch contentType {
					case "application/yaml", "application/x-yaml", "text/yaml":
						fileType = Yaml
					case "application/json":
						fileType = Json
					case "text/x-toml", "application/toml", "text/toml":
						fileType = Toml
					default:
						resp.Body.Close()
						return nil, configType, fmt.Errorf("could not automatically determine config format from extension %q or content-type %q", ext, contentType)

					}

				}
			}

			return resp.Body, fileType, err
		}

		c.order = append(c.order, configFile)
		return nil
	}
}

// FromConfigFileFlagPath tells confy to load file from path as specified by cli flag
// cliFlagName: string cli option that defines config filepath
// configType: ConfigType, what type the config file is expected to be, use `Auto` if you dont care and just want it to choose for you. Supports yaml, toml and json
func FromConfigFileFlagPath(cliFlagName, defaultPath, description string, configType ConfigType) OptionFunc {
	return func(c *options) error {

		if c.cli.commandLine == nil {
			c.cli.commandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		}

		configPath := c.cli.commandLine.String(cliFlagName, defaultPath, description)
		if err := c.cli.commandLine.Parse(os.Args[1:]); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return flag.ErrHelp
			}

			// We will get a lot of random "flag not defined" errors,as our flags are defined much later (if at all) in the Cli component
			configPath = &defaultPath
		}

		return FromConfigFile(*configPath, configType)(c)
	}
}

func WithNonStructCliFlag() OptionFunc {
	return func(o *options) error {
		return nil
	}
}

// WithStrictParsing parse config files in a strict way, do not allow unknown fields
func WithStrictParsing() OptionFunc {
	return func(c *options) error {
		c.config.strictParsing = true
		return nil
	}
}

// WithConfigRequired causes failure to load the configuration from file/bytes/url to become fatal rather than just warning
func WithConfigRequired() OptionFunc {
	return func(c *options) error {
		c.config.required = true
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
func FromEnvs(delimiter string) OptionFunc {
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
func WithEnvTransform(t Transform) OptionFunc {
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
func FromCli(delimiter string) OptionFunc {
	return func(c *options) error {

		if c.cli.commandLine == nil {
			c.cli.commandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		}

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
func WithCliTransform(t Transform) OptionFunc {
	return func(c *options) error {
		if t == nil {
			logger.Warn("WithCliTranform was used, but transform was nil")
		}

		c.cli.transform = t
		return nil
	}
}
