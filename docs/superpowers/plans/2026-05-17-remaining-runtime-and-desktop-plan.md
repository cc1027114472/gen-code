# Gen-Code Remaining Runtime And Desktop Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the remaining gaps between the current `gen-code` runtime shell and the target product shape described in `AGENTS.md`, with priority on trustworthy runtime entry, usable workspace/thread workflows, stronger desktop acceptance, and staged Codex/CC/MCP capability expansion.

**Architecture:** Keep the current split between `cmd/cli`, `cmd/server`, `desktop`, `internal/core`, and `internal/appserver`, but tighten the contracts between them. The next phase should stabilize one canonical runtime path, make thread/task workflows first-class in the desktop shell, then expand tool and MCP capabilities on top of a verified end-to-end baseline.

**Tech Stack:** Go, Gin, SQLite, Wails, React, TypeScript, SSE, provider-backed Responses API execution, Playwright for desktop/browser acceptance.

---

## 1. Current State Summary

### Milestone A status update

Milestone A is now functionally complete as of May 17, 2026.

Closed in Milestone A:

- canonical runtime source semantics across CLI, runtime status, and desktop
- desktop workflow visibility for workspace, thread, task, approval, and write execution state
- one repeatable Python Playwright desktop acceptance lane

Still deferred beyond Milestone A:

- Codex / CC capability matrix expansion
- MCP productization
- broader `agent.run` structural cleanup beyond the already-verified scope

Based on the current repository state, the following capabilities are already present and should be treated as the baseline rather than rebuilt:

- Shared runtime model for `workspace`, `thread`, `task`, `message`, `tool call`, `artifact`, `event`, `approval`, and `write execution`
- Project-local SQLite persistence under `.gen-code/state.db`
- CLI, app-server, and desktop shell entrypoints
- `model.response.create`
- `agent.run` phase-2 orchestration
- approval, resume, write-audit, and rollback pipeline
- `common` / `codex` / `cc` skill grouping concepts
- desktop runtime status dashboard and preview shell

The remaining work is mainly about:

- runtime trust and environment cleanup
- missing user workflow polish
- incomplete acceptance lanes
- incomplete Codex / CC / MCP capability coverage
- incomplete productization of the multi-thread workspace model

## 2. Planning Principles

The remaining work should follow these rules:

- Do not replace working orchestration code unless a boundary is actively blocking the next phase.
- Prefer tightening contracts and acceptance around existing paths over adding new parallel paths.
- Do not expand Codex / CC tool coverage until the runtime entry path is trustworthy.
- Do not treat browser automation instability as a backend blocker; isolate it as a separate acceptance lane.
- Every new capability must be classified as `implemented`, `verified`, or `not verified yet`.
- Desktop and CLI must expose the same runtime truth for thread/task/approval/write-execution state.

## 3. Recommended Delivery Order

Implementation should proceed in this order:

1. Runtime entry cleanup and canonical source selection
2. Workspace/thread/task workflow completion
3. Desktop acceptance lane hardening
4. `agent.run` structural cleanup for future extensibility
5. Codex / CC capability coverage matrix and staged integration
6. MCP productization
7. Release baseline, documentation, and regression gates

This order is deliberate:

- it removes false signals first
- then makes the current product usable
- then makes it testable
- then makes it easier to expand safely

## 4. Delivery Phases

### Phase 1: Canonical Runtime Entry

**Objective**

Make one runtime path the default, trustworthy entry for CLI, desktop, and acceptance.

**Problems being solved**

- stale `127.0.0.1:10008` confusion
- remote runtime and desktop fallback feel interchangeable when they should not
- team members cannot easily tell whether they are testing current code or stale state

**Scope**

- unify runtime port and discovery rules
- define one canonical default runtime endpoint
- surface `runtimeSource`, `stateStore`, `statePath`, and trust hints consistently
- document when fallback should be used and when it is considered degraded mode
- remove or quarantine stale runtime assumptions from acceptance docs and scripts

**Likely files**

