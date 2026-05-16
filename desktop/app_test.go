package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestAppGetAppInfo(t *testing.T) {
	app := NewApp()
	defer app.shutdown(nil)

	if got := app.GetAppInfo(); got != "gen-code desktop shell ready" {
		t.Fatalf("GetAppInfo() = %q, want %q", got, "gen-code desktop shell ready")
	}
}

func TestDesktopFallbackThreadTaskFlow(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "desktop-state.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	initial := app.GetRuntimeStatus()
	if !initial.RuntimeReady {
		t.Fatalf("expected fallback runtime ready, got false with message %q", initial.RuntimeMessage)
	}
	if initial.RuntimeSource != "desktop-local" {
		t.Fatalf("expected runtime source desktop-local, got %q", initial.RuntimeSource)
	}
	if initial.StateStore != "sqlite" {
		t.Fatalf("expected sqlite state store, got %q", initial.StateStore)
	}
	if initial.StatePath == "" {
		t.Fatal("expected fallback state path")
	}
	if !initial.UsesProjectLocalStore {
		t.Fatal("expected project-local store flag")
	}

	afterCreateThread := app.CreateThread("Desktop Thread")
	if afterCreateThread.ThreadCount != 1 {
		t.Fatalf("expected thread count 1, got %d", afterCreateThread.ThreadCount)
	}
	if afterCreateThread.ActiveThreadID == "" {
		t.Fatal("expected active thread id after creating thread")
	}

	afterCreateTask := app.CreateTask(afterCreateThread.ActiveThreadID, `{"title":"Organize runtime panel","kind":"prompt","input":"Show active thread runtime state"}`)
	if len(afterCreateTask.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(afterCreateTask.Tasks))
	}
	if afterCreateTask.Tasks[0].Status != "queued" {
		t.Fatalf("expected queued task, got %q", afterCreateTask.Tasks[0].Status)
	}
	if afterCreateTask.Tasks[0].Kind != "prompt" {
		t.Fatalf("expected prompt kind, got %q", afterCreateTask.Tasks[0].Kind)
	}
	if afterCreateTask.Tasks[0].Input != "Show active thread runtime state" {
		t.Fatalf("expected input to persist, got %q", afterCreateTask.Tasks[0].Input)
	}

	afterAdvance := app.AdvanceTask(afterCreateTask.Tasks[0].ID)
	if len(afterAdvance.Tasks) != 1 {
		t.Fatalf("expected 1 task after advance, got %d", len(afterAdvance.Tasks))
	}
	if afterAdvance.Tasks[0].Status != "completed" {
		t.Fatalf("expected completed task, got %q", afterAdvance.Tasks[0].Status)
	}
	if !strings.Contains(afterAdvance.Tasks[0].ResultSummary, "Task completed") {
		t.Fatalf("expected result summary after run, got %q", afterAdvance.Tasks[0].ResultSummary)
	}
	if len(afterAdvance.Messages) < 2 {
		t.Fatalf("expected fallback messages after create/run, got %d", len(afterAdvance.Messages))
	}
	if len(afterAdvance.ToolCalls) == 0 {
		t.Fatalf("expected fallback tool call after run, got %d", len(afterAdvance.ToolCalls))
	}
	if afterAdvance.ToolCalls[0].ToolID != "task.run" {
		t.Fatalf("expected latest tool call task.run, got %q", afterAdvance.ToolCalls[0].ToolID)
	}
	if len(afterAdvance.Artifacts) != 0 {
		t.Fatalf("expected no artifacts in fallback flow, got %d", len(afterAdvance.Artifacts))
	}
	if len(afterAdvance.Events) == 0 {
		t.Fatal("expected events after task transition")
	}
	if !strings.Contains(afterAdvance.RecoverySummary, "Recovered") {
		t.Fatalf("expected recovery summary, got %q", afterAdvance.RecoverySummary)
	}
	if tools, ok := afterAdvance.ToolsByGroup["runtime"]; !ok || len(tools) == 0 {
		t.Fatal("expected runtime tools summary in fallback status")
	} else if !strings.Contains(strings.Join(tools, " "), "read-only") {
		t.Fatalf("expected fallback tool labels to include read-only metadata, got %v", tools)
	}
}

