package main

import "testing"

func TestAppGetAppInfo(t *testing.T) {
	app := NewApp()

	if got := app.GetAppInfo(); got != "gen-code desktop shell ready" {
		t.Fatalf("GetAppInfo() = %q, want %q", got, "gen-code desktop shell ready")
	}
}

func TestDesktopFallbackThreadTaskFlow(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")

	app := NewApp()

	initial := app.GetRuntimeStatus()
	if !initial.RuntimeReady {
		t.Fatalf("expected fallback runtime ready, got false with message %q", initial.RuntimeMessage)
	}
	if initial.RuntimeSource != "desktop-local" {
		t.Fatalf("expected runtime source desktop-local, got %q", initial.RuntimeSource)
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
}

func TestCheckBridgeFallsBackLocally(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")

	app := NewApp()
	result := app.CheckBridge()

	if !result.OK {
		t.Fatalf("expected local bridge check to succeed, got false with message %q", result.Message)
	}
	if result.Message == "" {
		t.Fatal("expected bridge check message")
	}
}
