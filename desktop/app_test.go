package main

import (
	"encoding/json"
	"os"
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
	if initial.RuntimeSource != "local-fallback" {
		t.Fatalf("expected runtime source local-fallback, got %q", initial.RuntimeSource)
	}
	if initial.RuntimeTrust != "degraded" {
		t.Fatalf("expected degraded runtime trust, got %q", initial.RuntimeTrust)
	}
	if !strings.Contains(initial.RuntimeSourceDetail, "canonical app-server runtime is unavailable") {
		t.Fatalf("expected degraded source detail, got %q", initial.RuntimeSourceDetail)
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
	if len(afterAdvance.Skills) == 0 {
		t.Fatal("expected fallback skill inventory")
	}
	if afterAdvance.Skills[0].VerificationStatus == "" {
		t.Fatal("expected fallback skills to include verification status")
	}
	if afterAdvance.Skills[0].IsolationStatus == "" {
		t.Fatal("expected fallback skills to include isolation status")
	}
	if afterAdvance.Skills[0].CapabilitySummary == "" {
		t.Fatal("expected fallback skills to include capability summary")
	}
	if len(afterAdvance.SkillGovernance) == 0 {
		t.Fatal("expected fallback skill governance summary")
	}
	if afterAdvance.SkillGovernance[0].Group == "" {
		t.Fatal("expected non-empty skill governance group")
	}
	if !strings.Contains(afterAdvance.RuntimeMessage, "manual refresh") {
		t.Fatalf("expected fallback runtime message to mention manual refresh, got %q", afterAdvance.RuntimeMessage)
	}
}

func TestDesktopFallbackRuntimeStatusShowsManualRefreshMode(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "fallback-refresh.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	status := app.GetRuntimeStatus()
	if status.RuntimeSource != "local-fallback" {
		t.Fatalf("expected local-fallback runtime source, got %q", status.RuntimeSource)
	}
	if status.RuntimeTrust != "degraded" {
		t.Fatalf("expected degraded runtime trust, got %q", status.RuntimeTrust)
	}
	if status.SupportsSSE {
		t.Fatal("expected fallback runtime to disable SSE")
	}
	if !strings.Contains(status.RuntimeMessage, "manual refresh") {
		t.Fatalf("expected manual refresh wording, got %q", status.RuntimeMessage)
	}
}

func TestBrowserCapabilitySummaryDifferentiatesRemoteAndFallback(t *testing.T) {
	remote := browserCapabilitySummary("remote-app-server")
	if !strings.Contains(remote, "verified-lanes=managed-authenticated-session,multi-target-public-web-read-only") {
		t.Fatalf("expected remote browser summary to advertise verified lanes, got %q", remote)
	}
	if !strings.Contains(remote, "compatibility=allowlist-only") {
		t.Fatalf("expected remote browser summary to advertise allowlist-only scope, got %q", remote)
	}

	fallback := browserCapabilitySummary("local-fallback")
	if !strings.Contains(fallback, "verified-lanes=supporting-evidence-local-preview") {
		t.Fatalf("expected fallback browser summary to stay evidence-only, got %q", fallback)
	}
	if !strings.Contains(fallback, "compatibility=non-canonical") {
		t.Fatalf("expected fallback browser summary to stay non-canonical, got %q", fallback)
	}
}

func TestBrowserWorkspaceStateFromRemoteUsesVerifiedLaneFallbackSummary(t *testing.T) {
	state := browserWorkspaceStateFromRemote(struct {
		ActiveTabID string `json:"activeTabId"`
		Tabs        []struct {
			ID           string `json:"id"`
			URL          string `json:"url"`
			Title        string `json:"title"`
			Loading      bool   `json:"loading"`
			CanGoBack    bool   `json:"canGoBack"`
			CanGoForward bool   `json:"canGoForward"`
		} `json:"tabs"`
		LatestActionSummary string `json:"latestActionSummary"`
		LatestActionError   string `json:"latestActionError"`
	}{
		ActiveTabID: "browser-tab-remote",
		Tabs: []struct {
			ID           string `json:"id"`
			URL          string `json:"url"`
			Title        string `json:"title"`
			Loading      bool   `json:"loading"`
			CanGoBack    bool   `json:"canGoBack"`
			CanGoForward bool   `json:"canGoForward"`
		}{
			{
				ID:    "browser-tab-remote",
				URL:   "https://example.com/",
				Title: "example.com",
			},
		},
	})
	if !strings.Contains(state.LatestActionSummary, "verified-lanes=managed-authenticated-session,multi-target-public-web-read-only") {
		t.Fatalf("expected remote browser workspace summary fallback, got %q", state.LatestActionSummary)
	}
}

func TestLocalSkillLocalizationCheckedRequiresFullyChineseAudit(t *testing.T) {
	root := t.TempDir()
	requireLocalized := func(name string, content string, want bool) {
		t.Helper()
		path := filepath.Join(root, name+".md")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
		if got := localSkillLocalizationChecked(root, name); got != want {
			t.Fatalf("localSkillLocalizationChecked(%q) = %t, want %t", name, got, want)
		}
	}

	requireLocalized("fully-localized", `---
name: fully-localized
description: metadata can remain machine-readable
---

# 中文技能

这是完整中文化审计样例。
- 先读取上下文。
- 再整理中文结果。
- 最后说明下一步。
`, true)

	requireLocalized("mixed-localized", `---
name: mixed-localized
description: metadata can remain machine-readable
---

# 中文标题

This step is still explained in English.
这里只有一行中文补充。
`, false)
}

func TestDesktopFallbackSkillGovernanceLocalizationPendingMatchesInventory(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "skill-governance.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	status := app.GetRuntimeStatus()
	if len(status.Skills) == 0 {
		t.Fatal("expected fallback skill inventory")
	}
	if len(status.SkillGovernance) == 0 {
		t.Fatal("expected fallback skill governance summary")
	}

	pendingByGroup := map[string]int{}
	capabilityPendingByGroup := map[string]int{}
	for _, item := range status.Skills {
		if !item.LocalizationChecked {
			group := strings.TrimSpace(item.Group)
			if group == "" {
				group = "common"
			}
			pendingByGroup[group]++
		}
		if !item.CapabilityVerified {
			group := strings.TrimSpace(item.Group)
			if group == "" {
				group = "common"
			}
			if group != "common" {
				capabilityPendingByGroup[group]++
			}
		}
	}
	for _, summary := range status.SkillGovernance {
		if got, want := summary.LocalizationPending, pendingByGroup[summary.Group]; got != want {
			t.Fatalf("group %q localization-pending = %d, want %d", summary.Group, got, want)
		}
		if got, want := summary.CapabilityPending, capabilityPendingByGroup[summary.Group]; got != want {
			t.Fatalf("group %q capability-pending = %d, want %d", summary.Group, got, want)
		}
	}
	if got := pendingByGroup["common"]; got != 0 {
		t.Fatalf("expected common localization-pending 0, got %d", got)
	}
	if got := pendingByGroup["codex"]; got != 0 {
		t.Fatalf("expected codex localization-pending 0, got %d", got)
	}
	if got := pendingByGroup["cc"]; got != 0 {
		t.Fatalf("expected cc localization-pending 0, got %d", got)
	}
}

func TestLocalSkillCatalogUsesProjectLocalCopiedSkills(t *testing.T) {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		t.Fatalf("findWorkspaceRoot: %v", err)
	}

	items := localSkillCatalog(workspaceRoot)
	if len(items) == 0 {
		t.Fatal("expected local skill catalog items")
	}

	found := map[string]SkillSummary{}
	for _, item := range mapSkills(items) {
		found[item.Group+":"+item.ID] = item
	}

	for _, key := range []string{
		"common:common.browser",
		"codex:code-review",
		"codex:test-tui",
		"cc:andrej-karpathy-skills",
		"cc:careful",
		"cc:connect-chrome",
		"cc:freeze",
		"cc:guard",
		"cc:land-and-deploy",
		"cc:qa",
		"cc:review",
		"cc:setup-browser-cookies",
		"cc:setup-deploy",
		"cc:ship",
		"cc:unfreeze",
		"cc:writing-skills",
	} {
		item, ok := found[key]
		if !ok {
			t.Fatalf("expected project-local copied skill %q", key)
		}
		if !item.LocalizationChecked {
			t.Fatalf("expected project-local copied skill %q to be localized", key)
		}
	}
	if !found["codex:code-review"].CapabilityVerified || found["codex:code-review"].CapabilitySummary == "" {
		t.Fatal("expected project-local copied codex skill to expose capability verification")
	}
	if !found["cc:writing-skills"].CapabilityVerified || found["cc:writing-skills"].CapabilitySummary == "" {
		t.Fatal("expected project-local copied cc skill to expose capability verification")
	}
	if !found["cc:careful"].CapabilityVerified || found["cc:freeze"].CapabilitySummary == "" || !found["cc:guard"].CapabilityVerified || !found["cc:unfreeze"].CapabilityVerified || !found["cc:setup-browser-cookies"].CapabilityVerified || !found["cc:connect-chrome"].CapabilityVerified || !found["cc:setup-deploy"].CapabilityVerified || !found["cc:qa"].CapabilityVerified || !found["cc:review"].CapabilityVerified || !found["cc:ship"].CapabilityVerified || !found["cc:land-and-deploy"].CapabilityVerified {
		t.Fatal("expected promoted gstack copied skills to expose stable capability verification")
	}
	if _, ok := found["cc:gstack"]; ok {
		t.Fatal("expected deferred gstack suite to stay out of the runtime-visible fallback inventory")
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

func TestDesktopFallbackTaskSummariesKeepParentAndWaitingFields(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "task-fields.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	created := app.CreateThread("Task Fields Thread")
	if created.ActiveThreadID == "" {
		t.Fatal("expected active thread")
	}

	store := app.store
	if store == nil || store.db == nil {
		t.Fatal("expected desktop store")
	}

	now := "2026-05-17T00:00:00Z"
	if _, err := store.db.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-parent", created.ActiveThreadID, "Agent run", "agent.run", `{"goal":"demo"}`, "waiting_for_approval", "agent waiting for approval", "pending", "", "waiting_for_approval", `{"taskId":"task-parent","threadId":"thread-1","stepIndex":1,"maxSteps":3,"waitingChildTaskId":"task-child","status":"waiting_for_approval","goal":"demo","planSummary":"Inspect workspace, patch docs, and verify output","currentStepTitle":"Inspect workspace","lastReasoning":"Need approval before continuing"}`, now, now); err != nil {
		t.Fatalf("insert parent task: %v", err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-child", created.ActiveThreadID, "Apply patch", "workspace.apply_patch", `{"path":"README.md","patch":"*** Begin Patch\n*** End Patch"}`, "needs_approval", "approval required", "pending", "task-parent", "", "", now, now); err != nil {
		t.Fatalf("insert child task: %v", err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-waiting", created.ActiveThreadID, "Wait for task", "agent.run", `{"goal":"wait on child"}`, "waiting_for_task", "waiting for child task", "pending", "task-parent", "waiting_for_task", `{"taskId":"task-waiting","threadId":"thread-1","stepIndex":2,"maxSteps":3,"waitingChildTaskId":"task-grandchild","status":"waiting_for_task","goal":"wait on child","planSummary":"Schedule follow-up work","currentStepTitle":"Await child task","lastReasoning":"Block until the child finishes"}`, now, now); err != nil {
		t.Fatalf("insert waiting task: %v", err)
	}

	status := app.GetRuntimeStatus()
	if len(status.Tasks) < 3 {
		t.Fatalf("expected at least 3 tasks, got %d", len(status.Tasks))
	}

	parent := findTaskByID(t, status, "task-parent")
	if parent.WaitingStatus != "waiting_for_approval" {
		t.Fatalf("expected parent waiting status, got %q", parent.WaitingStatus)
	}
	if parent.WaitingTaskID != "task-child" {
		t.Fatalf("expected waiting child task id task-child, got %q", parent.WaitingTaskID)
	}
	if parent.AgentStep != 1 || parent.AgentMaxSteps != 3 {
		t.Fatalf("expected agent step 1/3, got %d/%d", parent.AgentStep, parent.AgentMaxSteps)
	}
	if !strings.Contains(parent.WaitingSummary, "waiting for approval") {
		t.Fatalf("expected waiting summary for approval, got %q", parent.WaitingSummary)
	}
	if !strings.Contains(parent.WorkflowLabel, "waiting_for_approval") {
		t.Fatalf("expected workflow label to include waiting_for_approval, got %q", parent.WorkflowLabel)
	}
	if !strings.Contains(parent.WorkflowLabel, "step 1/3") {
		t.Fatalf("expected agent step label, got %q", parent.WorkflowLabel)
	}
	if strings.TrimSpace(parent.AgentPlanSummary) == "" {
		t.Fatal("expected non-empty agent plan summary")
	}
	if parent.LatestChildTaskID == "" {
		t.Fatal("expected latest child task id")
	}
	if !containsString(parent.ChildTaskIDs, "task-child") {
		t.Fatalf("expected child task ids to include task-child, got %+v", parent.ChildTaskIDs)
	}
	if !containsString(parent.ChildTaskIDs, "task-waiting") {
		t.Fatalf("expected child task ids to include task-waiting, got %+v", parent.ChildTaskIDs)
	}
	waiting := findTaskByID(t, status, "task-waiting")
	if waiting.WaitingStatus != "waiting_for_task" {
		t.Fatalf("expected waiting-for-task status, got %q", waiting.WaitingStatus)
	}
	if !strings.Contains(waiting.WaitingSummary, "waiting for child task task-grandchild") {
		t.Fatalf("expected waiting-for-task summary, got %q", waiting.WaitingSummary)
	}
	if !strings.Contains(waiting.WorkflowLabel, "waiting_for_task") {
		t.Fatalf("expected waiting-for-task workflow label, got %q", waiting.WorkflowLabel)
	}
	child := findTaskByID(t, status, "task-child")
	if child.ParentTaskID != "task-parent" {
		t.Fatalf("expected child parent task id task-parent, got %q", child.ParentTaskID)
	}
	if !strings.Contains(child.WorkflowLabel, "child task") {
		t.Fatalf("expected child workflow label, got %q", child.WorkflowLabel)
	}
	if !strings.Contains(child.WorkflowLabel, "approval required") {
		t.Fatalf("expected child approval label, got %q", child.WorkflowLabel)
	}

	if status.ActiveThreadSummary.ID != created.ActiveThreadID {
		t.Fatalf("expected active thread summary id %q, got %q", created.ActiveThreadID, status.ActiveThreadSummary.ID)
	}
	if !strings.Contains(status.ActiveThreadSummary.Summary, "task(s)") {
		t.Fatalf("expected active thread summary, got %q", status.ActiveThreadSummary.Summary)
	}
	if !strings.Contains(status.ActiveThreadSummary.Summary, "waiting") {
		t.Fatalf("expected active thread summary to mention waiting states, got %q", status.ActiveThreadSummary.Summary)
	}
	if status.ActiveThreadSummary.WaitingForApproval != 1 {
		t.Fatalf("expected waiting-for-approval count 1, got %d", status.ActiveThreadSummary.WaitingForApproval)
	}
	if status.ActiveThreadSummary.WaitingForTaskCount != 1 {
		t.Fatalf("expected waiting-for-task count 1, got %d", status.ActiveThreadSummary.WaitingForTaskCount)
	}
	if status.ActiveThreadSummary.ApprovalRequiredCount != 1 {
		t.Fatalf("expected approval-required count 1, got %d", status.ActiveThreadSummary.ApprovalRequiredCount)
	}
	if status.ActiveThreadSummary.ChildTaskCount < 1 {
		t.Fatalf("expected at least one child task, got %d", status.ActiveThreadSummary.ChildTaskCount)
	}
	if status.WorkspaceSummary.ActiveThreadID != created.ActiveThreadID {
		t.Fatalf("expected workspace summary active thread id %q, got %q", created.ActiveThreadID, status.WorkspaceSummary.ActiveThreadID)
	}
	if !strings.Contains(status.WorkspaceSummary.Summary, "active thread Task Fields Thread") {
		t.Fatalf("expected active workspace summary to name active thread, got %q", status.WorkspaceSummary.Summary)
	}
	if !strings.Contains(status.WorkspaceSummary.Summary, "waiting task(s)") {
		t.Fatalf("expected active workspace summary to mention waiting tasks, got %q", status.WorkspaceSummary.Summary)
	}
	if status.WorkspaceSummary.WaitingTaskCount == 0 {
		t.Fatal("expected workspace summary waiting task count")
	}
	if status.WorkspaceSummary.ApprovalRequiredCount != 1 {
		t.Fatalf("expected workspace approval-required count 1, got %d", status.WorkspaceSummary.ApprovalRequiredCount)
	}
}

func TestDesktopFallbackApprovePatchTaskWritesFile(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "approval-state.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Approval Thread")
	if createdThread.ActiveThreadID == "" {
		t.Fatal("expected active thread after create")
	}
	workspaceRoot := createdThread.WorkspaceRoot
	if workspaceRoot == "" {
		t.Fatal("expected workspace root in runtime status")
	}

	relativePath := filepath.ToSlash(filepath.Join(".tmp-desktop-tests", "approve-task.txt"))
	absolutePath := filepath.Join(workspaceRoot, filepath.FromSlash(relativePath))
	_ = os.Remove(absolutePath)
	t.Cleanup(func() {
		_ = os.Remove(absolutePath)
		_ = os.Remove(filepath.Dir(absolutePath))
	})

	patch := "*** Begin Patch\n*** Add File: .tmp-desktop-tests/approve-task.txt\n+approved from desktop fallback\n*** End Patch\n"
	payload := mustPatchTaskPayload(t, "Apply approval patch", relativePath, patch)
	createdTask := app.CreateTask(createdThread.ActiveThreadID, payload)
	if len(createdTask.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d with runtime message %q", len(createdTask.Tasks), createdTask.RuntimeMessage)
	}
	task := createdTask.Tasks[0]
	if task.Status != "needs_approval" {
		t.Fatalf("expected needs_approval status, got %q", task.Status)
	}
	if task.ApprovalStatus != "pending" {
		t.Fatalf("expected pending approval, got %q", task.ApprovalStatus)
	}
	if len(createdTask.Approvals) != 1 {
		t.Fatalf("expected 1 approval row, got %d", len(createdTask.Approvals))
	}
	if _, err := os.Stat(absolutePath); !os.IsNotExist(err) {
		t.Fatalf("expected file to be absent before approval, stat err=%v", err)
	}

	approved := app.ApproveTask(createdThread.ActiveThreadID, task.ID)
	if len(approved.Tasks) != 1 {
		t.Fatalf(
			"expected 1 task after approval, got %d (ready=%t state=%q message=%q approvals=%d events=%d)",
			len(approved.Tasks),
			approved.RuntimeReady,
			approved.RuntimeState,
			approved.RuntimeMessage,
			len(approved.Approvals),
			len(approved.Events),
		)
	}
	if approved.Tasks[0].Status != "completed" {
		t.Fatalf("expected completed task after approval, got %q", approved.Tasks[0].Status)
	}
	if approved.Tasks[0].ApprovalStatus != "executed" {
		t.Fatalf("expected executed approval status, got %q", approved.Tasks[0].ApprovalStatus)
	}
	if approved.Tasks[0].ApprovalSummary == "" {
		t.Fatal("expected approval summary on task for UI")
	}
	if approved.Tasks[0].WriteExecutionSummary == "" {
		t.Fatal("expected write execution summary on task for UI")
	}
	if len(approved.Approvals) != 1 {
		t.Fatalf("expected 1 approval after approval flow, got %d", len(approved.Approvals))
	}
	if approved.Approvals[0].Status != "executed" {
		t.Fatalf("expected approval executed status, got %q", approved.Approvals[0].Status)
	}
	if approved.Approvals[0].Summary == "" {
		t.Fatal("expected approval summary to drive UI")
	}
	if len(approved.WriteExecutions) != 1 {
		t.Fatalf("expected 1 write execution after approval, got %d", len(approved.WriteExecutions))
	}
	if approved.WriteExecutions[0].TaskID != task.ID {
		t.Fatalf("expected write execution task id %q, got %q", task.ID, approved.WriteExecutions[0].TaskID)
	}
	if approved.WriteExecutions[0].Status != "completed" {
		t.Fatalf("expected completed write execution, got %q", approved.WriteExecutions[0].Status)
	}
	if len(approved.WriteExecutions[0].TargetPaths) != 1 || approved.WriteExecutions[0].TargetPaths[0] != relativePath {
		t.Fatalf("expected write execution target path %q, got %+v", relativePath, approved.WriteExecutions[0].TargetPaths)
	}
	if !strings.Contains(approved.WriteExecutions[0].PatchSummary, "applied patch") {
		t.Fatalf("expected patch summary in write execution, got %q", approved.WriteExecutions[0].PatchSummary)
	}
	if !strings.Contains(approved.WriteExecutions[0].AfterSummary, "exists") {
		t.Fatalf("expected after snapshot summary, got %q", approved.WriteExecutions[0].AfterSummary)
	}
	if approved.WriteExecutions[0].ResultSummary == "" {
		t.Fatal("expected write execution result summary to drive UI")
	}
	approvedTask := findTaskByID(t, approved, task.ID)
	if approvedTask.ApprovalID == "" {
		t.Fatal("expected task approval id")
	}
	if approvedTask.WriteExecutionID == "" {
		t.Fatal("expected task write execution id")
	}
	if approvedTask.ApprovalID != approved.Approvals[0].ID {
		t.Fatalf("expected task approval id %q, got %q", approved.Approvals[0].ID, approvedTask.ApprovalID)
	}
	if approvedTask.WriteExecutionID != approved.WriteExecutions[0].ID {
		t.Fatalf("expected task write execution id %q, got %q", approved.WriteExecutions[0].ID, approvedTask.WriteExecutionID)
	}
	if !strings.Contains(approvedTask.ApprovalSummary, "approval") {
		t.Fatalf("expected task approval summary, got %q", approvedTask.ApprovalSummary)
	}
	if !strings.Contains(approvedTask.WriteExecutionSummary, "applied patch") {
		t.Fatalf("expected task write execution summary, got %q", approvedTask.WriteExecutionSummary)
	}
	if approved.ActiveThreadSummary.PendingApprovalCount != 0 {
		t.Fatalf("expected no pending approvals after execution, got %d", approved.ActiveThreadSummary.PendingApprovalCount)
	}
	if approved.ActiveThreadSummary.WriteExecutionCount != 1 {
		t.Fatalf("expected active thread write execution count 1, got %d", approved.ActiveThreadSummary.WriteExecutionCount)
	}
	if approved.ActiveThreadSummary.LatestApprovalTaskID != task.ID {
		t.Fatalf("expected latest approval task id %q, got %q", task.ID, approved.ActiveThreadSummary.LatestApprovalTaskID)
	}
	if approved.ActiveThreadSummary.LatestWriteTaskID != task.ID {
		t.Fatalf("expected latest write task id %q, got %q", task.ID, approved.ActiveThreadSummary.LatestWriteTaskID)
	}
	if approved.WorkspaceSummary.WriteExecutionCount != 1 {
		t.Fatalf("expected workspace write execution count 1, got %d", approved.WorkspaceSummary.WriteExecutionCount)
	}
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		t.Fatalf("expected approved patch file to exist: %v", err)
	}
	if string(content) != "approved from desktop fallback" {
		t.Fatalf("unexpected file content after approval: %q", string(content))
	}
	if !strings.Contains(approved.Tasks[0].ResultSummary, "applied patch to .tmp-desktop-tests/approve-task.txt") {
		t.Fatalf("expected patch result summary, got %q", approved.Tasks[0].ResultSummary)
	}
	if len(approved.ToolCalls) == 0 || approved.ToolCalls[0].ToolID != "workspace.apply_patch" {
		t.Fatalf("expected workspace.apply_patch tool call, got %+v", approved.ToolCalls)
	}
}

func TestDesktopFallbackRejectPatchTaskLeavesFileUntouched(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "reject-state.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Reject Thread")
	if createdThread.ActiveThreadID == "" {
		t.Fatal("expected active thread after create")
	}
	workspaceRoot := createdThread.WorkspaceRoot
	if workspaceRoot == "" {
		t.Fatal("expected workspace root in runtime status")
	}

	relativePath := filepath.ToSlash(filepath.Join(".tmp-desktop-tests", "reject-task.txt"))
	absolutePath := filepath.Join(workspaceRoot, filepath.FromSlash(relativePath))
	_ = os.Remove(absolutePath)
	t.Cleanup(func() {
		_ = os.Remove(absolutePath)
		_ = os.Remove(filepath.Dir(absolutePath))
	})

	patch := "*** Begin Patch\n*** Add File: .tmp-desktop-tests/reject-task.txt\n+should not be written\n*** End Patch\n"
	payload := mustPatchTaskPayload(t, "Reject approval patch", relativePath, patch)
	createdTask := app.CreateTask(createdThread.ActiveThreadID, payload)
	if len(createdTask.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d with runtime message %q", len(createdTask.Tasks), createdTask.RuntimeMessage)
	}
	task := createdTask.Tasks[0]
	if task.Status != "needs_approval" {
		t.Fatalf("expected needs_approval status, got %q", task.Status)
	}

	rejected := app.RejectTask(createdThread.ActiveThreadID, task.ID)
	if rejected.Tasks[0].Status != "failed" {
		t.Fatalf("expected failed task after rejection, got %q", rejected.Tasks[0].Status)
	}
	if rejected.Tasks[0].ApprovalStatus != "rejected" {
		t.Fatalf("expected rejected approval status, got %q", rejected.Tasks[0].ApprovalStatus)
	}
	if len(rejected.Approvals) != 1 || rejected.Approvals[0].Status != "rejected" {
		t.Fatalf("expected rejected approval summary, got %+v", rejected.Approvals)
	}
	if len(rejected.WriteExecutions) != 0 {
		t.Fatalf("expected no write execution after rejection, got %+v", rejected.WriteExecutions)
	}
	rejectedTask := findTaskByID(t, rejected, task.ID)
	if rejectedTask.ApprovalID == "" {
		t.Fatal("expected rejected task to retain latest approval id")
	}
	if !strings.Contains(rejectedTask.ApprovalSummary, "approval") {
		t.Fatalf("expected rejected task approval summary, got %q", rejectedTask.ApprovalSummary)
	}
	if rejected.ActiveThreadSummary.LatestApprovalTaskID != task.ID {
		t.Fatalf("expected latest approval task id %q after rejection, got %q", task.ID, rejected.ActiveThreadSummary.LatestApprovalTaskID)
	}
	if rejected.ActiveThreadSummary.LatestWriteTaskID != "" {
		t.Fatalf("expected no latest write task after rejection, got %q", rejected.ActiveThreadSummary.LatestWriteTaskID)
	}
	if _, err := os.Stat(absolutePath); !os.IsNotExist(err) {
		t.Fatalf("expected file to remain absent after rejection, stat err=%v", err)
	}
	if !strings.Contains(rejected.Tasks[0].ResultSummary, "approval rejected") {
		t.Fatalf("expected rejection summary, got %q", rejected.Tasks[0].ResultSummary)
	}
}

