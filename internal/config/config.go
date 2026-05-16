package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config contains the runtime configuration required by the service.
type Config struct {
	App       AppConfig
	Log       LogConfig
	Providers ProvidersConfig
}

// AppConfig defines the HTTP service runtime settings.
type AppConfig struct {
	Name            string
	Env             string
	Port            int
	Debug           bool
	ShutdownTimeout time.Duration
	TrustedProxies  []string
}

// LogConfig controls application logging behavior.
type LogConfig struct {
	Level      string
	HTTPAccess bool
}

// ProvidersConfig defines model provider configuration loaded from env.
type ProvidersConfig struct {
	DefaultProvider string
	Anthropic       ProviderConfig
	OpenAI          ProviderConfig
	Gemini          ProviderConfig
}

// ProviderConfig contains shared provider connection settings and model aliases.
type ProviderConfig struct {
	Enabled   bool
	BaseURL   string
	AuthToken string
	Models    ProviderModels
}

// ProviderModels stores a provider's preferred model aliases.
type ProviderModels struct {
	Default string
	Haiku   string
	Sonnet  string
	Opus    string
}

// Load reads configuration from the environment and optional .env file.
func Load() (Config, error) {
	loadDotEnv()

	appDebug, err := optionalBool(false, "APP_DEBUG")
	if err != nil {
		return Config{}, err
	}

	appName := optionalString("llm-trace", "APP_NAME")
	appEnv := optionalString("dev", "APP_ENV")
	appTrustedProxies := optionalCSV("127.0.0.1", "APP_TRUSTED_PROXIES")

	port, err := optionalInt(10008, "APP_PORT")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := optionalPositiveDuration(10*time.Second, "APP_SHUTDOWN_TIMEOUT")
	if err != nil {
		return Config{}, err
	}

	defaultLogLevel := "info"
	if appDebug {
		defaultLogLevel = "debug"
	}
	logLevel := optionalString(defaultLogLevel, "LOG_LEVEL")

	logHTTPAccess, err := optionalBool(true, "LOG_HTTP_ACCESS")
	if err != nil {
		return Config{}, err
	}

	providers, err := loadProviders()
	if err != nil {
		return Config{}, err
	}

	return Config{
		App: AppConfig{
			Name:            appName,
			Env:             appEnv,
			Port:            port,
			Debug:           appDebug,
			ShutdownTimeout: shutdownTimeout,
			TrustedProxies:  appTrustedProxies,
		},
		Log: LogConfig{
			Level:      logLevel,
			HTTPAccess: logHTTPAccess,
		},
		Providers: providers,
	}, nil
}

func loadProviders() (ProvidersConfig, error) {
	anthropic, err := loadProviderConfig(
		[]string{"ANTHROPIC_ENABLED"},
		[]string{"ANTHROPIC_BASE_URL"},
		[]string{"ANTHROPIC_AUTH_TOKEN", "ANTHROPIC_API_KEY"},
		ProviderModels{
			Default: optionalString("", "ANTHROPIC_MODEL"),
			Haiku:   optionalString("", "ANTHROPIC_DEFAULT_HAIKU_MODEL"),
			Sonnet:  optionalString("", "ANTHROPIC_DEFAULT_SONNET_MODEL"),
			Opus:    optionalString("", "ANTHROPIC_DEFAULT_OPUS_MODEL"),
		},
	)
	if err != nil {
		return ProvidersConfig{}, err
	}

	openAI, err := loadProviderConfig(
		[]string{"OPENAI_ENABLED"},
		[]string{"OPENAI_BASE_URL"},
		[]string{"OPENAI_AUTH_TOKEN", "OPENAI_API_KEY"},
		ProviderModels{
			Default: optionalString("", "OPENAI_MODEL"),
			Haiku:   optionalString("", "OPENAI_DEFAULT_MINI_MODEL"),
			Sonnet:  optionalString("", "OPENAI_DEFAULT_MODEL"),
			Opus:    optionalString("", "OPENAI_DEFAULT_REASONING_MODEL"),
		},
	)
	if err != nil {
		return ProvidersConfig{}, err
	}

	gemini, err := loadProviderConfig(
		[]string{"GEMINI_ENABLED"},
		[]string{"GEMINI_BASE_URL"},
		[]string{"GEMINI_AUTH_TOKEN", "GEMINI_API_KEY"},
		ProviderModels{
			Default: optionalString("", "GEMINI_MODEL"),
			Haiku:   optionalString("", "GEMINI_DEFAULT_FLASH_MODEL"),
			Sonnet:  optionalString("", "GEMINI_DEFAULT_PRO_MODEL"),
			Opus:    optionalString("", "GEMINI_DEFAULT_ULTRA_MODEL"),
		},
	)
	if err != nil {
		return ProvidersConfig{}, err
	}

	return ProvidersConfig{
		DefaultProvider: optionalString("", "MODEL_PROVIDER", "LLM_PROVIDER", "PROVIDER_DEFAULT"),
		Anthropic:       anthropic,
		OpenAI:          openAI,
		Gemini:          gemini,
	}, nil
}

