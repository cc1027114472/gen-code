package runner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/core/session"
)

func TestRunnerExecutesThreadLocalMessageTask(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{Name: "Runner"})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Append message",
		Kind:  KindMessageAppend,
		Input: `{"role":"assistant","content":"done"}`,
	})
	require.True(t, ok)

	result, err := New(registry).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "message appended")

	messages, ok := registry.Messages(thread.ID)
	require.True(t, ok)
	require.Len(t, messages, 1)
	require.Equal(t, "assistant", messages[0].Role)

	toolCalls, ok := registry.ToolCalls(thread.ID)
	require.True(t, ok)
	require.Len(t, toolCalls, 2)
	require.Equal(t, "running", toolCalls[0].Status)
	require.Equal(t, "completed", toolCalls[1].Status)
}

func TestRunnerRecoversInterruptedRunningTasks(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{Name: "Runner"})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Interrupted task",
		Kind:  KindRuntimeFlagSet,
		Input: `{"key":"mode","value":"draft"}`,
	})
	require.True(t, ok)
	_, err := registry.UpdateTaskStatus(thread.ID, task.ID, session.UpdateTaskStatusInput{Status: "running"})
	require.NoError(t, err)

	err = New(registry).RecoverInterruptedTasks()
	require.NoError(t, err)

	reloaded, err := registry.Task(thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", reloaded.Status)
	require.Equal(t, restartFailureLabel, reloaded.ResultSummary)
}
