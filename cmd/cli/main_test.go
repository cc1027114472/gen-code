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
	require.Contains(t, output, "primary workflow: agent run --thread=<threadId> --goal=...")
	require.Contains(t, output, "inspect agent progress: tasks list --thread=<threadId>")
	require.Contains(t, output, "runtime status")
	require.Contains(t, output, "threads write-executions --id=<threadId>")
	require.Contains(t, output, "tasks list --thread=<threadId>")
	require.Contains(t, output, "tasks update-status --thread=<threadId> --task=<taskId> --status=<status>")
	require.Contains(t, output, "model run --thread=<threadId> --input=...")
	require.Contains(t, output, "rollback latest --thread=<threadId>")
	require.Contains(t, output, "tasks create --thread=<threadId> --kind=<kind>")
}

func TestRuntimeStatusUsesRemoteSourceWhenServerIsAvailable(t *testing.T) {
	serverURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","runtimeSourceDetail":"canonical shared runtime served by the app-server entry","runtimeTrust":"canonical","canonicalRuntimeUrl":"` + serverURL + `","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":2,"activeThreadId":"thread-1","taskCount":3,"eventCount":5}}`))
		case "/api/skills":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"common.browser","group":"common","name":"Browser","description":"Shared browser skill","source":"common","verificationStatus":"implemented","localizationChecked":true},{"id":"codex.review","group":"codex","name":"Review","description":"Codex review skill","source":"codex","verificationStatus":"implemented","localizationChecked":true}]}}`))
		case "/api/tools":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"workspace.read_file","name":"Read File","description":"Read a file from workspace","permissionMode":"read-only","source":"runtime","kind":"workspace.read_file","readOnly":true,"executable":true}]}}`))
		case "/api/mcp/servers":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"filesystem","source":"node_modules","enabled":true,"toolCount":2,"resourceCount":1,"status":"enabled"}]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	serverURL = server.URL

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"runtime", "status"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "source: remote-app-server")
	require.Contains(t, output, "canonical runtime target: "+server.URL)
	require.Contains(t, output, "source trust: canonical")
	require.Contains(t, output, "source detail: canonical shared runtime served by the app-server entry")
	require.Contains(t, output, "mcp metadata verification: metadata health only; end-to-end MCP execution is not verified")
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
	require.Contains(t, output, "source trust: degraded")
	require.Contains(t, output, "source detail: project-local SQLite fallback because the canonical app-server runtime is unavailable")
	require.Contains(t, output, "kind=thread.message.append")
}

func TestToolsListPrintsExecutionMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tools":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"workspace.read_file","name":"Read File","description":"Read a file from workspace","permissionMode":"read-only","source":"runtime","kind":"workspace.read_file","readOnly":true,"executable":true},{"id":"workspace.stat_file","name":"Stat File","description":"Stat a workspace file","permissionMode":"read-only","source":"runtime","kind":"workspace.stat_file","readOnly":true,"executable":true},{"id":"workspace.read_files_batch","name":"Read Files Batch","description":"Read workspace files","permissionMode":"read-only","source":"runtime","kind":"workspace.read_files_batch","readOnly":true,"executable":true},{"id":"workspace.list_files_filtered","name":"List Files Filtered","description":"List filtered files","permissionMode":"read-only","source":"runtime","kind":"workspace.list_files_filtered","readOnly":true,"executable":true},{"id":"workspace.search_text_detailed","name":"Search Text Detailed","description":"Search text with details","permissionMode":"read-only","source":"runtime","kind":"workspace.search_text_detailed","readOnly":true,"executable":true},{"id":"browser.state","name":"Browser State","description":"Inspect the current browser workspace state","permissionMode":"read-only","source":"runtime","kind":"browser.state","readOnly":true,"executable":true},{"id":"browser.open","name":"Browser Open","description":"Open a new controlled local browser tab for a URL","permissionMode":"read-only","source":"runtime","kind":"browser.open","readOnly":true,"executable":true},{"id":"browser.navigate","name":"Browser Navigate","description":"Navigate an existing controlled local browser tab to a URL","permissionMode":"read-only","source":"runtime","kind":"browser.navigate","readOnly":true,"executable":true}]}}`))
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
	require.Contains(t, output, "source trust: canonical")
	require.Contains(t, output, "source detail: canonical shared runtime served by the app-server entry")
	require.Contains(t, output, "permission=read-only")
	require.Contains(t, output, "kind=workspace.read_file")
	require.Contains(t, output, "kind=workspace.stat_file")
	require.Contains(t, output, "kind=workspace.read_files_batch")
	require.Contains(t, output, "kind=workspace.list_files_filtered")
	require.Contains(t, output, "kind=workspace.search_text_detailed")
	require.Contains(t, output, "kind=browser.state")
	require.Contains(t, output, "kind=browser.open")
	require.Contains(t, output, "kind=browser.navigate")
	require.Contains(t, output, "executable=true")
	require.Contains(t, output, "readOnly=true")
}

func TestToolsListIncludesRepresentativeBrowserTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tools":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"browser.state","name":"Browser State","description":"Inspect the current browser workspace state","permissionMode":"read-only","source":"runtime","kind":"browser.state","readOnly":true,"executable":false},{"id":"browser.open","name":"Browser Open","description":"Open a new browser tab for a URL","permissionMode":"ask-user","source":"runtime","kind":"browser.open","readOnly":false,"executable":false},{"id":"browser.navigate","name":"Browser Navigate","description":"Navigate an existing browser tab to a URL","permissionMode":"ask-user","source":"runtime","kind":"browser.navigate","readOnly":false,"executable":false}]}}`))
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","runtimeSourceDetail":"canonical shared runtime served by the app-server entry","runtimeTrust":"canonical","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":1,"activeThreadId":"thread-1","taskCount":0,"eventCount":0}}`))
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

	require.Contains(t, output, "browser.state")
	require.Contains(t, output, "browser.open")
	require.Contains(t, output, "browser.navigate")
	require.Contains(t, output, "permission=ask-user")
	require.Contains(t, output, "kind=browser.open")
}

func TestSkillsListPrintsGovernanceBaseline(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"skills", "list"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "skills list")
	require.Contains(t, output, "source: local-fallback")
	require.Contains(t, output, "source trust: degraded")
	require.Contains(t, output, "governance fields: skill id, group, source, verification status, localization checked, capability verified, isolation status")
	require.Contains(t, output, "skill governance:")
	require.Contains(t, output, "common: implemented=")
	require.Contains(t, output, "codex: implemented=")
	require.Contains(t, output, "cc: implemented=")
	require.Contains(t, output, "capability-pending=")
	require.Contains(t, output, "common:")
	require.Contains(t, output, "codex:")
	require.Contains(t, output, "cc:")
	require.Contains(t, output, "verification=implemented")
	require.Contains(t, output, "localization=checked")
	require.Contains(t, output, "capability=")
	require.Contains(t, output, "isolation=shared-common")
}

func TestSkillsListUsesRemoteRuntimeSourceWhenServerIsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/skills":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"common.browser","group":"common","name":"Browser","description":"Shared browser skill","source":"common","verificationStatus":"implemented","localizationChecked":true,"isolationStatus":"shared-common","capabilityVerified":false,"capabilitySummary":"capability baseline not tracked for built-in shared common skill"},{"id":"codex.review","group":"codex","name":"Review","description":"Codex review skill","source":"codex","verificationStatus":"verified","localizationChecked":true,"isolationStatus":"isolated","capabilityVerified":true,"capabilitySummary":"capability verified"}]}}`))
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","runtimeSourceDetail":"canonical shared runtime served by the app-server entry","runtimeTrust":"canonical","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":1,"activeThreadId":"thread-1","taskCount":0,"eventCount":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"skills", "list"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "source: remote-app-server")
	require.Contains(t, output, "source trust: canonical")
	require.Contains(t, output, "common: implemented=1 verified=0 localization-pending=0 capability-pending=0")
	require.Contains(t, output, "codex: implemented=1 verified=1 localization-pending=0 capability-pending=0")
	require.Contains(t, output, "cc: implemented=0 verified=0 localization-pending=0 capability-pending=0")
	require.Contains(t, output, "verification=verified")
	require.Contains(t, output, "localization=checked")
	require.Contains(t, output, "capability=verified")
	require.Contains(t, output, "isolation=shared-common")
	require.Contains(t, output, "isolation=isolated")
}

