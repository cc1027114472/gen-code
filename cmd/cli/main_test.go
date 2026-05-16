package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
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
	require.Contains(t, output, "model run --thread=<threadId> --input=...")
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

func TestProvidersListPrintsRecommendedAPIStyle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/providers":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"kind":"anthropic","enabled":true,"baseUrl":"http://localhost:1314","defaultModel":"gpt-5.4-A","hasAuthToken":true,"supportsChat":true,"supportsResponses":true,"preferredApiStyle":"openai-responses","recommended":true,"recommendedReason":"gpt models should use responses"}]}}`))
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":1,"activeThreadId":"thread-1","taskCount":0,"eventCount":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"providers", "list"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "source: remote-app-server")
	require.Contains(t, output, "preferredApi=openai-responses")
	require.Contains(t, output, "recommended=true")
}

func TestProviderProbePrintsProbeResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/providers/anthropic/probe":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"kind":"anthropic","reachable":true,"preferredApiStyle":"openai-responses","message":"provider reachable","details":{"supported_endpoint_types":["openai"]}}}`))
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":1,"activeThreadId":"thread-1","taskCount":0,"eventCount":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"provider", "probe", "--kind=anthropic"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "kind: anthropic")
	require.Contains(t, output, "reachable: true")
	require.Contains(t, output, "preferred api: openai-responses")
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
	require.Contains(t, output, "input source: inline --input")
	require.Contains(t, output, "recommended input: use --input-file=<path> for JSON payloads")
	require.Contains(t, output, "inline fallback: PowerShell JSON can use --input='{\"path\":\"go.mod\"}'")
}

func TestCreateTaskReadsInputFromInputFile(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")

	dir := t.TempDir()
	inputPath := filepath.Join(dir, "task-input.json")
	require.NoError(t, os.WriteFile(inputPath, []byte("{\"path\":\"README.md\"}\n"), 0o600))

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"threads", "create", "--name=File Input Thread"})
		require.NoError(t, err)
		err = run(context.Background(), []string{"tasks", "create", "--thread=thread-1", "--title=Read README", "--kind=workspace.read_file", "--input-file=" + inputPath})
		require.NoError(t, err)
	})

	require.Contains(t, output, "input source: input file task-input.json")
	require.Contains(t, output, "input: {\"path\":\"README.md\"}")
}

func TestCreatePatchTaskBuildsJSONFromPatchFileAndPath(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")

	dir := t.TempDir()
	patchPath := filepath.Join(dir, "sample.patch")
	patch := "*** Begin Patch\n*** Add File: docs/sample.txt\n+hello\n*** End Patch\n"
	require.NoError(t, os.WriteFile(patchPath, []byte(patch), 0o600))

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"threads", "create", "--name=Patch Thread", "--permission=ask-user"})
		require.NoError(t, err)
		err = run(context.Background(), []string{
			"tasks", "create",
			"--thread=thread-1",
			"--title=Apply patch",
			"--kind=workspace.apply_patch",
			"--patch-file=" + patchPath,
			"--path=docs/sample.txt",
		})
		require.NoError(t, err)
	})

	require.Contains(t, output, "kind: workspace.apply_patch")
	require.Contains(t, output, "approval: pending")
	require.Contains(t, output, "input source: patch file sample.patch -> docs/sample.txt")
	require.Contains(t, output, "docs/sample.txt")
	require.Contains(t, output, "recommended patch input: use --patch-file=<path> --path=<workspace-relative-path>")
}

func TestCreateTaskRejectsConflictingInputFlags(t *testing.T) {
	err := run(context.Background(), []string{
		"tasks", "create",
		"--thread=thread-1",
		"--kind=workspace.read_file",
		"--input={\"path\":\"go.mod\"}",
		"--input-file=payload.json",
	})
	require.EqualError(t, err, "use either --input or --input-file, not both")
}

func TestModelRunExecutesThroughRemoteTaskAPI(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/threads/thread-1/tasks":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var payload struct {
				Title string `json:"title"`
				Kind  string `json:"kind"`
				Input string `json:"input"`
			}
			require.NoError(t, json.Unmarshal(body, &payload))
			require.Equal(t, "model.response.create", payload.Kind)
			var taskInput map[string]any
			require.NoError(t, json.Unmarshal([]byte(payload.Input), &taskInput))
			require.Equal(t, "anthropic", taskInput["provider"])
			require.Equal(t, "gpt-5.4-A", taskInput["model"])
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":"task-9","threadId":"thread-1","title":"Ask model","status":"queued","kind":"model.response.create","inputSummary":"{\"provider\":\"anthropic\",\"model\":\"gpt-5.4-A\",\"input\":\"hello\",\"maxOutputTokens\":128}","createdAt":"2026-05-16T00:00:00Z","updatedAt":"2026-05-16T00:00:00Z"}}`))
		case "/api/threads/thread-1/tasks/task-9/run":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":"task-9","threadId":"thread-1","title":"Ask model","status":"completed","kind":"model.response.create","resultSummary":"response from gpt-5.4-A: hello back","createdAt":"2026-05-16T00:00:00Z","updatedAt":"2026-05-16T00:00:01Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{
			"model", "run",
			"--thread=thread-1",
			"--title=Ask model",
			"--provider=anthropic",
			"--model=gpt-5.4-A",
			"--input=hello",
			"--max-output-tokens=128",
		})
		require.NoError(t, err)
	})

	require.Contains(t, output, "model task created")
	require.Contains(t, output, "kind: model.response.create")
	require.Contains(t, output, "model task executed")
	require.Contains(t, output, "result: response from gpt-5.4-A: hello back")
}

func TestNormalizeTaskInputAcceptsLoosePowerShellObject(t *testing.T) {
	assertJSONEqual(t, `{"path":"go.mod"}`, normalizeTaskInput(`{path:go.mod}`))
	assertJSONEqual(t, `{"query":"workspace","path":"internal"}`, normalizeTaskInput(`{query:workspace,path:internal}`))
	assertJSONEqual(t, `{"path":"go.mod","recursive":true}`, normalizeTaskInput(`{path:'go.mod',recursive:true}`))
}

func TestNormalizeTaskInputLeavesValidJSONUntouched(t *testing.T) {
	assertJSONEqual(t, `{"path":"go.mod"}`, normalizeTaskInput(`{"path":"go.mod"}`))
}

func TestResolveTaskCreateInputRejectsPatchFlagsForNonPatchKind(t *testing.T) {
	_, _, err := resolveTaskCreateInput("workspace.read_file", taskCreateInputOptions{
		PatchFile: "demo.patch",
		PatchPath: "README.md",
	})
	require.EqualError(t, err, "--patch-file and --path are only supported for --kind=workspace.apply_patch")
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

	var buffer bytes.Buffer
	var copyWG sync.WaitGroup
	copyWG.Add(1)
	go func() {
		defer copyWG.Done()
		_, _ = io.Copy(&buffer, reader)
	}()

	fn()

	require.NoError(t, writer.Close())
	copyWG.Wait()
	require.NoError(t, reader.Close())

	return buffer.String()
}

func assertJSONEqual(t *testing.T, expected string, actual string) {
	t.Helper()

	var expectedValue any
	var actualValue any
	require.NoError(t, json.Unmarshal([]byte(expected), &expectedValue))
	require.NoError(t, json.Unmarshal([]byte(actual), &actualValue))
	require.Equal(t, expectedValue, actualValue)
}
