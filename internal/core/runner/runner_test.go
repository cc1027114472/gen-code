package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"llmtrace/internal/core/browser"
	"llmtrace/internal/core/mcp"
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

type scriptedModelExecutor struct {
	results []provider.ResponseResult
	err     error
	index   int
}

func (s *scriptedModelExecutor) CreateResponse(context.Context, provider.ResponseRequest) (provider.ResponseResult, error) {
	if s.err != nil {
		return provider.ResponseResult{}, s.err
	}
	if s.index >= len(s.results) {
		return provider.ResponseResult{}, errors.New("no scripted response available")
	}
	result := s.results[s.index]
	s.index++
	return result, nil
}

type stubBrowserCore struct {
	activeID string
	tabs     map[string]browser.Tab
	order    []string
	nextID   int
	values   map[string]string
}

func (s *stubBrowserCore) State(context.Context) (browser.Snapshot, error) {
	return s.snapshot(), nil
}

func (s *stubBrowserCore) Open(_ context.Context, request browser.OpenRequest) (browser.Snapshot, error) {
	if !stubBrowserURLAllowed(request.URL) {
		return browser.Snapshot{}, browser.ErrURLNotAllowed
	}
	s.ensureTabs()
	s.nextID++
	tabID := fmt.Sprintf("browser-tab-%d", s.nextID)
	s.tabs[tabID] = browser.Tab{ID: tabID, URL: request.URL}
	s.order = append(s.order, tabID)
	s.activeID = tabID
	return s.snapshot(), nil
}

func (s *stubBrowserCore) Navigate(_ context.Context, request browser.NavigateRequest) (browser.Snapshot, error) {
	if !stubBrowserURLAllowed(request.URL) {
		return browser.Snapshot{}, browser.ErrURLNotAllowed
	}
	tab, err := s.lookup(request.TabID)
	if err != nil {
		return browser.Snapshot{}, err
	}
	tab.URL = request.URL
	s.tabs[request.TabID] = tab
	s.activeID = request.TabID
	return s.snapshot(), nil
}

func (s *stubBrowserCore) Back(_ context.Context, request browser.TabRequest) (browser.Snapshot, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.Snapshot{}, err
	}
	s.activeID = request.TabID
	return s.snapshot(), nil
}

func (s *stubBrowserCore) Forward(_ context.Context, request browser.TabRequest) (browser.Snapshot, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.Snapshot{}, err
	}
	s.activeID = request.TabID
	return s.snapshot(), nil
}

func (s *stubBrowserCore) Reload(_ context.Context, request browser.TabRequest) (browser.Snapshot, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.Snapshot{}, err
	}
	s.activeID = request.TabID
	return s.snapshot(), nil
}

func (s *stubBrowserCore) CloseTab(_ context.Context, request browser.TabRequest) (browser.Snapshot, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.Snapshot{}, err
	}
	delete(s.tabs, request.TabID)
	filtered := make([]string, 0, len(s.order))
	for _, id := range s.order {
		if id != request.TabID {
			filtered = append(filtered, id)
		}
	}
	s.order = filtered
	if s.activeID == request.TabID {
		s.activeID = ""
		if len(s.order) > 0 {
			s.activeID = s.order[0]
		}
	}
	return s.snapshot(), nil
}

func (s *stubBrowserCore) ActivateTab(_ context.Context, request browser.TabRequest) (browser.Snapshot, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.Snapshot{}, err
	}
	s.activeID = request.TabID
	return s.snapshot(), nil
}

func (s *stubBrowserCore) Click(_ context.Context, request browser.ClickRequest) (browser.Snapshot, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.Snapshot{}, err
	}
	if strings.TrimSpace(request.Selector) == "" || strings.Contains(request.Selector, "missing") {
		return browser.Snapshot{}, browser.ErrSelectorNotFound
	}
	snapshot := s.snapshot()
	snapshot.ActiveTabID = request.TabID
	snapshot.LatestActionSummary = fmt.Sprintf("browser click executed: %s", request.TabID)
	return snapshot, nil
}

func (s *stubBrowserCore) Type(_ context.Context, request browser.TypeRequest) (browser.Snapshot, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.Snapshot{}, err
	}
	if strings.TrimSpace(request.Selector) == "" || strings.Contains(request.Selector, "missing") {
		return browser.Snapshot{}, browser.ErrSelectorNotFound
	}
	if s.values == nil {
		s.values = map[string]string{}
	}
	s.values[request.Selector] = request.Text
	snapshot := s.snapshot()
	snapshot.ActiveTabID = request.TabID
	snapshot.LatestActionSummary = fmt.Sprintf("browser type executed: %s", request.TabID)
	return snapshot, nil
}

func (s *stubBrowserCore) Extract(_ context.Context, request browser.ExtractRequest) (browser.ExtractResult, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.ExtractResult{}, err
	}
	if strings.TrimSpace(request.Selector) == "" || strings.Contains(request.Selector, "missing") {
		return browser.ExtractResult{}, browser.ErrSelectorNotFound
	}
	text := ""
	if s.values != nil {
		text = s.values[request.Selector]
		if text == "" && request.Selector == `[data-testid="result"]` {
			text = s.values[`[data-testid="name"]`]
		}
	}
	snapshot := s.snapshot()
	snapshot.ActiveTabID = request.TabID
	snapshot.LatestActionSummary = fmt.Sprintf("browser extract completed: %s", request.TabID)
	return browser.ExtractResult{Snapshot: snapshot, Text: text}, nil
}

func (s *stubBrowserCore) Screenshot(_ context.Context, request browser.ScreenshotRequest) (browser.ScreenshotResult, error) {
	if _, err := s.lookup(request.TabID); err != nil {
		return browser.ScreenshotResult{}, err
	}
	snapshot := s.snapshot()
	snapshot.ActiveTabID = request.TabID
	snapshot.LatestActionSummary = fmt.Sprintf("browser screenshot captured: %s", request.TabID)
	return browser.ScreenshotResult{Snapshot: snapshot, Bytes: []byte("stub")}, nil
}

func (s *stubBrowserCore) ensureTabs() {
	if s.tabs == nil {
		s.tabs = map[string]browser.Tab{}
	}
}

func (s *stubBrowserCore) lookup(tabID string) (browser.Tab, error) {
	s.ensureTabs()
	tab, ok := s.tabs[tabID]
	if !ok {
		return browser.Tab{}, browser.ErrTabNotFound
	}
	return tab, nil
}

func (s *stubBrowserCore) snapshot() browser.Snapshot {
	s.ensureTabs()
	tabs := make([]browser.Tab, 0, len(s.order))
	for _, id := range s.order {
		tab, ok := s.tabs[id]
		if ok {
			tabs = append(tabs, tab)
		}
	}
	return browser.Snapshot{
		ActiveTabID: s.activeID,
		Tabs:        tabs,
	}
}

func stubBrowserURLAllowed(raw string) bool {
	raw = strings.TrimSpace(raw)
	return strings.HasPrefix(raw, "http://localhost") || strings.HasPrefix(raw, "http://127.0.0.1")
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
	originalNow := recoveryNow
	recoveryNow = func() time.Time {
		return time.Now().UTC().Add(recoveryGracePeriod + time.Second)
	}
	t.Cleanup(func() {
		recoveryNow = originalNow
	})

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

func TestRunnerSkipsFreshRunningTasksDuringRecovery(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Runner",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Fresh task",
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
	require.Equal(t, "running", reloaded.Status)
}

func TestRunnerExecutesWorkspaceReadOnlyTools(t *testing.T) {
	projectRoot := t.TempDir()
	readmePath := filepath.Join(projectRoot, "README.md")
	require.NoError(t, os.WriteFile(readmePath, []byte("hello workspace"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "notes.txt"), []byte("batch text"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(projectRoot, "internal"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "internal", "demo.go"), []byte("package internal\n\nconst target = \"workspace\"\n"), 0o644))

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

	statTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Stat file",
		Kind:  KindWorkspaceStat,
		Input: `{"path":"README.md"}`,
	})
	require.True(t, ok)
	result, err = New(registry, nil).RunTask(context.Background(), thread.ID, statTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "stat README.md: file")

	batchTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Read files batch",
		Kind:  KindWorkspaceReadBatch,
		Input: `{"paths":["README.md","notes.txt"]}`,
	})
	require.True(t, ok)
	result, err = New(registry, nil).RunTask(context.Background(), thread.ID, batchTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "read 2 files")
	require.Contains(t, result.ResultSummary, "README.md")

	filteredTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "List files filtered",
		Kind:  KindWorkspaceListFiltered,
		Input: `{"path":".","pattern":"*.go","includeDirs":false}`,
	})
	require.True(t, ok)
	result, err = New(registry, nil).RunTask(context.Background(), thread.ID, filteredTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "listed 1 filtered entries")
	require.Contains(t, result.ResultSummary, "internal/demo.go")

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

	detailedTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Search text detailed",
		Kind:  KindWorkspaceSearchDetailed,
		Input: `{"query":"workspace","path":".","limit":20}`,
	})
	require.True(t, ok)
	result, err = New(registry, nil).RunTask(context.Background(), thread.ID, detailedTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "found 2 detailed matches")
	require.Contains(t, result.ResultSummary, "README.md:1")
}