func TestDesktopFallbackPersistsAcrossAppRestart(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	statePath := filepath.Join(t.TempDir(), "restart-state.sqlite")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", statePath)

	first := NewApp()
	defer first.shutdown(nil)
	created := first.CreateThread("Persistent Thread")
	if created.ActiveThreadID == "" {
		t.Fatal("expected active thread after first create")
	}
	created = first.CreateTask(created.ActiveThreadID, `{"title":"Resume after restart","kind":"spec","input":"Restore task metadata after desktop relaunch"}`)
	if len(created.Tasks) != 1 {
		t.Fatalf("expected 1 task before restart, got %d", len(created.Tasks))
	}

	second := NewApp()
	defer second.shutdown(nil)
	reloaded := second.GetRuntimeStatus()
	if reloaded.StatePath != statePath {
		t.Fatalf("expected persisted state path %q, got %q", statePath, reloaded.StatePath)
	}
	if reloaded.ThreadCount != 1 {
		t.Fatalf("expected 1 restored thread, got %d", reloaded.ThreadCount)
	}
	if reloaded.ActiveThreadID == "" {
		t.Fatal("expected restored active thread")
	}
	if len(reloaded.Tasks) != 1 {
		t.Fatalf("expected 1 restored task, got %d", len(reloaded.Tasks))
	}
	if reloaded.Messages == nil || reloaded.ToolCalls == nil || reloaded.Artifacts == nil {
		t.Fatal("expected recovered thread context collections")
	}
	if reloaded.Tasks[0].Title != "Resume after restart" {
		t.Fatalf("expected restored task title, got %q", reloaded.Tasks[0].Title)
	}
	if reloaded.Tasks[0].Kind != "spec" {
		t.Fatalf("expected restored task kind, got %q", reloaded.Tasks[0].Kind)
	}
	if reloaded.Tasks[0].Input != "Restore task metadata after desktop relaunch" {
		t.Fatalf("expected restored task input, got %q", reloaded.Tasks[0].Input)
	}
	if !strings.Contains(reloaded.RecoverySummary, "Recovered 1 thread") {
		t.Fatalf("expected restart recovery summary, got %q", reloaded.RecoverySummary)
	}
}

func TestCheckBridgeFallsBackLocally(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "bridge-state.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)
	result := app.CheckBridge()

	if !result.OK {
		t.Fatalf("expected local bridge check to succeed, got false with message %q", result.Message)
	}
	if result.Message == "" {
		t.Fatal("expected bridge check message")
	}
	if !strings.Contains(result.RuntimeHint, "desktop-local") {
		t.Fatalf("expected local runtime hint, got %q", result.RuntimeHint)
	}
}

func TestBrowserWorkspaceFlow(t *testing.T) {
	app := NewApp()
	defer app.shutdown(nil)

	initial := app.BrowserState()
	if !initial.IsOpen {
		t.Fatal("expected browser workspace open by default")
	}
	if len(initial.Tabs) != 1 {
		t.Fatalf("expected 1 default tab, got %d", len(initial.Tabs))
	}
	if initial.ActiveTabID == "" {
		t.Fatal("expected active browser tab id")
	}
	if initial.Tabs[0].URL == "" {
		t.Fatal("expected default browser url")
	}

	opened := app.BrowserOpen("http://127.0.0.1:5174/")
	if len(opened.Tabs) != 2 {
		t.Fatalf("expected 2 tabs after open, got %d", len(opened.Tabs))
	}
	activeID := opened.ActiveTabID

	navigated := app.BrowserNavigate(activeID, "http://localhost:10008/")
	if navigated.ActiveTabID != activeID {
		t.Fatalf("expected active tab %q, got %q", activeID, navigated.ActiveTabID)
	}
	if navigated.Tabs[len(navigated.Tabs)-1].URL != "http://localhost:10008/" {
		t.Fatalf("expected navigated URL, got %q", navigated.Tabs[len(navigated.Tabs)-1].URL)
	}

	reloaded := app.BrowserReload(activeID)
	if reloaded.ActiveTabID != activeID {
		t.Fatalf("expected active tab after reload, got %q", reloaded.ActiveTabID)
	}

	activated := app.BrowserActivateTab(initial.Tabs[0].ID)
	if activated.ActiveTabID != initial.Tabs[0].ID {
		t.Fatalf("expected first tab active, got %q", activated.ActiveTabID)
	}

	closed := app.BrowserCloseTab(initial.Tabs[0].ID)
	if len(closed.Tabs) != 1 {
		t.Fatalf("expected 1 tab after close, got %d", len(closed.Tabs))
	}
	if closed.ActiveTabID == "" {
		t.Fatal("expected remaining active tab after close")
	}
}