func TestDesktopFallbackThreadToolCallAppendCreatesToolCallRecord(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "toolcall-append.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Tool Call Append Thread")
	createdTask := app.CreateTask(createdThread.ActiveThreadID, mustThreadToolCallTaskPayload(t, "Append tool call", "workspace.search_text", "completed", "search finished"))
	if len(createdTask.Tasks) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(createdTask.Tasks))
	}

	completed := app.AdvanceTask(createdTask.Tasks[0].ID)
	task := findTaskByID(t, completed, createdTask.Tasks[0].ID)
	if task.Status != "completed" {
		t.Fatalf("expected completed task, got %q", task.Status)
	}
	if task.ResultSummary != "tool call workspace.search_text appended" {
		t.Fatalf("unexpected task summary: %q", task.ResultSummary)
	}

	appended := findToolCallByFields(t, completed, "workspace.search_text", "completed", "search finished")
	if appended.ThreadID != createdThread.ActiveThreadID {
		t.Fatalf("expected tool call thread id %q, got %q", createdThread.ActiveThreadID, appended.ThreadID)
	}
	if completed.ActiveThreadSummary.TaskCount != 1 {
		t.Fatalf("expected active thread task count 1, got %d", completed.ActiveThreadSummary.TaskCount)
	}
	if completed.WorkspaceSummary.TaskCount != 1 {
		t.Fatalf("expected workspace task count 1, got %d", completed.WorkspaceSummary.TaskCount)
	}
}