func TestRunnerExecutesBrowserTasks(t *testing.T) {
	registry := session.NewRegistry(t.TempDir())
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Browser",
		PermissionMode: policy.ReadOnly,
	})
	runner := New(registry, nil).WithBrowser(&stubBrowserCore{})

	openTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Open tab",
		Kind:  KindBrowserOpen,
		Input: `{"url":"http://localhost:3000"}`,
	})
	require.True(t, ok)
	result, err := runner.RunTask(context.Background(), thread.ID, openTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "browser tab opened: browser-tab-1", result.ResultSummary)

	navigateTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Navigate tab",
		Kind:  KindBrowserNavigate,
		Input: `{"tabId":"browser-tab-1","url":"http://127.0.0.1:4000/page"}`,
	})
	require.True(t, ok)
	result, err = runner.RunTask(context.Background(), thread.ID, navigateTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "browser tab navigated: browser-tab-1", result.ResultSummary)

	for _, testCase := range []struct {
		title   string
		kind    string
		input   string
		summary string
	}{
		{title: "State", kind: KindBrowserState, input: `{}`, summary: "browser state captured"},
		{title: "Back", kind: KindBrowserBack, input: `{"tabId":"browser-tab-1"}`, summary: "browser tab went back: browser-tab-1"},
		{title: "Forward", kind: KindBrowserForward, input: `{"tabId":"browser-tab-1"}`, summary: "browser tab went forward: browser-tab-1"},
		{title: "Reload", kind: KindBrowserReload, input: `{"tabId":"browser-tab-1"}`, summary: "browser tab reloaded: browser-tab-1"},
		{title: "Activate", kind: KindBrowserActivateTab, input: `{"tabId":"browser-tab-1"}`, summary: "browser tab activated: browser-tab-1"},
		{title: "Click", kind: KindBrowserClick, input: `{"tabId":"browser-tab-1","selector":"[data-testid=\"apply\"]"}`, summary: "browser click executed: browser-tab-1"},
		{title: "Type", kind: KindBrowserType, input: `{"tabId":"browser-tab-1","selector":"[data-testid=\"name\"]","text":"browser"}`, summary: "browser type executed: browser-tab-1"},
		{title: "Extract", kind: KindBrowserExtract, input: `{"tabId":"browser-tab-1","selector":"[data-testid=\"result\"]"}`, summary: "browser extract completed: browser-tab-1 | text=browser"},
		{title: "Close", kind: KindBrowserCloseTab, input: `{"tabId":"browser-tab-1"}`, summary: "browser tab closed: browser-tab-1"},
	} {
		task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
			Title: testCase.title,
			Kind:  testCase.kind,
			Input: testCase.input,
		})
		require.True(t, ok)
		result, err = runner.RunTask(context.Background(), thread.ID, task.ID)
		require.NoError(t, err)
		require.Equal(t, "completed", result.Status)
		require.Equal(t, testCase.summary, result.ResultSummary)
	}

	toolCalls, ok := registry.ToolCalls(thread.ID)
	require.True(t, ok)
	require.Len(t, toolCalls, 22)
	require.Equal(t, KindBrowserOpen, toolCalls[0].ToolID)
	require.Equal(t, "running", toolCalls[0].Status)
	require.Equal(t, "completed", toolCalls[15].Status)
}

func TestRunnerExecutesBrowserScreenshotTaskAndAppendsArtifact(t *testing.T) {
	registry := session.NewRegistry(t.TempDir())
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Browser Screenshot",
		PermissionMode: policy.ReadOnly,
	})
	runner := New(registry, nil).WithBrowser(&stubBrowserCore{})

	openTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Open tab",
		Kind:  KindBrowserOpen,
		Input: `{"url":"http://localhost:3000"}`,
	})
	require.True(t, ok)
	_, err := runner.RunTask(context.Background(), thread.ID, openTask.ID)
	require.NoError(t, err)

	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Screenshot",
		Kind:  KindBrowserScreenshot,
		Input: `{"tabId":"browser-tab-1"}`,
	})
	require.True(t, ok)

	result, err := runner.RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "browser screenshot captured:")

	artifacts, ok := registry.Artifacts(thread.ID)
	require.True(t, ok)
	require.Len(t, artifacts, 1)
	require.Equal(t, "browser.screenshot", artifacts[0].Kind)
	require.FileExists(t, artifacts[0].Path)
}

func TestRunnerFailsBrowserOpenForBadURL(t *testing.T) {
	registry := session.NewRegistry(t.TempDir())
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Browser Bad URL",
		PermissionMode: policy.ReadOnly,
	})
	runner := New(registry, nil).WithBrowser(&stubBrowserCore{})

	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Open disallowed URL",
		Kind:  KindBrowserOpen,
		Input: `{"url":"https://example.com"}`,
	})
	require.True(t, ok)

	result, err := runner.RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ResultSummary, browser.ErrURLNotAllowed.Error())

	toolCalls, ok := registry.ToolCalls(thread.ID)
	require.True(t, ok)
	require.Len(t, toolCalls, 2)
	require.Equal(t, "running", toolCalls[0].Status)
	require.Equal(t, "failed", toolCalls[1].Status)
	require.Contains(t, toolCalls[1].Summary, browser.ErrURLNotAllowed.Error())
}

func TestRunnerFailsBrowserTaskForMissingTab(t *testing.T) {
	registry := session.NewRegistry(t.TempDir())
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Browser Missing Tab",
		PermissionMode: policy.ReadOnly,
	})
	runner := New(registry, nil).WithBrowser(&stubBrowserCore{})

	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Reload missing tab",
		Kind:  KindBrowserReload,
		Input: `{"tabId":"browser-tab-404"}`,
	})
	require.True(t, ok)

	result, err := runner.RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ResultSummary, browser.ErrTabNotFound.Error())

	toolCalls, ok := registry.ToolCalls(thread.ID)
	require.True(t, ok)
	require.Len(t, toolCalls, 2)
	require.Equal(t, "running", toolCalls[0].Status)
	require.Equal(t, "failed", toolCalls[1].Status)
	require.Contains(t, toolCalls[1].Summary, browser.ErrTabNotFound.Error())
}

func TestRunnerAllowsReadToolForAskUserMode(t *testing.T) {
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
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "hello")
}

func TestRunnerExecutesMCPToolInvokeTask(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "MCP Runner",
		PermissionMode: policy.ReadOnly,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Invoke external fixture",
		Kind:  KindMCPToolInvoke,
		Input: `{"serverId":"external-fixture","toolName":"echo","arguments":{"message":"hello"}}`,
	})
	require.True(t, ok)

	manager := mcp.NewManager([]mcp.ServerDescriptor{{
		ID:            "external-fixture",
		Source:        "fixture",
		Enabled:       true,
		ToolCount:     2,
		ResourceCount: 0,
		Status:        "enabled",
		Command:       runnerFixtureCommand(t),
		Tools:         []string{"echo", "sum", "fail"},
	}})

	result, err := New(registry, nil).WithMCP(manager).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "mcp tool external-fixture/echo executed", result.ResultSummary)

	toolCalls, ok := registry.ToolCalls(thread.ID)
	require.True(t, ok)
	require.Len(t, toolCalls, 2)
	require.Equal(t, "running", toolCalls[0].Status)
	require.Equal(t, "completed", toolCalls[1].Status)

	approvals, ok := registry.Approvals(thread.ID)
	require.True(t, ok)
	require.Len(t, approvals, 0)
	writeExecutions, ok := registry.WriteExecutions(thread.ID)
	require.True(t, ok)
	require.Len(t, writeExecutions, 0)
}

func TestRunnerExecutesSDKMCPToolInvokeTask(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "MCP SDK Runner",
		PermissionMode: policy.ReadOnly,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Invoke sdk fixture",
		Kind:  KindMCPToolInvoke,
		Input: `{"serverId":"sdk-external-fixture","toolName":"echo","arguments":{"message":"hello-sdk"}}`,
	})
	require.True(t, ok)

	manager := mcp.NewManager([]mcp.ServerDescriptor{{
		ID:            "sdk-external-fixture",
		Source:        "sdk",
		Enabled:       true,
		ToolCount:     2,
		ResourceCount: 0,
		Status:        "enabled",
		Command:       []string{"node", filepath.Join(runnerRepoRoot(t), "scripts", "mcp_sdk_server.js")},
		Tools:         []string{"echo", "sum"},
		Transport:     "stdio-sdk",
	}})

	result, err := New(registry, nil).WithMCP(manager).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "mcp tool sdk-external-fixture/echo executed", result.ResultSummary)
}

func TestRunnerExecutesThirdPartyTimeMCPToolInvokeTask(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "MCP Third Party Runner",
		PermissionMode: policy.ReadOnly,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Invoke third-party time",
		Kind:  KindMCPToolInvoke,
		Input: `{"serverId":"third-party-time","toolName":"get_current_time","arguments":{"timezone":"UTC"}}`,
	})
	require.True(t, ok)

	manager := mcp.NewManager([]mcp.ServerDescriptor{{
		ID:            "third-party-time",
		Source:        "third-party",
		Enabled:       true,
		ToolCount:     1,
		ResourceCount: 0,
		Status:        "enabled",
		Command:       []string{"node", filepath.Join(runnerRepoRoot(t), "scripts", "mcp_third_party_time_server.js")},
		Tools:         []string{"get_current_time"},
		Transport:     "stdio-third-party",
	}})

	result, err := New(registry, nil).WithMCP(manager).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "mcp tool third-party-time/get_current_time executed", result.ResultSummary)
}

func TestRunnerFailsMCPToolInvokeForUnknownServer(t *testing.T) {
	registry := session.NewRegistry(`D:\GOWorks\gen-code-heji\gen-code`)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "MCP Failure",
		PermissionMode: policy.ReadOnly,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Invoke missing server",
		Kind:  KindMCPToolInvoke,
		Input: `{"serverId":"missing","toolName":"echo","arguments":{}}`,
	})
	require.True(t, ok)

	result, err := New(registry, nil).WithMCP(mcp.NewManager(nil)).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ResultSummary, "mcp server not found")

	toolCalls, ok := registry.ToolCalls(thread.ID)
	require.True(t, ok)
	require.Len(t, toolCalls, 2)
	require.Equal(t, "failed", toolCalls[1].Status)
	require.Contains(t, toolCalls[1].Summary, "mcp server not found")
}

func runnerFixtureCommand(t *testing.T) []string {
	t.Helper()
	scriptPath := filepath.Join(runnerRepoRoot(t), "scripts", "mcp_stdio_fixture.py")
	if runtime.GOOS == "windows" {
		return []string{"python", scriptPath}
	}
	return []string{"python3", scriptPath}
}

func runnerRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
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

func TestRunnerRecordsWriteExecutionForWorkspaceApplyPatch(t *testing.T) {
	projectRoot := t.TempDir()
	targetPath := filepath.Join(projectRoot, "README.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("old\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Writer",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Patch README",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)

	result, err := New(registry, nil).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "applied patch to README.md: updated 2 line(s)", result.ResultSummary)

	writeExecutions, ok := registry.WriteExecutions(thread.ID)
	require.True(t, ok)
	require.Len(t, writeExecutions, 1)
	require.Equal(t, task.ID, writeExecutions[0].TaskID)
	require.Equal(t, "completed", writeExecutions[0].Status)
	require.Equal(t, []string{"README.md"}, writeExecutions[0].TargetPaths)
	require.Equal(t, "2 patch line(s)", writeExecutions[0].PatchSummary)
	require.Equal(t, result.ResultSummary, writeExecutions[0].ResultSummary)
	require.NotEmpty(t, writeExecutions[0].PatchHash)
	require.Contains(t, writeExecutions[0].BeforeSnapshotSummary, "exists")
	require.Contains(t, writeExecutions[0].AfterSnapshotSummary, "exists")
}

func TestRunnerRejectTaskDoesNotCreateWriteExecution(t *testing.T) {
	registry := session.NewRegistry(t.TempDir())
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Approval",
		PermissionMode: policy.AskUser,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Patch README",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Add File: README.md\n+hello\n*** End Patch"}`,
		Status:         "needs_approval",
		ResultSummary:  "approval required for workspace.apply_patch on README.md; 1 patch line(s)",
		ApprovalStatus: "pending",
	})
	require.True(t, ok)
	_, err := registry.CreateApproval(thread.ID, session.CreateApprovalInput{
		TaskID:      task.ID,
		ToolKind:    KindWorkspaceApplyPatch,
		Status:      "pending",
		Summary:     "approval required for workspace.apply_patch on README.md; 1 patch line(s)",
		TargetPaths: []string{"README.md"},
	})
	require.NoError(t, err)

	result, err := New(registry, nil).RejectTask(thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Equal(t, "rejected", result.ApprovalStatus)

	writeExecutions, ok := registry.WriteExecutions(thread.ID)
	require.True(t, ok)
	require.Len(t, writeExecutions, 0)
}

func TestRunnerRollsBackLatestUpdatedFileExecution(t *testing.T) {
	projectRoot := t.TempDir()
	targetPath := filepath.Join(projectRoot, "README.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("old\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Rollback",
		PermissionMode: policy.WorkspaceWrite,
	})

	applyTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Apply patch",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)

	applyResult, err := New(registry, nil).RunTask(context.Background(), thread.ID, applyTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", applyResult.Status)

	applyExecutions, ok := registry.WriteExecutions(thread.ID)
	require.True(t, ok)
	require.Len(t, applyExecutions, 1)
	require.Equal(t, "apply", applyExecutions[0].Operation)
	require.Len(t, applyExecutions[0].RollbackPayload, 1)
	require.Equal(t, "old\n", applyExecutions[0].RollbackPayload[0].BeforeContent)

	rollbackInput := `{"writeExecutionId":"` + applyExecutions[0].ID + `"}`
	rollbackTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Rollback patch",
		Kind:           KindWorkspaceApplyPatchRollback,
		Input:          rollbackInput,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)

	rollbackResult, err := New(registry, nil).RunTask(context.Background(), thread.ID, rollbackTask.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", rollbackResult.Status)
	require.Contains(t, rollbackResult.ResultSummary, "rolled back patch on README.md")

	content, err := os.ReadFile(targetPath)
	require.NoError(t, err)
	require.Equal(t, "old\n", string(content))

	writeExecutions, ok := registry.WriteExecutions(thread.ID)
	require.True(t, ok)
	require.Len(t, writeExecutions, 2)
	require.Equal(t, "rollback", writeExecutions[1].Operation)
	require.Equal(t, writeExecutions[0].ID, writeExecutions[1].RelatedExecutionID)
	require.Equal(t, rollbackTask.ID, writeExecutions[1].TaskID)
}

func TestRunnerRollsBackLatestAddedFileExecutionByDeletingFile(t *testing.T) {
	projectRoot := t.TempDir()
	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Rollback Add",
		PermissionMode: policy.WorkspaceWrite,
	})

	applyTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Add file",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"docs/sample.txt","patch":"*** Begin Patch\n*** Add File: docs/sample.txt\n+hello\n*** End Patch"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)
	_, err := New(registry, nil).RunTask(context.Background(), thread.ID, applyTask.ID)
	require.NoError(t, err)

	applyExecutions, ok := registry.WriteExecutions(thread.ID)
	require.True(t, ok)
	require.Len(t, applyExecutions, 1)

	rollbackTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Rollback add",
		Kind:           KindWorkspaceApplyPatchRollback,
		Input:          `{"writeExecutionId":"` + applyExecutions[0].ID + `"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)
	_, err = New(registry, nil).RunTask(context.Background(), thread.ID, rollbackTask.ID)
	require.NoError(t, err)

	_, statErr := os.Stat(filepath.Join(projectRoot, "docs", "sample.txt"))
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestRunnerExecutesAgentRunWithReadAndRespond(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("hello agent\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"Read README and answer","maxSteps":4}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"read_file","path":"README.md","reasoningSummary":"Read the file first"}`},
			{OutputText: `{"type":"respond","response":"README says hello agent.","reasoningSummary":"Done"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "agent completed")

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 2)
	require.Equal(t, KindWorkspaceRead, tasks[1].Kind)
	require.Equal(t, task.ID, tasks[1].ParentTaskID)

	messages, ok := registry.Messages(thread.ID)
	require.True(t, ok)
	require.NotEmpty(t, messages)
	require.Equal(t, "assistant", messages[len(messages)-1].Role)
	require.Equal(t, "README says hello agent.", messages[len(messages)-1].Content)
}

func TestRunnerExecutesAgentRunWithBrowserSequence(t *testing.T) {
	registry := session.NewRegistry(t.TempDir())
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Browser",
		PermissionMode: policy.ReadOnly,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent browser run",
		Kind:  KindAgentRun,
		Input: `{"goal":"Open the local preview in the browser, type into the form, extract the result, capture a screenshot, then answer","maxSteps":6}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"browser_open","url":"http://127.0.0.1:3000","reasoningSummary":"Open the local preview"}`},
			{OutputText: `{"type":"browser_type","tabId":"browser-tab-1","selector":"input","text":"browser","reasoningSummary":"Fill the input"}`},
			{OutputText: `{"type":"browser_click","tabId":"browser-tab-1","selector":"button","reasoningSummary":"Apply the form"}`},
			{OutputText: `{"type":"browser_extract","tabId":"browser-tab-1","selector":"#result","reasoningSummary":"Read the result"}`},
			{OutputText: `{"type":"browser_screenshot","tabId":"browser-tab-1","reasoningSummary":"Capture a screenshot"}`},
			{OutputText: `{"type":"respond","response":"Browser workflow complete.","reasoningSummary":"Done"}`},
		},
	}

	runner := New(registry, models).WithBrowser(&stubBrowserCore{})
	result, err := runner.RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "agent completed")
	resultState, err := parseAgentRunState(result.AgentState)
	require.NoError(t, err)
	require.Equal(t, "browser_then_respond", resultState.Plan.Mode)

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 6)
	require.Equal(t, KindBrowserOpen, tasks[1].Kind)
	var browserOpenPayload map[string]string
	require.NoError(t, json.Unmarshal([]byte(tasks[1].Input), &browserOpenPayload))
	require.Equal(t, "http://127.0.0.1:5174/?gcPreview=1&pane=acceptance-browser&threadId=thread-1", browserOpenPayload["url"])
	require.Equal(t, KindBrowserType, tasks[2].Kind)
	var browserTypePayload map[string]string
	require.NoError(t, json.Unmarshal([]byte(tasks[2].Input), &browserTypePayload))
	require.Equal(t, "[data-testid='controlled-browser-input']", browserTypePayload["selector"])
	require.Equal(t, KindBrowserClick, tasks[3].Kind)
	var browserClickPayload map[string]string
	require.NoError(t, json.Unmarshal([]byte(tasks[3].Input), &browserClickPayload))
	require.Equal(t, "[data-testid='controlled-browser-apply']", browserClickPayload["selector"])
	require.Equal(t, KindBrowserExtract, tasks[4].Kind)
	var browserExtractPayload map[string]string
	require.NoError(t, json.Unmarshal([]byte(tasks[4].Input), &browserExtractPayload))
	require.Equal(t, "[data-testid='controlled-browser-result']", browserExtractPayload["selector"])
	require.Equal(t, KindBrowserScreenshot, tasks[5].Kind)
	require.Equal(t, task.ID, tasks[5].ParentTaskID)

	artifacts, ok := registry.Artifacts(thread.ID)
	require.True(t, ok)
	require.Len(t, artifacts, 1)
	require.Equal(t, "browser.screenshot", artifacts[0].Kind)

	messages, ok := registry.Messages(thread.ID)
	require.True(t, ok)
	require.NotEmpty(t, messages)
	require.Equal(t, "Browser workflow complete.", messages[len(messages)-1].Content)
}

func TestDeriveAgentExecutionPlanUsesBrowserThenRespondForControlledBrowserGoal(t *testing.T) {
	plan := DeriveAgentExecutionPlanForRuntime("Open the controlled browser fixture, type browser demo text, click apply, extract the result, and reply.")
	require.Equal(t, "browser_then_respond", plan.Mode)
	require.Equal(t, []string{"browser_open", "browser_type", "browser_click", "browser_extract", "respond"}, plan.RequiredSequence)
	require.Equal(t, "Open the controlled browser page", plan.Steps[0].Title)
	require.Equal(t, "Answer with the browser result", plan.Steps[len(plan.Steps)-1].Title)
}

func TestDeriveAgentExecutionPlanAddsOptionalBrowserScreenshotStep(t *testing.T) {
	plan := DeriveAgentExecutionPlanForRuntime("Open the local preview, type into the form, click apply, extract the result, take a screenshot, and reply.")
	require.Equal(t, "browser_then_respond", plan.Mode)
	require.Equal(t, []string{"browser_open", "browser_type", "browser_click", "browser_extract", "browser_screenshot", "respond"}, plan.RequiredSequence)
	require.Equal(t, "Capture a browser screenshot", plan.Steps[4].Title)
}

func TestNormalizeAgentActionForStateRewritesBrowserOpenToCanonicalFixture(t *testing.T) {
	action := normalizeAgentActionForState(
		AgentAction{
			Type: "browser_open",
			URL:  "http://127.0.0.1:3000",
		},
		AgentRunState{
			ThreadID: "thread-acceptance",
			Plan: AgentExecutionPlan{
				Mode: "browser_then_respond",
			},
		},
	)

	require.Equal(t, "http://127.0.0.1:5174/?gcPreview=1&pane=acceptance-browser&threadId=thread-acceptance", action.URL)
}

func TestBuildAgentPromptIncludesCanonicalBrowserFixtureGuidance(t *testing.T) {
	prompt := buildAgentPrompt(nil, nil, AgentRunState{
		ThreadID: "thread-browser",
		Goal:     "Open the controlled browser fixture and reply.",
		MaxSteps: 6,
		Plan: AgentExecutionPlan{
			Summary:          "Open the controlled browser page, interact with it, extract the stable result, and then answer.",
			Mode:             "browser_then_respond",
			RequiredSequence: []string{"browser_open", "browser_type", "browser_click", "browser_extract", "respond"},
		},
		CurrentStepTitle: "Open the controlled browser page",
	})

	require.Contains(t, prompt, "Browser fixture guidance:")
	require.Contains(t, prompt, "http://127.0.0.1:5174/?gcPreview=1&pane=acceptance-browser&threadId=thread-browser")
	require.Contains(t, prompt, "Do not substitute another localhost port such as 3000")
	require.Contains(t, prompt, "[data-testid='controlled-browser-input']")
	require.Contains(t, prompt, "[data-testid='controlled-browser-apply']")
	require.Contains(t, prompt, "[data-testid='controlled-browser-result']")
}

func TestNormalizeAgentActionForStateRewritesGenericControlledBrowserSelectors(t *testing.T) {
	state := AgentRunState{
		ThreadID: "thread-browser",
		Plan: AgentExecutionPlan{
			Mode: "browser_then_respond",
		},
	}

	typeExpectation := map[string]string{
		"browser_type":    "[data-testid='controlled-browser-input']",
		"browser_click":   "[data-testid='controlled-browser-apply']",
		"browser_extract": "[data-testid='controlled-browser-result']",
	}
	inputSelectors := map[string]string{
		"browser_type":    "input",
		"browser_click":   "button",
		"browser_extract": "[data-testid='stable-result']",
	}

	for actionType, expectedSelector := range typeExpectation {
		action := normalizeAgentActionForState(AgentAction{Type: actionType, Selector: inputSelectors[actionType]}, state)
		require.Equal(t, expectedSelector, action.Selector)
	}
}

func TestValidateAgentActionSequenceRejectsBrowserClickBeforeType(t *testing.T) {
	err := validateAgentActionSequence(
		AgentRunState{
			Plan: AgentExecutionPlan{
				Mode:             "browser_then_respond",
				RequiredSequence: []string{"browser_open", "browser_type", "browser_click", "browser_extract", "respond"},
			},
			CompletedActions: []string{"browser_open"},
		},
		AgentAction{Type: "browser_click"},
	)
	require.EqualError(t, err, "agent action violates required sequence: browser_type must happen before browser_click")
}

func TestValidateAgentActionSequenceRejectsBrowserRespondBeforeExtract(t *testing.T) {
	err := validateAgentActionSequence(
		AgentRunState{
			Plan: AgentExecutionPlan{
				Mode:             "browser_then_respond",
				RequiredSequence: []string{"browser_open", "browser_type", "browser_click", "browser_extract", "respond"},
			},
			CompletedActions: []string{"browser_open", "browser_type", "browser_click"},
		},
		AgentAction{Type: "respond", Response: "done"},
	)
	require.EqualError(t, err, "agent action violates required sequence: browser_extract must happen before respond")
}

func TestRunnerExecutesAgentRunWithSecondBatchReadTools(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("hello agent\n"), 0o644))
	require.NoError(t, os.Mkdir(filepath.Join(projectRoot, "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "pkg", "demo.go"), []byte("package pkg\n\nconst target = \"hello agent\"\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Batch",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"Inspect workspace and answer","maxSteps":5}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"stat_file","path":"README.md","reasoningSummary":"Check the file first"}`},
			{OutputText: `{"type":"search_text_detailed","query":"hello agent","path":".","limit":20,"reasoningSummary":"Find all references"}`},
			{OutputText: `{"type":"respond","response":"Inspection complete.","reasoningSummary":"Done"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "agent completed")

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 3)
	require.Equal(t, KindWorkspaceStat, tasks[1].Kind)
	require.Equal(t, KindWorkspaceSearchDetailed, tasks[2].Kind)
}

func TestRunnerAgentRunWaitsForApprovalAndResumesAfterApprove(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("old\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Approval",
		PermissionMode: policy.AskUser,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent patch",
		Kind:  KindAgentRun,
		Input: `{"goal":"Patch README and confirm","maxSteps":4}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"apply_patch","path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch","reasoningSummary":"Apply the required patch"}`},
			{OutputText: `{"type":"respond","response":"Patch applied and verified.","reasoningSummary":"Done"}`},
		},
	}

	runner := New(registry, models)
	waiting, err := runner.RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "waiting_for_approval", waiting.Status)
	require.Equal(t, waitingStatusApproval, waiting.WaitingStatus)
	waitingState, err := parseAgentRunState(waiting.AgentState)
	require.NoError(t, err)
	require.Equal(t, "patch_then_respond", waitingState.Plan.Mode)
	require.Equal(t, "Answer with the result", waitingState.CurrentStepTitle)

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 2)
	child := tasks[1]
	require.Equal(t, KindWorkspaceApplyPatch, child.Kind)
	require.Equal(t, "needs_approval", child.Status)
	require.Equal(t, task.ID, child.ParentTaskID)

	resumed, err := runner.ApproveTask(context.Background(), thread.ID, child.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", resumed.Status)
	require.Contains(t, resumed.ResultSummary, "agent completed")

	reloadedParent, err := registry.Task(thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedParent.Status)
	reloadedState, err := parseAgentRunState(reloadedParent.AgentState)
	require.NoError(t, err)
	require.Equal(t, "patch_then_respond", reloadedState.Plan.Mode)
	require.Equal(t, "Answer with the result", reloadedState.CurrentStepTitle)
	require.Equal(t, "completed", reloadedState.Status)
	require.Empty(t, reloadedState.WaitingChildTaskID)
	require.Empty(t, reloadedState.FailureReason)

	content, err := os.ReadFile(filepath.Join(projectRoot, "README.md"))
	require.NoError(t, err)
	require.Equal(t, "new", string(content))
}

func TestRunnerRecoverInterruptedAgentRunResumesWaitingChildCompletion(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("hello\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Recover Agent",
		PermissionMode: policy.WorkspaceWrite,
	})
	parent, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:         "Agent recover",
		Kind:          KindAgentRun,
		Input:         `{"goal":"Read README and answer","maxSteps":3}`,
		Status:        "waiting_for_task",
		WaitingStatus: waitingStatusTask,
		AgentState:    `{"taskId":"task-1","threadId":"thread-1","stepIndex":1,"maxSteps":3,"waitingChildTaskId":"task-2","lastAction":{"type":"read_file","path":"README.md"},"status":"waiting_for_task","goal":"Read README and answer"}`,
	})
	require.True(t, ok)
	child, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:         "Read file README.md",
		Kind:          KindWorkspaceRead,
		Input:         `{"path":"README.md"}`,
		Status:        "completed",
		ResultSummary: "read README.md: hello",
		ParentTaskID:  parent.ID,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"respond","response":"Recovered and finished.","reasoningSummary":"Done"}`},
		},
	}

	runner := New(registry, models)
	err := runner.RecoverInterruptedTasks()
	require.NoError(t, err)

	reloadedParent, err := registry.Task(thread.ID, parent.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedParent.Status)
	require.Contains(t, reloadedParent.ResultSummary, "agent completed")
	reloadedState, err := parseAgentRunState(reloadedParent.AgentState)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedState.Status)
	require.Empty(t, reloadedState.WaitingChildTaskID)

	reloadedChild, err := registry.Task(thread.ID, child.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedChild.Status)
}

