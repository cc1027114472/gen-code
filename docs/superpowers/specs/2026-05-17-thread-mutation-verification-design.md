# Thread Mutation Verification Design

**Date:** May 17, 2026

**Goal:** Add explicit verification coverage for the remaining built-in thread mutation tools so they can be promoted from `implemented` to `verified` in the project capability baseline.

## Scope

This design covers only these existing runtime tools:

- `thread.toolcall.append`
- `thread.artifact.append`
- `thread.runtimeflag.set`

It does not introduce new runtime capabilities, new UI concepts, or new product workflows. The work is verification-first and should preserve current behavior.

## Why This Slice

Milestone A already proved the canonical runtime path, approval flow, write execution flow, rollback flow, and several read-oriented workspace tools. The remaining gap is narrower: some thread mutation tools already exist in the runtime, but they are not yet explicitly covered by the primary verification story.

That leaves the project in an awkward state where code exists, but the capability claim is weaker than the implementation footprint. This slice fixes that without expanding scope into MCP, broader agent refactors, or new desktop features.

## Recommended Approach

Use a two-layer verification strategy:

1. Add deterministic fallback-mode Go tests in `desktop/app_test.go` for all three thread mutation tools.
2. Add explicit direct-task acceptance coverage in `scripts/verify-desktop-live-refresh.py` so the main desktop lane exercises the same tools against the canonical app-server runtime.
3. Update the capability baseline document to reflect the stronger verification state.

This is the best option because it gives fast local regression coverage through Go tests while also satisfying the requirement for full entry-to-exit workflow verification through the main acceptance lane.

## Alternatives Considered

### Option A: Go tests only

Pros:

- fastest to implement
- low flakiness

Cons:

- does not satisfy the project rule that tool integration should not be considered complete from local-only tests
- would still leave the capability matrix weaker than desired

### Option B: Acceptance script only

Pros:

- strongest product-level confidence

Cons:

- slower debugging loop
- harder to isolate failures
- less precise for state-shape regressions

### Option C: Two-layer verification

Pros:

- balances deterministic unit-level coverage with end-to-end proof
- keeps scope narrow
- upgrades capability claims cleanly

Cons:

- touches both Go tests and the Python acceptance script

**Recommendation:** Option C.

## Design

### 1. Desktop fallback verification

Extend `desktop/app_test.go` with one focused test per tool:

- `thread.toolcall.append` should create a task, run it, and verify the resulting `ToolCalls` collection contains the expected tool ID, status, summary, and thread linkage.
- `thread.artifact.append` should create a task, run it, and verify the resulting `Artifacts` collection contains the expected path, kind, and thread linkage.
- `thread.runtimeflag.set` should create a task, run it, and verify the resulting `RuntimeFlags` collection contains the expected key, value, and thread linkage.

These tests should follow the existing desktop fallback style:

- force local fallback with `GENCODE_RUNTIME_BASE_URL=http://127.0.0.1:1`
- use a temp desktop state DB
- assert on `RuntimeStatus` rather than on implementation internals

### 2. Canonical runtime acceptance coverage

Extend `scripts/verify-desktop-live-refresh.py` with three new direct scenarios using the existing `run_direct_tool_scenario(...)` helper shape, or a close sibling where needed.

The new acceptance scenarios should:

- create and run a `thread.toolcall.append` task
- create and run a `thread.artifact.append` task
- create and run a `thread.runtimeflag.set` task

The assertions should verify:

- task completion
- the correct collection is updated through the API
- the UI shows at least the task title or a stable identity signal

The acceptance lane should continue preferring stable identity signals over brittle summary prose.

### 3. Capability baseline update

Update the capability matrix document so these three tools move from `implemented` to `verified` once both layers pass.

If helpful, also add a short note in the acceptance report describing that the desktop lane now explicitly covers all current built-in thread mutation tools.

## Data Flow

The expected verification flow is:

1. Create thread
2. Create mutation task
3. Run task
4. Runtime persists the resulting tool call, artifact, or runtime flag
5. Runtime status and thread collections expose the new state
6. Desktop acceptance confirms API state and visible UI linkage

No schema changes are expected. This slice should reuse the existing runtime contract and desktop payload surfaces.

## Error Handling

Failures should remain categorized by layer:

- Go test failure means the fallback payload or local runtime behavior regressed
- acceptance runtime/API failure means the canonical runtime path regressed
- acceptance UI failure means the browser-rendered workflow no longer exposes enough stable identity for the task

The Python script should continue surfacing failures in structured JSON with a clear `category`.

## Testing Plan

Required verification after implementation:

```powershell
GOTOOLCHAIN=auto go test ./desktop
```

Then:

```powershell
GOTOOLCHAIN=auto go test ./internal/core/runtime ./cmd/cli
```

Then run the primary acceptance lane against the canonical runtime:

```powershell
$env:GEN_CODE_API_BASE_URL='http://127.0.0.1:10018'
python .\scripts\verify-desktop-live-refresh.py
```

If the wrapper is the preferred entry in the current environment, the equivalent wrapper run should also remain valid.

## Risks And Controls

### Risk: Acceptance script becomes more text-fragile again

Control:

- use task titles, IDs, or collection-level API checks first
- only use summary substring checks where the summary is already stable and intentional

### Risk: Tool-specific verification duplicates too much helper logic

Control:

- reuse the existing direct-scenario helper patterns
- add a tiny specialized helper only if one collection needs post-task polling beyond the current generic flow

### Risk: Capability baseline gets ahead of proof

Control:

- do not mark the matrix entries as `verified` until both Go tests and the acceptance lane pass

## Done Criteria

This slice is complete when:

- the three thread mutation tools each have explicit desktop fallback Go test coverage
- the primary desktop acceptance lane explicitly exercises all three tools
- the capability matrix records them as `verified`
- verification commands pass in the current environment