- [cmd/cli/main.go](D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main.go)
- [desktop/app.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
- [internal/core/runtime/status.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status.go)
- [internal/handler/runtime_handler.go](D:/GOWorks/gen-code-heji/gen-code/internal/handler/runtime_handler.go)
- [docs/superpowers/plans/2026-05-17-agent-run-phase2-acceptance-report.md](D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-agent-run-phase2-acceptance-report.md)

**Acceptance**

- CLI shows exactly which runtime it is using and why
- desktop shows exactly which runtime it is using and why
- remote app-server path is the default verified path
- fallback path is clearly marked as fallback, not equivalent to the primary runtime

### Phase 2: Workspace And Thread Workflow Completion

**Objective**

Turn the current state model into a complete user workflow: open project, inspect workspace, create threads, switch threads, create tasks, review approvals, inspect artifacts, and resume work.

**Problems being solved**

- the data model is ahead of the product workflow
- thread isolation exists, but the UX still feels diagnostic rather than operational
- long-lived task/thread lifecycle handling is incomplete

**Scope**

- complete workspace initialization and status presentation
- support thread lifecycle operations beyond create/activate
- improve task hierarchy visibility for parent-child relationships
- expose waiting states and approval states more clearly
- unify timeline rendering of task, event, tool call, message, and artifact context
- expose thread-level configuration such as model and permission mode in a first-class way

**Likely files**

- [desktop/frontend/src/App.tsx](D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx)
- [desktop/app.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
- [internal/core/session/session.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/session/session.go)
- [internal/core/runtime/status.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status.go)
- [internal/appserver/runtimecontract/contract.go](D:/GOWorks/gen-code-heji/gen-code/internal/appserver/runtimecontract/contract.go)

**Acceptance**

- a user can understand the active workspace without opening the database or logs
- parent and child tasks are clearly visible in desktop and CLI output
- approval-required tasks are distinguishable from failed tasks and waiting tasks
- active thread switching preserves thread-local context and preview ownership

### Phase 3: Desktop Acceptance Lane Hardening

**Objective**

Create one reliable end-to-end desktop acceptance lane that the team can rerun after meaningful changes.

**Problems being solved**

- current desktop implementation is ahead of its visual acceptance evidence
- `agent-browser` is not stable enough on this environment to be the only acceptance path
- regressions in approval/resume/rollback flows could slip through

**Scope**

- pick a primary automation lane, preferably Python Playwright
- automate thread creation, agent task creation, approval, auto resume, rollback, and preview refresh checks
- pin a clean runtime target and test data
- preserve one remote-runtime lane and one local-fallback lane where both are intentional

**Likely files**

- [scripts/verify-desktop-live-refresh.py](D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
- [scripts/run-desktop-live-refresh-check.ps1](D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
- [desktop/app_test.go](D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)
- [docs/superpowers/plans/2026-05-17-desktop-live-refresh-check.md](D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-desktop-live-refresh-check.md)

**Acceptance**

- one command or one short script can run the primary desktop acceptance flow
- the flow proves agent approval and rollback visually or through machine assertions
- failures point to runtime, UI, or automation layers clearly

### Phase 4: Agent Runtime Structural Cleanup

**Objective**

Refactor `agent.run` enough to support additional actions and better debugging without changing the already-verified behavior.

**Problems being solved**

- action parsing, action dispatch, state mutation, and child-task orchestration are still tightly packed
- future action growth will otherwise keep expanding `agent_loop.go` and `runner.go`
- debugging failed or stuck agents is still more manual than it should be

**Scope**

- isolate agent action schema and validation
- isolate action dispatch from state progression
- improve persisted agent progress metadata
- improve recovery diagnostics for interrupted parent-child flows
- preserve current action set and current approval semantics during refactor

**Likely files**

- [internal/core/runner/agent_loop.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/agent_loop.go)
- [internal/core/runner/runner.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/runner.go)
- [internal/core/runner/runner_test.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/runner_test.go)

**Acceptance**

- no regression in current `agent.run` tests
- clearer internal boundaries for action parse, action plan, action execute, and state update
- easier addition of new actions without editing the same large switch repeatedly

### Phase 5: Codex / CC Capability Coverage Matrix

**Objective**

Replace vague “support Codex and CC” language with a tracked matrix of concrete capabilities, groups, and verification status.

**Problems being solved**

- no single source of truth for what is implemented vs merely planned
- tool and skill coverage can drift from AGENTS expectations
- capability work may be duplicated or expanded without verification discipline

**Scope**

- define capability inventory by group: `common`, `codex`, `cc`
- classify each capability as `implemented`, `verified`, `not implemented`, or `blocked`
- map each capability to runtime descriptors, UI visibility, and tests
- prioritize high-frequency capabilities first

**Likely outputs**

- a new capability matrix document under `docs/`
- updates to tool and skill descriptors surfaced by runtime APIs

**Acceptance**

- the team can answer “is capability X really supported?” without reading source code
- the next implementation phase can pick items from the matrix instead of from memory

### Phase 6: MCP Productization

**Objective**

Promote MCP support from static metadata management to a visible, diagnosable, user-meaningful capability surface.

**Problems being solved**

- MCP manager exists, but the runtime experience is still thin
- failures and missing resources are not yet easy to inspect
- there is no strong acceptance story for MCP configuration and exposure

**Scope**

- define MCP configuration and enable/disable lifecycle clearly
- expose server health and diagnostic messages
- improve server visibility in desktop and CLI
- add minimal probe and acceptance coverage for configured servers

**Likely files**

- [internal/core/mcp/manager.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/mcp/manager.go)
- [internal/core/runtime/status.go](D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status.go)
- [desktop/frontend/src/App.tsx](D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx)

**Acceptance**

- enabled vs disabled vs degraded MCP servers are visible
- tool/resource counts are not the only exposed state
- at least one configured MCP path is verified end-to-end

### Phase 7: Release Baseline And Regression Gates

**Objective**

Lock in a repeatable pre-merge and pre-release baseline for the whole product.

**Problems being solved**

- test coverage exists but is not yet presented as a single release gate
- environment drift can invalidate acceptance confidence
- docs are scattered across plan artifacts and source exploration

**Scope**

- define required test commands by layer
- define required acceptance commands by lane
- document runtime ports, source precedence, fallback semantics, and provider support
- document what is experimental vs what is stable

**Acceptance**

- there is a short release checklist the team can actually run
- runtime, desktop, and provider expectations are documented in one place
- remaining known gaps are explicit rather than hidden

## 5. First Milestone Recommendation

The first milestone should be intentionally narrow and produce a stronger base for all later work.

**Milestone A: Trustworthy Runtime And Usable Desktop Core**

Include only:

- Phase 1 in full
- the critical path parts of Phase 2
- the primary acceptance lane from Phase 3

**Milestone A deliverables**

- one canonical runtime target
- clean runtime source reporting across CLI and desktop
- complete thread/task/approval workflow in desktop for daily use
- one reliable desktop acceptance script proving approval, auto resume, and rollback

**Milestone A non-goals**

- full Codex / CC capability parity
- full MCP productization
- broad agent action expansion

This milestone is the best next investment because it converts the project from “promising but environment-sensitive” to “stable enough to build on”.

## 6. Risks And Controls

### Risk: Runtime ambiguity remains

Control:

- do not begin new capability work until the canonical runtime path is documented and enforced

### Risk: Desktop work outruns acceptance again

Control:

- require one maintained desktop E2E lane before approving more large UI/runtime work

### Risk: Agent code grows into another large monolith

Control:

- split by responsibility before adding additional agent actions

### Risk: Capability expansion becomes unbounded

Control:

- use the capability matrix and explicit verification state before implementation

## 7. Definition Of Done For The Remaining Program

The remaining program should be considered complete only when all of the following are true:

- the default runtime entry path is canonical and trustworthy
- desktop, CLI, and app-server report the same runtime truth
- workspace/thread/task flows are operational, not just inspectable
- one stable desktop E2E lane is rerunnable in the current environment
- Codex / CC capability coverage is explicitly tracked and partially or fully verified by priority
- MCP support is visible and diagnosable
- release and regression expectations are documented and executable

## 8. Immediate Next Steps

- [x] Finalize the canonical runtime target and deprecate stale port assumptions
- [x] Write a focused implementation plan for Milestone A only
- [x] Execute Milestone A before expanding tool or MCP surface area
- [ ] Start the next post-Milestone-A plan, beginning with capability matrix and release-baseline follow-up

## 9. Suggested Follow-Up Plan Split

This document is the umbrella roadmap. Execution should be split into smaller implementation plans:

1. `runtime-entry-cleanup`
2. `desktop-thread-workflow`
3. `desktop-e2e-acceptance`
4. `agent-runtime-refactor`
5. `capability-matrix-and-tool-coverage`
6. `mcp-productization`

Plan complete and saved to `docs/superpowers/plans/2026-05-17-remaining-runtime-and-desktop-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