func TestRunnerRecoverInterruptedAgentRunKeepsWaitingApprovalWhenChildStillPending(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("old\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Recover Approval Pending",
		PermissionMode: policy.AskUser,
	})
	parent, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent recover approval",
		Kind:  KindAgentRun,
		Input: `{"goal":"Patch README and confirm","maxSteps":4}`,
	})
	require.True(t, ok)
	child, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Apply patch README.md",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch"}`,
		Status:         "needs_approval",
		ResultSummary:  "approval required for README.md; 1 patch line(s)",
		ApprovalStatus: "pending",
		ParentTaskID:   parent.ID,
	})
	require.True(t, ok)
	agentState := fmt.Sprintf(`{"taskId":%q,"threadId":%q,"stepIndex":1,"maxSteps":4,"waitingChildTaskId":%q,"lastAction":{"type":"apply_patch","path":"README.md"},"status":"waiting_for_approval","goal":"Patch README and confirm","currentStepTitle":"Answer with the result","completedActions":["apply_patch"],"plan":{"summary":"Apply the requested patch first, then answer with the result.","mode":"patch_then_respond","requiredSequence":["apply_patch","respond"]}}`, parent.ID, thread.ID, child.ID)
	_, err := registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "waiting_for_approval",
		ResultSummary: child.ResultSummary,
		WaitingStatus: strPtr(waitingStatusApproval),
		AgentState:    strPtr(agentState),
	})
	require.NoError(t, err)

	runner := New(registry, &scriptedModelExecutor{})
	err = runner.RecoverInterruptedTasks()
	require.NoError(t, err)

	reloadedParent, err := registry.Task(thread.ID, parent.ID)
	require.NoError(t, err)
	require.Equal(t, "waiting_for_approval", reloadedParent.Status)
	require.Equal(t, waitingStatusApproval, reloadedParent.WaitingStatus)
	reloadedState, err := parseAgentRunState(reloadedParent.AgentState)
	require.NoError(t, err)
	require.Equal(t, waitingStatusApproval, reloadedState.Status)
	require.Equal(t, child.ID, reloadedState.WaitingChildTaskID)

	reloadedChild, err := registry.Task(thread.ID, child.ID)
	require.NoError(t, err)
	require.Equal(t, "needs_approval", reloadedChild.Status)
}

func TestRunnerRecoverInterruptedAgentRunResumesAfterApprovedChildCompleted(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("new"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Recover Approval Completed",
		PermissionMode: policy.AskUser,
	})
	parent, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent recover approved child",
		Kind:  KindAgentRun,
		Input: `{"goal":"Patch README and confirm","maxSteps":4}`,
	})
	require.True(t, ok)
	child, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Apply patch README.md",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch"}`,
		Status:         "completed",
		ResultSummary:  "applied patch to README.md: updated 1 line(s)",
		ApprovalStatus: "executed",
		ParentTaskID:   parent.ID,
	})
	require.True(t, ok)
	agentState := fmt.Sprintf(`{"taskId":%q,"threadId":%q,"stepIndex":1,"maxSteps":4,"waitingChildTaskId":%q,"lastAction":{"type":"apply_patch","path":"README.md"},"status":"waiting_for_approval","goal":"Patch README and confirm","currentStepTitle":"Answer with the result","completedActions":["apply_patch"],"plan":{"summary":"Apply the requested patch first, then answer with the result.","mode":"patch_then_respond","requiredSequence":["apply_patch","respond"]}}`, parent.ID, thread.ID, child.ID)
	_, err := registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "waiting_for_approval",
		ResultSummary: child.ResultSummary,
		WaitingStatus: strPtr(waitingStatusApproval),
		AgentState:    strPtr(agentState),
	})
	require.NoError(t, err)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"respond","response":"Patch applied and verified.","reasoningSummary":"Done"}`},
		},
	}

	runner := New(registry, models)
	err = runner.RecoverInterruptedTasks()
	require.NoError(t, err)

	reloadedParent, err := registry.Task(thread.ID, parent.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedParent.Status)
	require.Contains(t, reloadedParent.ResultSummary, "agent completed")
	reloadedState, err := parseAgentRunState(reloadedParent.AgentState)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedState.Status)
	require.Empty(t, reloadedState.WaitingChildTaskID)

	reloadedChild, err := registry.Task(thread.ID, child.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedChild.Status)
}

func TestRunnerRecoverInterruptedAgentRunFailsWhenApprovalChildMissing(t *testing.T) {
	registry := session.NewRegistry(t.TempDir())
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Recover Approval Missing Child",
		PermissionMode: policy.AskUser,
	})
	parent, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent recover missing child",
		Kind:  KindAgentRun,
		Input: `{"goal":"Patch README and confirm","maxSteps":4}`,
	})
	require.True(t, ok)
	agentState := fmt.Sprintf(`{"taskId":%q,"threadId":%q,"stepIndex":1,"maxSteps":4,"waitingChildTaskId":"task-missing","lastAction":{"type":"apply_patch","path":"README.md"},"status":"waiting_for_approval","goal":"Patch README and confirm","currentStepTitle":"Answer with the result","completedActions":["apply_patch"],"plan":{"summary":"Apply the requested patch first, then answer with the result.","mode":"patch_then_respond","requiredSequence":["apply_patch","respond"]}}`, parent.ID, thread.ID)
	_, err := registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "waiting_for_approval",
		ResultSummary: "approval required for README.md",
		WaitingStatus: strPtr(waitingStatusApproval),
		AgentState:    strPtr(agentState),
	})
	require.NoError(t, err)

	runner := New(registry, &scriptedModelExecutor{})
	err = runner.RecoverInterruptedTasks()
	require.NoError(t, err)

	reloadedParent, err := registry.Task(thread.ID, parent.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", reloadedParent.Status)
	require.Contains(t, reloadedParent.ResultSummary, "approval child task not found")
	reloadedState, err := parseAgentRunState(reloadedParent.AgentState)
	require.NoError(t, err)
	require.Equal(t, "failed", reloadedState.Status)
	require.Contains(t, reloadedState.FailureReason, "approval child task not found")
}

func TestRunnerResumeAgentRunCompletesAfterApprovedPatchStep(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("old\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Resume Approval Again",
		PermissionMode: policy.AskUser,
	})
	parent, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:         "Agent resume approval",
		Kind:          KindAgentRun,
		Input:         `{"goal":"Patch README and confirm","maxSteps":4}`,
		Status:        "waiting_for_approval",
		WaitingStatus: waitingStatusApproval,
		AgentState:    `{"taskId":"task-agent","threadId":"thread-1","stepIndex":1,"maxSteps":4,"waitingChildTaskId":"task-child","lastAction":{"type":"apply_patch","path":"README.md"},"status":"waiting_for_approval","goal":"Patch README and confirm","currentStepTitle":"Answer with the result","completedActions":["apply_patch"],"plan":{"summary":"Apply the requested patch first, then answer with the result.","mode":"patch_then_respond","requiredSequence":["apply_patch","respond"]}}`,
	})
	require.True(t, ok)
	child, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Apply patch README.md",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch"}`,
		Status:         "completed",
		ResultSummary:  "applied patch to README.md: updated 2 line(s)",
		ApprovalStatus: "executed",
		ParentTaskID:   parent.ID,
	})
	require.True(t, ok)
	parentState := fmt.Sprintf(`{"taskId":%q,"threadId":%q,"stepIndex":1,"maxSteps":4,"waitingChildTaskId":%q,"lastAction":{"type":"apply_patch","path":"README.md"},"status":"waiting_for_approval","goal":"Patch README and confirm","currentStepTitle":"Answer with the result","completedActions":["apply_patch"],"plan":{"summary":"Apply the requested patch first, then answer with the result.","mode":"patch_then_respond","requiredSequence":["apply_patch","respond"]}}`, parent.ID, thread.ID, child.ID)
	_, err := registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "waiting_for_approval",
		ResultSummary: child.ResultSummary,
		WaitingStatus: strPtr(waitingStatusApproval),
		AgentState:    strPtr(parentState),
	})
	require.NoError(t, err)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"respond","response":"Patch applied and confirmed.","reasoningSummary":"Report the result"}`},
		},
	}

	runner := New(registry, models)
	resumed, err := runner.resumeAgentRun(context.Background(), thread.ID, parent)
	require.NoError(t, err)
	require.Equal(t, "completed", resumed.Status)
	require.Contains(t, resumed.ResultSummary, "agent completed")

	reloadedParent, err := registry.Task(thread.ID, parent.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedParent.Status)
	reloadedState, err := parseAgentRunState(reloadedParent.AgentState)
	require.NoError(t, err)
	require.Equal(t, "completed", reloadedState.Status)
	require.Empty(t, reloadedState.WaitingChildTaskID)

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 2)
}

func TestShouldRecoverRunningTask(t *testing.T) {
	originalNow := recoveryNow
	recoveryNow = func() time.Time {
		return time.Date(2026, 5, 17, 5, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		recoveryNow = originalNow
	})

	require.False(t, shouldRecoverRunningTask(session.Task{
		Status:    "running",
		UpdatedAt: recoveryNow(),
	}))
	require.True(t, shouldRecoverRunningTask(session.Task{
		Status:    "running",
		UpdatedAt: recoveryNow().Add(-recoveryGracePeriod - time.Second),
	}))
}

func TestParseAgentActionExtractsFirstJSONObject(t *testing.T) {
	action, err := parseAgentAction("I will do this now.\n{\"type\":\"respond\",\"response\":\"done\",\"reasoningSummary\":\"ok\"}\n{\"ignored\":true}")
	require.NoError(t, err)
	require.Equal(t, "respond", action.Type)
	require.Equal(t, "done", action.Response)
}

func TestParseAgentActionAcceptsResponseAlias(t *testing.T) {
	action, err := parseAgentAction(`{"type":"response","response":"done","reasoningSummary":"ok"}`)
	require.NoError(t, err)
	require.Equal(t, "respond", action.Type)
	require.Equal(t, "done", action.Response)
}

func TestParseAgentActionAcceptsResultAlias(t *testing.T) {
	action, err := parseAgentAction(`{"type":"result","response":"done","reasoningSummary":"ok"}`)
	require.NoError(t, err)
	require.Equal(t, "respond", action.Type)
	require.Equal(t, "done", action.Response)
}

func TestParseAgentActionAcceptsToolResultAlias(t *testing.T) {
	action, err := parseAgentAction(`{"type":"tool_result","response":"done","reasoningSummary":"ok"}`)
	require.NoError(t, err)
	require.Equal(t, "respond", action.Type)
	require.Equal(t, "done", action.Response)
}

func TestParseAgentActionTreatsToolCallAsInferableAlias(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"type":"tool_call","path":"go.mod","reasoningSummary":"Inspect file metadata first"}`,
		AgentRunState{
			Plan: AgentExecutionPlan{
				Steps: []AgentPlanStep{
					{Title: "Check file status", ExpectedActionTypes: []string{"stat_file"}},
				},
			},
			CurrentStepTitle: "Check file status",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "stat_file", action.Type)
	require.Equal(t, "go.mod", action.Path)
}

