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
	App AppConfig
	Log LogConfig
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
	}, nil
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
