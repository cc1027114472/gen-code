package main

import (
	"net/http"
	"testing"
	"time"
)

func TestAppGetAppInfo(t *testing.T) {
	app := NewApp()

	if got := app.GetAppInfo(); got != "gen-code desktop shell ready" {
		t.Fatalf("GetAppInfo() = %q, want %q", got, "gen-code desktop shell ready")
	}
}

func TestAppRuntimeThreadFlow(t *testing.T) {
	client := http.Client{Timeout: time.Second}
	if _, err := client.Get("http://127.0.0.1:10008/api/workspace"); err != nil {
		t.Skip("runtime server not available for desktop bridge flow test")
	}
	if _, err := client.Get("http://127.0.0.1:10008/api/threads/thread-1/tasks"); err != nil {
		t.Skip("runtime server not available for desktop bridge flow test")
	}

	app := NewApp()

	before := app.GetRuntimeStatus()
	if before.WorkspaceID == "" {
		t.Fatal("expected workspace id in runtime status")
	}

	afterCreate := app.CreateThread("Desktop Thread")
	if afterCreate.ThreadCount < 1 {
		t.Fatalf("expected thread count >= 1, got %d", afterCreate.ThreadCount)
	}
	if len(afterCreate.Threads) == 0 {
		t.Fatal("expected created thread to appear in runtime status")
	}

	activated := app.ActivateThread(afterCreate.Threads[0].ID)
	if activated.ActiveThreadID != afterCreate.Threads[0].ID {
		t.Fatalf("expected active thread id %q, got %q", afterCreate.Threads[0].ID, activated.ActiveThreadID)
	}

	afterTask := app.CreateTask(activated.ActiveThreadID, "Draft spec")
	if len(afterTask.Tasks) == 0 {
		t.Fatal("expected task to appear in runtime status")
	}
	if len(afterTask.Events) == 0 {
		t.Fatal("expected events to appear in runtime status")
	}
}
