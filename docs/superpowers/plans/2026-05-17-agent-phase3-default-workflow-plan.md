# Agent Phase 3 Default Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `agent.run` the easiest default thread workflow while keeping remote and desktop local-fallback behavior understandable, testable, and consistently visible across CLI and desktop.

**Architecture:** Keep the current `thread -> task -> tool call -> approval -> write execution -> event -> runtime status` model unchanged, and improve the product surface around it instead of adding new execution backdoors. The work is split into three layers: runtime summary tightening, CLI/desktop workflow presentation, and side-by-side acceptance evidence for `remote-app-server` versus `local-fallback`.

**Tech Stack:** Go, SQLite, Wails desktop shell, React + TypeScript, existing runtime bridge, existing Playwright/Python verification scripts

---

## File Structure

- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\status.go`
  - Keep task summary derivation stable for parent/child agent workflows and expose a cleaner “default workflow” summary surface without changing the execution model.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\status_test.go`
  - Lock summary semantics for parent `agent.run`, latest child task, waiting reason, and final result strings.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
  - Improve `agent run`, `tasks list`, and `runtime status` output so the default path is readable without requiring users to understand every raw task kind.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`
  - Verify usage text and task output examples for the default agent path.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
  - Make the desktop workbench present `agent.run` as the default working mode, clarify waiting reasons, and show remote/fallback state consistently.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src/styles.css`
  - Adjust labels/chips/layout styles only as needed to support clearer runtime source, waiting state, and parent/child workflow presentation.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\verify-desktop-live-refresh.py`
  - Add a structured remote-vs-fallback acceptance path summary without weakening current remote default assertions.
- Add: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-agent-phase3-acceptance-report.md`
  - Record the side-by-side acceptance conclusions for remote and fallback behavior once the code path is verified.

### Task 1: Tighten runtime workflow summaries for the default agent path

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\status.go`
- Test: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\status_test.go`

- [ ] **Step 1: Write the failing summary tests**

Add focused assertions that cover:

- a parent `agent.run` with a latest child task
- a parent waiting on approval
- a parent waiting on a child task
- a completed parent with final assistant result

Test shape to add in `status_test.go`:

```go
func TestTaskDescriptorAgentWorkflowSummary(t *testing.T) {
	status := RuntimeStatus{
		Tasks: []runtimecontract.TaskDescriptor{
			{
				ID:                "task-agent",
				Kind:              "agent.run",
				Status:            "waiting_for_approval",
				WaitingStatus:     "waiting_for_approval",
				LatestChildTaskID: "task-child",
				AgentStep:         2,
				AgentMaxSteps:     4,
				ResultSummary:     "agent step 2/4: waiting for approval",
			},
			{
				ID:           "task-child",
				Kind:         "workspace.apply_patch",
				Status:       "needs_approval",
				ParentTaskID: "task-agent",
			},
		},
	}

	parent := findTask(status.Tasks, "task-agent")
	if parent.LatestChildTaskID != "task-child" {
		t.Fatalf("expected latest child task id task-child, got %q", parent.LatestChildTaskID)
	}
	if parent.WaitingStatus != "waiting_for_approval" {
		t.Fatalf("expected waiting_for_approval, got %q", parent.WaitingStatus)
	}
}
```

- [ ] **Step 2: Run runtime tests to verify the new assertions fail or expose gaps**

Run:

```powershell
$env:GOTOOLCHAIN='auto'; go test ./internal/core/runtime -run "TaskDescriptorAgentWorkflowSummary|Agent" -v
```

Expected: either FAIL on missing/default summary semantics or PASS with clear signal that no code change is needed for that case.

- [ ] **Step 3: Make the runtime summary logic minimal and explicit**

In `status.go`, normalize a single derived summary path for `agent.run` so callers can directly consume:

- latest child task id
- waiting reason
- step progress
- final result summary

Keep the implementation small and stable, for example:

```go
func summarizeAgentTask(task runtimecontract.TaskDescriptor) string {
	if strings.TrimSpace(task.ResultSummary) != "" {
		return strings.TrimSpace(task.ResultSummary)
	}
	if task.WaitingStatus == "waiting_for_approval" {
		return fmt.Sprintf("agent waiting for approval at step %d/%d", task.AgentStep, task.AgentMaxSteps)
	}
	if task.WaitingStatus == "waiting_for_task" {
		return fmt.Sprintf("agent waiting for child task %s at step %d/%d", fallbackText(task.LatestChildTaskID, "unknown"), task.AgentStep, task.AgentMaxSteps)
	}
	return "agent queued"
}
```