func TestParseAgentActionNormalizesWorkspaceToolKind(t *testing.T) {
	action, err := parseAgentAction(`{"type":"workspace.read_files_batch","paths":["go.mod"],"reasoningSummary":"Read file content next"}`)
	require.NoError(t, err)
	require.Equal(t, "read_files_batch", action.Type)
	require.Equal(t, []string{"go.mod"}, action.Paths)
}

func TestParseAgentActionInfersDetailedSearchTypeFromCurrentStep(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"query":"KindWorkspaceStat","path":"internal/core/runner","limit":20,"reasoningSummary":"Inspect line hits"}`,
		AgentRunState{
			Plan: AgentExecutionPlan{
				Steps: []AgentPlanStep{
					{Title: "Search for broad matches", ExpectedActionTypes: []string{"search_text"}},
					{Title: "Inspect detailed matches", ExpectedActionTypes: []string{"search_text_detailed"}},
				},
			},
			CompletedActions: []string{"search_text"},
			CurrentStepTitle: "Inspect detailed matches",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "search_text_detailed", action.Type)
}

func TestParseAgentActionDetailedSearchInheritsPreviousQueryAndPath(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"type":"search_text_detailed","limit":20,"reasoningSummary":"Inspect line hits"}`,
		AgentRunState{
			LastAction: AgentAction{
				Type:  "search_text",
				Query: "KindWorkspaceStat",
				Path:  "internal/core/runner",
			},
			Plan: AgentExecutionPlan{
				Steps: []AgentPlanStep{
					{Title: "Search for broad matches", ExpectedActionTypes: []string{"search_text"}},
					{Title: "Inspect detailed matches", ExpectedActionTypes: []string{"search_text_detailed"}},
				},
			},
			CompletedActions: []string{"search_text"},
			CurrentStepTitle: "Inspect detailed matches",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "search_text_detailed", action.Type)
	require.Equal(t, "KindWorkspaceStat", action.Query)
	require.Equal(t, "internal/core/runner", action.Path)
}

