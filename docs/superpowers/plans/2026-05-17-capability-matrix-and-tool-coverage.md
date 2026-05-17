# Gen-Code Capability Matrix And Tool Coverage

**Date:** May 17, 2026

**Purpose:** One verified baseline for what `gen-code` supports, what has been exercised end-to-end, and what remains incomplete.

## 1. Runtime Entry And Trust Baseline

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| Canonical app-server runtime status endpoint | verified | `internal/core/runtime/status.go`, `internal/appserver/runtimecontract/contract.go`, `scripts/verify-desktop-live-refresh.py` | Exposes `runtimeSource`, `runtimeTrust`, `runtimeSourceDetail`, and `canonicalRuntimeUrl`. |
| Canonical runtime URL follows actual server port | verified | `internal/appserver/appserver.go`, `internal/core/runtime/status_test.go` | Verified against the current acceptance lane port. |
| Desktop local fallback runtime | verified | `desktop/app.go`, `desktop/app_test.go`, `desktop/frontend/src/runtimeBridge.ts` | Fallback stays `local-fallback` with degraded trust and manual refresh wording. |
| CLI / desktop / app-server source semantics alignment | verified | `cmd/cli/main.go`, `cmd/cli/main_test.go`, `desktop/frontend/src/runtimeBridge.ts`, `internal/core/runtime/status.go` | Source/detail/trust semantics stay aligned across surfaces. |

## 2. Workspace, Thread, And Task Model

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| Project-local workspace state store | implemented | `.gen-code/state.db`, `internal/core/state`, `internal/core/session` | SQLite-backed persistence is present. |
| Multiple threads under one workspace | implemented | `internal/core/session`, `runtimecontract.ThreadDescriptor` | Data model supports multi-thread organization. |
| Active thread switching | implemented | runtime contract and desktop status payloads | Existing UI and runtime expose active thread semantics. |
| Parent-child task relationships | verified | `desktop/app.go`, `desktop/app_test.go`, `desktop/frontend/src/App.tsx`, `scripts/verify-desktop-live-refresh.py` | Desktop surfaces stable parent/child/waiting relationships across fallback and canonical lanes. |
| Resume and interrupted-task recovery | verified | `runner.New(...).RecoverInterruptedTasks()`, `desktop/app_test.go`, `scripts/verify-desktop-live-refresh.py` | Verified with fallback restart evidence and canonical resumed-to-completion browser acceptance. |

## 3. Built-In Runtime Tool Coverage

### Read and inspect tools

| Tool Kind | Status | Verification | Notes |
| --- | --- | --- | --- |
| `workspace.read_file` | verified | direct task + agent scenario | Verified in both direct and agent paths. |
| `workspace.list_files` | verified | direct task + agent scenario | Verified in both direct and agent paths. |
| `workspace.search_text` | verified | Playwright acceptance agent scenario | Used in the `search_then_detailed` scenario. |
| `workspace.stat_file` | verified | direct task + agent scenario | Verified in both direct and agent paths. |
| `workspace.read_files_batch` | verified | direct task + agent scenario | Verified in both direct and agent paths. |
| `workspace.list_files_filtered` | verified | direct task + agent scenario | Verified in both direct and agent paths. |
| `workspace.search_text_detailed` | verified | direct task + agent scenario | Verified in both direct and agent paths. |

### Write and thread mutation tools

| Tool Kind | Status | Verification | Notes |
| --- | --- | --- | --- |
| `workspace.apply_patch` | verified | Playwright acceptance apply flow | Approval, execution, and write audit are covered. |
| `workspace.apply_patch.rollback` | verified | Playwright acceptance rollback flow | Verified with persisted write execution linkage. |
| `thread.message.append` | verified | runtime tests + Playwright acceptance | Covered in runtime and acceptance artifacts. |
| `thread.toolcall.append` | verified | desktop fallback tests + Playwright acceptance | Verified in both fallback and canonical evidence. |
| `thread.artifact.append` | verified | desktop fallback tests + Playwright acceptance | Verified in both fallback and canonical evidence. |
| `thread.runtimeflag.set` | verified | desktop fallback tests + Playwright acceptance | Verified in both fallback and canonical evidence. |

### Runtime inspection tools