func loadProviderConfig(enabledKeys, baseURLKeys, authTokenKeys []string, models ProviderModels) (ProviderConfig, error) {
	enabled, hasExplicitEnabled, err := optionalBoolWithPresence(enabledKeys...)
	if err != nil {
		return ProviderConfig{}, err
	}

	cfg := ProviderConfig{
		BaseURL:   optionalString("", baseURLKeys...),
		AuthToken: optionalString("", authTokenKeys...),
		Models:    models,
	}

	if hasExplicitEnabled {
		cfg.Enabled = enabled
		return cfg, nil
	}

	cfg.Enabled = cfg.BaseURL != "" || cfg.AuthToken != "" || cfg.Models.Default != "" || cfg.Models.Haiku != "" || cfg.Models.Sonnet != "" || cfg.Models.Opus != ""
	return cfg, nil
}

// requiredString reads the first non-empty value from the provided env keys.
func requiredString(keys ...string) (string, error) {
	for _, key := range keys {
		value := os.Getenv(key)
		if value != "" {
			return value, nil
		}
	}

	return "", fmt.Errorf("missing required env: %s", strings.Join(keys, " or "))
}

// requiredInt parses a required integer environment variable.
func requiredInt(keys ...string) (int, error) {
	value, err := requiredString(keys...)
	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", keys[0], err)
	}

	return n, nil
}

// optionalString returns the first non-empty env value or the default.
func optionalString(defaultValue string, keys ...string) string {
	for _, key := range keys {
		value := os.Getenv(key)
		if value != "" {
			return value
		}
	}

	return defaultValue
}

// optionalInt parses an optional integer environment variable.
func optionalInt(defaultValue int, keys ...string) (int, error) {
	for _, key := range keys {
		value := os.Getenv(key)
		if value == "" {
			continue
		}

		n, err := strconv.Atoi(value)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", key, err)
		}

		return n, nil
	}

	return defaultValue, nil
}

// optionalBool parses an optional boolean environment variable.
func optionalBool(defaultValue bool, keys ...string) (bool, error) {
	for _, key := range keys {
		value := os.Getenv(key)
		if value == "" {
			continue
		}

		b, err := strconv.ParseBool(value)
		if err != nil {
			return false, fmt.Errorf("parse %s: %w", key, err)
		}

		return b, nil
	}

	return defaultValue, nil
}

// optionalBoolWithPresence parses an optional boolean env var and reports whether any key was set.
func optionalBoolWithPresence(keys ...string) (bool, bool, error) {
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		if !ok || value == "" {
			continue
		}

		b, err := strconv.ParseBool(value)
		if err != nil {
			return false, true, fmt.Errorf("parse %s: %w", key, err)
		}

		return b, true, nil
	}

	return false, false, nil
}

// optionalDuration parses an optional duration environment variable.
func optionalDuration(defaultValue time.Duration, keys ...string) (time.Duration, error) {
	for _, key := range keys {
		value := os.Getenv(key)
		if value == "" {
			continue
		}

		d, err := time.ParseDuration(value)
		if err != nil {
			return 0, fmt.Errorf("parse %s: %w", key, err)
		}

		return d, nil
	}

	return defaultValue, nil
}

// optionalPositiveInt ensures the resolved integer is greater than zero.
func optionalPositiveInt(defaultValue int, keys ...string) (int, error) {
	value, err := optionalInt(defaultValue, keys...)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", keys[0])
	}

	return value, nil
}

// optionalPositiveDuration ensures the resolved duration is greater than zero.
func optionalPositiveDuration(defaultValue time.Duration, keys ...string) (time.Duration, error) {
	value, err := optionalDuration(defaultValue, keys...)
	if err != nil {
		return 0, err
	}
	if value <= 0 {
		return 0, fmt.Errorf("%s must be greater than zero", keys[0])
	}

	return value, nil
}

// optionalCSV parses a comma-separated env var into a string slice.
func optionalCSV(defaultValue string, keys ...string) []string {
	value := optionalString(defaultValue, keys...)
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		result = append(result, item)
	}

	return result
}

// loadDotEnv supplements env vars from a local .env file when present.
func loadDotEnv() {
	path := filepath.Join(".", ".env")
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			continue
		}

		if _, exists := os.LookupEnv(key); exists {
			continue
		}

		_ = os.Setenv(key, value)
	}
}