func TestDesktopFallbackThreadArtifactAppendCreatesArtifactRecord(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "artifact-append.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Artifact Append Thread")
	createdTask := app.CreateTask(createdThread.ActiveThreadID, mustThreadArtifactTaskPayload(t, "Append artifact", "artifacts/notes.md", "markdown"))
	if len(createdTask.Tasks) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(createdTask.Tasks))
	}

	completed := app.AdvanceTask(createdTask.Tasks[0].ID)
	task := findTaskByID(t, completed, createdTask.Tasks[0].ID)
	if task.Status != "completed" {
		t.Fatalf("expected completed task, got %q", task.Status)
	}
	if task.ResultSummary != "artifact markdown appended" {
		t.Fatalf("unexpected task summary: %q", task.ResultSummary)
	}

	artifact := findArtifactByPath(t, completed, "artifacts/notes.md")
	if artifact.ThreadID != createdThread.ActiveThreadID {
		t.Fatalf("expected artifact thread id %q, got %q", createdThread.ActiveThreadID, artifact.ThreadID)
	}
	if artifact.Kind != "markdown" {
		t.Fatalf("expected artifact kind markdown, got %q", artifact.Kind)
	}
}

func TestDesktopFallbackThreadRuntimeFlagSetCreatesRuntimeFlagRecord(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "runtimeflag-set.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Runtime Flag Thread")
	createdTask := app.CreateTask(createdThread.ActiveThreadID, mustThreadRuntimeFlagTaskPayload(t, "Set runtime flag", "preview.mode", "threaded"))
	if len(createdTask.Tasks) != 1 {
		t.Fatalf("expected 1 created task, got %d", len(createdTask.Tasks))
	}

	completed := app.AdvanceTask(createdTask.Tasks[0].ID)
	task := findTaskByID(t, completed, createdTask.Tasks[0].ID)
	if task.Status != "completed" {
		t.Fatalf("expected completed task, got %q", task.Status)
	}
	if task.ResultSummary != "runtime flag preview.mode updated" {
		t.Fatalf("unexpected task summary: %q", task.ResultSummary)
	}

	flag := findRuntimeFlagByKey(t, completed, "preview.mode")
	if flag.ThreadID != createdThread.ActiveThreadID {
		t.Fatalf("expected runtime flag thread id %q, got %q", createdThread.ActiveThreadID, flag.ThreadID)
	}
	if flag.Value != "threaded" {
		t.Fatalf("expected runtime flag value threaded, got %q", flag.Value)
	}

	reloaded := app.GetRuntimeStatus()
	reloadedFlag := findRuntimeFlagByKey(t, reloaded, "preview.mode")
	if reloadedFlag.Value != "threaded" {
		t.Fatalf("expected persisted runtime flag value threaded, got %q", reloadedFlag.Value)
	}
}