func TestMCPListPrintsHealthStatusAndTrust(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/mcp/servers":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"external-fixture","source":"fixture","enabled":true,"toolCount":3,"resourceCount":0,"status":"enabled"},{"id":"sdk-external-fixture","source":"sdk","enabled":true,"toolCount":2,"resourceCount":0,"status":"enabled"},{"id":"third-party-time","source":"third-party","enabled":true,"toolCount":1,"resourceCount":0,"status":"enabled"},{"id":"stale-bridge","source":"node_modules","enabled":true,"toolCount":1,"resourceCount":0,"status":"unreachable"}]}}`))
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","runtimeSourceDetail":"canonical shared runtime served by the app-server entry","runtimeTrust":"canonical","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":1,"activeThreadId":"thread-1","taskCount":0,"eventCount":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"mcp", "list"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "mcp list")
	require.Contains(t, output, "source: remote-app-server")
	require.Contains(t, output, "source trust: canonical")
	require.Contains(t, output, "metadata verification: metadata health only; end-to-end MCP execution is not verified")
	require.Contains(t, output, "execution baseline: multi-server stdio external execution baseline verified")
	require.Contains(t, output, "configured servers: 4")
	require.Contains(t, output, "external-fixture (enabled, metadata health: enabled) [source=fixture, tools=3, resources=0]")
	require.Contains(t, output, "sdk-external-fixture (enabled, metadata health: enabled) [source=sdk, tools=2, resources=0]")
	require.Contains(t, output, "third-party-time (enabled, metadata health: enabled) [source=third-party, tools=1, resources=0]")
	require.Contains(t, output, "stale-bridge (enabled, metadata health: unreachable) [source=node_modules, tools=1, resources=0]")
	require.Contains(t, output, "verified execution lanes:")
	require.Contains(t, output, "external-fixture: fixture regression lane (configured)")
	require.Contains(t, output, "sdk-external-fixture: official SDK external lane (configured)")
	require.Contains(t, output, "third-party-time: third-party time lane (configured)")
}

func TestMCPInvokePrintsTaskIdentityAndSummary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/threads/thread-1/tasks":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":"task-mcp","threadId":"thread-1","title":"Invoke MCP sdk-external-fixture/echo","status":"queued","kind":"mcp.tool.invoke","createdAt":"2026-05-17T00:00:00Z","updatedAt":"2026-05-17T00:00:00Z"}}`))
		case "/api/threads/thread-1/tasks/task-mcp/run":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":"task-mcp","threadId":"thread-1","title":"Invoke MCP sdk-external-fixture/echo","status":"completed","kind":"mcp.tool.invoke","resultSummary":"mcp tool sdk-external-fixture/echo executed","updatedAt":"2026-05-17T00:00:01Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{
			"mcp", "invoke",
			"--thread=thread-1",
			"--server=sdk-external-fixture",
			"--tool=echo",
			`--arguments={"message":"hello"}`,
		})
		require.NoError(t, err)
	})

	require.Contains(t, output, "mcp task created")
	require.Contains(t, output, "server id: sdk-external-fixture")
	require.Contains(t, output, "tool name: echo")
	require.Contains(t, output, "task id: task-mcp")
	require.Contains(t, output, "status: completed")
	require.Contains(t, output, "result summary: mcp tool sdk-external-fixture/echo executed")
}

func TestTasksListPrintsAgentPlanMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/threads/thread-1/tasks":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"task-agent","threadId":"thread-1","title":"Agent run","status":"running","kind":"agent.run","resultSummary":"agent step 2/4: Read the selected files","parentTaskId":"","waitingStatus":"waiting_for_task","agentStep":2,"agentMaxSteps":4,"latestChildTaskId":"task-child","agentPlanSummary":"Filter matching files first, then read the selected files, then answer.","agentPlanMode":"filter_then_read","agentCurrentStepTitle":"Answer with the findings","agentLastReasoning":"Read the selected files","createdAt":"2026-05-17T00:00:00Z","updatedAt":"2026-05-17T00:00:02Z"},{"id":"task-child","threadId":"thread-1","title":"Read files","status":"completed","kind":"workspace.read_files_batch","resultSummary":"read 2 files: README.md, go.mod","parentTaskId":"task-agent","waitingStatus":"","createdAt":"2026-05-17T00:00:01Z","updatedAt":"2026-05-17T00:00:01Z"}]}}`))
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":1,"activeThreadId":"thread-1","taskCount":1,"eventCount":0}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"tasks", "list", "--thread=thread-1"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "plan: Filter matching files first, then read the selected files, then answer.")
	require.Contains(t, output, "plan mode: filter_then_read")
	require.Contains(t, output, "current step: Answer with the findings")
	require.Contains(t, output, "last reasoning: Read the selected files")
	require.Contains(t, output, "progress: 2/4")
	require.Contains(t, output, "latest child: task-child")
	require.Contains(t, output, "workflow: default goal-oriented agent parent")
	require.Contains(t, output, "waiting on: child task task-child to finish")
	require.Contains(t, output, "workflow: child task of task-agent")
	require.Contains(t, output, "kind=workspace.read_files_batch")
}

