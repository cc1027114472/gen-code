# Agent Failure Browser Acceptance Closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the `agent.run` browser acceptance gap by adding explicit failure-state coverage for both `remote-app-server` and `desktop local-fallback`, while keeping fallback as an evidence lane rather than a canonical gate.

**Architecture:** Reuse the existing desktop workbench and `verify-desktop-live-refresh.py` as the single browser acceptance entrypoint. Extend the acceptance script into a small scenario matrix for success, approval rejection, child-task failure, and recovery-failed evidence; keep desktop UI changes minimal and driven only by existing runtime fields so the browser assertions stay aligned with runtime semantics.

**Tech Stack:** Go, desktop fallback tests, React + TypeScript, Playwright Python sync API, PowerShell smoke runners

---

## File Structure

**Primary files**
- Modify: `scripts/verify-desktop-live-refresh.py`
  - Own the browser acceptance scenario matrix, summary JSON layout, and lane split between canonical remote and fallback evidence.
- Modify: `desktop/frontend/src/App.tsx`
  - Own the desktop workbench presentation for `agent.run` failure states using existing runtime fields only.
- Modify: `desktop/app_test.go`
  - Own the fallback persistence/recovery evidence assertions for `waiting_for_approval`, `waiting_for_task`, and recovery-failed visibility.

**Secondary files**
- Modify: `scripts/run-desktop-live-refresh-check.ps1`
  - Keep the local operator entrypoint aligned with the browser acceptance mode if new environment flags are needed.
- Modify: `scripts/run-desktop-smoke-check.ps1`
  - Keep smoke lane language aligned if summary field names or mode notes change.
- Modify: `docs/superpowers/plans/2026-05-17-desktop-remote-acceptance-entrypoint.md`
  - Clarify that remote remains canonical and fallback remains evidence-only.
- Modify: `docs/superpowers/plans/2026-05-17-desktop-copy-encoding-acceptance-report.md`
  - Record the new failure-state coverage and the explicit fallback boundary.

**Non-goals**
- Do not add new HTTP routes.
- Do not expand the `agent.run` tool whitelist.
- Do not promote `desktop local-fallback` to a same-level browser gate.
- Do not add rollback to the automatic agent action set.

---

### Task 1: Define the Agent Failure Acceptance Matrix

**Files:**
- Modify: `scripts/verify-desktop-live-refresh.py`
- Modify: `docs/superpowers/plans/2026-05-17-desktop-remote-acceptance-entrypoint.md`

- [ ] **Step 1: Add a single acceptance matrix section near the top of the browser script**

```python
AGENT_FAILURE_MATRIX = {
    "success_resume_baseline": {"lane": "remote-canonical", "required": True},
    "approval_rejected": {"lane": "remote-canonical", "required": True},
    "child_task_failed": {"lane": "remote-canonical", "required": True},
    "recovered_as_failed": {"lane": "remote-canonical", "required": True},
}

FALLBACK_EVIDENCE_MATRIX = {
    "approval_rejected": {"lane": "fallback-evidence", "required": True},
    "child_task_failed": {"lane": "fallback-evidence", "required": True},
    "recovered_as_failed": {"lane": "fallback-evidence", "required": True},
}
```

- [ ] **Step 2: Keep the lane boundary explicit in the acceptance entrypoint doc**

```markdown
- `remote-app-server`: canonical browser gate
- `desktop local-fallback`: browser-visible evidence lane only
- Failure-state coverage is reported for both lanes, but only remote decides the default browser pass/fail gate
```

- [ ] **Step 3: Verify the acceptance matrix is referenced from the browser script**

Run:

```powershell
Select-String -Path .\scripts\verify-desktop-live-refresh.py -Pattern "AGENT_FAILURE_MATRIX|FALLBACK_EVIDENCE_MATRIX"
```

Expected:
- Both constants are present and readable from the script.

- [ ] **Step 4: Commit**

```bash
git add scripts/verify-desktop-live-refresh.py docs/superpowers/plans/2026-05-17-desktop-remote-acceptance-entrypoint.md
git commit -m "docs: define agent failure browser acceptance matrix"
```

