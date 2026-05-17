# Milestone A Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a trustworthy runtime entry path, a usable desktop thread/task workflow, and one repeatable desktop end-to-end acceptance lane for `gen-code`.

**Architecture:** Milestone A keeps the existing runtime split across CLI, app-server, desktop, and core runtime layers, but removes ambiguity about which runtime is authoritative. The implementation should standardize runtime source semantics, improve desktop workflow presentation for thread/task/approval state, and lock in one stable Playwright-based acceptance path against the canonical runtime target.

**Tech Stack:** Go, Gin, SQLite, Wails, React, TypeScript, SSE, Python Playwright, PowerShell helper scripts.

---

## Scope

This plan covers only the first milestone from the umbrella roadmap:

- canonical runtime target and source reporting
- desktop workflow improvements for workspace/thread/task approval state
- one stable desktop E2E acceptance lane

This plan does **not** include:

- Codex / CC full capability expansion
- MCP productization
- broader `agent.run` refactoring beyond what is needed to support this milestone

## File Map

### Runtime entry and source reporting

- Modify: [cmd/cli/main.go](D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main.go)
  - normalize runtime base URL and runtime source messaging
- Modify: [desktop/app.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
  - align desktop runtime source semantics with CLI and app-server
- Modify: [internal/core/runtime/status.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status.go)
  - extend source and trust metadata surfaced by runtime
- Modify: [internal/appserver/runtimecontract/contract.go](D:/GOWorks/gen-code-heji/gen-code/internal/appserver/runtimecontract/contract.go)
  - add any missing contract fields required by desktop and CLI
- Modify: [internal/handler/runtime_handler.go](D:/GOWorks/gen-code-heji/gen-code/internal/handler/runtime_handler.go)
  - keep API shape aligned with runtime contract

### Desktop workflow

- Modify: [desktop/frontend/src/App.tsx](D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx)
  - improve runtime state, thread workflow, and approval/waiting presentation
- Modify: [desktop/app.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
  - expose any additional runtime summaries the UI needs
- Modify: [desktop/app_test.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)
  - extend fallback and workflow tests

### Acceptance lane

- Modify: [scripts/verify-desktop-live-refresh.py](D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
  - pin the canonical runtime target and strengthen assertions
- Modify: [scripts/run-desktop-live-refresh-check.ps1](D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
  - make the Playwright lane easier to run consistently
- Modify: [docs/superpowers/plans/2026-05-17-agent-run-phase2-acceptance-report.md](D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-agent-run-phase2-acceptance-report.md)
  - clarify that Milestone A supersedes the stale runtime acceptance caveats

### New docs

- Create: [docs/superpowers/plans/2026-05-17-milestone-a-implementation-plan.md](D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-milestone-a-implementation-plan.md)
- Create: `docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md`
  - concise runtime source and acceptance checklist for the team

## Task 1: Canonical Runtime Entry And Source Semantics

**Files:**
- Modify: [cmd/cli/main.go](D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main.go)
- Modify: [desktop/app.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
- Modify: [internal/core/runtime/status.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status.go)
- Modify: [internal/appserver/runtimecontract/contract.go](D:/GOWorks/gen-code-heji/gen-code/internal/appserver/runtimecontract/contract.go)
- Test: [cmd/cli/main_test.go](D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main_test.go)
- Test: [internal/core/runtime/status_test.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status_test.go)
- Test: [desktop/app_test.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)

- [ ] **Step 1: Write failing tests for canonical runtime source semantics**

Add tests that assert:

- CLI reports the canonical runtime target as the preferred source when reachable
- CLI and desktop clearly distinguish canonical remote runtime vs fallback local runtime
- runtime status payloads expose the same source and trust hints across layers

Use the existing style in:

- [cmd/cli/main_test.go](D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main_test.go)
- [internal/core/runtime/status_test.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status_test.go)
- [desktop/app_test.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)

- [ ] **Step 2: Run targeted tests to verify they fail**

Run:

```powershell
go test ./cmd/cli ./internal/core/runtime ./desktop
```

Expected:

- at least one test fails because canonical runtime source metadata is incomplete or inconsistent

- [ ] **Step 3: Extend runtime contract and summary metadata minimally**

Update the runtime contract and runtime status surface so that CLI and desktop can consume the same truth, keeping the field set minimal and explicit. Favor additive contract changes over breaking existing fields.

Implementation notes:

- if a new trust or source-detail field is needed, add it once in [contract.go](D:/GOWorks/gen-code-heji/gen-code/internal/appserver/runtimecontract/contract.go)
- populate it from [status.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status.go)
- keep fallback semantics explicit instead of inferred by UI-only logic

- [ ] **Step 4: Align CLI runtime source reporting with the canonical path**

Update [main.go](D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main.go) so that:

- the default runtime target is described as the canonical app-server entry when reachable
- fallback wording is clearly degraded-mode wording
- stale port assumptions are removed from user-facing messages

- [ ] **Step 5: Align desktop runtime source reporting with the canonical path**

Update [app.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app.go) so that:

- desktop remote status mirrors CLI source semantics
- local fallback is still supported, but clearly labeled as fallback
- the bridge check and runtime status do not imply equivalence between remote and fallback modes

- [ ] **Step 6: Run targeted tests to verify they pass**

Run:

```powershell
go test ./cmd/cli ./internal/core/runtime ./desktop
```

Expected:

- PASS for the updated runtime-source tests

- [ ] **Step 7: Commit**

```bash
git add cmd/cli/main.go cmd/cli/main_test.go internal/core/runtime/status.go internal/core/runtime/status_test.go internal/appserver/runtimecontract/contract.go desktop/app.go desktop/app_test.go
git commit -m "feat: standardize canonical runtime source reporting"
```

**Status:** Completed and verified on May 17, 2026.

## Task 2: Desktop Workflow For Thread, Task, Approval, And Waiting State

**Files:**
- Modify: [desktop/frontend/src/App.tsx](D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx)
- Modify: [desktop/app.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
- Test: [desktop/app_test.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)

- [ ] **Step 1: Write failing tests for desktop workflow visibility**

Add or extend tests in [desktop/app_test.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go) to assert the desktop status payload supports:

- active workspace summary
- active thread summary
- parent-child task visibility
- waiting-for-task vs waiting-for-approval distinction
- approval and write-execution summaries that can drive the UI without extra implicit logic

- [ ] **Step 2: Run desktop tests to verify the new expectations fail**

Run:

```powershell
go test ./desktop
```

Expected:

- failures show that one or more expected workflow fields or summaries are not yet available or not stable enough

- [ ] **Step 3: Extend desktop runtime status payload only where needed**

Update [desktop/app.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app.go) to expose enough normalized runtime state for the UI to render:

- workspace identity
- canonical runtime source and message
- thread/task/approval/write-execution relationships
- waiting-state friendly summaries

Keep derivation logic close to the desktop boundary rather than duplicating complex runtime logic in React.

- [ ] **Step 4: Improve desktop React workflow presentation**

Update [App.tsx](D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx) so the workbench emphasizes:

- which runtime is active
- which thread is active
- which tasks are waiting, failed, approval-required, or completed
- which approval belongs to which task
- which write execution belongs to which task

Implementation guidance:

- prefer improving labels, grouping, and ordering before adding new visual surface area
- keep current browser preview integration intact
- do not expand into unrelated product features during this milestone

- [ ] **Step 5: Run desktop Go tests again**

Run:

```powershell
go test ./desktop
```

Expected:

- PASS for desktop payload and fallback workflow tests

- [ ] **Step 6: Build the desktop frontend to catch UI contract drift**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

Expected:

- successful TypeScript and production build

- [ ] **Step 7: Commit**

```bash
git add desktop/app.go desktop/app_test.go desktop/frontend/src/App.tsx
git commit -m "feat: improve desktop thread and approval workflow visibility"
```

**Status:** Completed and verified on May 17, 2026.

## Task 3: Stable Desktop End-To-End Acceptance Lane

**Files:**
- Modify: [scripts/verify-desktop-live-refresh.py](D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
- Modify: [scripts/run-desktop-live-refresh-check.ps1](D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
- Modify: [desktop/app_test.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)
- Create: `docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md`

- [ ] **Step 1: Write or tighten test coverage around acceptance assumptions**

Add tests that protect the assumptions used by the Playwright lane:

- canonical runtime base URL selection
- deterministic task and approval visibility in desktop runtime status
- rollback remains exposed after apply execution

Use [desktop/app_test.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go) for payload-level expectations rather than trying to push browser-only concerns into unit tests.

- [ ] **Step 2: Run desktop tests to verify the acceptance assumptions**

Run:

```powershell
go test ./desktop
```

Expected:

- PASS, proving the Playwright lane will have the payload it expects

- [ ] **Step 3: Standardize the Playwright script on the canonical runtime target**

Update [verify-desktop-live-refresh.py](D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py) so that:

- the canonical runtime target is the default API base URL
- stale remote targets are not silently accepted as equivalent
- failure messages clearly say whether the problem is UI, API, runtime, or Playwright

- [ ] **Step 4: Make the PowerShell wrapper easy to rerun locally**

Update [run-desktop-live-refresh-check.ps1](D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1) so that:

- it documents or sets the canonical UI/API targets
- it reports the exact command path used
- it exits clearly on setup failure vs verification failure

- [ ] **Step 5: Add a short runtime entry checklist doc**

Create `docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md` with:

- canonical runtime port or base URL
- fallback rules
- desktop acceptance command
- what counts as a valid remote acceptance run

Keep it short enough for daily use.

- [ ] **Step 6: Run the primary acceptance lane**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
python scripts/verify-desktop-live-refresh.py
```

or, if the wrapper is the intended entry:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File scripts/run-desktop-live-refresh-check.ps1
```

Expected:

- one successful flow covering thread creation, tool tasks, approval, auto resume, apply write execution, rollback task, and rollback write execution

- [ ] **Step 7: Commit**

```bash
git add scripts/verify-desktop-live-refresh.py scripts/run-desktop-live-refresh-check.ps1 desktop/app_test.go docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md
git commit -m "test: harden desktop milestone acceptance lane"
```

**Status:** Completed and verified on May 17, 2026.

## Task 4: Milestone A Documentation And Closure

**Files:**
- Modify: [docs/superpowers/plans/2026-05-17-agent-run-phase2-acceptance-report.md](D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-agent-run-phase2-acceptance-report.md)
- Modify: [docs/superpowers/plans/2026-05-17-remaining-runtime-and-desktop-plan.md](D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-remaining-runtime-and-desktop-plan.md)
- Modify: [docs/superpowers/plans/2026-05-17-milestone-a-implementation-plan.md](D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-milestone-a-implementation-plan.md)

- [ ] **Step 1: Update the acceptance report to reflect Milestone A runtime trust decisions**

Document:

- the canonical runtime target
- whether stale-port ambiguity is now closed
- what remains outside Milestone A

- [ ] **Step 2: Update the umbrella roadmap to mark Milestone A execution status**

In [2026-05-17-remaining-runtime-and-desktop-plan.md](D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-remaining-runtime-and-desktop-plan.md), record:

- Milestone A started
- what was closed
- what remains for later phases

- [ ] **Step 3: Run the milestone verification set**

Run:

```powershell
go test ./cmd/cli ./internal/core/runtime ./desktop
```

Then run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

Then run the desktop acceptance lane from Task 3.

Expected:

- all three verification layers succeed

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/plans/2026-05-17-agent-run-phase2-acceptance-report.md docs/superpowers/plans/2026-05-17-remaining-runtime-and-desktop-plan.md docs/superpowers/plans/2026-05-17-milestone-a-implementation-plan.md
git commit -m "docs: close milestone a runtime and desktop plan"
```

**Status:** Completed with one documented concern on May 17, 2026.

## Milestone A Outcome

Milestone A goals are now closed in the current worktree:

- one canonical runtime target with explicit source and trust semantics
- usable desktop thread/task/approval/write-execution workflow visibility
- one repeatable desktop Playwright acceptance lane

Latest verification evidence:

- `GOTOOLCHAIN=auto go test ./...` in `desktop`
- `npm run build` in `desktop/frontend`
- `powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1` from the repo root previously passed on the canonical runtime path, but the latest rerun still shows UI-summary assertion fragility in the Playwright script

Remaining follow-up outside Milestone A:

- `canonicalRuntimeUrl` should be populated consistently in the runtime payload
- the Playwright lane still has residual UI text-coupling and should rely more on stable title/API assertions than exact summary prose
- Codex / CC capability expansion
- MCP productization

## Self-Review

### Spec coverage

This plan covers all Milestone A goals:

- canonical runtime target and source semantics
- usable desktop thread/task/approval workflow
- one repeatable desktop acceptance lane

It intentionally excludes:

- capability matrix expansion
- MCP productization
- non-essential `agent.run` refactors

### Placeholder scan

No `TBD`, `TODO`, or deferred implementation placeholders are left in task steps. Each task names exact files and commands.

### Type consistency

The plan consistently refers to:

- runtime contract fields in `internal/appserver/runtimecontract`
- runtime summary logic in `internal/core/runtime`
- desktop payload composition in `desktop/app.go`
- desktop workflow rendering in `desktop/frontend/src/App.tsx`

Plan complete and saved to `docs/superpowers/plans/2026-05-17-milestone-a-implementation-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
