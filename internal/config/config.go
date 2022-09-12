// In general, configs are loaded via yaml files, that will be placed on the server.
// Therefore implement handling to load yaml from a file and validate the config based on rules
// which apply for the corresponding service implementation.

package config

import (
	"os"
	// "gopkg.in/yaml.v2"
)

type (
	Application struct {
		Service ServiceConfig `yaml:"serviceConfig"`
		IsCorsDisabled bool `yaml:"corsDisabled"`
	}

	ServiceConfig struct {
		ServiceName string `yaml:"serviceName"`
		Port        int    `env:"servicePort"`
	}
)

func UnmarshalFromYamlConfiguration(file *os.File) (*Application, error) {

	// Load configuration from a yaml file

	// example:
	// defer file.Close()
	// d := yaml.NewDecoder(file)
	//
	// config := &Config{}
	// if err := d.Decode(&config); err != nil {
	// 		return nil, err
	// }

	// return config, nil

	return nil, nil
}