### Task 2: Factor Remote Browser Scenarios for Success and Failure States

**Files:**
- Modify: `scripts/verify-desktop-live-refresh.py`

- [ ] **Step 1: Write the failing shape for the scenario helpers**

```python
def run_agent_approval_rejected_scenario(page, thread_id: str, run_id: str) -> dict:
    raise NotImplementedError("approval rejected scenario not implemented")


def run_agent_child_task_failed_scenario(page, thread_id: str, run_id: str) -> dict:
    raise NotImplementedError("child task failed scenario not implemented")


def run_agent_recovered_as_failed_scenario(page, thread_id: str, run_id: str) -> dict:
    raise NotImplementedError("recovered_as_failed scenario not implemented")
```

- [ ] **Step 2: Run a syntax check to confirm the script still parses**

Run:

```powershell
python -m py_compile .\scripts\verify-desktop-live-refresh.py
```

Expected:
- PASS, because the helper stubs are syntactically valid.

- [ ] **Step 3: Implement the remote approval-rejected scenario**

```python
def run_agent_approval_rejected_scenario(page, thread_id: str, run_id: str) -> dict:
    created = create_task(thread_id, f"Agent approval rejected {run_id}", "agent.run", {
        "goal": "Apply a patch to approval-rejected-note.txt and then reply in one short sentence.",
        "maxSteps": 4,
    })
    task_id = created["id"]
    run_task(thread_id, task_id)
    waiting_parent = wait_for_task(thread_id, lambda item: item["id"] == task_id and item["status"] == "waiting_for_approval")
    waiting_child = find_latest_child_task(thread_id, task_id, "needs_approval")
    rejected_child = api("POST", f"/api/threads/{thread_id}/tasks/{waiting_child['id']}/reject", {})
    failed_parent = wait_for_task_terminal(thread_id, task_id, timeout_seconds=60.0)
    return {
        "parentTaskId": task_id,
        "childTaskId": waiting_child["id"],
        "childStatusAfterReject": rejected_child["status"],
        "parentStatusAfterReject": failed_parent["status"],
        "resultSummary": failed_parent.get("resultSummary", ""),
    }
```

- [ ] **Step 4: Implement the remote child-task-failed scenario**

```python
def run_agent_child_task_failed_scenario(page, thread_id: str, run_id: str) -> dict:
    created = create_task(thread_id, f"Agent child failed {run_id}", "agent.run", {
        "goal": "Use list_files_filtered with an empty pattern and then summarize the result.",
        "maxSteps": 3,
    })
    task_id = created["id"]
    run_task(thread_id, task_id)
    failed_parent = wait_for_task_terminal(thread_id, task_id, timeout_seconds=60.0)
    child_tasks = [item for item in api("GET", f"/api/threads/{thread_id}/tasks")["items"] if item.get("parentTaskId") == task_id]
    failed_child = next(item for item in child_tasks if item.get("status") == "failed")
    return {
        "parentTaskId": task_id,
        "childTaskId": failed_child["id"],
        "childKind": failed_child["kind"],
        "parentStatus": failed_parent["status"],
        "parentSummary": failed_parent.get("resultSummary", ""),
        "childSummary": failed_child.get("resultSummary", ""),
    }
```

- [ ] **Step 5: Implement the remote recovered-as-failed scenario**

```python
def run_agent_recovered_as_failed_scenario(page, thread_id: str, run_id: str) -> dict:
    evidence = collect_recovery_failure_runtime_evidence(thread_id, run_id)
    refresh_thread_view(page, thread_id)
    expect(page.get_by_test_id("latest-agent-card")).to_contain_text("task.recovered_as_failed", timeout=15000)
    return evidence
```

- [ ] **Step 6: Wire the new scenarios into the full acceptance result**

```python
"agentFailureMatrix": {
    "successResumeBaseline": recovery_result,
    "approvalRejected": approval_rejected_result,
    "childTaskFailed": child_failed_result,
    "recoveredAsFailed": recovery_failed_result,
}
```