func mustPatchTaskPayload(t *testing.T, title string, relativePath string, patch string) string {
	t.Helper()

	input, err := json.Marshal(map[string]string{
		"path":  relativePath,
		"patch": patch,
	})
	if err != nil {
		t.Fatalf("marshal patch input: %v", err)
	}

	payload, err := json.Marshal(map[string]string{
		"title": title,
		"kind":  "workspace.apply_patch",
		"input": string(input),
	})
	if err != nil {
		t.Fatalf("marshal task payload: %v", err)
	}
	return string(payload)
}

func mustThreadToolCallTaskPayload(t *testing.T, title string, toolID string, status string, summary string) string {
	t.Helper()

	input, err := json.Marshal(map[string]string{
		"toolId":  toolID,
		"status":  status,
		"summary": summary,
	})
	if err != nil {
		t.Fatalf("marshal tool call input: %v", err)
	}

	payload, err := json.Marshal(map[string]string{
		"title": title,
		"kind":  "thread.toolcall.append",
		"input": string(input),
	})
	if err != nil {
		t.Fatalf("marshal tool call task payload: %v", err)
	}
	return string(payload)
}

func mustThreadArtifactTaskPayload(t *testing.T, title string, path string, kind string) string {
	t.Helper()

	input, err := json.Marshal(map[string]string{
		"path": path,
		"kind": kind,
	})
	if err != nil {
		t.Fatalf("marshal artifact input: %v", err)
	}

	payload, err := json.Marshal(map[string]string{
		"title": title,
		"kind":  "thread.artifact.append",
		"input": string(input),
	})
	if err != nil {
		t.Fatalf("marshal artifact task payload: %v", err)
	}
	return string(payload)
}

func mustThreadRuntimeFlagTaskPayload(t *testing.T, title string, key string, value string) string {
	t.Helper()

	input, err := json.Marshal(map[string]string{
		"key":   key,
		"value": value,
	})
	if err != nil {
		t.Fatalf("marshal runtime flag input: %v", err)
	}

	payload, err := json.Marshal(map[string]string{
		"title": title,
		"kind":  "thread.runtimeflag.set",
		"input": string(input),
	})
	if err != nil {
		t.Fatalf("marshal runtime flag task payload: %v", err)
	}
	return string(payload)
}

func mustRollbackTaskPayload(t *testing.T, title string, writeExecutionID string) string {
	t.Helper()

	input, err := json.Marshal(map[string]string{
		"writeExecutionId": writeExecutionID,
	})
	if err != nil {
		t.Fatalf("marshal rollback input: %v", err)
	}

	payload, err := json.Marshal(map[string]string{
		"title": title,
		"kind":  "workspace.apply_patch.rollback",
		"input": string(input),
	})
	if err != nil {
		t.Fatalf("marshal rollback task payload: %v", err)
	}
	return string(payload)
}

func setDesktopThreadPermissionMode(t *testing.T, app *App, threadID string, mode string) {
	t.Helper()
	if app == nil || app.store == nil || app.store.db == nil {
		t.Fatal("desktop store is not ready")
	}
	if _, err := app.store.db.Exec(`UPDATE threads SET permission_mode = ? WHERE id = ?`, mode, threadID); err != nil {
		t.Fatalf("set thread permission mode: %v", err)
	}
}

func findTaskByID(t *testing.T, status RuntimeStatus, taskID string) TaskSummary {
	t.Helper()
	for _, item := range status.Tasks {
		if item.ID == taskID {
			return item
		}
	}
	t.Fatalf("task %q not found in runtime status", taskID)
	return TaskSummary{}
}

func findApprovalByTaskID(t *testing.T, status RuntimeStatus, taskID string) ApprovalSummary {
	t.Helper()
	for _, item := range status.Approvals {
		if item.TaskID == taskID {
			return item
		}
	}
	t.Fatalf("approval for task %q not found in runtime status", taskID)
	return ApprovalSummary{}
}

func findWriteExecutionByTaskID(t *testing.T, status RuntimeStatus, taskID string) WriteExecutionSummary {
	t.Helper()
	for _, item := range status.WriteExecutions {
		if item.TaskID == taskID {
			return item
		}
	}
	t.Fatalf("write execution for task %q not found in runtime status", taskID)
	return WriteExecutionSummary{}
}

func findToolCallByFields(t *testing.T, status RuntimeStatus, toolID string, taskStatus string, summary string) ToolCallSummary {
	t.Helper()
	for _, item := range status.ToolCalls {
		if item.ToolID == toolID && item.Status == taskStatus && item.Summary == summary {
			return item
		}
	}
	t.Fatalf("tool call %q/%q/%q not found in runtime status", toolID, taskStatus, summary)
	return ToolCallSummary{}
}

func findArtifactByPath(t *testing.T, status RuntimeStatus, path string) ArtifactSummary {
	t.Helper()
	for _, item := range status.Artifacts {
		if item.Path == path {
			return item
		}
	}
	t.Fatalf("artifact path %q not found in runtime status", path)
	return ArtifactSummary{}
}