func TestParseAgentActionSearchUsesGoalFallbackQueryAndPath(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"type":"search_text","reasoningSummary":"Search broadly first"}`,
		AgentRunState{
			Goal: "Use search_text in internal/core/runner for KindWorkspaceStat, then use search_text_detailed in internal/core/runner for KindWorkspaceStat with line details, then answer in one short sentence.",
			Plan: AgentExecutionPlan{
				Mode: "search_then_detailed",
				Steps: []AgentPlanStep{
					{Title: "Search for broad matches", ExpectedActionTypes: []string{"search_text"}},
					{Title: "Inspect detailed matches", ExpectedActionTypes: []string{"search_text_detailed"}},
				},
			},
			CurrentStepTitle: "Search for broad matches",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "search_text", action.Type)
	require.Equal(t, "KindWorkspaceStat", action.Query)
	require.Equal(t, "internal/core/runner", action.Path)
}

func TestParseAgentActionDetailedSearchUsesGoalFallbackQueryAndPath(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"type":"search_text_detailed","limit":20,"reasoningSummary":"Inspect line hits"}`,
		AgentRunState{
			Goal: "Use search_text in internal/core/runner for KindWorkspaceStat, then use search_text_detailed in internal/core/runner for KindWorkspaceStat with line details, then answer in one short sentence.",
			Plan: AgentExecutionPlan{
				Mode: "search_then_detailed",
				Steps: []AgentPlanStep{
					{Title: "Search for broad matches", ExpectedActionTypes: []string{"search_text"}},
					{Title: "Inspect detailed matches", ExpectedActionTypes: []string{"search_text_detailed"}},
				},
			},
			CompletedActions: []string{"search_text"},
			CurrentStepTitle: "Inspect detailed matches",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "search_text_detailed", action.Type)
	require.Equal(t, "KindWorkspaceStat", action.Query)
	require.Equal(t, "internal/core/runner", action.Path)
}

func TestParseAgentActionRecoversMalformedRespondJSONOnRespondStep(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"type":"respond","response":"go.mod exists and uses Go 1.25.0","reasoningSummary":"Answer with the findings","path"}`,
		AgentRunState{
			Plan: AgentExecutionPlan{
				Mode: "stat_then_read",
				Steps: []AgentPlanStep{
					{Title: "Check file status", ExpectedActionTypes: []string{"stat_file"}},
					{Title: "Read the file content", ExpectedActionTypes: []string{"read_file", "read_files_batch"}},
					{Title: "Answer with the findings", ExpectedActionTypes: []string{"respond"}},
				},
			},
			CompletedActions: []string{"stat_file", "read_files_batch"},
			CurrentStepTitle: "Answer with the findings",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "respond", action.Type)
	require.Equal(t, "go.mod exists and uses Go 1.25.0", action.Response)
}

func TestParseAgentActionDefaultsToReadFilesBatchAndGoalPathsForFilterThenReadStep(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"reasoningSummary":"Read the selected files next"}`,
		AgentRunState{
			Goal: "Use list_files_filtered on internal/core/runner with pattern *.go, then use read_files_batch on internal/core/runner/agent_loop.go and internal/core/runner/runner.go, then answer in one short sentence about what files you inspected.",
			Plan: AgentExecutionPlan{
				Mode: "filter_then_read",
				Steps: []AgentPlanStep{
					{Title: "Filter matching files", ExpectedActionTypes: []string{"list_files_filtered"}},
					{Title: "Read the selected files", ExpectedActionTypes: []string{"read_files_batch"}},
					{Title: "Answer with the findings", ExpectedActionTypes: []string{"respond"}},
				},
			},
			CompletedActions: []string{"list_files_filtered"},
			CurrentStepTitle: "Read the selected files",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "read_files_batch", action.Type)
	require.Equal(t, []string{"internal/core/runner/agent_loop.go", "internal/core/runner/runner.go"}, action.Paths)
}

func TestParseAgentActionDefaultsToStatFileAndGoalPathForStatThenReadStep(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"reasoningSummary":"Check the file metadata first"}`,
		AgentRunState{
			Goal: "Use stat_file on go.mod to confirm it exists and inspect metadata, then use read_files_batch on go.mod to read the content, then answer in one short sentence about what you found.",
			Plan: AgentExecutionPlan{
				Mode: "stat_then_read",
				Steps: []AgentPlanStep{
					{Title: "Check file status", ExpectedActionTypes: []string{"stat_file"}},
					{Title: "Read the file content", ExpectedActionTypes: []string{"read_file", "read_files_batch"}},
					{Title: "Answer with the findings", ExpectedActionTypes: []string{"respond"}},
				},
			},
			CurrentStepTitle: "Check file status",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "stat_file", action.Type)
	require.Equal(t, "go.mod", action.Path)
}

func TestDeriveDeterministicAgentActionReturnsBrowserScreenshotFallback(t *testing.T) {
	action, ok := deriveDeterministicAgentAction(AgentRunState{
		ThreadID:         "thread-browser",
		Goal:             "Open the controlled browser fixture page for the current thread. Use [data-testid='controlled-browser-input'] to type browser demo text, click [data-testid='controlled-browser-apply'], extract [data-testid='controlled-browser-result'], take a screenshot, and then answer in one short sentence with the extracted result.",
		CompletedActions: []string{"browser_open", "browser_type", "browser_click", "browser_extract"},
		Plan: AgentExecutionPlan{
			Mode: "browser_then_respond",
			Steps: []AgentPlanStep{
				{ExpectedActionTypes: []string{"browser_open"}},
				{ExpectedActionTypes: []string{"browser_type"}},
				{ExpectedActionTypes: []string{"browser_click"}},
				{ExpectedActionTypes: []string{"browser_extract"}},
				{ExpectedActionTypes: []string{"browser_screenshot"}},
				{ExpectedActionTypes: []string{"respond"}},
			},
		},
	})
	require.True(t, ok)
	require.Equal(t, "browser_screenshot", action.Type)
}

func TestDeriveDeterministicAgentActionReturnsPatchFallback(t *testing.T) {
	action, ok := deriveDeterministicAgentAction(AgentRunState{
		Goal:             "Update the existing file playwright-recovery-note.txt by adding one line saying recovery evidence, then answer in one short sentence about what you changed.",
		CompletedActions: []string{},
		Plan: AgentExecutionPlan{
			Mode: "patch_then_respond",
			Steps: []AgentPlanStep{
				{ExpectedActionTypes: []string{"apply_patch"}},
				{ExpectedActionTypes: []string{"respond"}},
			},
		},
	})
	require.True(t, ok)
	require.Equal(t, "apply_patch", action.Type)
	require.Equal(t, "playwright-recovery-note.txt", action.Path)
	require.Contains(t, action.Patch, "recovery evidence")
}

func TestDeriveDeterministicAgentActionReturnsRespondFallback(t *testing.T) {
	action, ok := deriveDeterministicAgentAction(AgentRunState{
		Goal:             "Open the controlled browser fixture page for the current thread. Use [data-testid='controlled-browser-input'] to type browser demo text, click [data-testid='controlled-browser-apply'], extract [data-testid='controlled-browser-result'], take a screenshot, and then answer in one short sentence with the extracted result.",
		CompletedActions: []string{"browser_open", "browser_type", "browser_click", "browser_extract", "browser_screenshot"},
		Plan: AgentExecutionPlan{
			Mode: "browser_then_respond",
			Steps: []AgentPlanStep{
				{ExpectedActionTypes: []string{"browser_open"}},
				{ExpectedActionTypes: []string{"browser_type"}},
				{ExpectedActionTypes: []string{"browser_click"}},
				{ExpectedActionTypes: []string{"browser_extract"}},
				{ExpectedActionTypes: []string{"browser_screenshot"}},
				{ExpectedActionTypes: []string{"respond"}},
			},
		},
	})
	require.True(t, ok)
	require.Equal(t, "respond", action.Type)
	require.Equal(t, "Controlled browser result: browser demo text", action.Response)
}

func TestParseAgentActionInfersStatFileTypeFromCurrentStep(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"path":"README.md","reasoningSummary":"Check existence first"}`,
		AgentRunState{
			Plan: AgentExecutionPlan{
				Steps: []AgentPlanStep{
					{Title: "Check file status", ExpectedActionTypes: []string{"stat_file"}},
				},
			},
			CurrentStepTitle: "Check file status",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "stat_file", action.Type)
}