- [ ] **Step 7: Run the targeted acceptance script in full mode**

Run:

```powershell
$env:GEN_CODE_UI_BASE_URL='http://127.0.0.1:5174/'
$env:GEN_CODE_API_BASE_URL='http://127.0.0.1:10008'
$env:GEN_CODE_ACCEPTANCE_MODE='full'
python .\scripts\verify-desktop-live-refresh.py
```

Expected:
- PASS with `acceptanceMode=full`
- The JSON output includes `agentFailureMatrix`
- `approvalRejected`, `childTaskFailed`, and `recoveredAsFailed` all report a visible parent task and a stable summary

- [ ] **Step 8: Commit**

```bash
git add scripts/verify-desktop-live-refresh.py
git commit -m "test: add remote agent failure browser scenarios"
```

### Task 3: Keep Desktop Failure-State Copy and Cards Consistent

**Files:**
- Modify: `desktop/frontend/src/App.tsx`

- [ ] **Step 1: Write a failing unit of presentation logic for the new failure summaries**

```tsx
function formatAgentStateSummary(task: RuntimeTask) {
  return "";
}
```

- [ ] **Step 2: Build the frontend to verify the placeholder compiles**

Run:

```powershell
cd .\desktop\frontend
npm run build
```

Expected:
- PASS, with no TypeScript errors from the placeholder.

- [ ] **Step 3: Implement explicit summary branches for failure and recovery states**

```tsx
if (resultSummary.startsWith("agent recovery failed:")) {
  return `状态：恢复失败（task.recovered_as_failed） / ${reason}`;
}
if (resultSummary.startsWith("agent failed: child approval rejected:")) {
  return `状态：审批已拒绝并中断续跑 / ${reason}`;
}
if (resultSummary.startsWith("agent child task failed:")) {
  return `状态：子任务失败 / ${reason}`;
}
```

- [ ] **Step 4: Keep the latest cards aligned with the same runtime-derived text**

```tsx
<ResultCard
  label="最新 Agent"
  title={latestAgentTask ? formatLatestTaskTitle(latestAgentTask) : "暂无 Agent 父任务"}
  body={latestAgentTask ? formatLatestTaskBody(latestAgentTask) : "当前线程还没有 agent.run 父任务"}
  testId="latest-agent-card"
/>
```

- [ ] **Step 5: Ensure child cards always show the task identity and parent linkage**

```tsx
const parts: string[] = [`任务 ${task.id}`];
if (task.parentTaskId) {
  parts.push(`父任务 ${task.parentTaskId}`);
}
if (task.latestChildTaskId) {
  parts.push(`最新子任务 ${task.latestChildTaskId}`);
}
```

- [ ] **Step 6: Rebuild the frontend**

Run:

```powershell
cd .\desktop\frontend
npm run build
```

Expected:
- PASS
- No TypeScript or Vite build regressions

- [ ] **Step 7: Commit**

```bash
git add desktop/frontend/src/App.tsx
git commit -m "feat: clarify desktop agent failure state visibility"
```

### Task 4: Strengthen Fallback Evidence Tests Without Promoting Fallback to Gate

**Files:**
- Modify: `desktop/app_test.go`

- [ ] **Step 1: Add failing assertions for fallback evidence labels**

```go
if !strings.Contains(parent.WorkflowLabel, "waiting_for_approval") {
    t.Fatalf("expected workflow label to include waiting_for_approval, got %q", parent.WorkflowLabel)
}
if strings.TrimSpace(parent.WaitingSummary) == "" {
    t.Fatal("expected non-empty parent waiting summary after restart")
}
```

- [ ] **Step 2: Add recovery-failed evidence coverage for the desktop fallback lane**

```go
func TestDesktopFallbackRecoveredAsFailedVisibleAfterRestart(t *testing.T) {
    // Seed a parent agent task with status failed and result_summary beginning with "agent recovery failed:"
    // Reload the desktop app and assert the task summary remains visible in the fallback runtime status.
}
```

- [ ] **Step 3: Run desktop tests**

Run:

```powershell
cd .\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
```

