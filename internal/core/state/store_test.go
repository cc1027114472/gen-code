package state

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestStorePersistsSnapshotTables(t *testing.T) {
	projectRoot := t.TempDir()

	store, err := Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, store.Close()) }()
	require.Equal(t, PathForProject(projectRoot), store.Path())

	createdAt := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(5 * time.Minute)

	require.NoError(t, store.SaveWorkspace(WorkspaceRecord{
		ID:             "workspace-1",
		ProjectRoot:    projectRoot,
		SharedDocsRoot: projectRoot + `\docs`,
		CreatedAt:      createdAt,
		ActiveThreadID: "thread-1",
	}))
	require.NoError(t, store.SaveThread(ThreadRecord{
		ID:             "thread-1",
		WorkspaceID:    "workspace-1",
		Name:           "Thread 1",
		Status:         "idle",
		ActiveModel:    "gpt-5",
		PermissionMode: "ask-user",
		CreatedAt:      createdAt,
	}))
	require.NoError(t, store.SaveTask(TaskRecord{
		ID:            "task-1",
		ThreadID:      "thread-1",
		Title:         "Draft spec",
		Status:        "running",
		Kind:          "thread.message.append",
		Input:         `{"role":"user","content":"Draft spec please"}`,
		ResultSummary: "message appended",
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}))
	require.NoError(t, store.SaveMessage(MessageRecord{
		ID:        "message-1",
		ThreadID:  "thread-1",
		Role:      "user",
		Content:   "Draft spec please",
		CreatedAt: createdAt,
	}))
	require.NoError(t, store.SaveToolCall(ToolCallRecord{
		ID:        "toolcall-1",
		ThreadID:  "thread-1",
		ToolID:    "bridge.check",
		Status:    "completed",
		Summary:   "Bridge reachable",
		CreatedAt: createdAt,
	}))
	require.NoError(t, store.SaveArtifact(ArtifactRecord{
		ID:        "artifact-1",
		ThreadID:  "thread-1",
		Path:      `D:\artifacts\spec.md`,
		Kind:      "markdown",
		CreatedAt: updatedAt,
	}))
	require.NoError(t, store.SaveRuntimeFlag(RuntimeFlagRecord{
		ThreadID:  "thread-1",
		Key:       "preview",
		Value:     "ready",
		UpdatedAt: updatedAt,
	}))
	require.NoError(t, store.SaveEvent(EventRecord{
		ID:        "event-1",
		ThreadID:  "thread-1",
		Type:      "task.updated",
		Message:   "Draft spec moved to running on Thread 1",
		CreatedAt: updatedAt,
	}))
	require.NoError(t, store.SaveWriteExecution(WriteExecutionRecord{
		ID:                    "writeexec-1",
		ThreadID:              "thread-1",
		TaskID:                "task-1",
		ApprovalID:            "approval-1",
		ToolKind:              "workspace.apply_patch",
		Operation:             "apply",
		RelatedExecutionID:    "",
		Status:                "completed",
		TargetPaths:           `["README.md"]`,
		PatchHash:             "abc123",
		PatchSummary:          "2 patch line(s)",
		BeforeSnapshotSummary: "exists, 1 line(s), 4 byte(s), sha256:oldhash123456",
		AfterSnapshotSummary:  "exists, 1 line(s), 4 byte(s), sha256:newhash123456",
		RollbackPayload:       `[{"path":"README.md","beforeExists":true,"beforeContent":"old\n","beforeHash":"oldhash123456","afterExists":true,"afterHash":"newhash123456"}]`,
		ResultSummary:         "applied patch to README.md: updated 2 line(s)",
		CreatedAt:             updatedAt,
		UpdatedAt:             updatedAt,
	}))

	snapshot, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, "workspace-1", snapshot.Workspace.ID)
	require.Equal(t, "thread-1", snapshot.Workspace.ActiveThreadID)
	require.Len(t, snapshot.Threads, 1)
	require.Len(t, snapshot.Tasks, 1)
	require.Len(t, snapshot.Messages, 1)
	require.Len(t, snapshot.ToolCalls, 1)
	require.Len(t, snapshot.Artifacts, 1)
	require.Len(t, snapshot.Flags, 1)
	require.Len(t, snapshot.Events, 1)
	require.Len(t, snapshot.WriteExecutions, 1)
	require.Equal(t, "running", snapshot.Tasks[0].Status)
	require.Equal(t, "thread.message.append", snapshot.Tasks[0].Kind)
	require.Equal(t, "message appended", snapshot.Tasks[0].ResultSummary)
	require.Equal(t, updatedAt, snapshot.Tasks[0].UpdatedAt)
	require.Equal(t, "Draft spec please", snapshot.Messages[0].Content)
	require.Equal(t, "bridge.check", snapshot.ToolCalls[0].ToolID)
	require.Equal(t, "markdown", snapshot.Artifacts[0].Kind)
	require.Equal(t, "ready", snapshot.Flags[0].Value)
	require.Equal(t, "workspace.apply_patch", snapshot.WriteExecutions[0].ToolKind)
	require.Equal(t, "approval-1", snapshot.WriteExecutions[0].ApprovalID)
	require.Equal(t, "apply", snapshot.WriteExecutions[0].Operation)
	require.Equal(t, `[{"path":"README.md","beforeExists":true,"beforeContent":"old\n","beforeHash":"oldhash123456","afterExists":true,"afterHash":"newhash123456"}]`, snapshot.WriteExecutions[0].RollbackPayload)
	require.Equal(t, "applied patch to README.md: updated 2 line(s)", snapshot.WriteExecutions[0].ResultSummary)
}

func TestMaxSuffix(t *testing.T) {
	require.Equal(t, 12, MaxSuffix([]string{"thread-2", "thread-12", "other-1"}, "thread-"))
	require.Equal(t, 0, MaxSuffix([]string{"other-1"}, "thread-"))
}

func TestStoreSupportsConcurrentOpenAndRead(t *testing.T) {
	projectRoot := t.TempDir()

	first, err := Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, first.Close()) }()

	createdAt := time.Date(2026, 5, 17, 5, 0, 0, 0, time.UTC)
	require.NoError(t, first.SaveWorkspace(WorkspaceRecord{
		ID:             "workspace-1",
		ProjectRoot:    projectRoot,
		SharedDocsRoot: filepath.Join(projectRoot, "docs"),
		CreatedAt:      createdAt,
		ActiveThreadID: "thread-1",
	}))
	require.NoError(t, first.SaveThread(ThreadRecord{
		ID:             "thread-1",
		WorkspaceID:    "workspace-1",
		Name:           "Thread 1",
		Status:         "idle",
		ActiveModel:    "",
		PermissionMode: "ask-user",
		CreatedAt:      createdAt,
	}))

	second, err := Open(projectRoot)
	require.NoError(t, err)
	defer func() { require.NoError(t, second.Close()) }()

	snapshot, err := second.Load()
	require.NoError(t, err)
	require.Equal(t, "workspace-1", snapshot.Workspace.ID)
	require.Len(t, snapshot.Threads, 1)
	require.Equal(t, "thread-1", snapshot.Threads[0].ID)
}
