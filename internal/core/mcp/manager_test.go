package mcp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewManagerNormalizesMetadataStatuses(t *testing.T) {
	manager := NewManager([]ServerDescriptor{
		{
			ID:            "filesystem",
			Source:        "node_modules",
			Enabled:       true,
			ToolCount:     2,
			ResourceCount: 1,
			Status:        "enabled",
		},
		{
			ID:            "proxy",
			Source:        "node_modules",
			Enabled:       true,
			ToolCount:     0,
			ResourceCount: 0,
		},
	})

	items := manager.List()
	require.Len(t, items, 2)
	require.Equal(t, "enabled", items[0].Status)
	require.Equal(t, "degraded", items[1].Status)
}

func TestMetadataHealthSummaryUsesNormalizedDescriptor(t *testing.T) {
	summary := MetadataHealthSummary(ServerDescriptor{
		ID:            "memory",
		Source:        "builtin",
		Enabled:       false,
		ToolCount:     1,
		ResourceCount: 1,
		Status:        "enabled",
	})

	require.Equal(t, "memory (disabled, metadata health: disabled)", summary)
}

func TestManagerInvokeExecutesFixtureServer(t *testing.T) {
	manager := NewManager([]ServerDescriptor{
		fixtureServerDescriptor(t, "external-fixture", "echo"),
	})

	result, err := manager.Invoke(context.Background(), InvokeRequest{
		ServerID: "external-fixture",
		ToolName: "echo",
		Arguments: map[string]any{
			"message": "hello",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "external-fixture", result.ServerID)
	require.Equal(t, "echo", result.ToolName)
	require.Equal(t, "stdio-fixture", result.Transport)
	require.Equal(t, "mcp tool external-fixture/echo executed", result.ResultSummary)
	require.Equal(t, "hello", result.Result["echo"])
}

func TestManagerInvokeRejectsUnknownServer(t *testing.T) {
	manager := NewManager(nil)

	_, err := manager.Invoke(context.Background(), InvokeRequest{
		ServerID: "missing",
		ToolName: "echo",
	})
	require.ErrorIs(t, err, ErrServerNotFound)
	require.Contains(t, err.Error(), "missing")
}

func TestManagerInvokeRejectsUnknownTool(t *testing.T) {
	manager := NewManager([]ServerDescriptor{
		fixtureServerDescriptor(t, "external-fixture", "echo"),
	})

	_, err := manager.Invoke(context.Background(), InvokeRequest{
		ServerID: "external-fixture",
		ToolName: "sum",
	})
	require.ErrorIs(t, err, ErrToolNotFound)
	require.Contains(t, err.Error(), "external-fixture/sum")
}

func TestManagerInvokePropagatesExternalFailure(t *testing.T) {
	manager := NewManager([]ServerDescriptor{
		fixtureServerDescriptor(t, "external-fixture", "fail"),
	})

	_, err := manager.Invoke(context.Background(), InvokeRequest{
		ServerID: "external-fixture",
		ToolName: "fail",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "external mcp server external-fixture failed")
	require.Contains(t, err.Error(), "fixture forced failure")
}

func fixtureServerDescriptor(t *testing.T, serverID string, tools ...string) ServerDescriptor {
	t.Helper()

	scriptPath := filepath.Join(repoRoot(t), "scripts", "mcp_stdio_fixture.py")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("fixture server script missing: %v", err)
	}
	return ServerDescriptor{
		ID:            serverID,
		Source:        "fixture",
		Enabled:       true,
		ToolCount:     len(tools),
		ResourceCount: 0,
		Status:        "enabled",
		Command:       fixtureCommand(scriptPath),
		Tools:         tools,
	}
}

func fixtureCommand(scriptPath string) []string {
	if runtime.GOOS == "windows" {
		return []string{"python", scriptPath}
	}
	return []string{"python3", scriptPath}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