Expected:
- PASS
- Fallback tests prove `waiting_for_approval`, `waiting_for_task`, and `recovered_as_failed` remain visible after restart

- [ ] **Step 4: Commit**

```bash
git add desktop/app_test.go
git commit -m "test: expand fallback agent failure evidence"
```

### Task 5: Report Remote Canonical vs Fallback Evidence Separately

**Files:**
- Modify: `scripts/verify-desktop-live-refresh.py`
- Modify: `docs/superpowers/plans/2026-05-17-desktop-copy-encoding-acceptance-report.md`

- [ ] **Step 1: Add explicit summary sections for remote canonical and fallback evidence**

```python
"remote": {
    "mode": "browser-live",
    "agentFailureMatrix": {...},
},
"fallback": {
    "mode": "evidence-only",
    "agentFailureEvidence": {...},
}
```

- [ ] **Step 2: Keep the fallback report boundary explicit**

```markdown
- Fallback browser-visible checks are evidence-only.
- Canonical pass/fail still comes from `remote-app-server`.
- `supportsSSE=false` in fallback remains a degraded/manual-refresh note, not an automatic failure.
```

- [ ] **Step 3: Run smoke and full acceptance back-to-back**

Run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-smoke-with-bootstrap.ps1
$env:GEN_CODE_UI_BASE_URL='http://127.0.0.1:5174/'
$env:GEN_CODE_API_BASE_URL='http://127.0.0.1:10008'
$env:GEN_CODE_ACCEPTANCE_MODE='full'
python .\scripts\verify-desktop-live-refresh.py
```

Expected:
- Smoke still emits `desktop-smoke-summary.json`
- Full still emits `desktop-full-summary.json`
- Full summary contains both `remote.agentFailureMatrix` and `fallback.agentFailureEvidence`

- [ ] **Step 4: Commit**

```bash
git add scripts/verify-desktop-live-refresh.py docs/superpowers/plans/2026-05-17-desktop-copy-encoding-acceptance-report.md
git commit -m "docs: split canonical and fallback agent failure evidence"
```

### Task 6: Final Regression and Closeout

**Files:**
- Modify: `scripts/run-desktop-live-refresh-check.ps1`
- Modify: `scripts/run-desktop-smoke-check.ps1`

- [ ] **Step 1: Keep the PowerShell entrypoints aligned with the acceptance modes**

```powershell
$env:GEN_CODE_ACCEPTANCE_MODE = "smoke"
& $pythonCommand.Source (Join-Path $projectRoot "scripts\verify-desktop-live-refresh.py")
```

- [ ] **Step 2: Run the final regression pack**

Run:

```powershell
cd .\desktop\frontend
npm run build

cd ..\..
cd .\desktop
$env:GOTOOLCHAIN='auto'
go test ./...

cd .. 
python -m py_compile .\scripts\verify-desktop-live-refresh.py
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-smoke-with-bootstrap.ps1
$env:GEN_CODE_UI_BASE_URL='http://127.0.0.1:5174/'
$env:GEN_CODE_API_BASE_URL='http://127.0.0.1:10008'
$env:GEN_CODE_ACCEPTANCE_MODE='full'
python .\scripts\verify-desktop-live-refresh.py
```

Expected:
- All commands pass
- Remote canonical lane covers success + failure matrix
- Fallback evidence remains visible but not promoted to a release gate

- [ ] **Step 3: Commit**

```bash
git add scripts/run-desktop-live-refresh-check.ps1 scripts/run-desktop-smoke-check.ps1
git commit -m "chore: close agent failure browser acceptance regression pack"
```

---

## Self-Review

- Spec coverage: the plan covers remote canonical success/failure scenarios, fallback evidence-only coverage, desktop UI visibility, acceptance summary layering, and regression commands.
- Placeholder scan: removed `TBD`-style placeholders and assigned concrete files, commands, and expected outputs to every task.
- Type consistency: the plan consistently uses `agentFailureMatrix`, `agentFailureEvidence`, `waiting_for_approval`, `waiting_for_task`, and `task.recovered_as_failed`.
