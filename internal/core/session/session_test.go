package session

import (
	"testing"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/state"
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

	first.MessageHistory = append(first.MessageHistory, MessageRecord{ID: "message-1"})
	first.ToolHistory = append(first.ToolHistory, ToolCallRecord{ID: "toolcall-1"})
	first.ArtifactPaths = append(first.ArtifactPaths, ArtifactRecord{ID: "artifact-1"})

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

func TestThreadContextAppendAndRestore(t *testing.T) {
	registry := NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(CreateThreadInput{Name: "Context"})

	message, err := registry.AppendMessage(thread.ID, AppendMessageInput{Role: "user", Content: "Hello"})
	require.NoError(t, err)
	require.Equal(t, "user", message.Role)

	toolCall, err := registry.AppendToolCall(thread.ID, AppendToolCallInput{ToolID: "bridge.check", Status: "completed", Summary: "ok"})
	require.NoError(t, err)
	require.Equal(t, "bridge.check", toolCall.ToolID)

	artifact, err := registry.AppendArtifact(thread.ID, AppendArtifactInput{Path: `D:\artifacts\hello.md`, Kind: "markdown"})
	require.NoError(t, err)
	require.Equal(t, "markdown", artifact.Kind)

	flag, err := registry.SetRuntimeFlag(thread.ID, SetRuntimeFlagInput{Key: "preview", Value: "ready"})
	require.NoError(t, err)
	require.Equal(t, "ready", flag.Value)

	reloaded, ok := registry.Thread(thread.ID)
	require.True(t, ok)
	require.Equal(t, 1, reloaded.MessageHistoryCount)
	require.Equal(t, 1, reloaded.ToolCallCount)
	require.Equal(t, 1, reloaded.ArtifactCount)

	flags, ok := registry.RuntimeFlags(thread.ID)
	require.True(t, ok)
	require.Len(t, flags, 1)
	require.Equal(t, "preview", flags[0].Key)
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

func TestTaskStatusUpdatesRemainThreadScoped(t *testing.T) {
	registry := NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)

	first := registry.CreateThread(CreateThreadInput{Name: "First"})
	second := registry.CreateThread(CreateThreadInput{Name: "Second"})

	firstTask, ok := registry.CreateTask(first.ID, CreateTaskInput{Title: "Draft spec"})
	require.True(t, ok)
	_, ok = registry.CreateTask(second.ID, CreateTaskInput{Title: "Review UI"})
	require.True(t, ok)

	updated, err := registry.UpdateTaskStatus(first.ID, firstTask.ID, UpdateTaskStatusInput{Status: "running"})
	require.NoError(t, err)
	require.Equal(t, "running", updated.Status)
	require.False(t, updated.UpdatedAt.IsZero())

	firstTasks, ok := registry.Tasks(first.ID)
	require.True(t, ok)
	require.Equal(t, "running", firstTasks[0].Status)

	secondTasks, ok := registry.Tasks(second.ID)
	require.True(t, ok)
	require.Equal(t, "queued", secondTasks[0].Status)
}

func TestUpdateTaskStatusRejectsInvalidStatus(t *testing.T) {
	registry := NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(CreateThreadInput{Name: "First"})
	task, ok := registry.CreateTask(thread.ID, CreateTaskInput{Title: "Draft spec"})
	require.True(t, ok)

	_, err := registry.UpdateTaskStatus(thread.ID, task.ID, UpdateTaskStatusInput{Status: "paused"})
	require.ErrorIs(t, err, ErrInvalidTaskStatus)
}

func TestRegistryRestoresAndPersistsViaSQLiteStore(t *testing.T) {
	projectRoot := t.TempDir()
	store, err := state.Open(projectRoot)
	require.NoError(t, err)

	registry, err := NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)
	require.Equal(t, state.StoreName, registry.StateStoreName())
	require.Equal(t, state.PathForProject(projectRoot), registry.StatePath())

	thread := registry.CreateThread(CreateThreadInput{Name: "First"})
	task, ok := registry.CreateTask(thread.ID, CreateTaskInput{Title: "Draft spec"})
	require.True(t, ok)

	updated, err := registry.UpdateTaskStatus(thread.ID, task.ID, UpdateTaskStatusInput{Status: "completed"})
	require.NoError(t, err)
	require.Equal(t, "completed", updated.Status)
	_, err = registry.AppendMessage(thread.ID, AppendMessageInput{Role: "user", Content: "persist this"})
	require.NoError(t, err)
	_, err = registry.AppendToolCall(thread.ID, AppendToolCallInput{ToolID: "bridge.check", Status: "completed", Summary: "ok"})
	require.NoError(t, err)
	_, err = registry.AppendArtifact(thread.ID, AppendArtifactInput{Path: `D:\artifacts\persist.md`, Kind: "markdown"})
	require.NoError(t, err)
	_, err = registry.SetRuntimeFlag(thread.ID, SetRuntimeFlagInput{Key: "draft", Value: "saved"})
	require.NoError(t, err)

	require.NoError(t, store.Close())
	store, err = state.Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	reloaded, err := NewRegistryWithStore(projectRoot, store)
	require.NoError(t, err)

	threads := reloaded.Threads()
	require.Len(t, threads, 1)
	require.Equal(t, "First", threads[0].Name)
	require.Equal(t, thread.ID, reloaded.ActiveThreadID())

	tasks, ok := reloaded.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 1)
	require.Equal(t, "completed", tasks[0].Status)

	messages, ok := reloaded.Messages(thread.ID)
	require.True(t, ok)
	require.Len(t, messages, 1)
	require.Equal(t, "persist this", messages[0].Content)

	toolCalls, ok := reloaded.ToolCalls(thread.ID)
	require.True(t, ok)
	require.Len(t, toolCalls, 1)
	require.Equal(t, "bridge.check", toolCalls[0].ToolID)

	artifacts, ok := reloaded.Artifacts(thread.ID)
	require.True(t, ok)
	require.Len(t, artifacts, 1)
	require.Equal(t, "markdown", artifacts[0].Kind)

	flags, ok := reloaded.RuntimeFlags(thread.ID)
	require.True(t, ok)
	require.Len(t, flags, 1)
	require.Equal(t, "saved", flags[0].Value)

	events, ok := reloaded.Events(thread.ID)
	require.True(t, ok)
	require.Len(t, events, 7)
}
