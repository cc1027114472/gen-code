package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunPrintsUsageWithoutArgs(t *testing.T) {
	output := captureOutput(t, func() {
		err := run(context.Background(), nil)
		require.NoError(t, err)
	})

	require.Contains(t, output, "gen-code commands:")
	require.Contains(t, output, "runtime status")
	require.Contains(t, output, "workspace show")
	require.Contains(t, output, "threads list")
}

func TestFallbackText(t *testing.T) {
	require.Equal(t, "fallback", fallbackText("", "fallback"))
	require.Equal(t, "fallback", fallbackText("   ", "fallback"))
	require.Equal(t, "value", fallbackText("value", "fallback"))
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	reader, writer, err := os.Pipe()
	require.NoError(t, err)

	os.Stdout = writer
	defer func() {
		os.Stdout = originalStdout
	}()

	fn()

	require.NoError(t, writer.Close())

	var buffer bytes.Buffer
	_, err = io.Copy(&buffer, reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())

	return buffer.String()
}
