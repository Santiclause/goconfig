Golang Configuration
====================

This package just makes it simpler to deal with configuration. Just define
a struct with the appropriate tags, and either embed the config.Config struct
type or manually implement the interface (I recommend the former), and then
you're essentially good to go. Observe:

```go
import (
    "github.com/santiclause/goconfig"
)
type Config struct {
	HttpPort            int           `yaml:"http_port" env:"HTTP_PORT"`
	ConnTimeout         time.Duration `yaml:"conn_timeout" env:"CONN_TIMEOUT" required:"true"`
	RequestGracePeriod  time.Duration `yaml:"request_grace_period" env:"REQUEST_GRACE_PERIOD"`
	goconfig.Config
}

func main() {
    // Make the struct, with a default value (since Load will overwrite it)
    config := &Config{
        HttpPort: 8080,
    }
    config.SetFilename("whatever.yaml")
    goconfig.Load(&config)
    // Optionally make it listen for SIGHUPs
    goconfig.ListenForSignals(&config)
}
```


Supported tags
--------------

* `env`: see https://github.com/caarlos0/env
* `yaml`: see https://github.com/go-yaml/yaml
* `required`: if this has a value of "true", Load will return an error if that
struct field has a zero value after parsing the yaml and environment variables.


Priority
--------

The provided yaml file will be loaded first (if it exists), and environment
variables will override the yaml files.


Debug level
-----------

The Config struct type also defines a debug level and reads it from yaml: `debug`
and env: `DEBUG` respectively, and you can check if the debug level is at least
at a certain level via `config.DebugLevel(level)` - e.g.:

```go
if config.DebugLevel("info") {
    log.Println("New connection from client")
}
```