- [ ] **Step 4: Run the runtime tests again**

Run:

```powershell
$env:GOTOOLCHAIN='auto'; go test ./internal/core/runtime -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/runtime/status.go internal/core/runtime/status_test.go
git commit -m "test: lock agent workflow summary semantics"
```

### Task 2: Make CLI treat `agent.run` as the primary default workflow

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
- Test: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`

- [ ] **Step 1: Write failing CLI output tests**

Add or extend tests that assert:

- `help` shows `agent run` as a first-class default path
- `tasks list` shows waiting reason and latest child task for agent parents
- `runtime status` keeps `10008` and runtime source visible

Suggested test shape:

```go
func TestHelpIncludesAgentDefaultWorkflow(t *testing.T) {
	output := captureHelpOutput(t)
	require.Contains(t, output, "agent run --thread=<threadId> --goal=...")
	require.Contains(t, output, "tasks list --thread=<threadId>")
	require.Contains(t, output, "runtime status")
}
```

- [ ] **Step 2: Run CLI tests to confirm the gap**

Run:

```powershell
$env:GOTOOLCHAIN='auto'; go test ./cmd/cli -run "Help|Agent|Tasks" -v
```

Expected: FAIL where wording or ordering still does not reflect the default workflow.

- [ ] **Step 3: Tighten help text and agent task rendering**

Update `main.go` so:

- `agent run` is presented as the preferred entry for goal-oriented work
- `tasks list` prints parent/child relationships and waiting reasons using existing descriptor fields
- `runtime status` keeps `source`, `source trust`, and base runtime chain easy to read

Implementation shape:

```go
func printTaskLine(task runtimecontract.TaskDescriptor) {
	label := fallbackText(task.Title, task.Kind)
	if task.Kind == "agent.run" {
		fmt.Printf("- %s [agent] step %d/%d", label, task.AgentStep, task.AgentMaxSteps)
		if task.WaitingStatus != "" {
			fmt.Printf(" waiting=%s", task.WaitingStatus)
		}
		if task.LatestChildTaskID != "" {
			fmt.Printf(" child=%s", task.LatestChildTaskID)
		}
		fmt.Println()
		return
	}
	fmt.Printf("- %s [%s]\n", label, task.Kind)
}
```

- [ ] **Step 4: Re-run CLI tests**

Run:

```powershell
$env:GOTOOLCHAIN='auto'; go test ./cmd/cli -v
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/cli/main.go cmd/cli/main_test.go
git commit -m "feat: present agent run as default cli workflow"
```

### Task 3: Make the desktop workbench explain the agent workflow at a glance

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src/styles.css`

- [ ] **Step 1: Write down the UI assertions before editing**

Capture the behaviors to preserve in code comments or a local checklist while editing:

- active thread still owns the whole workflow
- parent `agent.run` remains visually distinct from child tasks
- waiting-for-approval and waiting-for-task use stable Chinese copy
- runtime source and SSE/manual-refresh state remain visible
- fallback remains readable without pretending it is canonical remote

Use a short comment near the workflow rendering helpers if needed:

```tsx
// Agent workflow cards must stay readable across remote and fallback snapshots.
```

- [ ] **Step 2: Refactor the desktop view model helpers before changing layout copy**

Extract or tighten helpers in `App.tsx` so the same derived text is reused for:

- center flow cards
- approval panel
- right-side result cards
- runtime source / SSE badges

Implementation shape:

```tsx
function formatAgentWorkflowHeadline(task: ExtendedRuntimeTask): string {
  if (task.waitingStatus === "waiting_for_approval") return "等待审批后自动续跑";
  if (task.waitingStatus === "waiting_for_task") return "等待子任务完成后续跑";
  if (task.status === "completed") return "Agent 已完成";
  if (task.status === "failed") return "Agent 已失败";
  return "Agent 执行中";
}
```

- [ ] **Step 3: Update the desktop labels and cards**

Adjust the UI so:

- the create-task form recommends entering a goal for `agent.run`
- parent cards clearly show step progress and latest child
- right-side cards clearly distinguish:
  - latest agent result
  - latest child task
  - latest approval
  - latest write execution
