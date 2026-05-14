package main

import "testing"

func TestAppGetAppInfo(t *testing.T) {
	app := NewApp()

	if got := app.GetAppInfo(); got != "gen-code desktop shell ready" {
		t.Fatalf("GetAppInfo() = %q, want %q", got, "gen-code desktop shell ready")
	}
}
