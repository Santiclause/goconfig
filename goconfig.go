package goconfig

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"

	"github.com/caarlos0/env"
	"gopkg.in/yaml.v2"
)

const (
	DebugError   = iota
	DebugWarning = iota
	DebugInfo    = iota
	DebugVerbose = iota
)

var (
	debugLevelMap = map[string]int{
		"error":   DebugError,
		"warning": DebugWarning,
		"info":    DebugInfo,
		"verbose": DebugVerbose,
	}
)

// Config will contain the config loaded from the yaml file.
type Config struct {
	Debug string `yaml:"debug" env:"DEBUG"`
	// filename is the path and filename of the yaml config file.
	filename  string
	listening bool
	// Mutex guards readwrite access to Config.
	sync.Mutex
}

func (c *Config) GetFilename() string {
	return c.filename
}

func (c *Config) SetFilename(filename string) {
	c.filename = filename
}

func (c *Config) IsListening() bool {
	return c.listening
}

func (c *Config) SetListening(listening bool) {
	c.listening = listening
}

func (c *Config) DebugLevel(level string) bool {
	return debugLevelMap[c.Debug] >= debugLevelMap[level]
}

type Configterface interface {
	GetFilename() string
	IsListening() bool
	SetListening(bool)
	Lock()
	Unlock()
}

type MissingRequiredStructFields struct {
	missing []string
}

func (e MissingRequiredStructFields) Error() string {
	return fmt.Sprintf("The following struct fields have missing values: %s", strings.Trim(fmt.Sprintf("%v", e.missing), "[]"))
}

// Loads (or reloads) the config file from disk.
func Load(c Configterface) error {
	if reflect.ValueOf(c).Kind() != reflect.Ptr {
		panic("Load only accepts pointers to structs")
	}
	data, err := ioutil.ReadFile(c.GetFilename())
	c.Lock()
	defer c.Unlock()
	if err == nil {
		if err := yaml.Unmarshal(data, c); err != nil {
			return err
		}
	}
	if err := env.Parse(c); err != nil {
		return err
	}
	if err := findMissingRequiredFields(c); err != nil {
		return err
	}
	return nil
}

// Reloads the config file on SIGHUP.
func ListenForSignals(c Configterface) {
	if reflect.ValueOf(c).Kind() != reflect.Ptr {
		panic("ListenForSignals only accepts pointers to structs")
	}
	c.Lock()
	defer c.Unlock()
	if c.IsListening() {
		return
	}
	c.SetListening(true)
	s := make(chan os.Signal, 1)
	signal.Notify(s, syscall.SIGHUP)
	go func() {
		for {
			<-s
			if err := Load(c); err != nil {
				panic(fmt.Sprintf("config file error: %s", err))
			}
		}
	}()
}

func findMissingRequiredFields(val interface{}) error {
	var missing []string
	value := reflect.ValueOf(val)
	for {
		switch value.Kind() {
		case reflect.Struct:
			for i := 0; i < value.NumField(); i++ {
				tag := value.Type().Field(i).Tag
				name := value.Type().Field(i).Name
				field := value.Field(i)
				if tag.Get("required") == "true" && isZero(field) {
					missing = append(missing, name)
				}
			}
			if missing != nil {
				return MissingRequiredStructFields{missing}
			}
			return nil
		case reflect.Ptr:
			if value.IsNil() {
				return errors.New("nil pointer!")
			}
			value = reflect.Indirect(value)
		default:
			return errors.New("Not a struct!")
		}
	}
}

// Shamelessly stolen from the 2nd answer of https://stackoverflow.com/questions/23555241/golang-reflection-how-to-get-zero-value-of-a-field-type
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Func, reflect.Map, reflect.Slice:
		return v.IsNil()
	case reflect.Array:
		z := true
		for i := 0; i < v.Len(); i++ {
			z = z && isZero(v.Index(i))
		}
		return z
	case reflect.Struct:
		z := true
		for i := 0; i < v.NumField(); i++ {
			if v.Field(i).CanSet() {
				z = z && isZero(v.Field(i))
			}
		}
		return z
	case reflect.Ptr:
		// Modified: we don't need to indirect the pointer.
		// If the pointer is set, but points at a zero value, that's fine -
		// we only care that it was set at all. This allows explicit empty values (e.g. "")
		return v.IsNil()
	}
	// Compare other types directly:
	z := reflect.Zero(v.Type())
	result := v.Interface() == z.Interface()

	return result
}
