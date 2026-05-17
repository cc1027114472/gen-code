# Gen-Code Capability Matrix And Tool Coverage

**Date:** May 17, 2026

**Purpose:** Establish one concrete baseline for what `gen-code` currently supports, what has been verified end-to-end, and what remains incomplete. This document is the follow-up starting point after Milestone A.

## Status Labels

- `implemented`: code path exists in the current repository
- `verified`: implementation has been exercised by tests or the Milestone A acceptance lane
- `partial`: some runtime or UI surface exists, but coverage or workflow completeness is still missing
- `not implemented`: expected by the target product shape, but not present yet

## 1. Runtime Entry And Trust Baseline

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| Canonical app-server runtime status endpoint | verified | `internal/core/runtime/status.go`, `internal/appserver/runtimecontract/contract.go`, `scripts/verify-desktop-live-refresh.py` | Runtime status now exposes `runtimeSource`, `runtimeTrust`, `runtimeSourceDetail`, and `canonicalRuntimeUrl`. |
| Canonical runtime URL follows actual server port | verified | `internal/appserver/appserver.go`, `internal/core/runtime/status_test.go` | Verified against `http://127.0.0.1:10018` in acceptance rerun. |
| Desktop local fallback runtime | partial | `desktop/app.go` | Fallback mode exists and is labeled degraded, but still needs release-level documentation and broader regression coverage. |
| CLI / desktop / app-server source semantics alignment | partial | code already modified in current worktree; Milestone A docs reference completion | Should be re-verified as part of the next release baseline once current unrelated worktree changes settle. |

## 2. Workspace, Thread, And Task Model

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| Project-local workspace state store | implemented | `.gen-code/state.db`, `internal/core/state`, `internal/core/session` | SQLite-backed persistence is present. |
| Multiple threads under one workspace | implemented | `internal/core/session`, `runtimecontract.ThreadDescriptor` | Data model supports multi-thread workspace organization. |
| Active thread switching | implemented | runtime contract and desktop status payloads | Existing UI and runtime expose active thread semantics. |
| Thread-local isolation for messages, tasks, approvals, tool calls, artifacts, events | implemented | `runtimecontract` descriptors and session registry APIs | Matches the target model in `AGENTS.md`. |
| Parent-child task relationships | partial | `ParentTaskID`, `LatestChildTaskID`, `WaitingStatus` fields | Runtime model exists; broader UX polish and release gating still remain. |
| Resume and interrupted-task recovery | partial | `runner.New(...).RecoverInterruptedTasks()` | Present in runtime path, but should be called out in future release verification. |

## 3. Built-In Runtime Tool Coverage

### Read and inspect tools

| Tool Kind | Status | Verification | Notes |
| --- | --- | --- | --- |
| `workspace.read_file` | implemented | indirect runtime coverage | Present in discovery and runner, but not yet explicitly called out in the primary acceptance report. |
| `workspace.list_files` | implemented | indirect runtime coverage | Implemented, not primary-lane verified. |
| `workspace.search_text` | verified | Playwright acceptance agent scenario | Used in the `search_then_detailed` scenario. |
| `workspace.stat_file` | verified | direct task + agent scenario in `verify-desktop-live-refresh.py` | Verified in both direct and agent paths. |
| `workspace.read_files_batch` | verified | direct task + agent scenario in `verify-desktop-live-refresh.py` | Verified in both direct and agent paths. |
| `workspace.list_files_filtered` | verified | direct task + agent scenario in `verify-desktop-live-refresh.py` | Verified in both direct and agent paths. |
| `workspace.search_text_detailed` | verified | direct task + agent scenario in `verify-desktop-live-refresh.py` | Verified in both direct and agent paths. |

### Write and thread mutation tools

| Tool Kind | Status | Verification | Notes |
| --- | --- | --- | --- |
| `workspace.apply_patch` | verified | Playwright acceptance apply flow | Approval, execution, and write audit are covered. |
| `workspace.apply_patch.rollback` | verified | Playwright acceptance rollback flow | Verified with persisted write execution linkage. |
| `thread.message.append` | verified | runtime tests + Playwright acceptance | Used in runtime test coverage and acceptance artifacts. |
| `thread.toolcall.append` | verified | desktop fallback Go tests + Playwright acceptance | Explicitly verified in fallback status and canonical runtime acceptance lane. |
| `thread.artifact.append` | verified | desktop fallback Go tests + Playwright acceptance | Explicitly verified in fallback status and canonical runtime acceptance lane. |
| `thread.runtimeflag.set` | verified | desktop fallback Go tests + Playwright acceptance | Explicitly verified in fallback status and canonical runtime acceptance lane. |