| Tool Kind | Status | Verification | Notes |
| --- | --- | --- | --- |
| `bridge.check` | implemented | runtime tests | Read-only runtime probe. |
| `skills.list` | verified | CLI output tests + runtime discovery surface | Grouped inventory fields are surfaced, but per-skill acceptance is still separate. |
| `mcp.servers.list` | verified | CLI output tests + runtime discovery surface | Metadata-only lane covers `enabled/disabled/degraded/unreachable` health states. |

## 4. Model And Agent Execution

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| `model.response.create` | implemented | `internal/core/runner/runner.go` | Core model-response execution exists. |
| `agent.run` task creation and persisted plan metadata | implemented | `runtimecontract.TaskDescriptor`, runner, runtime tests | Plan mode, step, and last-reasoning fields are exposed. |
| `agent.run` child-task orchestration | verified | Playwright acceptance scenarios | Verified for `filter_then_read`, `search_then_detailed`, and `stat_then_read`. |

## 5. Skills And Group Isolation

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| `common`, `codex`, `cc` skill grouping model | verified | `internal/core/runtime/status.go`, `discovery.go`, CLI and desktop payloads | Group isolation baseline is surfaced consistently. |
| Shared runtime exposure of skill groups | verified | runtime status, discovery, `/api/skills`, `skills list` output | Inventory fields and grouped governance summaries are visible for verification. |
| Grouped skill import audit against `AGENTS.md` localization and isolation rules | verified | `internal/core/runtime/discovery.go`, `desktop/app.go`, `cmd/cli/main.go`, governance tests | Current baseline audits project-local copied skills under `internal/core/skill/catalog` for `localization checked` and `isolation status` without claiming per-skill capability acceptance. |
| Full 1:1 Chinese localization audit for copied skills | verified | `internal/core/runtime/discovery.go`, `desktop/app.go`, governance tests, baseline docs | `localizationChecked` now reflects an explicit 1:1 Chinese-localization audit conclusion for copied skills in `gen-code/internal/core/skill/catalog` instead of a loose heuristic. |
| Capability-level verification for each grouped skill | not implemented | no matrix before this doc | This document starts the inventory but does not verify each skill individually. |

### Skill Governance Boundary

- `skill discovered` is not `skill accepted`.
- Grouped inventory is separate from runtime release acceptance.
- Localization status stays tracked, not assumed complete.
- For copied `codex` / `cc` skills, the governance truth now comes from `gen-code/internal/core/skill/catalog`, not sibling source directories.

## 6. MCP Surface

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| MCP server metadata listing | implemented | `runtimecontract.MCPServer`, runtime discovery | Tool/resource counts and enabled state are exposed. |
| MCP server health status contract | verified | `internal/core/mcp/manager.go`, `internal/core/runtime/status.go`, runtime tests, CLI output tests | `enabled`, `disabled`, `degraded`, and `unreachable` stay explicit. |
| End-to-end MCP execution acceptance lane | verified | `internal/core/mcp/manager.go`, `internal/core/runner/runner.go`, `cmd/cli/main.go`, `scripts/verify-desktop-live-refresh.py` | Verified with a fixture-backed stdio external MCP server through the canonical task lane. |

## 7. Desktop Product Surface

| Capability | Status | Evidence | Notes |
| --- | --- | --- | --- |
| Runtime dashboard with source and trust hints | verified | `desktop/app.go`, `desktop/frontend/src/App.tsx`, `desktop/frontend/src/runtimeBridge.ts` | Desktop reuses bridge-derived source/trust/refresh semantics. |
| Thread/task/approval/write-execution workflow visibility | partial | desktop payloads and acceptance lane | Core flow works; polish remains. |
| Release-grade desktop regression gate | verified | `docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md` | One maintained checklist covers desktop Go, frontend build, and canonical live acceptance. |

## 8. What Is Already Verified End-To-End

- canonical runtime status exposes trust metadata
- `canonicalRuntimeUrl` tracks the actual server port
- desktop acceptance can create a thread, create tasks, require approval, approve execution, record write execution, and execute rollback
- desktop acceptance covers the implemented built-in thread mutation tools
- `agent.run` completes at least three read-oriented action plans

## 9. Highest-Priority Remaining Gaps

1. Decide whether to extend the current verified MCP baseline beyond the fixture-backed external execution lane.
2. Define how and when to add capability-level verification for each grouped skill.
