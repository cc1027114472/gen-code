package config

import (
	"os"
	"path/filepath"
	"strings"
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
	require.False(t, cfg.Providers.Anthropic.Enabled)
	require.False(t, cfg.Providers.OpenAI.Enabled)
	require.False(t, cfg.Providers.Gemini.Enabled)
}

func TestLoadProvidersFromAnthropicEnv(t *testing.T) {
	clearAllConfigEnv(t)
	t.Setenv("MODEL_PROVIDER", "anthropic")
	t.Setenv("ANTHROPIC_BASE_URL", "http://localhost:1314")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "test-token")
	t.Setenv("ANTHROPIC_MODEL", "gpt-5.4-A")
	t.Setenv("ANTHROPIC_DEFAULT_HAIKU_MODEL", "gpt-5.4-A")
	t.Setenv("ANTHROPIC_DEFAULT_SONNET_MODEL", "gpt-5.4-A")
	t.Setenv("ANTHROPIC_DEFAULT_OPUS_MODEL", "gpt-5.4-A")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "anthropic", cfg.Providers.DefaultProvider)
	require.True(t, cfg.Providers.Anthropic.Enabled)
	require.Equal(t, "http://localhost:1314", cfg.Providers.Anthropic.BaseURL)
	require.Equal(t, "test-token", cfg.Providers.Anthropic.AuthToken)
	require.Equal(t, "gpt-5.4-A", cfg.Providers.Anthropic.Models.Default)
	require.Equal(t, "gpt-5.4-A", cfg.Providers.Anthropic.Models.Haiku)
	require.Equal(t, "gpt-5.4-A", cfg.Providers.Anthropic.Models.Sonnet)
	require.Equal(t, "gpt-5.4-A", cfg.Providers.Anthropic.Models.Opus)
}

func TestLoadProvidersRespectsExplicitDisable(t *testing.T) {
	clearAllConfigEnv(t)
	t.Setenv("ANTHROPIC_ENABLED", "false")
	t.Setenv("ANTHROPIC_BASE_URL", "http://localhost:1314")
	t.Setenv("ANTHROPIC_AUTH_TOKEN", "test-token")
	t.Setenv("ANTHROPIC_MODEL", "gpt-5.4-A")

	cfg, err := Load()
	require.NoError(t, err)
	require.False(t, cfg.Providers.Anthropic.Enabled)
	require.Equal(t, "http://localhost:1314", cfg.Providers.Anthropic.BaseURL)
}

func TestLoadReturnsErrorWhenProviderBoolMalformed(t *testing.T) {
	clearAllConfigEnv(t)
	t.Setenv("ANTHROPIC_ENABLED", "sometimes")

	_, err := Load()
	require.EqualError(t, err, "parse ANTHROPIC_ENABLED: strconv.ParseBool: parsing \"sometimes\": invalid syntax")
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

func TestLoadReadsDotEnvFromParentDirectory(t *testing.T) {
	originalValues := captureEnv(
		"MODEL_PROVIDER",
		"ANTHROPIC_ENABLED",
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
	)
	for key := range originalValues {
		_ = os.Unsetenv(key)
	}
	defer restoreEnv(originalValues)

	root := t.TempDir()
	err := os.WriteFile(filepath.Join(root, ".env"), []byte(strings.Join([]string{
		"MODEL_PROVIDER=anthropic",
		"ANTHROPIC_BASE_URL=http://localhost:1314",
		"ANTHROPIC_AUTH_TOKEN=test-token",
		"ANTHROPIC_MODEL=gpt-5.4-A",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL=gpt-5.4-A",
		"ANTHROPIC_DEFAULT_SONNET_MODEL=gpt-5.4-A",
		"ANTHROPIC_DEFAULT_OPUS_MODEL=gpt-5.4-A",
	}, "\n")), 0o600)
	require.NoError(t, err)

	nested := filepath.Join(root, "cmd", "cli")
	require.NoError(t, os.MkdirAll(nested, 0o755))

	originalWD, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(nested))
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "anthropic", cfg.Providers.DefaultProvider)
	require.True(t, cfg.Providers.Anthropic.Enabled)
	require.Equal(t, "http://localhost:1314", cfg.Providers.Anthropic.BaseURL)
	require.Equal(t, "test-token", cfg.Providers.Anthropic.AuthToken)
	require.Equal(t, "gpt-5.4-A", cfg.Providers.Anthropic.Models.Default)
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
		"MODEL_PROVIDER",
		"LLM_PROVIDER",
		"PROVIDER_DEFAULT",
		"ANTHROPIC_ENABLED",
		"ANTHROPIC_BASE_URL",
		"ANTHROPIC_AUTH_TOKEN",
		"ANTHROPIC_API_KEY",
		"ANTHROPIC_MODEL",
		"ANTHROPIC_DEFAULT_HAIKU_MODEL",
		"ANTHROPIC_DEFAULT_SONNET_MODEL",
		"ANTHROPIC_DEFAULT_OPUS_MODEL",
		"OPENAI_ENABLED",
		"OPENAI_BASE_URL",
		"OPENAI_AUTH_TOKEN",
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
		"OPENAI_DEFAULT_MINI_MODEL",
		"OPENAI_DEFAULT_MODEL",
		"OPENAI_DEFAULT_REASONING_MODEL",
		"GEMINI_ENABLED",
		"GEMINI_BASE_URL",
		"GEMINI_AUTH_TOKEN",
		"GEMINI_API_KEY",
		"GEMINI_MODEL",
		"GEMINI_DEFAULT_FLASH_MODEL",
		"GEMINI_DEFAULT_PRO_MODEL",
		"GEMINI_DEFAULT_ULTRA_MODEL",
	}

	for _, key := range keys {
		t.Setenv(key, "")
	}
}

func captureEnv(keys ...string) map[string]*string {
	values := make(map[string]*string, len(keys))
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			copied := value
			values[key] = &copied
			continue
		}
		values[key] = nil
	}
	return values
}

func restoreEnv(values map[string]*string) {
	for key, value := range values {
		if value == nil {
			_ = os.Unsetenv(key)
			continue
		}
		_ = os.Setenv(key, *value)
	}
}