func findRuntimeFlagByKey(t *testing.T, status RuntimeStatus, key string) RuntimeFlagSummary {
	t.Helper()
	for _, item := range status.RuntimeFlags {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("runtime flag key %q not found in runtime status", key)
	return RuntimeFlagSummary{}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestDesktopFallbackWriteExecutionsPersistAcrossRestart(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	statePath := filepath.Join(t.TempDir(), "write-executions-restart.sqlite")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", statePath)

	first := NewApp()
	defer first.shutdown(nil)

	createdThread := first.CreateThread("Write Execution Restart")
	relativePath := filepath.ToSlash(filepath.Join(".tmp-desktop-tests", "write-exec-restart.txt"))
	absolutePath := filepath.Join(createdThread.WorkspaceRoot, filepath.FromSlash(relativePath))
	_ = os.Remove(absolutePath)
	t.Cleanup(func() {
		_ = os.Remove(absolutePath)
		_ = os.Remove(filepath.Dir(absolutePath))
	})

	patch := "*** Begin Patch\n*** Add File: .tmp-desktop-tests/write-exec-restart.txt\n+persist write execution\n*** End Patch\n"
	createdTask := first.CreateTask(createdThread.ActiveThreadID, mustPatchTaskPayload(t, "Persist write execution", relativePath, patch))
	if len(createdTask.Tasks) != 1 {
		t.Fatalf("expected 1 task before restart, got %d", len(createdTask.Tasks))
	}
	approved := first.ApproveTask(createdThread.ActiveThreadID, createdTask.Tasks[0].ID)
	if len(approved.WriteExecutions) != 1 {
		t.Fatalf("expected 1 write execution before restart, got %d", len(approved.WriteExecutions))
	}

	second := NewApp()
	defer second.shutdown(nil)
	reloaded := second.GetRuntimeStatus()
	if len(reloaded.WriteExecutions) != 1 {
		t.Fatalf("expected 1 persisted write execution after restart, got %d", len(reloaded.WriteExecutions))
	}
	if reloaded.WriteExecutions[0].TaskID != createdTask.Tasks[0].ID {
		t.Fatalf("expected persisted write execution task id %q, got %q", createdTask.Tasks[0].ID, reloaded.WriteExecutions[0].TaskID)
	}
	if !strings.Contains(reloaded.RecoverySummary, "write execution") {
		t.Fatalf("expected recovery summary to mention write executions, got %q", reloaded.RecoverySummary)
	}
}

func TestDesktopFallbackAgentWaitingForApprovalPersistsAcrossRestart(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	statePath := filepath.Join(t.TempDir(), "agent-waiting-approval-restart.sqlite")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", statePath)

	first := NewApp()
	defer first.shutdown(nil)

	created := first.CreateThread("Agent Approval Restart")
	if created.ActiveThreadID == "" {
		t.Fatal("expected active thread after create")
	}
	store := first.store
	if store == nil || store.db == nil {
		t.Fatal("expected desktop store")
	}

	now := "2026-05-17T00:00:00Z"
	if _, err := store.db.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-parent-approval", created.ActiveThreadID, "Agent approval parent", "agent.run", `{"goal":"update docs"}`, "waiting_for_approval", "agent waiting for approval", "", "", "waiting_for_approval", `{"taskId":"task-parent-approval","threadId":"thread-1","stepIndex":2,"maxSteps":5,"waitingChildTaskId":"task-child-approval","latestChildTaskId":"task-child-approval","status":"waiting_for_approval","goal":"update docs","planSummary":"Inspect docs, patch README, and verify output","currentStepTitle":"Patch README","lastReasoning":"Need approval before applying the patch","childTaskIds":["task-child-approval"]}`, now, now); err != nil {
		t.Fatalf("insert parent task: %v", err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-child-approval", created.ActiveThreadID, "Apply patch child", "workspace.apply_patch", `{"path":"README.md","patch":"*** Begin Patch\n*** End Patch"}`, "needs_approval", "approval required for README.md", "pending", "task-parent-approval", "", "", now, now); err != nil {
		t.Fatalf("insert child task: %v", err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO thread_approvals(id, thread_id, task_id, tool_kind, status, summary, target_paths, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "approval-pending-1", created.ActiveThreadID, "task-child-approval", "workspace.apply_patch", "pending", "approval required for README.md", `["README.md"]`, now, now); err != nil {
		t.Fatalf("insert approval: %v", err)
	}

	second := NewApp()
	defer second.shutdown(nil)

	reloaded := second.GetRuntimeStatus()
	if reloaded.RuntimeSource != "local-fallback" {
		t.Fatalf("expected local-fallback runtime source after restart, got %q", reloaded.RuntimeSource)
	}
	if reloaded.RuntimeTrust != "degraded" {
		t.Fatalf("expected degraded runtime trust after restart, got %q", reloaded.RuntimeTrust)
	}
	if !strings.Contains(reloaded.RuntimeSourceDetail, "canonical app-server runtime is unavailable") {
		t.Fatalf("expected canonical runtime fallback detail after restart, got %q", reloaded.RuntimeSourceDetail)
	}
	if !strings.Contains(reloaded.RuntimeMessage, "manual refresh") {
		t.Fatalf("expected manual refresh runtime message after restart, got %q", reloaded.RuntimeMessage)
	}
	if reloaded.StatePath != statePath {
		t.Fatalf("expected persisted state path %q, got %q", statePath, reloaded.StatePath)
	}
	parent := findTaskByID(t, reloaded, "task-parent-approval")
	if parent.Status != "waiting_for_approval" {
		t.Fatalf("expected waiting_for_approval parent status, got %q", parent.Status)
	}
	if parent.WaitingStatus != "waiting_for_approval" {
		t.Fatalf("expected waiting_for_approval field, got %q", parent.WaitingStatus)
	}
	if parent.WaitingTaskID != "task-child-approval" {
		t.Fatalf("expected waiting child task id task-child-approval, got %q", parent.WaitingTaskID)
	}
	if parent.LatestChildTaskID != "task-child-approval" {
		t.Fatalf("expected latest child task id task-child-approval, got %q", parent.LatestChildTaskID)
	}
	if parent.AgentStep != 2 || parent.AgentMaxSteps != 5 {
		t.Fatalf("expected agent step 2/5, got %d/%d", parent.AgentStep, parent.AgentMaxSteps)
	}
	if !strings.Contains(parent.WaitingSummary, "waiting for approval") {
		t.Fatalf("expected approval waiting summary, got %q", parent.WaitingSummary)
	}
	if !strings.Contains(parent.WorkflowLabel, "waiting_for_approval") {
		t.Fatalf("expected workflow label to include waiting_for_approval, got %q", parent.WorkflowLabel)
	}
	if !strings.Contains(parent.WorkflowLabel, "step 2/5") {
		t.Fatalf("expected workflow label to include step 2/5, got %q", parent.WorkflowLabel)
	}
	if strings.TrimSpace(parent.WaitingSummary) == "" {
		t.Fatal("expected non-empty parent waiting summary after restart")
	}
	if !containsString(parent.ChildTaskIDs, "task-child-approval") {
		t.Fatalf("expected child task ids to include task-child-approval, got %+v", parent.ChildTaskIDs)
	}
	child := findTaskByID(t, reloaded, "task-child-approval")
	if child.ParentTaskID != "task-parent-approval" {
		t.Fatalf("expected child parent task id task-parent-approval, got %q", child.ParentTaskID)
	}
	if child.Status != "needs_approval" {
		t.Fatalf("expected child needs_approval status, got %q", child.Status)
	}
	if child.ApprovalStatus != "pending" {
		t.Fatalf("expected child approval status pending, got %q", child.ApprovalStatus)
	}
	if !strings.Contains(child.WorkflowLabel, "child task") {
		t.Fatalf("expected child workflow label to include child task, got %q", child.WorkflowLabel)
	}
	if !strings.Contains(child.WorkflowLabel, "needs_approval") {
		t.Fatalf("expected child workflow label to include needs_approval, got %q", child.WorkflowLabel)
	}
	approval := findApprovalByTaskID(t, reloaded, "task-child-approval")
	if approval.Status != "pending" {
		t.Fatalf("expected pending approval after restart, got %q", approval.Status)
	}
	if !containsString(approval.TargetPaths, "README.md") {
		t.Fatalf("expected approval target paths to include README.md, got %+v", approval.TargetPaths)
	}
	if reloaded.ActiveThreadSummary.WaitingForApproval != 1 {
		t.Fatalf("expected waiting-for-approval count 1, got %d", reloaded.ActiveThreadSummary.WaitingForApproval)
	}
	if reloaded.ActiveThreadSummary.PendingApprovalCount != 1 {
		t.Fatalf("expected pending approval count 1, got %d", reloaded.ActiveThreadSummary.PendingApprovalCount)
	}
	if !strings.Contains(reloaded.RecoverySummary, "approval") {
		t.Fatalf("expected recovery summary to mention approvals, got %q", reloaded.RecoverySummary)
	}
}

func TestDesktopFallbackAgentWaitingForTaskPersistsAcrossRestart(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	statePath := filepath.Join(t.TempDir(), "agent-waiting-task-restart.sqlite")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", statePath)

	first := NewApp()
	defer first.shutdown(nil)

	created := first.CreateThread("Agent Child Wait Restart")
	if created.ActiveThreadID == "" {
		t.Fatal("expected active thread after create")
	}
	store := first.store
	if store == nil || store.db == nil {
		t.Fatal("expected desktop store")
	}

	now := "2026-05-17T00:00:00Z"
	if _, err := store.db.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-parent-wait", created.ActiveThreadID, "Agent child wait parent", "agent.run", `{"goal":"inspect source tree"}`, "waiting_for_task", "waiting for child task", "", "", "waiting_for_task", `{"taskId":"task-parent-wait","threadId":"thread-1","stepIndex":1,"maxSteps":4,"waitingChildTaskId":"task-child-read","latestChildTaskId":"task-child-read","status":"waiting_for_task","goal":"inspect source tree","planSummary":"List files, read package metadata, and summarize findings","currentStepTitle":"Read go.mod","lastReasoning":"Wait for the child read task to finish","childTaskIds":["task-child-read"]}`, now, now); err != nil {
		t.Fatalf("insert parent task: %v", err)
	}
	if _, err := store.db.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-child-read", created.ActiveThreadID, "Read go.mod child", "workspace.read_file", `{"path":"go.mod"}`, "completed", "read go.mod: module gen-code", "direct", "task-parent-wait", "", "", now, now); err != nil {
		t.Fatalf("insert child task: %v", err)
	}

	second := NewApp()
	defer second.shutdown(nil)

	reloaded := second.GetRuntimeStatus()
	if reloaded.RuntimeSource != "local-fallback" {
		t.Fatalf("expected local-fallback runtime source after restart, got %q", reloaded.RuntimeSource)
	}
	if reloaded.RuntimeTrust != "degraded" {
		t.Fatalf("expected degraded runtime trust after restart, got %q", reloaded.RuntimeTrust)
	}
	if !strings.Contains(reloaded.RuntimeSourceDetail, "canonical app-server runtime is unavailable") {
		t.Fatalf("expected canonical runtime fallback detail after restart, got %q", reloaded.RuntimeSourceDetail)
	}
	if !strings.Contains(reloaded.RuntimeMessage, "manual refresh") {
		t.Fatalf("expected manual refresh runtime message after restart, got %q", reloaded.RuntimeMessage)
	}
	parent := findTaskByID(t, reloaded, "task-parent-wait")
	if parent.Status != "waiting_for_task" {
		t.Fatalf("expected waiting_for_task parent status, got %q", parent.Status)
	}
	if parent.WaitingStatus != "waiting_for_task" {
		t.Fatalf("expected waiting_for_task field, got %q", parent.WaitingStatus)
	}
	if parent.WaitingTaskID != "task-child-read" {
		t.Fatalf("expected waiting child task id task-child-read, got %q", parent.WaitingTaskID)
	}
	if parent.LatestChildTaskID != "task-child-read" {
		t.Fatalf("expected latest child task id task-child-read, got %q", parent.LatestChildTaskID)
	}
	if parent.AgentStep != 1 || parent.AgentMaxSteps != 4 {
		t.Fatalf("expected agent step 1/4, got %d/%d", parent.AgentStep, parent.AgentMaxSteps)
	}
	if !strings.Contains(parent.WaitingSummary, "waiting for child task task-child-read") {
		t.Fatalf("expected waiting-for-task summary, got %q", parent.WaitingSummary)
	}
	if !strings.Contains(parent.WorkflowLabel, "waiting_for_task") {
		t.Fatalf("expected workflow label to include waiting_for_task, got %q", parent.WorkflowLabel)
	}
	if !strings.Contains(parent.WorkflowLabel, "step 1/4") {
		t.Fatalf("expected workflow label to include step 1/4, got %q", parent.WorkflowLabel)
	}
	if strings.TrimSpace(parent.WaitingSummary) == "" {
		t.Fatal("expected non-empty parent waiting summary after restart")
	}
	if !containsString(parent.ChildTaskIDs, "task-child-read") {
		t.Fatalf("expected child task ids to include task-child-read, got %+v", parent.ChildTaskIDs)
	}
	child := findTaskByID(t, reloaded, "task-child-read")
	if child.ParentTaskID != "task-parent-wait" {
		t.Fatalf("expected child parent task id task-parent-wait, got %q", child.ParentTaskID)
	}
	if child.Status != "completed" {
		t.Fatalf("expected child completed status after restart, got %q", child.Status)
	}
	if !strings.Contains(child.WorkflowLabel, "child task") {
		t.Fatalf("expected child workflow label to include child task, got %q", child.WorkflowLabel)
	}
	if !strings.Contains(child.WorkflowLabel, "completed") {
		t.Fatalf("expected child workflow label to include completed, got %q", child.WorkflowLabel)
	}
	if reloaded.ActiveThreadSummary.WaitingForTaskCount != 1 {
		t.Fatalf("expected waiting-for-task count 1, got %d", reloaded.ActiveThreadSummary.WaitingForTaskCount)
	}
	if reloaded.ActiveThreadSummary.WaitingTaskCount != 1 {
		t.Fatalf("expected waiting task count 1, got %d", reloaded.ActiveThreadSummary.WaitingTaskCount)
	}
	if !strings.Contains(reloaded.RecoverySummary, "Recovered 1 thread") {
		t.Fatalf("expected restart recovery summary, got %q", reloaded.RecoverySummary)
	}
}

func TestDesktopFallbackAgentRecoveredAsFailedPersistsAcrossRestart(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	statePath := filepath.Join(t.TempDir(), "agent-recovered-failed-restart.sqlite")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", statePath)

	first := NewApp()
	defer first.shutdown(nil)

	created := first.CreateThread("Agent Recovery Failed Restart")
	if created.ActiveThreadID == "" {
		t.Fatal("expected active thread after create")
	}
	store := first.store
	if store == nil || store.db == nil {
		t.Fatal("expected desktop store")
	}

	now := "2026-05-17T00:00:00Z"
	if _, err := store.db.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "task-recovered-failed", created.ActiveThreadID, "Agent recovered as failed", "agent.run", `{"goal":"resume interrupted agent run"}`, "failed", "agent recovery failed after desktop restart", "", "", "", `{"taskId":"task-recovered-failed","threadId":"thread-1","stepIndex":3,"maxSteps":5,"status":"recovered_as_failed","goal":"resume interrupted agent run","planSummary":"Recover prior agent state and resume execution","currentStepTitle":"Resume execution","lastReasoning":"agent recovery failed after desktop restart"}`, now, now); err != nil {
		t.Fatalf("insert recovered-as-failed task: %v", err)
	}

	second := NewApp()
	defer second.shutdown(nil)

	reloaded := second.GetRuntimeStatus()
	if reloaded.RuntimeSource != "local-fallback" {
		t.Fatalf("expected local-fallback runtime source after restart, got %q", reloaded.RuntimeSource)
	}
	if reloaded.RuntimeTrust != "degraded" {
		t.Fatalf("expected degraded runtime trust after restart, got %q", reloaded.RuntimeTrust)
	}
	if !strings.Contains(reloaded.RuntimeSourceDetail, "canonical app-server runtime is unavailable") {
		t.Fatalf("expected canonical runtime fallback detail after restart, got %q", reloaded.RuntimeSourceDetail)
	}
	if !strings.Contains(reloaded.RuntimeMessage, "manual refresh") {
		t.Fatalf("expected manual refresh runtime message after restart, got %q", reloaded.RuntimeMessage)
	}

	task := findTaskByID(t, reloaded, "task-recovered-failed")
	if task.Status != "failed" {
		t.Fatalf("expected failed task status after restart, got %q", task.Status)
	}
	if task.WaitingStatus != "" {
		t.Fatalf("expected no waiting status for recovered failure, got %q", task.WaitingStatus)
	}
	if task.AgentStep != 3 || task.AgentMaxSteps != 5 {
		t.Fatalf("expected agent step 3/5, got %d/%d", task.AgentStep, task.AgentMaxSteps)
	}
	if !strings.Contains(task.WorkflowLabel, "failed") {
		t.Fatalf("expected workflow label to include failed, got %q", task.WorkflowLabel)
	}
	if !strings.Contains(task.WorkflowLabel, "recovered_as_failed") {
		t.Fatalf("expected workflow label to include recovered_as_failed, got %q", task.WorkflowLabel)
	}
	if !strings.Contains(task.WorkflowLabel, "step 3/5") {
		t.Fatalf("expected workflow label to include step 3/5, got %q", task.WorkflowLabel)
	}
	if !strings.Contains(task.ResultSummary, "agent recovery failed") {
		t.Fatalf("expected result summary to mention agent recovery failed, got %q", task.ResultSummary)
	}
	if !strings.Contains(task.AgentLastReasoning, "agent recovery failed") {
		t.Fatalf("expected agent last reasoning to mention agent recovery failed, got %q", task.AgentLastReasoning)
	}
	if strings.TrimSpace(task.AgentPlanSummary) == "" {
		t.Fatal("expected non-empty agent plan summary after restart")
	}
	if reloaded.ActiveThreadSummary.FailedTaskCount != 1 {
		t.Fatalf("expected failed task count 1 in active thread summary, got %d", reloaded.ActiveThreadSummary.FailedTaskCount)
	}
	if reloaded.WorkspaceSummary.FailedTaskCount != 1 {
		t.Fatalf("expected failed task count 1 in workspace summary, got %d", reloaded.WorkspaceSummary.FailedTaskCount)
	}
	if !strings.Contains(reloaded.RecoverySummary, "Recovered 1 thread") {
		t.Fatalf("expected restart recovery summary, got %q", reloaded.RecoverySummary)
	}
}

func TestDesktopFallbackRollbackLatestApplyRestoresUpdatedFile(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "rollback-update.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Rollback Update Thread")
	setDesktopThreadPermissionMode(t, app, createdThread.ActiveThreadID, "workspace-write")

	relativePath := filepath.ToSlash(filepath.Join(".tmp-desktop-tests", "rollback-update.txt"))
	absolutePath := filepath.Join(createdThread.WorkspaceRoot, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
		t.Fatalf("mkdir test dir: %v", err)
	}
	if err := os.WriteFile(absolutePath, []byte("before\nline2"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(absolutePath)
		_ = os.Remove(filepath.Dir(absolutePath))
	})

	patch := "*** Begin Patch\n*** Update File: .tmp-desktop-tests/rollback-update.txt\n@@\n-before\n+after\n line2\n*** End Patch\n"
	createdTask := app.CreateTask(createdThread.ActiveThreadID, mustPatchTaskPayload(t, "Update file for rollback", relativePath, patch))
	if createdTask.Tasks[0].Status != "queued" || createdTask.Tasks[0].ApprovalStatus != "direct" {
		t.Fatalf("expected direct queued patch task, got status=%q approval=%q", createdTask.Tasks[0].Status, createdTask.Tasks[0].ApprovalStatus)
	}

	applied := app.AdvanceTask(createdTask.Tasks[0].ID)
	applyTask := findTaskByID(t, applied, createdTask.Tasks[0].ID)
	if applyTask.Status != "completed" {
		t.Fatalf("expected completed apply task, got %q", applyTask.Status)
	}
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if string(content) != "after\nline2" {
		t.Fatalf("unexpected content after apply: %q", string(content))
	}
	applyExecution := findWriteExecutionByTaskID(t, applied, createdTask.Tasks[0].ID)
	if applyExecution.Operation != "apply" {
		t.Fatalf("expected apply operation, got %q", applyExecution.Operation)
	}

	rollbackCreated := app.CreateTask(createdThread.ActiveThreadID, mustRollbackTaskPayload(t, "Rollback latest update", applyExecution.ID))
	if rollbackCreated.Tasks[0].Status != "queued" || rollbackCreated.Tasks[0].ApprovalStatus != "direct" {
		t.Fatalf("expected direct queued rollback task, got status=%q approval=%q", rollbackCreated.Tasks[0].Status, rollbackCreated.Tasks[0].ApprovalStatus)
	}
	rolledBack := app.AdvanceTask(rollbackCreated.Tasks[0].ID)
	rollbackTask := findTaskByID(t, rolledBack, rollbackCreated.Tasks[0].ID)
	if rollbackTask.Status != "completed" {
		t.Fatalf("expected completed rollback task, got %q", rollbackTask.Status)
	}
	content, err = os.ReadFile(absolutePath)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(content) != "before\nline2" {
		t.Fatalf("unexpected content after rollback: %q", string(content))
	}
	rollbackExecution := findWriteExecutionByTaskID(t, rolledBack, rollbackCreated.Tasks[0].ID)
	if rollbackExecution.Operation != "rollback" {
		t.Fatalf("expected rollback operation, got %q", rollbackExecution.Operation)
	}
	if rollbackExecution.RelatedExecutionID != applyExecution.ID {
		t.Fatalf("expected rollback related execution %q, got %q", applyExecution.ID, rollbackExecution.RelatedExecutionID)
	}
	if !strings.Contains(rollbackTask.ResultSummary, "rolled back patch on .tmp-desktop-tests/rollback-update.txt") {
		t.Fatalf("expected rollback summary, got %q", rollbackTask.ResultSummary)
	}
	if len(rolledBack.ToolCalls) == 0 || rolledBack.ToolCalls[0].ToolID != "workspace.apply_patch.rollback" {
		t.Fatalf("expected latest tool call workspace.apply_patch.rollback, got %+v", rolledBack.ToolCalls)
	}
}

