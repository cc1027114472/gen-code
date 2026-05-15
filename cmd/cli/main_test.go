package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
	require.Contains(t, output, "tasks list --thread=<threadId>")
	require.Contains(t, output, "tasks update-status --thread=<threadId> --task=<taskId> --status=<status>")
}

func TestRuntimeStatusUsesRemoteSourceWhenServerIsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/runtime/status", r.URL.Path)
		_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":2,"activeThreadId":"thread-1","taskCount":3,"eventCount":5}}`))
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"runtime", "status"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "source: remote-app-server")
	require.Contains(t, output, "source detail: shared runtime from the running app-server")
	require.Contains(t, output, "active thread task count: 3")
	require.Contains(t, output, "active thread event count: 5")
}

func TestTasksListFallsBackLocallyWhenServerIsUnavailable(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"threads", "create", "--name=Thread A"})
		require.NoError(t, err)
		err = run(context.Background(), []string{"tasks", "create", "--thread=thread-1", "--title=Draft spec", "--kind=thread.message.append", "--input={\"role\":\"user\",\"content\":\"Draft spec\"}"})
		require.NoError(t, err)
		err = run(context.Background(), []string{"tasks", "list", "--thread=thread-1"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "source: local-fallback")
	require.Contains(t, output, "source detail: project-local SQLite fallback because app-server is unavailable")
	require.Contains(t, output, "kind=thread.message.append")
}

func TestToolsListPrintsExecutionMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tools":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"workspace.read_file","name":"Read File","description":"Read a file from workspace","permissionMode":"read-only","source":"runtime","kind":"workspace.read_file","readOnly":true,"executable":true}]}}`))
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":1,"activeThreadId":"thread-1","taskCount":0,"eventCount":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"tools", "list"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "source: remote-app-server")
	require.Contains(t, output, "source detail: shared runtime from the running app-server")
	require.Contains(t, output, "permission=read-only")
	require.Contains(t, output, "kind=workspace.read_file")
	require.Contains(t, output, "executable=true")
	require.Contains(t, output, "readOnly=true")
}

func TestCreateTaskPrintsPowerShellInputHint(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"threads", "create", "--name=ReadOnly Thread", "--permission=read-only"})
		require.NoError(t, err)
		err = run(context.Background(), []string{"tasks", "create", "--thread=thread-1", "--title=Read go.mod", "--kind=workspace.read_file", "--input={\"path\":\"go.mod\"}"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "source detail: project-local SQLite fallback because app-server is unavailable")
	require.Contains(t, output, "input: {\"path\":\"go.mod\"}")
	require.Contains(t, output, "PowerShell JSON can use --input='{\"path\":\"go.mod\"}'")
}

func TestFallbackText(t *testing.T) {
	require.Equal(t, "fallback", fallbackText("", "fallback"))
	require.Equal(t, "fallback", fallbackText("   ", "fallback"))
	require.Equal(t, "value", fallbackText("value", "fallback"))
}

func TestRuntimeSourceDetail(t *testing.T) {
	require.Equal(t, "shared runtime from the running app-server", runtimeSourceDetail("remote-app-server"))
	require.Equal(t, "project-local SQLite fallback because app-server is unavailable", runtimeSourceDetail("local-fallback"))
	require.Equal(t, "unknown runtime source", runtimeSourceDetail("mystery"))
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
