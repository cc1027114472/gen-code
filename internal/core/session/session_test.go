package session

import (
	"testing"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/core/policy"
)

func TestNewRegistryCreatesSingleWorkspace(t *testing.T) {
	registry := NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)

	workspace := registry.Workspace()
	require.Equal(t, "gen-code", workspace.ID)
	require.Equal(t, `D:\GOWorks\gen-code-heji\gen-code`, workspace.ProjectRoot)
	require.Equal(t, `D:\GOWorks\gen-code-heji\gen-code\docs`, workspace.SharedDocsRoot)
	require.Equal(t, 0, workspace.ActiveThreadCount)
}

func TestCreateThreadAttachesToWorkspaceWithDefaultMode(t *testing.T) {
	registry := NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)

	thread := registry.CreateThread(CreateThreadInput{})

	require.Equal(t, "thread-1", thread.ID)
	require.Equal(t, "gen-code", thread.WorkspaceID)
	require.Equal(t, "Thread 1", thread.Name)
	require.Equal(t, policy.AskUser, thread.PermissionMode)
	require.True(t, thread.IsActive)
	require.Equal(t, 1, registry.Workspace().ActiveThreadCount)
}

func TestThreadsRemainIsolated(t *testing.T) {
	registry := NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)

	first := registry.CreateThread(CreateThreadInput{Name: "First"})
	second := registry.CreateThread(CreateThreadInput{Name: "Second"})

	first.MessageHistory = append(first.MessageHistory, "hello")
	first.ToolHistory = append(first.ToolHistory, "bridge.check")
	first.ArtifactPaths = append(first.ArtifactPaths, "a.txt")

	gotFirst, ok := registry.Thread(first.ID)
	require.True(t, ok)
	require.Equal(t, 0, gotFirst.MessageHistoryCount)
	require.Equal(t, 0, gotFirst.ToolCallCount)
	require.Equal(t, 0, gotFirst.ArtifactCount)

	gotSecond, ok := registry.Thread(second.ID)
	require.True(t, ok)
	require.Equal(t, 0, gotSecond.MessageHistoryCount)
	require.Equal(t, 0, gotSecond.ToolCallCount)
	require.Equal(t, 0, gotSecond.ArtifactCount)
}

func TestActivateThreadOnlySwitchesPointer(t *testing.T) {
	registry := NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)

	first := registry.CreateThread(CreateThreadInput{Name: "First"})
	second := registry.CreateThread(CreateThreadInput{Name: "Second"})

	require.Equal(t, first.ID, registry.ActiveThreadID())

	activated, ok := registry.ActivateThread(second.ID)
	require.True(t, ok)
	require.Equal(t, second.ID, activated.ID)
	require.True(t, activated.IsActive)
	require.Equal(t, second.ID, registry.ActiveThreadID())

	reloadedFirst, ok := registry.Thread(first.ID)
	require.True(t, ok)
	require.False(t, reloadedFirst.IsActive)
}
