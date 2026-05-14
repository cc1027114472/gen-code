package xhttpclient

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestDefaultConfigProvidesExpectedDefaults ????????????
func TestDefaultConfigProvidesExpectedDefaults(t *testing.T) {
	cfg := DefaultConfig()

	require.Equal(t, DefaultWorkers, cfg.Workers)
	require.Equal(t, DefaultQueueSize, cfg.QueueSize)
	require.Equal(t, DefaultTimeout, cfg.Timeout)
	require.Equal(t, DefaultRetry, cfg.Retry)
	require.Equal(t, DefaultMaxRequestLogBody, cfg.MaxRequestLogBody)
	require.Equal(t, DefaultMaxResponseLogBody, cfg.MaxResponseLogBody)
}

// TestConfigNormalizeFillsInvalidValues ????????????
func TestConfigNormalizeFillsInvalidValues(t *testing.T) {
	cfg := Config{
		Workers:            0,
		QueueSize:          -1,
		Timeout:            0,
		Retry:              -1,
		MaxRequestLogBody:  0,
		MaxResponseLogBody: -5,
	}.Normalize()

	require.Equal(t, DefaultWorkers, cfg.Workers)
	require.Equal(t, DefaultQueueSize, cfg.QueueSize)
	require.Equal(t, 10*time.Second, cfg.Timeout)
	require.Equal(t, DefaultRetry, cfg.Retry)
	require.Equal(t, DefaultMaxRequestLogBody, cfg.MaxRequestLogBody)
	require.Equal(t, DefaultMaxResponseLogBody, cfg.MaxResponseLogBody)
}