func TestParseAgentActionInfersRespondTypeFromCurrentStep(t *testing.T) {
	action, err := parseAgentActionWithState(
		`{"response":"done","reasoningSummary":"Answer now"}`,
		AgentRunState{
			Plan: AgentExecutionPlan{
				Steps: []AgentPlanStep{
					{Title: "Answer with the findings", ExpectedActionTypes: []string{"respond"}},
				},
			},
			CurrentStepTitle: "Answer with the findings",
		},
	)
	require.NoError(t, err)
	require.Equal(t, "respond", action.Type)
	require.Equal(t, "done", action.Response)
}

func TestBuildAgentPromptGuidesSecondBatchReadTools(t *testing.T) {
	prompt := buildAgentPrompt(nil, nil, AgentRunState{
		Goal:      "Inspect the repository",
		StepIndex: 0,
		MaxSteps:  4,
		Plan: AgentExecutionPlan{
			Summary:          "Filter matching files first, then read the selected files, then answer.",
			Mode:             "filter_then_read",
			RequiredSequence: []string{"list_files_filtered", "read_files_batch", "respond"},
		},
		CurrentStepTitle: "Filter matching files",
	})

	require.Contains(t, prompt, "Allowed actions: respond, read_file, list_files, stat_file, read_files_batch, list_files_filtered, search_text, search_text_detailed, apply_patch, browser_open, browser_state, browser_click, browser_type, browser_extract, browser_screenshot.")
	require.Contains(t, prompt, "use stat_file for one file's existence or metadata")
	require.Contains(t, prompt, "read_files_batch for multiple text files")
	require.Contains(t, prompt, "list_files_filtered for directory filtering by pattern")
	require.Contains(t, prompt, "search_text_detailed when you need file and line matches")
	require.Contains(t, prompt, "Plan mode: filter_then_read")
	require.Contains(t, prompt, "Plan summary: Filter matching files first, then read the selected files, then answer.")
	require.Contains(t, prompt, "Required action sequence: list_files_filtered -> read_files_batch -> respond")
	require.Contains(t, prompt, "Current required step: Filter matching files")
	require.Contains(t, prompt, "Mode guidance: If the goal asks to filter by pattern first")
	require.Contains(t, prompt, "first call list_files_filtered, then call read_files_batch for the selected files, then respond")
	require.Contains(t, prompt, "Do not skip the required step when the goal explicitly asks for it")
	require.Contains(t, prompt, "If you violate the required action sequence, the run fails immediately.")
	require.Contains(t, prompt, "Return JSON only")
}

func TestBuildAgentPromptGuidesPatchThenRespondPlan(t *testing.T) {
	prompt := buildAgentPrompt(nil, nil, AgentRunState{
		Goal:      "Patch README.md and confirm the result",
		StepIndex: 0,
		MaxSteps:  4,
		Plan: AgentExecutionPlan{
			Summary:          "Apply the requested patch first, then answer with the result.",
			Mode:             "patch_then_respond",
			RequiredSequence: []string{"apply_patch", "respond"},
		},
		CurrentStepTitle: "Apply the requested patch",
	})

	require.Contains(t, prompt, "Plan mode: patch_then_respond")
	require.Contains(t, prompt, "Plan summary: Apply the requested patch first, then answer with the result.")
	require.Contains(t, prompt, "Required action sequence: apply_patch -> respond")
	require.Contains(t, prompt, "Current required step: Apply the requested patch")
	require.Contains(t, prompt, "Mode guidance: If the goal explicitly asks you to modify a file and then report the outcome")
	require.Contains(t, prompt, "apply_patch when the goal explicitly asks for a code or file modification")
	require.Contains(t, prompt, "call apply_patch before respond")
}

func TestBuildAgentPromptGuidesBrowserThenRespondPlan(t *testing.T) {
	prompt := buildAgentPrompt(nil, nil, AgentRunState{
		Goal:      "Open the controlled browser page, type browser demo text, click apply, extract the result, capture a screenshot, then answer.",
		StepIndex: 0,
		MaxSteps:  6,
		Plan: AgentExecutionPlan{
			Summary:          "Open the controlled browser page, interact with it, extract the stable result, and then answer.",
			Mode:             "browser_then_respond",
			RequiredSequence: []string{"browser_open", "browser_type", "browser_click", "browser_extract", "browser_screenshot", "respond"},
		},
		CurrentStepTitle: "Open the controlled browser page",
	})

	require.Contains(t, prompt, "Plan mode: browser_then_respond")
	require.Contains(t, prompt, "Required action sequence: browser_open -> browser_type -> browser_click -> browser_extract -> browser_screenshot -> respond")
	require.Contains(t, prompt, "Mode guidance: If the goal asks you to open a controlled page, type, click, extract the result, and then answer")
	require.Contains(t, prompt, "browser_* actions only for controlled browser workflows on allowlisted local or verified HTTPS pages")
}

func TestParseAgentActionUsesContentFallbackForRespond(t *testing.T) {
	action, err := parseAgentActionWithState(`{"type":"respond","content":"Final answer from content","reasoningSummary":"backup"}`, AgentRunState{})
	require.NoError(t, err)
	require.Equal(t, "respond", action.Type)
	require.Equal(t, "Final answer from content", action.Response)
}

func TestParseAgentActionSanitizesLiteralNewlinesInsidePatchJSON(t *testing.T) {
	action, err := parseAgentActionWithState(`{"type":"apply_patch","path":"README.md","patch":"*** Begin Patch
*** Update File: README.md
@@
-old
+new
*** End Patch","reasoningSummary":"Apply patch"}`, AgentRunState{})
	require.NoError(t, err)
	require.Equal(t, "apply_patch", action.Type)
	require.Contains(t, action.Patch, "*** Update File: README.md")
	require.Contains(t, action.Patch, "+new")
}

