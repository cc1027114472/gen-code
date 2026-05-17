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

func TestManagerInvokeExecutesSDKServer(t *testing.T) {
	manager := NewManager([]ServerDescriptor{
		sdkServerDescriptor(t, "sdk-external-fixture", "echo"),
	})

	result, err := manager.Invoke(context.Background(), InvokeRequest{
		ServerID: "sdk-external-fixture",
		ToolName: "echo",
		Arguments: map[string]any{
			"message": "hello-sdk",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "sdk-external-fixture", result.ServerID)
	require.Equal(t, "echo", result.ToolName)
	require.Equal(t, "stdio-sdk", result.Transport)
	require.Equal(t, "mcp tool sdk-external-fixture/echo executed", result.ResultSummary)
	require.Equal(t, "hello-sdk", result.Result["echo"])
}

func TestManagerInvokeExecutesThirdPartyTimeServer(t *testing.T) {
	manager := NewManager([]ServerDescriptor{
		thirdPartyTimeDescriptor(t, "third-party-time", "get_current_time"),
	})

	result, err := manager.Invoke(context.Background(), InvokeRequest{
		ServerID: "third-party-time",
		ToolName: "get_current_time",
		Arguments: map[string]any{
			"timezone": "UTC",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "third-party-time", result.ServerID)
	require.Equal(t, "get_current_time", result.ToolName)
	require.Equal(t, "stdio-third-party", result.Transport)
	require.Equal(t, "mcp tool third-party-time/get_current_time executed", result.ResultSummary)
	require.Equal(t, "UTC", result.Result["timezone"])
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

func TestManagerInvokeRejectsInvalidJSONResponse(t *testing.T) {
	manager := NewManager([]ServerDescriptor{{
		ID:            "invalid-json",
		Source:        "fixture",
		Enabled:       true,
		ToolCount:     1,
		ResourceCount: 0,
		Status:        "enabled",
		Command:       invalidJSONCommand(t),
		Tools:         []string{"echo"},
		Transport:     "stdio-fixture",
	}})

	_, err := manager.Invoke(context.Background(), InvokeRequest{
		ServerID: "invalid-json",
		ToolName: "echo",
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "returned invalid json")
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
		Transport:     "stdio-fixture",
	}
}

func sdkServerDescriptor(t *testing.T, serverID string, tools ...string) ServerDescriptor {
	t.Helper()

	scriptPath := filepath.Join(repoRoot(t), "scripts", "mcp_sdk_server.js")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("sdk server script missing: %v", err)
	}
	return ServerDescriptor{
		ID:            serverID,
		Source:        "sdk",
		Enabled:       true,
		ToolCount:     len(tools),
		ResourceCount: 0,
		Status:        "enabled",
		Command:       []string{"node", scriptPath},
		Tools:         tools,
		Transport:     "stdio-sdk",
	}
}

func thirdPartyTimeDescriptor(t *testing.T, serverID string, tools ...string) ServerDescriptor {
	t.Helper()

	scriptPath := filepath.Join(repoRoot(t), "scripts", "mcp_third_party_time_server.js")
	if _, err := os.Stat(scriptPath); err != nil {
		t.Fatalf("third-party time server script missing: %v", err)
	}
	return ServerDescriptor{
		ID:            serverID,
		Source:        "third-party",
		Enabled:       true,
		ToolCount:     len(tools),
		ResourceCount: 0,
		Status:        "enabled",
		Command:       []string{"node", scriptPath},
		Tools:         tools,
		Transport:     "stdio-third-party",
	}
}

func fixtureCommand(scriptPath string) []string {
	if runtime.GOOS == "windows" {
		return []string{"python", scriptPath}
	}
	return []string{"python3", scriptPath}
}

func invalidJSONCommand(t *testing.T) []string {
	t.Helper()

	scriptPath := filepath.Join(t.TempDir(), "invalid-json.js")
	content := `process.stdout.write("not-json");`
	require.NoError(t, os.WriteFile(scriptPath, []byte(content), 0o644))
	return []string{"node", scriptPath}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}
