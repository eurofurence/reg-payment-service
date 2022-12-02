package config

import (
	"io"

	"gopkg.in/yaml.v3"
)

type (
	DatabaseType string
)

const (
	Mysql    DatabaseType = "mysql"
	Inmemory DatabaseType = "inmemory"
)

type (
	// Application is the root configuration type
	// that holds all other subconfiguration types
	Application struct {
		Service        ServiceConfig  `yaml:"service"`
		Server         ServerConfig   `yaml:"server"`
		Database       DatabaseConfig `yaml:"database"`
		IsCorsDisabled bool           `yaml:"cors_disabled"`
	}

	// ServerConfig contains all values for
	// http releated configuration
	ServerConfig struct {
		BaseAddress  string `yaml:"address"`
		Port         int    `yaml:"port"`
		ReadTimeout  int    `yaml:"read_timeout_seconds"`
		WriteTimeout int    `yaml:"write_timeout_seconds"`
		IdleTimeout  int    `yaml:"idle_timeout_seconds"`
	}

	// ServiceConfig contains configuration values
	// for service related tasks. E.g. URL to payment provider adapter
	ServiceConfig struct {
		ServiceName  string `yaml:"name"`
		ServiceToken string `yaml:"token"`
	}

	// DatabaseConfig contains all required configuration
	// values for database related tasks
	DatabaseConfig struct {
		Use        DatabaseType `yaml:"use"`
		Username   string       `yaml:"username"`
		Password   string       `yaml:"password"`
		Database   string       `yaml:"database"`
		Parameters []string     `yaml:"parameters"`
	}
)

// UnmarshalFromYamlConfiguration decodes yaml data from an `io.Reader` interface.
func UnmarshalFromYamlConfiguration(file io.Reader) (*Application, error) {
	d := yaml.NewDecoder(file)

	var conf Application

	if err := d.Decode(&conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