func TestDesktopFallbackRollbackAddFileDeletesCreatedFileAfterApproval(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "rollback-add.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Rollback Add Thread")
	relativePath := filepath.ToSlash(filepath.Join(".tmp-desktop-tests", "rollback-add.txt"))
	absolutePath := filepath.Join(createdThread.WorkspaceRoot, filepath.FromSlash(relativePath))
	_ = os.Remove(absolutePath)
	t.Cleanup(func() {
		_ = os.Remove(absolutePath)
		_ = os.Remove(filepath.Dir(absolutePath))
	})

	patch := "*** Begin Patch\n*** Add File: .tmp-desktop-tests/rollback-add.txt\n+created for rollback\n*** End Patch\n"
	createdTask := app.CreateTask(createdThread.ActiveThreadID, mustPatchTaskPayload(t, "Create file for rollback", relativePath, patch))
	if createdTask.Tasks[0].Status != "needs_approval" {
		t.Fatalf("expected patch task to need approval, got %q", createdTask.Tasks[0].Status)
	}
	applied := app.ApproveTask(createdThread.ActiveThreadID, createdTask.Tasks[0].ID)
	applyExecution := findWriteExecutionByTaskID(t, applied, createdTask.Tasks[0].ID)
	if _, err := os.Stat(absolutePath); err != nil {
		t.Fatalf("expected created file after apply approval: %v", err)
	}

	rollbackCreated := app.CreateTask(createdThread.ActiveThreadID, mustRollbackTaskPayload(t, "Rollback created file", applyExecution.ID))
	if rollbackCreated.Tasks[0].Status != "needs_approval" || rollbackCreated.Tasks[0].ApprovalStatus != "pending" {
		t.Fatalf("expected rollback approval task, got status=%q approval=%q", rollbackCreated.Tasks[0].Status, rollbackCreated.Tasks[0].ApprovalStatus)
	}
	rollbackApproval := findApprovalByTaskID(t, rollbackCreated, rollbackCreated.Tasks[0].ID)
	if rollbackApproval.ToolKind != "workspace.apply_patch.rollback" {
		t.Fatalf("expected rollback approval tool kind, got %q", rollbackApproval.ToolKind)
	}

	rolledBack := app.ApproveTask(createdThread.ActiveThreadID, rollbackCreated.Tasks[0].ID)
	rollbackTask := findTaskByID(t, rolledBack, rollbackCreated.Tasks[0].ID)
	if rollbackTask.Status != "completed" {
		t.Fatalf("expected completed rollback task, got %q", rollbackTask.Status)
	}
	if _, err := os.Stat(absolutePath); !os.IsNotExist(err) {
		t.Fatalf("expected created file to be removed by rollback, stat err=%v", err)
	}
	rollbackExecution := findWriteExecutionByTaskID(t, rolledBack, rollbackCreated.Tasks[0].ID)
	if rollbackExecution.Operation != "rollback" || rollbackExecution.RelatedExecutionID != applyExecution.ID {
		t.Fatalf("unexpected rollback execution: %+v", rollbackExecution)
	}
	if !strings.Contains(rollbackTask.ResultSummary, "rolled back patch on .tmp-desktop-tests/rollback-add.txt") {
		t.Fatalf("expected rollback summary, got %q", rollbackTask.ResultSummary)
	}
	if rolledBack.ActiveThreadSummary.WriteExecutionCount != 2 {
		t.Fatalf("expected active thread write execution count 2 after apply+rollback, got %d", rolledBack.ActiveThreadSummary.WriteExecutionCount)
	}
	if rolledBack.WorkspaceSummary.WriteExecutionCount != 2 {
		t.Fatalf("expected workspace write execution count 2 after apply+rollback, got %d", rolledBack.WorkspaceSummary.WriteExecutionCount)
	}
	if rollbackTask.WriteExecutionID == "" {
		t.Fatal("expected rollback task write execution id")
	}
	if rollbackTask.WriteExecutionID != rollbackExecution.ID {
		t.Fatalf("expected rollback task write execution id %q, got %q", rollbackExecution.ID, rollbackTask.WriteExecutionID)
	}
	if !strings.Contains(rollbackTask.WriteExecutionSummary, "rolled back patch") {
		t.Fatalf("expected rollback task write execution summary, got %q", rollbackTask.WriteExecutionSummary)
	}
}

