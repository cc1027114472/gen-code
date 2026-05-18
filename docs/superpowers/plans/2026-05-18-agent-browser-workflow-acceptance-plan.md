# Agent Browser Workflow Acceptance Plan Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `agent.run` use the existing controlled browser toolset predictably on canonical local pages, and close the acceptance loop from desktop UI creation through browser child tasks to final assistant response.

**Architecture:** Reuse the existing `browser.*` tool family, `agent.run` parent/child task model, desktop workbench, and canonical Playwright verifier. Do not add new routes or a new browser protocol. Instead, tighten browser action planning and sequencing inside `agent.run`, keep child-task summaries and artifacts readable, and add one stable UI-first browser agent scenario to the canonical `full` lane.

**Tech Stack:** Go, `internal/core/runner`, existing runtime task descriptors, React desktop frontend, Python Playwright verifier, PowerShell acceptance wrapper.

---

## File Structure

- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runner\agent_loop.go`
  Purpose: Tighten `agent.run` browser action planning, sequencing, correction, and failure summaries.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runner\runner_test.go`
  Purpose: Lock browser child-task orchestration and failure behavior with focused tests.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\discovery.go`
  Purpose: Keep browser tool descriptions and `agent.run` discovery wording aligned if planner hints surface there.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
  Purpose: Keep `agent run`, `tasks list`, and browser-child visibility readable from existing runtime fields.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`
  Purpose: Lock CLI wording and browser-child status visibility.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
  Purpose: Make browser child tasks, extract summaries, and screenshot artifacts easier to read in the existing workbench without inventing a new page.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\verify-desktop-live-refresh.py`
  Purpose: Add a canonical UI-first `agent.run` browser scenario and report it in the existing summary JSON.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\run-desktop-live-refresh-check.ps1`
  Purpose: Keep the canonical wrapper aligned with the new browser-agent scenario, while preserving current bootstrap and artifact behavior.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-capability-matrix-and-tool-coverage.md`
  Purpose: Record that browser tools are not only direct-task capable, but also agent-orchestrated on controlled local pages.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-runtime-entry-checklist.md`
  Purpose: Document the new browser-agent full-lane acceptance boundary.

## Non-Goals

- Do not add new HTTP routes.
- Do not allow arbitrary public-web agent browsing.
- Do not add login automation beyond the existing controlled fixture/session policy.
- Do not add a second browser protocol or desktop-private execution path.
- Do not add rollback to the automatic browser agent action set.

### Task 1: Tighten `agent.run` Browser Planning

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runner\agent_loop.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runner\runner_test.go`

- [ ] **Step 1: Write failing tests for browser-oriented `agent.run` plans**

Add focused cases that prove:
- a controlled-page goal chooses browser actions instead of read-file tools
- a form-style goal prefers `browser_open -> browser_type -> browser_click -> browser_extract -> respond`
- an optional screenshot goal adds `browser_screenshot` before `respond`

Target test shape:

```go
func TestDeriveAgentExecutionPlanUsesBrowserSequenceForControlledPageGoal(t *testing.T) {
	goal := "Open the controlled browser fixture, type browser demo text, click apply, extract the result, and reply."
	plan := deriveAgentExecutionPlan(goal, 5)
	require.Equal(t, "browser_then_respond", plan.Mode)
	require.Equal(t, []string{"browser_open", "browser_type", "browser_click", "browser_extract", "respond"}, plan.RequiredSequence)
}
```

- [ ] **Step 2: Run the focused runner tests to verify they fail first**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runner -run "Agent|Browser|Plan" -v
```

Expected: FAIL because browser-oriented planning is not fully locked yet.

- [ ] **Step 3: Implement the minimal browser planning mode**

In `agent_loop.go`, add one explicit plan mode for controlled browser goals:

```go
AgentExecutionPlan{
	Mode: "browser_then_respond",
	Summary: "open controlled page, interact, extract stable result, then answer",
	RequiredSequence: []string{
		"browser_open",
		"browser_type",
		"browser_click",
		"browser_extract",
		"respond",
	},
}
```

