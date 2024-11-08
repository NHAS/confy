package confy

import "fmt"

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

	order []preference
}

func Config[T any](config T, suppliedOptions ...Option) (result T, err error) {

	o := options{}
	for _, optFunc := range suppliedOptions {
		err := optFunc(&o)
		if err != nil {
			return result, err
		}
	}

	if len(o.order) == 0 {
		return result, fmt.Errorf("no configuration sources specified (no options given to Config() )")
	}

	orderLoadOpts := map[preference]loader[T]{
		cli:        loadCli[T],
		env:        nil,
		configFile: loadConfig[T],
	}

	for _, p := range o.order {

		f, ok := orderLoadOpts[p]
		if !ok {
			panic("unknown preference option: " + p)
		}

		if f == nil {
			continue
		}

		err := f(o, &result)
		if err != nil {
			return result, fmt.Errorf("%s: %s", p, err)
		}
	}

	return
}

func Defaults(path string) Option {
	return func(c *options) error {

		FromCli(".")(c)
		FromEnvs("_")(c)
		FromConfigFile(path, false, Auto)(c)

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