func TestRunnerCorrectsAgentRunAfterSequenceViolation(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("TODO one\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "notes.txt"), []byte("TODO two\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Search Corrected",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"先 search TODO，再看 detailed line matches 并回答","maxSteps":5}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"search_text_detailed","query":"TODO","path":".","limit":20,"reasoningSummary":"Jump ahead"}`},
			{OutputText: `{"type":"search_text","query":"TODO","path":".","reasoningSummary":"Search first after correction"}`},
			{OutputText: `{"type":"search_text_detailed","query":"TODO","path":".","limit":20,"reasoningSummary":"Inspect detailed matches"}`},
			{OutputText: `{"type":"respond","response":"Found the TODO matches.","reasoningSummary":"Answer now"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 3)
	require.Equal(t, KindWorkspaceSearch, tasks[1].Kind)
	require.Equal(t, KindWorkspaceSearchDetailed, tasks[2].Kind)
}

func TestRunnerCorrectsAgentRunWriteGoalToPatchThenRespond(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("old\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Patch Corrected",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"Patch README.md and confirm the result","maxSteps":4}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"read_file","path":"README.md","reasoningSummary":"Inspect the file first"}`},
			{OutputText: `{"type":"apply_patch","path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch","reasoningSummary":"Apply the requested patch first"}`},
			{OutputText: `{"type":"respond","response":"Patch applied and confirmed.","reasoningSummary":"Report the result"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Equal(t, "agent completed: Patch applied and confirmed.", result.ResultSummary)

	reloadedTask, err := registry.Task(thread.ID, task.ID)
	require.NoError(t, err)
	reloadedState, err := parseAgentRunState(reloadedTask.AgentState)
	require.NoError(t, err)
	require.Equal(t, "patch_then_respond", reloadedState.Plan.Mode)

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 2)
	require.Equal(t, KindWorkspaceApplyPatch, tasks[1].Kind)

	content, err := os.ReadFile(filepath.Join(projectRoot, "README.md"))
	require.NoError(t, err)
	require.Equal(t, "new", string(content))
}

func TestDeriveAgentExecutionPlanForFilterThenReadGoal(t *testing.T) {
	plan := deriveAgentExecutionPlan("先筛出 internal/core/runner 下的 *.go，再读取筛出的文件并回答")
	require.Equal(t, "Filter matching files first, then read the selected files, then answer.", plan.Summary)
	require.Equal(t, []string{"list_files_filtered", "read_files_batch", "respond"}, plan.RequiredSequence)
	require.Len(t, plan.Steps, 3)
	require.Equal(t, "Filter matching files", plan.Steps[0].Title)
}

func TestDeriveAgentExecutionPlanForPatchThenRespondGoal(t *testing.T) {
	plan := deriveAgentExecutionPlan("Patch README.md and confirm the result")
	require.Equal(t, "patch_then_respond", plan.Mode)
	require.Equal(t, "Apply the requested patch first, then answer with the result.", plan.Summary)
	require.Equal(t, []string{"apply_patch", "respond"}, plan.RequiredSequence)
	require.Len(t, plan.Steps, 2)
	require.Equal(t, "Apply the requested patch", plan.Steps[0].Title)
}

func TestDeriveAgentExecutionPlanForSearchThenDetailedGoal(t *testing.T) {
	plan := deriveAgentExecutionPlan("先 search TODO，再看 detailed line matches 和行号")
	require.Equal(t, "search_then_detailed", plan.Mode)
	require.Equal(t, []string{"search_text", "search_text_detailed", "respond"}, plan.RequiredSequence)
	require.Equal(t, "Search for broad matches", plan.Steps[0].Title)
}

func TestDeriveAgentExecutionPlanForStatThenReadGoal(t *testing.T) {
	plan := deriveAgentExecutionPlan("先确认 README.md 是否存在和 metadata，再读取内容并回答")
	require.Equal(t, "stat_then_read", plan.Mode)
	require.Equal(t, []string{"stat_file", "read_file|read_files_batch", "respond"}, plan.RequiredSequence)
	require.Equal(t, "Check file status", plan.Steps[0].Title)
}

func TestValidateAgentActionSequenceRejectsSkippedDiscoveryStep(t *testing.T) {
	state := AgentRunState{
		Plan: AgentExecutionPlan{
			RequiredSequence: []string{"list_files_filtered", "read_files_batch", "respond"},
		},
	}
	err := validateAgentActionSequence(state, AgentAction{Type: "read_files_batch"})
	require.EqualError(t, err, "agent action skipped required discovery step")
}

func TestValidateAgentActionSequenceRejectsWrongFollowupAction(t *testing.T) {
	state := AgentRunState{
		CompletedActions: []string{"list_files_filtered"},
		Plan: AgentExecutionPlan{
			RequiredSequence: []string{"list_files_filtered", "read_files_batch", "respond"},
		},
	}
	err := validateAgentActionSequence(state, AgentAction{Type: "search_text"})
	require.EqualError(t, err, "agent action violates required sequence: expected read_files_batch, got search_text")
}

func TestRunnerFailsAgentRunWhenRequiredSequenceIsViolated(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "internal", "core", "runner"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "internal", "core", "runner", "agent_loop.go"), []byte("package runner\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Sequence",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"Use list_files_filtered on internal/core/runner with pattern *.go, then use read_files_batch on the selected files, then answer.","maxSteps":4}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"read_files_batch","paths":["internal/core/runner/agent_loop.go"],"reasoningSummary":"Skip ahead"}`},
			{OutputText: `{"type":"read_files_batch","paths":["internal/core/runner/agent_loop.go"],"reasoningSummary":"Still skipping ahead"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Equal(t, "agent action skipped required discovery step", result.ResultSummary)
}

func TestRunnerExposesAgentPlanMetadataInState(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(projectRoot, "internal", "core", "runner"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "internal", "core", "runner", "agent_loop.go"), []byte("package runner\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "internal", "core", "runner", "runner.go"), []byte("package runner\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Metadata",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"Use list_files_filtered on internal/core/runner with pattern *.go, then use read_files_batch on internal/core/runner/agent_loop.go and internal/core/runner/runner.go, then answer.","maxSteps":4}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"list_files_filtered","path":"internal/core/runner","pattern":"*.go","reasoningSummary":"Filter matching Go files first"}`},
			{OutputText: `{"type":"read_files_batch","paths":["internal/core/runner/agent_loop.go","internal/core/runner/runner.go"],"reasoningSummary":"Read the selected files"}`},
			{OutputText: `{"type":"respond","response":"Done.","reasoningSummary":"Answer now"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)

	reloaded, err := registry.Task(thread.ID, task.ID)
	require.NoError(t, err)
	state, err := parseAgentRunState(reloaded.AgentState)
	require.NoError(t, err)
	require.Equal(t, "Filter matching files first, then read the selected files, then answer.", state.Plan.Summary)
	require.Equal(t, "Answer with the findings", state.CurrentStepTitle)
	require.Equal(t, "Answer now", state.LastReasoning)
	require.Equal(t, []string{"list_files_filtered", "read_files_batch", "respond"}, state.CompletedActions)
}

func TestRunnerFailsAgentRunWhenSearchDetailedSequenceIsViolated(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("TODO item\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Search Sequence",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"先 search TODO，再看 detailed line matches 并回答","maxSteps":4}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"search_text_detailed","query":"TODO","path":".","limit":20,"reasoningSummary":"Jump to detailed matches"}`},
			{OutputText: `{"type":"search_text_detailed","query":"TODO","path":".","limit":20,"reasoningSummary":"Still jumping to detailed matches"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Equal(t, "agent action skipped required discovery step", result.ResultSummary)
}

func TestRunnerFailsAgentRunWhenStatReadSequenceIsViolated(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("hello\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Stat Sequence",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"先确认 README.md 是否存在和 metadata，再读取内容并回答","maxSteps":4}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"read_file","path":"README.md","reasoningSummary":"Skip stat and read directly"}`},
			{OutputText: `{"type":"read_file","path":"README.md","reasoningSummary":"Still skipping stat and reading directly"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Equal(t, "agent action skipped required discovery step", result.ResultSummary)
}

func TestRunnerExecutesAgentRunWithSearchThenDetailedSequence(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("TODO one\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "notes.txt"), []byte("TODO two\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Search Detailed",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"先 search TODO，再看 detailed line matches 并回答","maxSteps":5}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"search_text","query":"TODO","path":".","reasoningSummary":"Search first"}`},
			{OutputText: `{"type":"search_text_detailed","query":"TODO","path":".","limit":20,"reasoningSummary":"Inspect detailed matches"}`},
			{OutputText: `{"type":"respond","response":"Found the TODO matches.","reasoningSummary":"Answer now"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 3)
	require.Equal(t, KindWorkspaceSearch, tasks[1].Kind)
	require.Equal(t, KindWorkspaceSearchDetailed, tasks[2].Kind)
}

func TestRunnerExecutesAgentRunWithStatThenReadSequence(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("hello metadata\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent Stat Read",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"先确认 README.md 是否存在和 metadata，再读取内容并回答","maxSteps":5}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"stat_file","path":"README.md","reasoningSummary":"Check file status first"}`},
			{OutputText: `{"type":"read_file","path":"README.md","reasoningSummary":"Read file content next"}`},
			{OutputText: `{"type":"respond","response":"README exists and was read.","reasoningSummary":"Answer now"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 3)
	require.Equal(t, KindWorkspaceStat, tasks[1].Kind)
	require.Equal(t, KindWorkspaceRead, tasks[2].Kind)
}

func TestRunnerExecutesAgentRunWithListThenReadSequence(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("hello listing\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "notes.txt"), []byte("extra\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Agent List Read",
		PermissionMode: policy.WorkspaceWrite,
	})
	task, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title: "Agent run",
		Kind:  KindAgentRun,
		Input: `{"goal":"Use list_files on . first, then read README.md, then answer.","maxSteps":5}`,
	})
	require.True(t, ok)

	models := &scriptedModelExecutor{
		results: []provider.ResponseResult{
			{OutputText: `{"type":"list_files","path":".","reasoningSummary":"List candidate files first"}`},
			{OutputText: `{"type":"read_file","path":"README.md","reasoningSummary":"Read the selected file next"}`},
			{OutputText: `{"type":"respond","response":"README was listed and read.","reasoningSummary":"Answer now"}`},
		},
	}

	result, err := New(registry, models).RunTask(context.Background(), thread.ID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)

	tasks, ok := registry.Tasks(thread.ID)
	require.True(t, ok)
	require.Len(t, tasks, 3)
	require.Equal(t, KindWorkspaceList, tasks[1].Kind)
	require.Equal(t, KindWorkspaceRead, tasks[2].Kind)

	reloaded, err := registry.Task(thread.ID, task.ID)
	require.NoError(t, err)
	state, err := parseAgentRunState(reloaded.AgentState)
	require.NoError(t, err)
	require.Equal(t, "list_then_read", state.Plan.Mode)
	require.Equal(t, []string{"list_files", "read_file", "respond"}, state.CompletedActions)
}

func TestRunnerRejectsRollbackWhenSourceIsNotLatestApply(t *testing.T) {
	projectRoot := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(projectRoot, "README.md"), []byte("old\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Rollback Latest Only",
		PermissionMode: policy.WorkspaceWrite,
	})

	firstTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "First patch",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+mid\n*** End Patch"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)
	_, err := New(registry, nil).RunTask(context.Background(), thread.ID, firstTask.ID)
	require.NoError(t, err)

	secondTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Second patch",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-mid\n+new\n*** End Patch"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)
	_, err = New(registry, nil).RunTask(context.Background(), thread.ID, secondTask.ID)
	require.NoError(t, err)

	writeExecutions, ok := registry.WriteExecutions(thread.ID)
	require.True(t, ok)
	require.Len(t, writeExecutions, 2)

	rollbackTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Rollback stale",
		Kind:           KindWorkspaceApplyPatchRollback,
		Input:          `{"writeExecutionId":"` + writeExecutions[0].ID + `"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)

	result, err := New(registry, nil).RunTask(context.Background(), thread.ID, rollbackTask.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ResultSummary, "only the latest completed apply execution can be rolled back")
}

func TestRunnerRejectsRollbackWhenFileHasDrifted(t *testing.T) {
	projectRoot := t.TempDir()
	targetPath := filepath.Join(projectRoot, "README.md")
	require.NoError(t, os.WriteFile(targetPath, []byte("old\n"), 0o644))

	registry := session.NewRegistry(projectRoot)
	thread := registry.CreateThread(session.CreateThreadInput{
		Name:           "Rollback Drift",
		PermissionMode: policy.WorkspaceWrite,
	})

	applyTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Apply patch",
		Kind:           KindWorkspaceApplyPatch,
		Input:          `{"path":"README.md","patch":"*** Begin Patch\n*** Update File: README.md\n@@\n-old\n+new\n*** End Patch"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)
	_, err := New(registry, nil).RunTask(context.Background(), thread.ID, applyTask.ID)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(targetPath, []byte("drifted\n"), 0o644))

	writeExecutions, ok := registry.WriteExecutions(thread.ID)
	require.True(t, ok)
	rollbackTask, ok := registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          "Rollback drifted",
		Kind:           KindWorkspaceApplyPatchRollback,
		Input:          `{"writeExecutionId":"` + writeExecutions[0].ID + `"}`,
		ApprovalStatus: "direct",
	})
	require.True(t, ok)

	result, err := New(registry, nil).RunTask(context.Background(), thread.ID, rollbackTask.ID)
	require.NoError(t, err)
	require.Equal(t, "failed", result.Status)
	require.Contains(t, result.ResultSummary, "file drift detected")
}