Keep scope narrow:
- only controlled local pages
- only existing browser actions
- no public-web agent mode

- [ ] **Step 4: Re-run the focused runner tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runner -run "Agent|Browser|Plan" -v
```

Expected: PASS for browser-plan derivation.

- [ ] **Step 5: Commit**

```bash
git add internal/core/runner/agent_loop.go internal/core/runner/runner_test.go
git commit -m "feat: add browser-oriented agent planning mode"
```

### Task 2: Stabilize Browser Child Task Sequencing And Failure Semantics

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runner\agent_loop.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runner\runner_test.go`

- [ ] **Step 1: Write failing tests for browser child-task orchestration**

Cover:
- `browser_open` creates a `browser.open` child task
- `browser_type`, `browser_click`, `browser_extract`, and optional `browser.screenshot` are emitted in order
- selector failures produce stable parent summaries
- extract failures surface as `agent child task failed: ...`

Example:

```go
func TestAgentRunCreatesBrowserChildTasksInStableOrder(t *testing.T) {
	// create thread
	// run agent goal against controlled browser fixture
	// assert child kinds are:
	// browser.open, browser.type, browser.click, browser.extract
}
```

- [ ] **Step 2: Run the focused orchestration tests to verify they fail first**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runner -run "BrowserChild|BrowserSequence|AgentChildTask" -v
```

Expected: FAIL until browser orchestration is fully stabilized.

- [ ] **Step 3: Implement the minimal sequencing and correction rules**

Add sequence validation/correction logic that keeps browser flows narrow:

```go
func validateAgentActionSequence(plan AgentExecutionPlan, actions []AgentAction) error
func correctAgentActionSequence(plan AgentExecutionPlan, action AgentAction) AgentAction
```

Rules:
- first browser action must be `browser_open`
- no `browser_click` before `browser_type` when the goal explicitly says "type then click"
- `browser_extract` must happen before `respond`
- `browser_screenshot` is optional and only allowed before `respond`

- [ ] **Step 4: Re-run the focused orchestration tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runner -run "BrowserChild|BrowserSequence|AgentChildTask" -v
```

Expected: PASS with stable browser child-task behavior.

- [ ] **Step 5: Commit**

```bash
git add internal/core/runner/agent_loop.go internal/core/runner/runner_test.go
git commit -m "feat: stabilize browser child task sequencing for agent run"
```

### Task 3: Make Browser Agent State Readable In CLI And Desktop

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`

- [ ] **Step 1: Write output expectations first**

CLI should show:
- parent `agent.run`
- latest browser child task
- waiting state if any
- extract summary if present
- screenshot artifact hint if present

Desktop should show, using existing runtime data only:
- latest browser child kind
- latest extract summary
- screenshot artifact path or basename

- [ ] **Step 2: Lock CLI output with tests**

Add a focused assertion in `cmd/cli/main_test.go`:

```go
func TestTasksListShowsBrowserChildAgentState(t *testing.T) {
	output := runCLI(t, "tasks", "list", "--thread=thread-browser")
	require.Contains(t, output, "agent.run")
	require.Contains(t, output, "browser.extract")
	require.Contains(t, output, "browser screenshot captured")
}
```

- [ ] **Step 3: Implement minimal CLI and desktop formatting**

Keep it purely presentational:

```go
// CLI
if task.Kind == "agent.run" && task.LatestChildTaskID != "" {
	// print latest child kind / waiting status / result summary
}
```

```tsx
// Desktop
const latestBrowserSummary = latestBrowserTask?.resultSummary ?? "";
const latestBrowserArtifact = extractArtifactPath(latestBrowserSummary);
```

- [ ] **Step 4: Validate presentation layers**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./cmd/cli -run "Task|Browser|Agent" -v
```

Expected: PASS for build and focused CLI tests.

- [ ] **Step 5: Commit**

```bash
git add cmd/cli/main.go cmd/cli/main_test.go desktop/frontend/src/App.tsx
git commit -m "feat: surface browser agent state in cli and desktop"
```