### Runtime inspection tools

| Tool Kind | Status | Verification | Notes |
| --- | --- | --- | --- |
| `bridge.check` | implemented | runtime tests | Present as a read-only runtime probe. |
| `skills.list` | implemented | runtime discovery surface | No dedicated acceptance lane yet. |
| `mcp.servers.list` | implemented | runtime discovery surface | Still metadata-level only. |

## 4. Model And Agent Execution

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| `model.response.create` | implemented | `internal/core/runner/runner.go` | Core model-response execution exists. |
| `agent.run` task creation and persisted plan metadata | implemented | `runtimecontract.TaskDescriptor`, runner, runtime tests | Plan mode, step, and last-reasoning fields are exposed. |
| `agent.run` child-task orchestration | verified | Playwright acceptance scenarios | Verified for `filter_then_read`, `search_then_detailed`, and `stat_then_read`. |
| Broader agent action extensibility cleanup | not implemented | roadmap only | Still deferred to the later `agent-runtime-refactor` phase. |

## 5. Skills And Group Isolation

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| `common`, `codex`, `cc` skill grouping model | implemented | `internal/core/runtime/status.go`, `discovery.go`, desktop payloads | Group isolation concept is present. |
| Shared runtime exposure of skill groups | implemented | runtime status and discovery | Surface exists for UI and CLI use. |
| Full 1:1 Chinese localization audit for copied skills | not implemented | not yet documented as completed | Required by `AGENTS.md`; needs a dedicated audit pass. |
| Capability-level verification for each grouped skill | not implemented | no matrix before this doc | This document starts the inventory but does not verify each skill individually. |

## 6. MCP Surface

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| MCP server metadata listing | implemented | `runtimecontract.MCPServer`, runtime discovery | Tool/resource counts and enabled state are exposed. |
| MCP server health and degraded diagnostics | not implemented | roadmap only | No strong health or probe UX yet. |
| End-to-end MCP execution acceptance lane | not implemented | none | Still a future milestone. |

## 7. Desktop Product Surface

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| Runtime dashboard with source and trust hints | partial | `desktop/app.go`, `desktop/frontend/src/App.tsx` | Present in current code, but current worktree has additional uncommitted UI changes that should be stabilized before claiming release-ready verification. |
| Thread/task/approval/write-execution workflow visibility | partial | desktop payloads and Milestone A acceptance lane | Core flow works; polish and regression hardening remain. |
| Browser preview shell integration | implemented | desktop browser workspace state in `desktop/app.go` | Functional shell exists. |
| Release-grade desktop regression gate | not implemented | no consolidated release checklist yet | Needs a single pre-merge baseline. |

## 8. What Is Already Verified End-To-End

The following are the strongest current verified paths:

- canonical runtime status is served by the app-server and exposes trust metadata
- custom app-server port is reflected in `canonicalRuntimeUrl`
- desktop acceptance can create a thread, create tasks, require approval, approve execution, record write execution, and execute rollback
- desktop acceptance now explicitly covers all currently implemented built-in thread mutation tools
- `agent.run` completes at least three concrete read-oriented action plans through child-task orchestration

## 9. Highest-Priority Remaining Gaps

These are the most important gaps after Milestone A:

1. Convert the current capability inventory into a maintained release checklist with exact commands.
2. Stabilize desktop UI verification around structure and identity rather than fragile summary prose.
3. Build a real MCP health and acceptance story instead of metadata-only listing.
4. Audit grouped skill imports against the `AGENTS.md` localization and isolation rules.

## 10. Recommended Next Execution Slice

The next smallest useful implementation plan should target:

- release baseline and regression gate documentation
- explicit verification coverage for the remaining implemented thread mutation tools
- one follow-up acceptance report that records all currently verified built-in tools

That slice is intentionally narrower than full Codex / CC or MCP expansion, and it keeps the project aligned with the requirement to prove complete tool capability coverage rather than only a small subset.
