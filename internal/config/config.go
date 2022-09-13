package config

import (
	"io"

	"gopkg.in/yaml.v3"
)

type (
	// Application is the root configuration type
	// that holds all other subconfiguration types
	Application struct {
		Service        ServiceConfig `yaml:"serviceConfig"`
		Server         ServerConfig  `yaml:"serverConfig"`
		IsCorsDisabled bool          `yaml:"corsDisabled"`
	}

	// ServerConfig contains all values for
	// http releated configuration
	ServerConfig struct {
		BaseAddress  string `yaml:"baseAddress"`
		Port         int    `yaml:"port"`
		ReadTimeout  int    `yaml:"readTimeoutSeconds"`
		WriteTimeout int    `yaml:"writeTimeoutSeconds"`
		IdleTimeout  int    `yaml:"idleTimeoutSeconds"`
	}

	// ServiceConfig contains configuration values
	// for service related tasks. E.g. URL to payment provider adapter
	ServiceConfig struct {
		ServiceName string `yaml:"name"`
	}

	// DatabaseConfig contains all required configuration
	// values for database related tasks
	DatabaseConfig struct {
		// TODO
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