### Task 4: Add A UI-First Canonical Browser Agent Acceptance Scenario

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\verify-desktop-live-refresh.py`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\run-desktop-live-refresh-check.ps1`

- [ ] **Step 1: Write the scenario shape first in the verifier**

Add one canonical browser-agent scenario that is created from the desktop UI flow:

```python
def run_agent_browser_controlled_scenario(page, thread_id: str, thread_name: str, run_id: str) -> dict:
    # create agent.run from UI or UI-first path
    # wait for browser.open child
    # wait for browser.type / click / extract
    # optionally wait for browser.screenshot
    # assert assistant message visible
```

Summary requirements:
- include parent task id
- child task ids
- child kinds
- extract summary
- screenshot artifact path if present
- assistant message id/content

- [ ] **Step 2: Run syntax validation first**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
python -m py_compile .\scripts\verify-desktop-live-refresh.py
```

Expected: PASS.

- [ ] **Step 3: Implement the minimal canonical agent-browser scenario**

Keep the goal explicit and controlled:

```python
goal = (
    "Open the controlled browser fixture, type controlled browser acceptance, "
    "click apply, extract the result text, take a screenshot, and reply in one short sentence."
)
```

Acceptance expectations:
- parent task kind is `agent.run`
- child tasks include `browser.open`, `browser.type`, `browser.click`, `browser.extract`
- optional `browser.screenshot` is allowed
- final assistant message is visible in the workbench

- [ ] **Step 4: Wire the scenario into the full acceptance summary**

Add one result block:

```python
"uiFirstCanonicalBrowserAgentScenario": browser_agent_result
```

Keep `smoke` unchanged.

- [ ] **Step 5: Run the canonical wrapper**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected:
- PASS in `full`
- summary includes `uiFirstCanonicalBrowserAgentScenario`
- existing direct browser lanes remain green

- [ ] **Step 6: Commit**

```bash
git add scripts/verify-desktop-live-refresh.py scripts/run-desktop-live-refresh-check.ps1
git commit -m "test: add canonical browser agent acceptance scenario"
```

### Task 5: Update Browser Capability And Runtime Entry Docs

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-capability-matrix-and-tool-coverage.md`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-runtime-entry-checklist.md`

- [ ] **Step 1: Add the new browser-agent baseline wording**

Document:
- direct browser tasks are already canonical
- `agent.run` can now orchestrate controlled browser flows
- public-web remains read-only and direct-task oriented unless explicitly expanded later

- [ ] **Step 2: Keep the acceptance lane boundary explicit**

Add wording like:

```markdown
- `smoke`: no browser interaction
- `full`: direct browser lanes + controlled browser `agent.run`
- `fallback`: evidence-only, not a canonical browser gate
```

- [ ] **Step 3: Sanity-check the docs**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
Select-String -Path .\docs\superpowers\plans\2026-05-17-capability-matrix-and-tool-coverage.md -Pattern "agent.run|browser"
Select-String -Path .\docs\superpowers\plans\2026-05-17-runtime-entry-checklist.md -Pattern "smoke|full|fallback|browser"
```

Expected:
- The updated browser-agent wording is present in both docs.

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md
git commit -m "docs: record browser agent acceptance baseline"
```

## Self-Review

- Spec coverage:
  - browser tools already exist: preserved
  - next step is agent-quality and acceptance closure: covered by Tasks 1-4
  - no new routes or browser protocol: preserved
  - canonical remote gate remains the same: documented in Task 5
- Placeholder scan:
  - no `TODO` or `implement later` placeholders remain
  - each task includes explicit files, commands, and expected outcomes
- Type consistency:
  - plan mode uses `browser_then_respond`
  - child actions use `browser_open`, `browser_type`, `browser_click`, `browser_extract`, `browser_screenshot`, `respond`
  - runtime/browser tool names remain `browser.open`, `browser.type`, `browser.click`, `browser.extract`, `browser.screenshot`

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-18-agent-browser-workflow-acceptance-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
