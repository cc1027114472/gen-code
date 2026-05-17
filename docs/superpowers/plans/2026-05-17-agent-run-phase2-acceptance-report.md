# Agent Run Phase 2 Acceptance Report

> Milestone A closure note: this report now reflects the post-Task-3 state, where the canonical remote runtime and the desktop Playwright lane were both re-verified on May 17, 2026.

## Scope

This report closes the current phase around:

- `agent.run`
- child read tasks
- `workspace.apply_patch`
- `ask-user` approval
- auto resume after approval
- write execution audit
- rollback
- remote app-server and local fallback consistency

This report now includes a passed browser-level Playwright acceptance lane and should be treated as the current Milestone A baseline.

## Final Verdict

The core execution surface for the `agent.run` phase 2 plan is functionally complete, and Milestone A is now closed for runtime trust, desktop workflow visibility, and one repeatable desktop acceptance lane.

Confirmed as completed:

- parent `agent.run` task orchestration
- child task creation and persistence
- `waiting_for_task` and `waiting_for_approval` state flow
- auto resume after approving a child write task
- failure propagation for approval rejection and permission denial
- write execution audit records for apply and rollback
- rollback through the same task and approval pipeline
- local fallback verification
- clean-port remote app-server verification

Still outside Milestone A:

- full Codex / CC capability expansion
- MCP productization
- stronger productization of `canonicalRuntimeUrl` reporting, which is still empty in the latest successful acceptance payload

## Real Verification Evidence

### 1. Local fallback chain

Verified thread:

- `thread-109`

Key records:

- parent task: `task-172`
- child patch task: `task-175`
- approval: `approval-55`
- write execution: `writeexec-26`
- assistant message: `message-8`

Observed result:

- parent task completed
- child patch task completed with `approval=executed`
- assistant message persisted
- tool calls persisted
- write execution persisted

Meaning:

- fallback path already proves `agent.run -> child write task -> approval -> auto resume -> final assistant output`

### 2. Remote app-server chain

Initial clean-port verification target:

- `http://127.0.0.1:10018`

Verified thread:

- `thread-114`

Key records:

- parent task: `task-176`
- child patch task: `task-177`
- rollback task: `task-178`
- approvals: `approval-56`, `approval-57`
- write executions: `writeexec-27`, `writeexec-28`
- assistant message: `message-9`

Observed result:

- `task-176` entered `waiting_for_approval`
- approving `task-177` executed the patch and auto resumed the parent agent task
- parent task completed and wrote an assistant message
- rollback task entered approval as expected under `ask-user`
- approving `task-178` completed rollback and removed the temporary file
- write execution history correctly recorded `apply -> rollback`

Meaning:

- remote app-server path now proves the same chain as fallback
- rollback is not only unit-tested, but also executed through the real remote task pipeline

### 3. Canonical remote runtime and browser acceptance lane

Canonical verification target:

- UI: `http://127.0.0.1:5174/`
- API: `http://127.0.0.1:10008`

Verified thread:

- `thread-138`

Key records:

- direct tool tasks: `task-239` through `task-242`
- parent agent task: `task-243`
- write task: `task-246`
- apply write execution: `writeexec-41`
- rollback task: `task-247`
- rollback write execution: `writeexec-42`

Observed result:

- `/api/runtime/status` reported `runtimeSource=remote-app-server`
- `/api/runtime/status` reported `runtimeTrust=canonical`
- Python Playwright created a fresh thread and completed direct tool checks
- `agent.run` completed with second-batch child tasks
- `workspace.apply_patch` entered approval and completed after approval
- rollback entered approval and completed after approval
- the wrapper script finished successfully from the repository root

Meaning:

- the default remote runtime on `127.0.0.1:10008` is now a trustworthy Milestone A acceptance target
- desktop browser-level click-through acceptance is now closed through the Python Playwright lane

## Test Baseline Completed

The following checks passed during this phase:

```text
go test ./internal/core/runner ./internal/core/runtime ./internal/core/session ./internal/router ./internal/handler ./cmd/cli
go test ./...   (in desktop)
npm run build   (in desktop/frontend)
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Additional real checks completed:

- CLI `runtime status`
- CLI `tools list`
- CLI `providers list`
- real `agent run` on fallback
- real `agent run` on remote app-server
- real approval and auto resume
- real rollback approval and execution
- real desktop Playwright approval and rollback flow on canonical runtime `10008`

## What Is Closed Against The Plan

### State machine and recovery

Closed:

- parent task states include `waiting_for_task` and `waiting_for_approval`
- parent-child linkage is persisted
- fallback recovery behavior was hardened to avoid false failure marking from short-lived CLI processes
- SQLite shared-state behavior was hardened for concurrent access

### Approval auto resume

Closed:

- child patch task waits for approval under `ask-user`
- parent `agent.run` waits instead of silently failing
- `ApproveTask` executes the child task and resumes the parent automatically
- approval rejection causes a stable parent failure result

### Runtime, API, CLI consistency

Closed:

- `TaskDescriptor` now carries parent/waiting/agent progress semantics needed by CLI and desktop
- CLI can display parent-child relationships and waiting states
- remote and fallback now expose the same task family and summary semantics
- runtime source semantics are aligned around `remote-app-server` vs `local-fallback`
- runtime trust semantics are aligned around `canonical` vs `degraded`

### Write audit and rollback

Closed:

- apply writes create write execution records
- rollback writes create independent rollback execution records
- rollback links back to the source apply execution
- latest-only rollback guard is active
- drift protection remains active

## What Is Not Fully Closed Yet

### 1. `canonicalRuntimeUrl` completeness

Status:

- partially closed

Reason:

- the runtime status now exposes `runtimeSource` and `runtimeTrust` correctly
- the latest successful acceptance payload still reported an empty `canonicalRuntimeUrl`

Risk:

- scripts currently allow this as long as source and trust are canonical, but productization is incomplete

### 2. Playwright UI assertion fragility

Status:

- partially closed

Reason:

- the primary Python Playwright lane passed against the canonical runtime during Milestone A work
- the latest rerun still failed on exact article text matching for agent-summary prose, even though API-side execution and the runtime/test layers remained healthy

Risk:

- the lane is usable, but its UI assertions are still more text-coupled than ideal

### 3. `agent-browser` environment

Status:

- not closed

Reason:

- `agent-browser` on this Windows machine still shows session cleanup and stability problems

Reference:

- [2026-05-17-agent-browser-min-repro.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-agent-browser-min-repro.md)

Impact:

- it should not be used as the pass/fail gate for this phase

## Residual Risks

- `canonicalRuntimeUrl` is not yet consistently populated in the latest acceptance payload
- existing repository encoding issues such as the already-garbled `AGENTS.md` content remain outside the core Milestone A closure work

## Recommended Next Step

Recommended priority order:

1. tighten `canonicalRuntimeUrl` reporting so the runtime contract is fully self-describing
2. continue to the next feature phase only after documenting the Milestone A closure state
3. keep `agent-browser` debugging separate from the Milestone A pass/fail gate

## Conclusion

For the `Agent Phase 2` objective and Milestone A scope, the execution core is complete and verified enough to move forward.

The remaining issues are now mostly outside Milestone A: capability expansion, MCP productization, and cleanup of a few contract-level polish gaps such as `canonicalRuntimeUrl`.