- runtime source badges use stable wording such as:
  - `远端运行时 / remote-app-server`
  - `本地回退 / desktop local-fallback`
  - `当前未接入 SSE，使用手动刷新`

Keep the structure intact and only refine copy and readability.

- [ ] **Step 4: Run the frontend build**

Run:

```powershell
npm run build
```

In:

```text
D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add desktop/frontend/src/App.tsx desktop/frontend/src/styles.css
git commit -m "feat: clarify desktop agent workflow and runtime source states"
```

### Task 4: Extend acceptance into a side-by-side remote versus fallback report

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\verify-desktop-live-refresh.py`
- Add: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-agent-phase3-acceptance-report.md`

- [ ] **Step 1: Add acceptance-report scaffolding to the verification script**

Extend the script so it can emit a concise structured result for:

- runtime source
- SSE support
- agent parent/child visibility
- approval visibility
- write execution visibility

Use a shape like:

```python
result = {
    "runtimeSource": runtime_source,
    "supportsSSE": supports_sse,
    "agentVisible": agent_visible,
    "childVisible": child_visible,
    "approvalVisible": approval_visible,
    "writeExecutionVisible": write_execution_visible,
}
```

- [ ] **Step 2: Run the existing remote default verification first**

Run:

```powershell
python .\scripts\verify-desktop-live-refresh.py
```

In:

```text
D:\GOWorks\gen-code-heji\gen-code
```

Expected: PASS on the default `5174 + 10008` chain and capture the resulting thread id / runtime source.

- [ ] **Step 3: Perform a fallback-oriented verification pass**

Run the same verification path against fallback conditions or a reduced fallback-specific flow, then capture:

- whether SSE is absent
- whether waiting/approval/result summaries remain readable
- whether parent/child relationships remain visible after refresh/reload

If browser automation cannot safely force fallback without destabilizing the session, record that limitation explicitly and use the passing Go fallback tests as the authoritative evidence.

- [ ] **Step 4: Write the side-by-side acceptance report**

Create `2026-05-17-agent-phase3-acceptance-report.md` with:

- scope
- remote default evidence
- fallback evidence
- differences that are intentional
- remaining coverage limits

Document structure:

```markdown
# 2026-05-17 Agent Phase 3 Acceptance Report

## Remote
- source: remote-app-server
- SSE: enabled

## Fallback
- source: local-fallback
- SSE: disabled/manual refresh

## Shared guarantees
- parent/child task visibility
- waiting reason visibility
- approval/write execution visibility
```

- [ ] **Step 5: Commit**

```bash
git add scripts/verify-desktop-live-refresh.py docs/superpowers/plans/2026-05-17-agent-phase3-acceptance-report.md
git commit -m "docs: add remote and fallback agent workflow acceptance report"
```

### Task 5: Run the full phase regression and produce the final handoff

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-agent-phase3-acceptance-report.md`
  - Only if final results require updating notes or risk statements.

- [ ] **Step 1: Run backend regressions**

Run:

```powershell
$env:GOTOOLCHAIN='auto'; go test ./internal/core/runtime ./internal/core/runner ./cmd/cli
```

Expected: PASS

- [ ] **Step 2: Run desktop regressions**

Run:

```powershell
$env:GOTOOLCHAIN='auto'; go test ./...
```

In:

```text
D:\GOWorks\gen-code-heji\gen-code\desktop
```

Expected: PASS

- [ ] **Step 3: Run the frontend build again**

Run:

```powershell
npm run build
```

In:

```text
D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
```

Expected: PASS

- [ ] **Step 4: Re-run the default live verification**

Run:

```powershell
python .\scripts\verify-desktop-live-refresh.py
```

Expected: PASS on default `5174 + 10008`

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/plans/2026-05-17-agent-phase3-acceptance-report.md
git commit -m "chore: finalize agent phase 3 verification handoff"
```

## Self-Review

- Spec coverage:
  - default `agent.run` path: covered by Tasks 1-3
  - remote/fallback clarity: covered by Tasks 3-4
  - regression and acceptance closure: covered by Task 5
- Placeholder scan:
  - all tasks include exact files and commands
  - fallback automation limitation is explicitly described instead of deferred vaguely
- Type consistency:
  - plan consistently uses `agent.run`, `LatestChildTaskID`, `WaitingStatus`, `remote-app-server`, and `local-fallback`

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-05-17-agent-phase3-default-workflow-plan.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
