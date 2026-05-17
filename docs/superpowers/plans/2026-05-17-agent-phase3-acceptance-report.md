# Agent Phase 3 Acceptance Report

## Scope

This report is the Phase B closeout for the currently verified acceptance surface. It consolidates:

- the canonical live regression lane on `http://127.0.0.1:5174/ + http://127.0.0.1:10008`
- the supported fallback verification lane for `desktop local-fallback`
- the currently verified capability set
- the residual risks that remain outside the release gate

This document does not claim new runtime behavior. It records the current verified state.

It also does not claim that grouped skill inventory or MCP metadata listing has become fully productized acceptance coverage.

## Final Verdict

The current phase is closed enough to use one explicit two-lane acceptance model:

- `remote-app-server` is the canonical browser-level release gate
- `desktop local-fallback` is the supporting persistence and restart-consistency lane

That is the current honest verified boundary. The remote lane is the pass/fail path. The fallback lane is supporting evidence, not a substitute.

## Verified Capabilities

### Canonical remote lane

Verified on the default target:

- UI: `http://127.0.0.1:5174/`
- API: `http://127.0.0.1:10008`

Verified through [verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py) and the wrapper script [run-desktop-live-refresh-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1):

- runtime status resolves to `runtimeSource = remote-app-server`
- runtime status resolves to `runtimeTrust = canonical`
- direct second-batch tool tasks complete for:
  - `workspace.stat_file`
  - `workspace.read_files_batch`
  - `workspace.list_files_filtered`
  - `workspace.search_text_detailed`
- `agent.run` completes for all currently verified constrained read plans:
  - `filter_then_read`
  - `search_then_detailed`
  - `stat_then_read`
- `workspace.apply_patch` enters approval, executes after approval, and records a write execution
- `workspace.apply_patch.rollback` enters approval, executes after approval, and records a rollback write execution
- browser-level workflow visibility is present for thread, task, approval, and latest write execution evidence

### Local fallback lane

Verified primarily by `desktop` Go tests rather than by browser automation.

Current verified fallback evidence includes:

- persisted thread and task state across app restart
- parent/child linkage and waiting-state summary persistence
- approval persistence for `waiting_for_approval`
- recovery persistence for `waiting_for_task`
- write execution persistence across restart

The strongest named tests remain:

- `TestDesktopFallbackPersistsAcrossAppRestart`
- `TestDesktopFallbackTaskSummariesKeepParentAndWaitingFields`
- `TestDesktopFallbackWriteExecutionsPersistAcrossRestart`
- `TestDesktopFallbackAgentWaitingForApprovalPersistsAcrossRestart`
- `TestDesktopFallbackAgentWaitingForTaskPersistsAcrossRestart`

## Release Baseline

The current short release checklist is now:

```text
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...

Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build

Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Interpretation:

- all three commands must pass for a release-candidate regression pass
- the live script must start on canonical remote runtime, not `local-fallback`
- fallback remains required evidence, but through Go-test coverage rather than a second browser gate

Reference checklist:

- [2026-05-17-runtime-entry-checklist.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md)

## Side-by-Side Summary

| Area | Remote app-server | Desktop local-fallback |
| --- | --- | --- |
| Primary acceptance method | Python Playwright against live UI and API | Focused Go tests over fallback persistence and restart behavior |
| Runtime source | `remote-app-server` | `local-fallback` |
| SSE expectation | Yes, part of the default browser lane | No, manual refresh mode is expected |
| Parent/child task visibility | Browser verified | Runtime snapshot and restart semantics verified in tests |
| Approval visibility | Browser verified on apply + rollback | Approval persistence and task state consistency verified in tests |
| Write execution visibility | Browser verified through latest write execution panel | Write execution persistence and restart recovery verified in tests |
| Pass/fail role in this phase | Canonical live acceptance gate | Supporting correctness and recovery evidence |

## Remote Canonical Lane

Canonical target:

- UI: `http://127.0.0.1:5174/`
- API: `http://127.0.0.1:10008`

Verification tool:

- [verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)

### What the script now reports

The script keeps its previous default behavior intact and now emits an additional structured `acceptanceReport.remote` section that captures:

- runtime source and runtime trust
- canonical runtime URL when present
- refresh-mode evidence:
  - `SSE 实时刷新`
  - `SSE 重连中`
  - or `手动刷新`
- parent/child task visibility for `agent.run`
- approval visibility for:
  - `workspace.apply_patch`
  - `workspace.apply_patch.rollback`
- write execution visibility through the right-side write execution panel
- direct tool visibility counts

### Latest verification outcome

The updated script was re-run successfully on May 17, 2026 against the default canonical lane.

Verified thread:

- `thread-174`

Observed outcome:

- runtime status resolved to `remote-app-server`
- runtime trust resolved to `canonical`
- the UI refresh signal resolved to `SSE 实时刷新`
- the Playwright lane still completed:
  - 4 direct second-batch read-tool tasks
  - 3 `agent.run` scenarios
  - 1 `workspace.apply_patch` approval flow
  - 1 rollback approval flow
- direct tool visibility was fully confirmed in the task stream:
  - 4 by title
  - 0 requiring tool-kind fallback matching
- `agent.run` parent visibility was confirmed for all 3 scenarios
- child-task visibility was confirmed for 2 of 3 scenarios in the live UI
- assistant message visibility was confirmed for all 3 scenarios
- approval visibility was confirmed for both apply and rollback
- write execution visibility was confirmed for both apply and rollback through the write execution panel
- the output now includes a structured `acceptanceReport.remote` object for reporting

Meaning:

- we preserved the existing remote acceptance path
- we now have structured evidence instead of relying only on ad hoc prose in the final JSON payload

Practical note:

- the `stat_then_read` scenario still completed correctly in runtime/API terms, but did not expose a stable child-task card signal in this particular browser run
- this is acceptable for the current phase because the parent task, assistant message, and runtime-side child records were all present

## Local Fallback Lane

Fallback target:

- desktop local store and runtime snapshot path

Primary evidence source:

- Go tests in [desktop/app_test.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)

### Why browser automation is not the primary lane here

Fallback is not the same product surface as the canonical remote app-server:

- it does not provide the same stable SSE behavior
- it is intentionally allowed to operate in manual refresh mode
- the higher-value risk on this lane is restart consistency and persisted workflow semantics, not browser click-through parity

Because of that, this phase does **not** treat full browser automation of fallback as a required pass condition.

### Current fallback evidence

The current fallback evidence is grounded in the following tests:

- `TestDesktopFallbackPersistsAcrossAppRestart`
- `TestDesktopFallbackTaskSummariesKeepParentAndWaitingFields`
- `TestDesktopFallbackWriteExecutionsPersistAcrossRestart`
- `TestDesktopFallbackAgentWaitingForApprovalPersistsAcrossRestart`
- `TestDesktopFallbackAgentWaitingForTaskPersistsAcrossRestart`

What these tests cover:

- persisted thread and task state across app restart
- stable `agent.run` waiting-state summaries
- parent/child linkage visibility in persisted descriptors
- approval-related waiting state persistence
- write execution persistence across restart
- local-fallback recovery consistency for the two important agent waiting states:
  - `waiting_for_approval`
  - `waiting_for_task`

Meaning:

- fallback correctness is currently evidenced at the state/recovery layer
- that is the right acceptance depth for this lane in the current phase

## Validation Run

The following verification was run after the Task 4 edits:

```text
python .\scripts\verify-desktop-live-refresh.py
GOTOOLCHAIN=auto go test ./...   (in desktop)
```

Summary:

- the remote browser lane remained green after the reporting changes
- the fallback lane remains covered by targeted Go-test evidence rather than new browser automation
- one earlier rerun hit a transient upstream provider `EOF`; a subsequent rerun completed successfully without code changes, so that failure is treated as an environment blip rather than a reporting regression

## What Changed in Task 4

Files changed:

- [verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
- [2026-05-17-agent-phase3-acceptance-report.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-agent-phase3-acceptance-report.md)

Behavioral change summary:

- no runtime behavior changed
- no backend API changed
- the default remote Playwright path still works as before
- the script now emits structured reporting for:
  - runtime source
  - refresh/SSE evidence
  - parent-child visibility
  - approval visibility
  - write execution visibility
- the phase now has an explicit written remote-vs-fallback acceptance split

## Residual Risks

The following remain open after this closeout:

1. A first-class browser automation lane for `desktop local-fallback` is still intentionally not closed.
2. Some browser assertions remain intentionally tolerant because the current desktop surface does not render every internal identifier with stable UI-level text.
3. Broader capability expansion remains outside this closeout:
   - full Codex / CC capability expansion
   - MCP end-to-end execution acceptance
   - broader skill-group verification and localization audit
   - per-skill governance verification across `common`, `codex`, and `cc`
4. Parallel work by other agents may continue elsewhere in the repository, so this report should be read as documentation of the current verified baseline rather than a claim that every in-flight change outside this docs scope is release-ready.

## Closeout Position

Phase B is closed for documentation purposes around the current verified execution surface:

- one fixed release checklist now exists
- one explicit acceptance model now exists
- the remote canonical gate and fallback supporting lane are clearly separated
- the verified capabilities and remaining risks are recorded in one place
- grouped skill governance and MCP health remain separately tracked baseline items rather than implied by this report
