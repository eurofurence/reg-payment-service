package config

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalConfig(t *testing.T) {
	s := []byte(`service:
  name: 'TestServiceName'
  attendee_service: 'http://localhost:9091'
  provider_adapter: 'http://localhost:9097'
server:
  port: 8080
  read_timeout_seconds: 30
  write_timeout_seconds: 40
  idle_timeout_seconds: 120
database:
  use: inmemory
security:
  fixed_token:
    api: 'some-api-token-must-be-long-enough'
  oidc:
    token_cookie_name: 'JWT'
    user_info_url: 'http://localhost/userinfo'
    admin_role: 'admin'
  cors:
    disable: true
    allow_origin: 'http://localhost:8000,http://localhost:8001'
logging:
  severity: INFO
`)

	b := bytes.NewBuffer(s)

	conf, err := UnmarshalFromYamlConfiguration(b)
	require.NoError(t, err)

	logRecording := strings.Builder{}
	logFunc := func(format string, v ...interface{}) {
		logRecording.WriteString(fmt.Sprintf(format, v...))
		logRecording.WriteString("\n")
	}
	err = Validate(conf, logFunc)
	require.Equal(t, "", logRecording.String())
	require.NoError(t, err)

	require.NotNil(t, conf)
	require.Equal(t, "TestServiceName", conf.Service.Name)
	require.Equal(t, "", conf.Server.BaseAddress)
	require.Equal(t, 8080, conf.Server.Port)
	require.Equal(t, 30, conf.Server.ReadTimeout)
	require.Equal(t, 40, conf.Server.WriteTimeout)
	require.Equal(t, 120, conf.Server.IdleTimeout)
	require.Equal(t, Inmemory, conf.Database.Use)
	require.Equal(t, "some-api-token-must-be-long-enough", conf.Security.Fixed.Api)
	require.Equal(t, "JWT", conf.Security.Oidc.TokenCookieName)
	require.Equal(t, "http://localhost/userinfo", conf.Security.Oidc.UserInfoURL)
	require.Equal(t, "admin", conf.Security.Oidc.AdminRole)
	require.True(t, conf.Security.Cors.DisableCors)
	require.Equal(t, "http://localhost:8000,http://localhost:8001", conf.Security.Cors.AllowOrigin)
}

func TestUnmarshalConfigInvalid(t *testing.T) {
	s := []byte(`---
service:
    name: 'TestServiceName' 
server:
port: 8080
read_timeout_seconds: 30
        write_timeout_seconds: 30
idle_timeout_seconds: 120
    cors_disabled: true
`)

	b := bytes.NewBuffer(s)

	conf, err := UnmarshalFromYamlConfiguration(b)
	require.Error(t, err)

	require.Nil(t, conf)
}

func TestUnmarshalUnknownFields(t *testing.T) {
	s := []byte(`service:
  name: 'TestServiceName' 
sucurity_with_typo_we_want_to_detect:
  fixed_token:
    api: 'some-api-token-must-be-long-enough'
  oidc:
    token_cookie_name: 'JWT'
    user_info_url: 'http://localhost/userinfo'
    admin_role: 'admin'
  cors:
    disable: true
    allow_origin: 'http://localhost:8000,http://localhost:8001'
`)

	b := bytes.NewBuffer(s)

	conf, err := UnmarshalFromYamlConfiguration(b)
	require.Error(t, err)
	require.Contains(t, err.Error(), "sucurity_with_typo_we_want_to_detect")

	require.Nil(t, conf)
}

func TestValidationErrors1(t *testing.T) {
	s := []byte(`service:
  name: 'TestServiceName'
  attendee_service: 'kittycat'
server:
  port: -77
  read_timeout_seconds: 0
  write_timeout_seconds: 8127368
  idle_timeout_seconds: -70
database:
  use: papyrus
security:
  fixed_token:
    api: 'too-short'
  oidc:
    token_cookie_name: 'JWT'
    token_public_keys_PEM:
      - |
        -----BEGIN PUBLIC KEY-----
        MIIBIjANBgkqhkiG9w
        -----END PUBLIC KEY-----
  cors:
    disable: true
    allow_origin: 'http://localhost:8000,http://localhost:8001'
logging:
  severity: CAT
`)

	b := bytes.NewBuffer(s)

	conf, err := UnmarshalFromYamlConfiguration(b)
	require.NoError(t, err)

	logRecording := strings.Builder{}
	logFunc := func(format string, v ...interface{}) {
		logRecording.WriteString(fmt.Sprintf(format, v...))
		logRecording.WriteString("\n")
	}
	err = Validate(conf, logFunc)

	expected := `configuration error: database.use: must be one of mysql, inmemory
configuration error: logging.severity: must be one of DEBUG, INFO, WARN, ERROR
configuration error: security.fixed_token.api: security.fixed_token.api field must be at least 16 and at most 256 characters long
configuration error: security.oidc.admin_role: security.oidc.admin_role field must be at least 1 and at most 256 characters long
configuration error: security.oidc.token_public_keys_PEM[0]: failed to parse RSA public key in PEM format: invalid key: Key must be a PEM encoded PKCS1 or PKCS8 key
configuration error: server.idle_timeout_seconds: server.idle_timeout_seconds field must be an integer at least 1 and at most 300
configuration error: server.port: server.port field must be an integer at least 1 and at most 65535
configuration error: server.read_timeout_seconds: server.read_timeout_seconds field must be an integer at least 1 and at most 300
configuration error: server.write_timeout_seconds: server.write_timeout_seconds field must be an integer at least 1 and at most 300
configuration error: service.attendee_service: base url must start with http:// or https:// and may not end in a /
configuration error: service.provider_adapter: base url must start with http:// or https:// and may not end in a /
`
	require.Equal(t, expected, logRecording.String())
	require.Error(t, err)
}