func TestDesktopFallbackRollbackNonLatestApplyFails(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "rollback-nonlatest.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Rollback NonLatest Thread")
	setDesktopThreadPermissionMode(t, app, createdThread.ActiveThreadID, "workspace-write")

	firstPath := filepath.ToSlash(filepath.Join(".tmp-desktop-tests", "rollback-nonlatest-1.txt"))
	secondPath := filepath.ToSlash(filepath.Join(".tmp-desktop-tests", "rollback-nonlatest-2.txt"))
	firstAbsolute := filepath.Join(createdThread.WorkspaceRoot, filepath.FromSlash(firstPath))
	secondAbsolute := filepath.Join(createdThread.WorkspaceRoot, filepath.FromSlash(secondPath))
	_ = os.Remove(firstAbsolute)
	_ = os.Remove(secondAbsolute)
	t.Cleanup(func() {
		_ = os.Remove(firstAbsolute)
		_ = os.Remove(secondAbsolute)
		_ = os.Remove(filepath.Dir(firstAbsolute))
	})

	firstTask := app.CreateTask(createdThread.ActiveThreadID, mustPatchTaskPayload(t, "First apply", firstPath, "*** Begin Patch\n*** Add File: .tmp-desktop-tests/rollback-nonlatest-1.txt\n+first\n*** End Patch\n"))
	firstApplied := app.AdvanceTask(firstTask.Tasks[0].ID)
	firstExecution := findWriteExecutionByTaskID(t, firstApplied, firstTask.Tasks[0].ID)

	secondTask := app.CreateTask(createdThread.ActiveThreadID, mustPatchTaskPayload(t, "Second apply", secondPath, "*** Begin Patch\n*** Add File: .tmp-desktop-tests/rollback-nonlatest-2.txt\n+second\n*** End Patch\n"))
	secondApplied := app.AdvanceTask(secondTask.Tasks[0].ID)
	secondExecution := findWriteExecutionByTaskID(t, secondApplied, secondTask.Tasks[0].ID)
	if secondExecution.ID == firstExecution.ID {
		t.Fatalf("expected distinct apply executions, got same id %q", secondExecution.ID)
	}

	rollbackCreated := app.CreateTask(createdThread.ActiveThreadID, mustRollbackTaskPayload(t, "Rollback first apply", firstExecution.ID))
	rolledBack := app.AdvanceTask(rollbackCreated.Tasks[0].ID)
	rollbackTask := findTaskByID(t, rolledBack, rollbackCreated.Tasks[0].ID)
	if rollbackTask.Status != "failed" {
		t.Fatalf("expected failed rollback task, got %q", rollbackTask.Status)
	}
	if !strings.Contains(rollbackTask.ResultSummary, "only the latest completed apply execution can be rolled back") {
		t.Fatalf("expected non-latest failure summary, got %q", rollbackTask.ResultSummary)
	}
	rollbackExecution := findWriteExecutionByTaskID(t, rolledBack, rollbackCreated.Tasks[0].ID)
	if rollbackExecution.Status != "failed" || rollbackExecution.Operation != "rollback" {
		t.Fatalf("expected failed rollback write execution, got %+v", rollbackExecution)
	}
}

