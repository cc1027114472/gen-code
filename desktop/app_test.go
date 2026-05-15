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

	afterCreateTask := app.CreateTask(afterCreateThread.ActiveThreadID, "Organize runtime panel")
	if len(afterCreateTask.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(afterCreateTask.Tasks))
	}
	if afterCreateTask.Tasks[0].Status != "queued" {
		t.Fatalf("expected queued task, got %q", afterCreateTask.Tasks[0].Status)
	}

	afterAdvance := app.AdvanceTask(afterCreateTask.Tasks[0].ID)
	if len(afterAdvance.Tasks) != 1 {
		t.Fatalf("expected 1 task after advance, got %d", len(afterAdvance.Tasks))
	}
	if afterAdvance.Tasks[0].Status != "running" {
		t.Fatalf("expected running task, got %q", afterAdvance.Tasks[0].Status)
	}
	if len(afterAdvance.Events) == 0 {
		t.Fatal("expected events after task transition")
	}
	if !strings.Contains(afterAdvance.RecoverySummary, "Recovered") {
		t.Fatalf("expected recovery summary, got %q", afterAdvance.RecoverySummary)
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
	created = first.CreateTask(created.ActiveThreadID, "Resume after restart")
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
	if reloaded.Tasks[0].Title != "Resume after restart" {
		t.Fatalf("expected restored task title, got %q", reloaded.Tasks[0].Title)
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
