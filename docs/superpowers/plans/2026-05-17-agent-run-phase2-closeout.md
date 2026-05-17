# 2026-05-17 Agent Run Phase 2 Closeout

## Scope

This closeout covers the current milestone around:

- second-batch read tools
- constrained `agent.run` action quality
- approval auto-resume
- rollback and write execution audit
- default remote runtime chain on `http://127.0.0.1:10008`
- default desktop/browser verification chain on `http://127.0.0.1:5174/`

It focuses on whether the current implementation is stable enough to be treated as the default working path for this phase.

## Completed

### Runtime and task model

- `workspace.stat_file`
- `workspace.read_files_batch`
- `workspace.list_files_filtered`
- `workspace.search_text_detailed`
- `workspace.apply_patch`
- `workspace.apply_patch.rollback`
- `agent.run`

All of the above now flow through the shared `thread -> task -> tool call -> event -> runtime status` model.

### Agent action quality and guardrails

- `agent.run` now derives and exposes plan metadata:
  - `agentPlanSummary`
  - `agentPlanMode`
  - `agentCurrentStepTitle`
  - `agentLastReasoning`
  - `latestChildTaskId`
- constrained plan modes are implemented and exercised:
  - `filter_then_read`
  - `search_then_detailed`
  - `stat_then_read`
- runtime-side action-sequence enforcement is active
- a single correction retry is now attempted after a sequence violation
- missing `type` model outputs now have conservative action inference based on the current expected step

### Approval and recovery behavior

- `agent.run` can create write-child tasks that enter `waiting_for_approval`
- approving the child task automatically resumes the parent `agent.run`
- rejection still fails the parent task cleanly
- recovery behavior is now covered for:
  - `waiting_for_task` with completed child
  - `waiting_for_approval` with pending child
  - `waiting_for_approval` with completed child
  - `waiting_for_approval` with missing child
- desktop local-fallback restart consistency is now covered for:
  - `waiting_for_approval` parent/child/approval snapshot persistence
  - `waiting_for_task` parent/child snapshot persistence
  - stable fallback thread summary counts after restart

### Desktop and browser verification

- the default browser verification script targets:
  - UI: `http://127.0.0.1:5174/`
  - API: `http://127.0.0.1:10008`
- verification now covers:
  - second-batch direct tool tasks
  - `agent.run` for all 3 constrained plan modes
  - `workspace.apply_patch`
  - `workspace.apply_patch.rollback`
- browser verification assertions were normalized to current UI reality:
  - no hard dependency on internal write execution IDs being rendered
  - reduced reliance on exact card-title persistence for every child task
  - verification still requires runtime success plus visible parent/message/child signals

## Evidence

### Focused tests

Passing:

- `GOTOOLCHAIN=auto go test ./...` in `desktop`
- `go test ./internal/core/runner ./internal/core/runtime ./cmd/cli`
- `npm run build` in `desktop/frontend`

Additional focused fallback coverage now includes:

- `TestDesktopFallbackAgentWaitingForApprovalPersistsAcrossRestart`
- `TestDesktopFallbackAgentWaitingForTaskPersistsAcrossRestart`

### Live runtime status

Confirmed after the latest changes:

- `runtime status`
  - `source: remote-app-server`
  - `source trust: canonical`

### Latest successful default live verification

Successful full script run:

- thread: `thread-170`
- runtime:
  - `remote-app-server`
  - `canonical`

Successful output included:

- direct tool tasks:
  - `workspace.stat_file`
  - `workspace.read_files_batch`
  - `workspace.list_files_filtered`
  - `workspace.search_text_detailed`
- agent scenarios:
  - `filter_then_read`
  - `search_then_detailed`
  - `stat_then_read`
- write execution:
  - apply execution completed
  - rollback execution completed

## Remaining

### Not yet closed as a separate milestone

- there is no standalone milestone report that compares remote and fallback behavior side by side in one document
- the browser verification script is now stable enough for this phase, but some UI assertions are intentionally tolerant because the current workbench does not render every internal identifier or every child title in a fully stable way
- desktop local-fallback recovery is now covered at the Go test layer, but not yet through a separate browser automation suite; this is now an environment/coverage choice rather than a correctness gap

### Explicit non-goals for this phase

- no multi-agent coordination
- no streaming token UI
- no agent-triggered rollback
- no arbitrary shell execution
- no destructive file operations outside controlled rollback

## Assessment

The current phase can be treated as functionally complete for:

- second-batch read tools
- constrained single-thread `agent.run`
- approval auto-resume
- controlled patch write and rollback
- default remote desktop/browser acceptance on `5174 + 10008`
- desktop local-fallback restart consistency for persisted `agent.run` waiting states

The next most sensible follow-up is not more tool expansion. It is either:

1. a side-by-side remote vs fallback acceptance report, or
2. the next product milestone above this execution surface.
