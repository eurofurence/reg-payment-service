package config

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"net/url"
	"regexp"
	"sort"
)

func Validate(conf *Application, logFunc func(format string, v ...interface{})) error {
	errs := url.Values{}
	validateServiceConfiguration(errs, conf.Service)
	validateServerConfiguration(errs, conf.Server)
	validateDatabaseConfiguration(errs, conf.Database)
	validateSecurityConfiguration(errs, conf.Security)
	validateLoggingConfiguration(errs, conf.Logging)

	if len(errs) > 0 {
		logValidationErrorDetails(errs, logFunc)
		return errors.New("configuration values failed to validate, bailing out")
	}

	return nil
}

const downstreamPattern = "^https?://.*[^/]$"

func validateServiceConfiguration(errs url.Values, c ServiceConfig) {
	if violatesPattern(downstreamPattern, c.AttendeeService) {
		errs.Add("service.attendee_service", "base url must start with http:// or https:// and may not end in a /")
	}
	if violatesPattern(downstreamPattern, c.ProviderAdapter) {
		errs.Add("service.provider_adapter", "base url must start with http:// or https:// and may not end in a /")
	}
}

func validateServerConfiguration(errs url.Values, c ServerConfig) {
	checkIntValueRange(errs, 1, 65535, "server.port", c.Port)
	checkIntValueRange(errs, 1, 300, "server.read_timeout_seconds", c.ReadTimeout)
	checkIntValueRange(errs, 1, 300, "server.write_timeout_seconds", c.WriteTimeout)
	checkIntValueRange(errs, 1, 300, "server.idle_timeout_seconds", c.IdleTimeout)
}

func validateSecurityConfiguration(errs url.Values, c SecurityConfig) {
	checkLength(&errs, 16, 256, "security.fixed_token.api", c.Fixed.Api)
	checkLength(&errs, 1, 256, "security.oidc.admin_role", c.Oidc.AdminRole)

	parsedKeySet = make([]*rsa.PublicKey, 0)
	for i, keyStr := range c.Oidc.TokenPublicKeysPEM {
		publicKeyPtr, err := jwt.ParseRSAPublicKeyFromPEM([]byte(keyStr))
		if err != nil {
			errs.Add(fmt.Sprintf("security.oidc.token_public_keys_PEM[%d]", i), fmt.Sprintf("failed to parse RSA public key in PEM format: %s", err.Error()))
		} else {
			parsedKeySet = append(parsedKeySet, publicKeyPtr)
		}
	}
}

var allowedDatabases = []DatabaseType{Mysql, Inmemory}

func validateDatabaseConfiguration(errs url.Values, c DatabaseConfig) {
	if notInAllowedValues(allowedDatabases[:], c.Use) {
		errs.Add("database.use", "must be one of mysql, inmemory")
	}
	if c.Use == Mysql {
		checkLength(&errs, 1, 256, "database.username", c.Username)
		checkLength(&errs, 1, 256, "database.password", c.Password)
		checkLength(&errs, 1, 256, "database.database", c.Database)
	}
}

var allowedSeverities = []string{"DEBUG", "INFO", "WARN", "ERROR"}

func validateLoggingConfiguration(errs url.Values, c LoggingConfig) {
	if notInAllowedValues(allowedSeverities[:], c.Severity) {
		errs.Add("logging.severity", "must be one of DEBUG, INFO, WARN, ERROR")
	}
}

func violatesPattern(pattern string, value string) bool {
	matched, err := regexp.MatchString(pattern, value)
	if err != nil {
		return true
	}
	return !matched
}

func checkLength(errs *url.Values, min int, max int, key string, value string) {
	if len(value) < min || len(value) > max {
		errs.Add(key, fmt.Sprintf("%s field must be at least %d and at most %d characters long", key, min, max))
	}
}

func checkIntValueRange(errs url.Values, min int, max int, key string, value int) {
	if value < min || value > max {
		errs.Add(key, fmt.Sprintf("%s field must be an integer at least %d and at most %d", key, min, max))
	}
}

func notInAllowedValues[T comparable](allowed []T, value T) bool {
	return !sliceContains(allowed, value)
}

func sliceContains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}

func logValidationErrorDetails(errs url.Values, logFunc func(format string, v ...interface{})) {
	var keys []string
	for key := range errs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, k := range keys {
		key := k
		val := errs[k]
		logFunc("configuration error: %s: %s", key, val[0])
	}
}
