package mcp

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManagerInvokeRunsRegisteredSyntheticTool(t *testing.T) {
	manager := NewManager([]ServerDescriptor{{
		ID:            "synthetic",
		Source:        "runtime.synthetic",
		Enabled:       true,
		ToolCount:     2,
		ResourceCount: 0,
		Status:        "enabled",
	}})
	manager.RegisterExecutor("synthetic", "echo", func(arguments map[string]any) (InvocationResult, error) {
		return InvocationResult{
			Content: fmt.Sprintf("echo %v", arguments["message"]),
			Data:    map[string]any{"arguments": arguments},
		}, nil
	})

	result, err := manager.Invoke(InvocationRequest{
		ServerID: "synthetic",
		ToolName: "echo",
		Arguments: map[string]any{
			"message": "hello",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "synthetic", result.ServerID)
	require.Equal(t, "echo", result.ToolName)
	require.Equal(t, "echo hello", result.Content)
	require.Equal(t, "hello", result.Data["arguments"].(map[string]any)["message"])
}

func TestManagerInvokeRejectsUnknownServerOrTool(t *testing.T) {
	manager := NewManager([]ServerDescriptor{{
		ID:            "synthetic",
		Source:        "runtime.synthetic",
		Enabled:       true,
		ToolCount:     1,
		ResourceCount: 0,
		Status:        "enabled",
	}})
	manager.RegisterExecutor("synthetic", "echo", func(arguments map[string]any) (InvocationResult, error) {
		return InvocationResult{Content: "ok"}, nil
	})

	_, err := manager.Invoke(InvocationRequest{ServerID: "missing", ToolName: "echo"})
	require.EqualError(t, err, "mcp server missing not found")

	_, err = manager.Invoke(InvocationRequest{ServerID: "synthetic", ToolName: "sum"})
	require.EqualError(t, err, "mcp tool synthetic/sum not found")
}
