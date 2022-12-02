package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalConfig(t *testing.T) {
	s := []byte(`service:
  name: 'TestServiceName' 
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
`)

	b := bytes.NewBuffer(s)

	conf, err := UnmarshalFromYamlConfiguration(b)
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
