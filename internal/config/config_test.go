package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadFromEnv(t *testing.T) {
	clearAllConfigEnv(t)
	t.Setenv("APP_NAME", "llm-trace")
	t.Setenv("APP_ENV", "dev")
	t.Setenv("APP_PORT", "8080")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "15s")
	t.Setenv("LOG_LEVEL", "info")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "llm-trace", cfg.App.Name)
	require.Equal(t, "dev", cfg.App.Env)
	require.Equal(t, 8080, cfg.App.Port)
	require.Equal(t, 15*time.Second, cfg.App.ShutdownTimeout)
	require.Equal(t, "info", cfg.Log.Level)
	require.True(t, cfg.Log.HTTPAccess)
	require.Equal(t, []string{"127.0.0.1"}, cfg.App.TrustedProxies)
	require.False(t, cfg.App.Debug)
}

func TestLoadAppliesDefaultsAndFlags(t *testing.T) {
	clearAllConfigEnv(t)
	t.Setenv("APP_DEBUG", "true")
	t.Setenv("LOG_HTTP_ACCESS", "false")
	t.Setenv("APP_TRUSTED_PROXIES", "127.0.0.1, 10.0.0.1")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "llm-trace", cfg.App.Name)
	require.Equal(t, "dev", cfg.App.Env)
	require.Equal(t, 10008, cfg.App.Port)
	require.Equal(t, 10*time.Second, cfg.App.ShutdownTimeout)
	require.True(t, cfg.App.Debug)
	require.Equal(t, "debug", cfg.Log.Level)
	require.False(t, cfg.Log.HTTPAccess)
	require.Equal(t, []string{"127.0.0.1", "10.0.0.1"}, cfg.App.TrustedProxies)
}

func TestLoadReturnsErrorWhenIntEnvMalformed(t *testing.T) {
	clearAllConfigEnv(t)
	t.Setenv("APP_PORT", "not-a-number")

	_, err := Load()
	require.EqualError(t, err, "parse APP_PORT: strconv.Atoi: parsing \"not-a-number\": invalid syntax")
}

func TestLoadReturnsErrorWhenDurationEnvMalformed(t *testing.T) {
	clearAllConfigEnv(t)
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "later")

	_, err := Load()
	require.EqualError(t, err, "parse APP_SHUTDOWN_TIMEOUT: time: invalid duration \"later\"")
}

func TestLoadReturnsErrorWhenDurationEnvNotPositive(t *testing.T) {
	clearAllConfigEnv(t)
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "0s")

	_, err := Load()
	require.EqualError(t, err, "APP_SHUTDOWN_TIMEOUT must be greater than zero")
}

func clearAllConfigEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"APP_NAME",
		"APP_ENV",
		"APP_PORT",
		"APP_DEBUG",
		"APP_SHUTDOWN_TIMEOUT",
		"APP_TRUSTED_PROXIES",
		"LOG_LEVEL",
		"LOG_HTTP_ACCESS",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}
