package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/provider"
	"llmtrace/internal/core/session"
)

type stubModelExecutor struct {
	result provider.ResponseResult
	err    error
}

func (s stubModelExecutor) CreateResponse(context.Context, provider.ResponseRequest) (provider.ResponseResult, error) {
	if s.err != nil {
		return provider.ResponseResult{}, s.err
	}
	return s.result, nil
}

func TestRunnerExecutesThreadLocalMessageTask(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Runner",
		PermissionMode: policy.WorkspaceWrite,
	})
	stream, cancel, err := registry.SubscribeEvents(thread.ID)
	require.NoError(t, err)
	defer cancel()
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Append message",
		Kind:  KindMessageAppend,
		Input: `{"role":"assistant","content":"done"}`,
	})
	require.True(t, ok)

	result, err := New(registry, nil).RunTask(context.Background(), thread.ID, task.ID)
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

	eventTypes := []string{}
collect:
	for {
		select {
		case item := <-stream:
			eventTypes = append(eventTypes, item.Type)
		default:
			break collect
		}
	}
	require.Contains(t, eventTypes, "task.started")
	require.Contains(t, eventTypes, "toolcall.started")
	require.Contains(t, eventTypes, "toolcall.completed")
	require.Contains(t, eventTypes, "task.completed")
}

func TestRunnerRecoversInterruptedRunningTasks(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Runner",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Interrupted task",
		Kind:  KindRuntimeFlagSet,
		Input: `{"key":"mode","value":"draft"}`,
	})
	require.True(t, ok)
	_, err := registry.UpdateTaskStatus(thread.ID, task.ID, session.UpdateTaskStatusInput{Status: "running"})
	require.NoError(t, err)

	err = New(registry, nil).RecoverInterruptedTasks()
	require.NoError(t, err)

	reloaded, err := registry.Task(thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", reloaded.Status)
	require.Equal(t, restartFailureLabel, reloaded.ResultSummary)

	events, ok := registry.Events(thread.ID)
	require.True(t, ok)
	found := false
	for _, item := range events {
		if item.Type == "task.recovered_as_failed" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestRunnerExecutesWorkspaceReadOnlyTools(t *testing.T) {
	projectRoot := t.TempDir()
	readmePath := filepath.Join(projectRoot, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("hello workspace"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Reader",
		PermissionMode: policy.ReadOnly,
	})

	readTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Read file",
		Kind:  KindWorkspaceRead,
		Input: `{"path":"README.md"}`,
	})
	require.True(t, ok)
	result, err := New(registry, nil).RunTask(context.Background(), thread.ID, readTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "hello workspace")

	listTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "List files",
		Kind:  KindWorkspaceList,
		Input: `{"path":"."}`,
	})
	require.True(t, ok)
	result, err = New(registry, nil).RunTask(context.Background(), thread.ID, listTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "README.md")

	searchTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Search text",
		Kind:  KindWorkspaceSearch,
		Input: `{"query":"workspace","path":"."}`,
	})
	require.True(t, ok)
	result, err = New(registry, nil).RunTask(context.Background(), thread.ID, searchTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "README.md")
}

func TestRunnerRejectsReadToolForAskUserMode(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("hello"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Restricted",
		PermissionMode: policy.AskUser,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Read file",
		Kind:  KindWorkspaceRead,
		Input: `{"path":"README.md"}`,
	})
	require.True(t, ok)

	result, err := New(registry, nil).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ResultSummary, "approval required")
}

func TestRunnerRejectsPathOutsideWorkspace(t *testing.T) {
	projectRoot := t.TempDir()
	outsideDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Reader",
		PermissionMode: policy.ReadOnly,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Read outside",
		Kind:  KindWorkspaceRead,
		Input: `{"path":"` + filepath.ToSlash(filepath.Join(outsideDir, "secret.txt")) + `"}`,
	})
	require.True(t, ok)

	result, err := New(registry, nil).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ResultSummary, ErrPathOutsideWorkspace.Error())
}

func TestRunnerExecutesModelResponseTask(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Model",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Ask model",
		Kind:  KindModelResponse,
		Input: `{"input":"hello model","model":"gpt-5.4-A"}`,
	})
	require.True(t, ok)

	result, err := New(registry, stubModelExecutor{
		result: provider.ResponseResult{
			ResponseID: "resp-1",
			Model:      "gpt-5.4-A",
			OutputText: "assistant answer",
			APIStyle:   provider.APIStyleOpenAIResponses,
		},
	}).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "response from gpt-5.4-A")

	messages, ok := registry.Messages(thread.ID)
	require.True(t, ok)
	require.NotEmpty(t, messages)
	require.Equal(t, "assistant", messages[len(messages)-1].Role)
	require.Equal(t, "assistant answer", messages[len(messages)-1].Content)

	toolCalls, ok := registry.ToolCalls(thread.ID)
	require.True(t, ok)
	require.Len(t, toolCalls, 2)
	require.Equal(t, KindModelResponse, toolCalls[0].ToolID)
	require.Equal(t, "completed", toolCalls[1].Status)
}

func TestRunnerFailsModelResponseTaskWhenProviderFails(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Model",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Ask model",
		Kind:  KindModelResponse,
		Input: `{"input":"hello model"}`,
	})
	require.True(t, ok)

	result, err := New(registry, stubModelExecutor{
		err: errors.New("gateway unavailable"),
	}).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ResultSummary, "provider error")
}