func TestAgentRunPrintsDefaultWorkflowGuidance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/threads/thread-1/tasks":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":"task-agent","threadId":"thread-1","title":"Goal run","status":"queued","kind":"agent.run","createdAt":"2026-05-17T00:00:00Z","updatedAt":"2026-05-17T00:00:00Z"}}`))
		case "/api/threads/thread-1/tasks/task-agent/run":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":"task-agent","threadId":"thread-1","title":"Goal run","status":"waiting_for_approval","kind":"agent.run","waitingStatus":"waiting_for_approval","resultSummary":"approval required for workspace.apply_patch on README.md; 2 patch line(s)","agentPlanSummary":"Apply the requested patch first, then answer with the result.","agentPlanMode":"patch_then_respond","agentCurrentStepTitle":"Answer with the result","agentLastReasoning":"Prepared a patch for README.md","agentStep":1,"agentMaxSteps":4,"latestChildTaskId":"task-child-patch","updatedAt":"2026-05-17T00:00:03Z"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{
			"agent", "run",
			"--thread=thread-1",
			"--title=Goal run",
			"--goal=Update README with the current workflow",
		})
		require.NoError(t, err)
	})

	require.Contains(t, output, "workflow: default goal-oriented workflow")
	require.Contains(t, output, "waiting on: approval for child task task-child-patch")
	require.Contains(t, output, "latest child: task-child-patch")
	require.Contains(t, output, "plan mode: patch_then_respond")
	require.Contains(t, output, "current step: Answer with the result")
	require.Contains(t, output, "next step: approve the child workspace.apply_patch task; the parent agent will auto-resume")
	require.Contains(t, output, "inspect: gen-code tasks list --thread=thread-1")
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

	require.Contains(t, output, "source detail: project-local SQLite fallback because the canonical app-server runtime is unavailable")
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

func TestThreadWriteExecutionsUsesRemoteSourceWhenServerIsAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/threads/thread-1/write-executions":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"write-1","threadId":"thread-1","taskId":"task-9","approvalId":"approval-1","toolKind":"workspace.apply_patch","status":"completed","targetPaths":["docs/sample.txt"],"patchSummary":"applied patch to docs/sample.txt: created 1 line(s)","beforeSummary":"docs/sample.txt missing before apply","afterSummary":"docs/sample.txt exists with 1 line(s), 5 byte(s), sha256:abc123","resultSummary":"applied patch to docs/sample.txt: created 1 line(s)","createdAt":"2026-05-16T00:00:00Z","updatedAt":"2026-05-16T00:00:01Z"}]}}`))
		case "/api/runtime/status":
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"state":"running","ready":true,"message":"remote ready","runtimeSource":"remote-app-server","workspaceId":"gen-code","projectRoot":"D:/repo/gen-code","threadCount":1,"activeThreadId":"thread-1","taskCount":1,"eventCount":4}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

	output := captureOutput(t, func() {
		err := run(context.Background(), []string{"threads", "write-executions", "--id=thread-1"})
		require.NoError(t, err)
	})

	require.Contains(t, output, "thread write executions")
	require.Contains(t, output, "source: remote-app-server")
	require.Contains(t, output, "op=apply")
	require.Contains(t, output, "tool=workspace.apply_patch")
	require.Contains(t, output, "docs/sample.txt")
	require.Contains(t, output, "patch: applied patch to docs/sample.txt: created 1 line(s)")
}

