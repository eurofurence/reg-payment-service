package config

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalConfig(t *testing.T) {
	s := []byte(`service:
  name: "TestServiceName" 
server:
  port: 8080
  read_timeout_seconds: 30
  write_timeout_seconds: 30
  idle_timeout_seconds: 120
cors_disabled: true
`)

	b := bytes.NewBuffer(s)

	conf, err := UnmarshalFromYamlConfiguration(b)
	require.NoError(t, err)

	require.NotNil(t, conf)
	require.Equal(t, "TestServiceName", conf.Service.ServiceName)
	require.Equal(t, "", conf.Server.BaseAddress)
	require.Equal(t, 8080, conf.Server.Port)
	require.Equal(t, 30, conf.Server.ReadTimeout)
	require.Equal(t, 30, conf.Server.WriteTimeout)
	require.Equal(t, 120, conf.Server.IdleTimeout)
	require.True(t, conf.IsCorsDisabled)

	// TODO Test db config

}

func TestUnmarshalConfigInvalid(t *testing.T) {
	s := []byte(`---
service:
    name: "TestServiceName" 
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