func TestDesktopFallbackRollbackDriftFails(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "rollback-drift.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	createdThread := app.CreateThread("Rollback Drift Thread")
	setDesktopThreadPermissionMode(t, app, createdThread.ActiveThreadID, "workspace-write")

	relativePath := filepath.ToSlash(filepath.Join(".tmp-desktop-tests", "rollback-drift.txt"))
	absolutePath := filepath.Join(createdThread.WorkspaceRoot, filepath.FromSlash(relativePath))
	_ = os.Remove(absolutePath)
	t.Cleanup(func() {
		_ = os.Remove(absolutePath)
		_ = os.Remove(filepath.Dir(absolutePath))
	})

	createdTask := app.CreateTask(createdThread.ActiveThreadID, mustPatchTaskPayload(t, "Create drift file", relativePath, "*** Begin Patch\n*** Add File: .tmp-desktop-tests/rollback-drift.txt\n+original\n*** End Patch\n"))
	applied := app.AdvanceTask(createdTask.Tasks[0].ID)
	applyExecution := findWriteExecutionByTaskID(t, applied, createdTask.Tasks[0].ID)
	if err := os.WriteFile(absolutePath, []byte("drifted"), 0o644); err != nil {
		t.Fatalf("write drift content: %v", err)
	}

	rollbackCreated := app.CreateTask(createdThread.ActiveThreadID, mustRollbackTaskPayload(t, "Rollback drift file", applyExecution.ID))
	rolledBack := app.AdvanceTask(rollbackCreated.Tasks[0].ID)
	rollbackTask := findTaskByID(t, rolledBack, rollbackCreated.Tasks[0].ID)
	if rollbackTask.Status != "failed" {
		t.Fatalf("expected failed rollback task, got %q", rollbackTask.Status)
	}
	if !strings.Contains(rollbackTask.ResultSummary, "file drift detected") {
		t.Fatalf("expected drift failure summary, got %q", rollbackTask.ResultSummary)
	}
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		t.Fatalf("read drifted file: %v", err)
	}
	if string(content) != "drifted" {
		t.Fatalf("expected drifted content to remain untouched, got %q", string(content))
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
	if !strings.Contains(result.RuntimeHint, "local-fallback") {
		t.Fatalf("expected local runtime hint, got %q", result.RuntimeHint)
	}
	if !strings.Contains(result.RuntimeHint, "degraded") {
		t.Fatalf("expected degraded runtime hint, got %q", result.RuntimeHint)
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
	if initial.LatestActionSummary == "" {
		t.Fatal("expected initial browser action summary")
	}

	opened := app.BrowserOpen("http://127.0.0.1:5174/")
	if len(opened.Tabs) < 1 {
		t.Fatalf("expected at least 1 tab after open, got %d", len(opened.Tabs))
	}
	activeID := opened.ActiveTabID
	if strings.TrimSpace(opened.LatestActionError) == "" && !strings.Contains(opened.LatestActionSummary, "opened") {
		t.Fatalf("expected open summary or stable error, got summary=%q error=%q", opened.LatestActionSummary, opened.LatestActionError)
	}

	navigated := app.BrowserNavigate(activeID, "http://localhost:10008/")
	if strings.TrimSpace(navigated.LatestActionError) == "" {
		if navigated.ActiveTabID != activeID {
			t.Fatalf("expected active tab %q, got %q", activeID, navigated.ActiveTabID)
		}
		activeTab := findBrowserTabByID(t, navigated, activeID)
		if activeTab.URL != "http://localhost:10008/" {
			t.Fatalf("expected navigated URL, got %q", activeTab.URL)
		}
		if !strings.Contains(navigated.LatestActionSummary, "navigated") {
			t.Fatalf("expected navigate summary, got %q", navigated.LatestActionSummary)
		}
	} else if navigated.LatestActionSummary == "" {
		t.Fatalf("expected navigate summary or stable error, got summary=%q error=%q", navigated.LatestActionSummary, navigated.LatestActionError)
	}

	reloaded := app.BrowserReload(activeID)
	if strings.TrimSpace(reloaded.LatestActionError) == "" {
		if reloaded.ActiveTabID != activeID {
			t.Fatalf("expected active tab after reload, got %q", reloaded.ActiveTabID)
		}
		if !strings.Contains(reloaded.LatestActionSummary, "reloaded") {
			t.Fatalf("expected reload summary, got %q", reloaded.LatestActionSummary)
		}
	} else if reloaded.LatestActionSummary == "" {
		t.Fatalf("expected reload summary or stable error, got summary=%q error=%q", reloaded.LatestActionSummary, reloaded.LatestActionError)
	}

	activated := app.BrowserActivateTab(initial.Tabs[0].ID)
	if strings.TrimSpace(activated.LatestActionError) == "" {
		if activated.ActiveTabID != initial.Tabs[0].ID {
			t.Fatalf("expected first tab active, got %q", activated.ActiveTabID)
		}
		if !strings.Contains(activated.LatestActionSummary, "activated") {
			t.Fatalf("expected activate summary, got %q", activated.LatestActionSummary)
		}
	} else if activated.LatestActionSummary == "" {
		t.Fatalf("expected activate summary or stable error, got summary=%q error=%q", activated.LatestActionSummary, activated.LatestActionError)
	}

	clicked := app.BrowserClick(activeID, "[data-testid='missing']")
	if clicked.LatestActionSummary == "" && clicked.LatestActionError == "" {
		t.Fatal("expected browser click to surface latest action result")
	}

	typed := app.BrowserType(activeID, "[data-testid='missing']", "hello")
	if typed.LatestActionSummary == "" && typed.LatestActionError == "" {
		t.Fatal("expected browser type to surface latest action result")
	}

	extracted := app.BrowserExtract(activeID, "")
	if extracted.LatestActionSummary == "" && extracted.LatestActionError == "" {
		t.Fatal("expected browser extract to surface latest action result")
	}

	screenshot := app.BrowserScreenshot(activeID)
	if screenshot.LatestActionSummary == "" && screenshot.LatestActionError == "" {
		t.Fatal("expected browser screenshot to surface latest action result")
	}

	closed := app.BrowserCloseTab(initial.Tabs[0].ID)
	if strings.TrimSpace(closed.LatestActionError) == "" {
		if len(closed.Tabs) < 1 {
			t.Fatalf("expected remaining tab after close, got %d", len(closed.Tabs))
		}
		if closed.ActiveTabID == "" {
			t.Fatal("expected remaining active tab after close")
		}
		if !strings.Contains(closed.LatestActionSummary, "closed") {
			t.Fatalf("expected close summary, got %q", closed.LatestActionSummary)
		}
	} else if closed.LatestActionSummary == "" {
		t.Fatalf("expected close summary or stable error, got summary=%q error=%q", closed.LatestActionSummary, closed.LatestActionError)
	}
}

func findBrowserTabByID(t *testing.T, state BrowserWorkspaceState, tabID string) BrowserTab {
	t.Helper()
	for _, item := range state.Tabs {
		if item.ID == tabID {
			return item
		}
	}
	t.Fatalf("browser tab %q not found", tabID)
	return BrowserTab{}
}
