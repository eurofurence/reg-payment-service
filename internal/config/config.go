package config

import (
	"crypto/rsa"
	"errors"
	"io"

	"gopkg.in/yaml.v3"
)

type (
	DatabaseType string
	LogStyle     string
)

const (
	Mysql    DatabaseType = "mysql"
	Inmemory DatabaseType = "inmemory"

	Plain LogStyle = "plain"
	ECS   LogStyle = "ecs" // default
)

var appConfig *Application

type (
	// Application is the root configuration type
	// that holds all other subconfiguration types
	Application struct {
		Service  ServiceConfig  `yaml:"service"`
		Server   ServerConfig   `yaml:"server"`
		Database DatabaseConfig `yaml:"database"`
		Security SecurityConfig `yaml:"security"`
		Logging  LoggingConfig  `yaml:"logging"`
	}

	// ServiceConfig contains configuration values
	// for service related tasks. E.g. URL to payment provider adapter
	ServiceConfig struct {
		Name                string   `yaml:"name"`
		AttendeeService     string   `yaml:"attendee_service"`
		ProviderAdapter     string   `yaml:"provider_adapter"`
		TransactionIDPrefix string   `yaml:"transaction_id_prefix"`
		AllowedCurrencies   []string `yaml:"allowed_currencies"`
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

	// DatabaseConfig configures which db to use (mysql, inmemory)
	// and how to connect to it (needed for mysql only)
	DatabaseConfig struct {
		Use        DatabaseType `yaml:"use"`
		Username   string       `yaml:"username"`
		Password   string       `yaml:"password"`
		Database   string       `yaml:"database"`
		Parameters []string     `yaml:"parameters"`
	}

	// SecurityConfig configures everything related to security
	SecurityConfig struct {
		Fixed        FixedTokenConfig    `yaml:"fixed_token"`
		Oidc         OpenIdConnectConfig `yaml:"oidc"`
		Cors         CorsConfig          `yaml:"cors"`
		RequireLogin bool                `yaml:"require_login_for_reg"`
	}

	FixedTokenConfig struct {
		Api string `yaml:"api"` // shared-secret for server-to-server backend authentication
	}

	OpenIdConnectConfig struct {
		TokenCookieName    string   `yaml:"token_cookie_name"`     // optional, if set, the jwt token is also read from this cookie (useful for mixed web application setups, see reg-auth-service)
		TokenPublicKeysPEM []string `yaml:"token_public_keys_PEM"` // a list of public RSA keys in PEM format, see https://github.com/Jumpy-Squirrel/jwks2pem for obtaining PEM from openid keyset endpoint
		UserInfoURL        string   `yaml:"user_info_url"`         // validation of admin accesses uses this endpoint to verify the token is still current and access has not been recently revoked
		AdminRole          string   `yaml:"admin_role"`            // the role/group claim that supplies admin rights
	}

	CorsConfig struct {
		DisableCors bool   `yaml:"disable"`
		AllowOrigin string `yaml:"allow_origin"`
	}

	// LoggingConfig configures logging
	LoggingConfig struct {
		Style    LogStyle `yaml:"style"`
		Severity string   `yaml:"severity"`
	}
)

var parsedKeySet []*rsa.PublicKey

func OidcKeySet() []*rsa.PublicKey {
	return parsedKeySet
}

// UnmarshalFromYamlConfiguration decodes yaml data from an `io.Reader` interface.
func UnmarshalFromYamlConfiguration(file io.Reader) (*Application, error) {
	d := yaml.NewDecoder(file)
	d.KnownFields(true) // strict

	var conf Application

	if err := d.Decode(&conf); err != nil {
		return nil, err
	}

	appConfig = &conf
	return &conf, nil
}

func GetApplicationConfig() (*Application, error) {
	if appConfig == nil {
		return nil, errors.New("config was not yet loaded")
	}

	return appConfig, nil
}