func TestRollbackLatestExecutesThroughRemoteTaskAPI(t *testing.T) {
	var createdBody []byte
	output := captureOutput(t, func() {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/threads/thread-1/write-executions":
				_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"items":[{"id":"writeexec-9","threadId":"thread-1","taskId":"task-apply","approvalId":"","toolKind":"workspace.apply_patch","operation":"apply","relatedExecutionId":"","status":"completed","targetPaths":["README.md"],"patchSummary":"2 patch line(s)","beforeSummary":"exists","afterSummary":"exists","resultSummary":"applied patch to README.md: updated 2 line(s)","createdAt":"2026-05-16T00:00:00Z","updatedAt":"2026-05-16T00:00:01Z"}]}}`))
			case "/api/threads/thread-1/tasks":
				require.Equal(t, http.MethodPost, r.Method)
				var err error
				createdBody, err = io.ReadAll(r.Body)
				require.NoError(t, err)
				_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":"task-rollback","threadId":"thread-1","title":"Rollback latest write execution","status":"queued","kind":"workspace.apply_patch.rollback","inputSummary":"{\"writeExecutionId\":\"writeexec-9\"}","approvalStatus":"direct","createdAt":"2026-05-16T00:00:02Z","updatedAt":"2026-05-16T00:00:02Z"}}`))
			case "/api/threads/thread-1/tasks/task-rollback/run":
				_, _ = w.Write([]byte(`{"code":0,"message":"ok","data":{"id":"task-rollback","threadId":"thread-1","title":"Rollback latest write execution","status":"completed","kind":"workspace.apply_patch.rollback","resultSummary":"rolled back patch on README.md: restored README.md","approvalStatus":"direct","createdAt":"2026-05-16T00:00:02Z","updatedAt":"2026-05-16T00:00:03Z"}}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		t.Setenv("GENCODE_RUNTIME_BASE_URL", server.URL)

		err := run(context.Background(), []string{"rollback", "latest", "--thread=thread-1"})
		require.NoError(t, err)
	})

	var payload struct {
		Title string `json:"title"`
		Kind  string `json:"kind"`
		Input string `json:"input"`
	}
	require.NoError(t, json.Unmarshal(createdBody, &payload))
	require.Equal(t, "workspace.apply_patch.rollback", payload.Kind)
	assertJSONEqual(t, `{"writeExecutionId":"writeexec-9"}`, payload.Input)
	require.Contains(t, output, "rollback task created")
	require.Contains(t, output, "rollback task executed")
	require.Contains(t, output, "rolled back patch on README.md")
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
	require.Equal(t, "canonical shared runtime served by the app-server entry", runtimeSourceDetail("remote-app-server"))
	require.Equal(t, "project-local SQLite fallback because the canonical app-server runtime is unavailable", runtimeSourceDetail("local-fallback"))
	require.Equal(t, "unknown runtime source", runtimeSourceDetail("mystery"))
}

func TestRuntimeSourceTrust(t *testing.T) {
	require.Equal(t, "canonical", runtimeSourceTrust("remote-app-server"))
	require.Equal(t, "degraded", runtimeSourceTrust("local-fallback"))
	require.Equal(t, "unknown", runtimeSourceTrust("mystery"))
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
