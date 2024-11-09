# Confy

`confy` is your one stop shop for configuration, it allows configuration from multiple sources including command-line arguments, environment variables, and configuration files in various formats (`JSON`, `YAML`, `TOML`).

## Features

- **Multiple Configuration Sources**: Populate settings from *CLI flags*, *environment variables*, and *config files*.
- **Automatic CLI Flag and Env Variable Mapping**: Fields are mapped automatically and can be renamed with the `confy:""` tag
- **Support for YAML, JSON, and TOML**: Automatically detect and parse configuration file format, no added effort to switch to a new format

## Installation

```sh
go get github.com/NHAS/confy
```

## Usage

### Tags
- `confy:"field_name;sensitive"`: Customize field names for env variables, CLI flags, and config files.
- `confy_description:"Field Description here"`: Set field descriptions for CLI parsing and help messages.

### Basic Examples

`config.json`:
```json
{
  "Database": {
    "Host": "localhost",
    "Port": 5432,
    "User": "dbuser",
    "Password": "securepassword"
  },
  "new_name": "some_value"
}
```

`config.yaml`:
```yaml
Database:
  Host: "localhost"
  Port: 5432
  User: "dbuser"
  Password: "securepassword"

new_name: "some_value"
```


```go
package main

import (
	"fmt"
	"github.com/NHAS/confy"
)

type Config struct {
	Database struct {
		Host     string
		Port     int
		User     string
		Password string
	}

    Renamed string `confy:"new_name"`
}

func main() {
    // Defaults will also load from env and cli!
	loadedConfig, _, err := confy.Config[Config](confy.Defaults("config.json"))
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

	fmt.Printf("Loaded JSON config: %+v\n", loadedConfig)

    yamlLoadedConfig, _, err := confy.Config[Config](confy.Defaults("config.yaml"))
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

    fmt.Printf("Loaded YAML config: %+v\n", yamlLoadedConfig)

    // They're the same!
    
}
```

Output
```sh
Loaded config: {Database:{Host:localhost Port:5432 User:dbuser Password:securepassword} Renamed:some_value}
Loaded config: {Database:{Host:localhost Port:5432 User:dbuser Password:securepassword} Renamed:some_value}
```


### Environment Variables
For struct:
```go
package main

import (
    "fmt"
    "github.com/NHAS/confy"
)

type Config struct {
    Server struct {
        Host string
    }
}

func main() {
	populatedConfig, _, err := confy.Config[Config](confy.FromEnvs(confy.DefaultENVDelimiter))
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

    fmt.Println(populatedConfig.Server.Host)
}
```
Expected environment variable:

```sh
export Server_Host="localhost"
$ ./test
localhost
```

### CLI Flags only
For struct:
```go
package main

import (
    "fmt"
    "github.com/NHAS/confy"
)

type Config struct {
    Database struct {
        Port int
    }
}

func main() {
	populatedConfig, _, err := confy.Config[Config](confy.FromCli(confy.DefaultCliDelimiter))
	if err != nil {
		fmt.Println("Error loading config:", err)
		return
	}

    fmt.Println(populatedConfig.Database.Port)
}

```
Expected CLI flag:
```sh
$ ./app -Database.Port=5432
5432
```


## Logging

Confy has logging capabilities using the `slog` package. Use the `WithLogLevel` option to adjust verbosity or disable logging.

## Configuration Options

Confy offers a variety of options for configuring your application's settings.

| Option | Description |
|----------|-------------|
| `Defaults(...)` | Loads configurations in the order: config file -> environment variables -> CLI flags. This sets a non-strict parsing mode for unknown fields in the config file. |
| `FromConfigFile(...)` | Load configuration from a file. Supports `YAML`, `JSON`, and `TOML`. |
| `FromConfigBytes(...)` | Load configuration from raw bytes, ideal for embedding configuration in code. |
| `FromEnvs(...)` | Load configuration from environment variables. Use the delimiter to denote nested fields. |
| `FromCli(...)` | Load configuration from CLI flags. Set a delimiter for nested struct parsing. |
| `WithLogLevel(...)` | Set logging level to control output verbosity. Useful for debugging. |
| `WithCliTransform(...)` | Takes a function to run against the generated CLI flag name, allows you to modify the flag name |
| `WithEnvTransform(...)` | Takes a function to run against the generated ENV variable name, allows you to modify the ENV name |

## Notes
- Complex structures must implement `encoding.TextUnmarshaler` and `encoding.TextMarshaler` for CLI/ENV parsing.
- CLI flags and environment variables use the delimiters (`.` for CLI, `_` for ENV by default) when handling nested fields.



