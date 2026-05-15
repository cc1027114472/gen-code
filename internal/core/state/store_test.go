package state

import (
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
		ID:        "task-1",
		ThreadID:  "thread-1",
		Title:     "Draft spec",
		Status:    "running",
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}))
	require.NoError(t, store.SaveEvent(EventRecord{
		ID:        "event-1",
		ThreadID:  "thread-1",
		Type:      "task.updated",
		Message:   "Draft spec moved to running on Thread 1",
		CreatedAt: updatedAt,
	}))

	snapshot, err := store.Load()
	require.NoError(t, err)
	require.Equal(t, "workspace-1", snapshot.Workspace.ID)
	require.Equal(t, "thread-1", snapshot.Workspace.ActiveThreadID)
	require.Len(t, snapshot.Threads, 1)
	require.Len(t, snapshot.Tasks, 1)
	require.Len(t, snapshot.Events, 1)
	require.Equal(t, "running", snapshot.Tasks[0].Status)
	require.Equal(t, updatedAt, snapshot.Tasks[0].UpdatedAt)
}

func TestMaxSuffix(t *testing.T) {
	require.Equal(t, 12, MaxSuffix([]string{"thread-2", "thread-12", "other-1"}, "thread-"))
	require.Equal(t, 0, MaxSuffix([]string{"other-1"}, "thread-"))
}
