package confy

import (
	"errors"
	"flag"
	"log/slog"
	"math"
	"os"
)

type Option func(*options) error

type options struct {
	config struct {
		strictParsing bool
		path          string
		fileType      ConfigType
	}

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

func Config[T any](config T, suppliedOptions ...Option) (result T, warnings []error, err error) {

	o := options{
		level: new(slog.LevelVar),
	}

	// disable logging by default
	o.level.Set(math.MaxInt)

	o.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: o.level,
	}))

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

func WithLogLevel(level slog.Level) Option {
	return func(c *options) error {
		c.level.Set(level)
		return nil
	}
}

func Defaults(path string) Option {
	return func(c *options) error {

		FromCli(".")(c)
		FromEnvs("_")(c)
		FromConfigFile(path, false, Auto)(c)

		WithLogLevel(slog.LevelError)

		return nil
	}
}

// Sets tells confy to load a config file from path
func FromConfigFile(path string, strictParsing bool, configType ConfigType) Option {
	return func(c *options) error {
		c.config.path = path
		c.config.fileType = configType
		c.config.strictParsing = strictParsing

		c.order = append(c.order, configFile)

		return nil
	}
}

// Confy will automatically look for environment variables
func FromEnvs(delimiter string) Option {
	return func(c *options) error {
		c.cli.delimiter = delimiter
		c.order = append(c.order, env)
		return nil
	}
}

// Confy will automatically look for variables from cli flags
func FromCli(delimiter string) Option {
	return func(c *options) error {
		c.cli.delimiter = delimiter
		c.order = append(c.order, cli)
		return nil
	}
}
