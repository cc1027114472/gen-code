package xhttpclient

import "time"

const (
	DefaultWorkers            = 32
	DefaultQueueSize          = 1000
	DefaultTimeout            = 10 * time.Second
	DefaultRetry              = 1
	DefaultMaxRequestLogBody  = 2048
	DefaultMaxResponseLogBody = 2048
)

// Config ?? HTTP ?????????
type Config struct {
	Workers            int
	QueueSize          int
	Timeout            time.Duration
	Retry              int
	MaxRequestLogBody  int
	MaxResponseLogBody int
}

// DefaultConfig ????????????
func DefaultConfig() Config {
	return Config{
		Workers:            DefaultWorkers,
		QueueSize:          DefaultQueueSize,
		Timeout:            DefaultTimeout,
		Retry:              DefaultRetry,
		MaxRequestLogBody:  DefaultMaxRequestLogBody,
		MaxResponseLogBody: DefaultMaxResponseLogBody,
	}
}

// Normalize ???????????
func (c Config) Normalize() Config {
	cfg := c
	defaults := DefaultConfig()

	if cfg.Workers <= 0 {
		cfg.Workers = defaults.Workers
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = defaults.QueueSize
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaults.Timeout
	}
	if cfg.Retry < 0 {
		cfg.Retry = defaults.Retry
	}
	if cfg.MaxRequestLogBody <= 0 {
		cfg.MaxRequestLogBody = defaults.MaxRequestLogBody
	}
	if cfg.MaxResponseLogBody <= 0 {
		cfg.MaxResponseLogBody = defaults.MaxResponseLogBody
	}

	return cfg
}
