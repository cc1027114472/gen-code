# Browser Authenticated And Public-Web Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the current browser verified baseline from controlled local pages only into a stable authenticated-session lane plus a constrained public-web read-only lane without changing the existing task and tool contracts.

**Architecture:** Keep `browser.open`, `browser.navigate`, `browser.click`, `browser.type`, `browser.extract`, and `browser.screenshot` as the only browser task kinds. Expand the shared Go browser core with an internal host policy plus optional session bootstrap configuration, then prove the new scope through deterministic acceptance helpers and minimal surface copy updates. The canonical lane remains task-driven and desktop-backed; this phase only broadens which URLs and session states that lane may exercise.

**Tech Stack:** Go, chromedp, Wails desktop bridge, React, TypeScript, Python Playwright, PowerShell.

---

## Scope

This phase adds two browser execution baselines:

- authenticated controlled-browser lane with pre-seeded session state
- constrained public-web read-only lane for one allowlisted HTTPS target

This phase does not include:

- arbitrary third-party website compatibility
- OAuth, popup login, SSO redirects, uploads, downloads, or file pickers
- browser-write actions on public websites
- a new browser-specific top-level runtime API
- new approval or write-execution semantics for browser tasks

## File Map

### Browser core and policy

- Create: `internal/core/browser/policy.go`
  - internal browser host policy, public-host allowlist parsing, and session profile lookup
- Create: `internal/core/browser/policy_test.go`
  - tests for localhost defaults, HTTPS allowlist admission, and policy rejection paths
- Modify: `internal/core/browser/core.go`
  - switch URL normalization from hard-coded localhost-only logic to policy-backed validation
- Modify: `internal/core/browser/chromedp_driver.go`
  - apply optional per-host session bootstrap before navigation and keep stable failure summaries
- Modify: `internal/core/browser/core_test.go`
  - extend baseline coverage for new allowlist and rejection semantics

### Runtime and desktop integration

- Modify: `internal/core/runtime/status.go`
  - expose minimal browser policy summary only if an existing browser status field can carry it without changing top-level contract shape
- Modify: `desktop/app.go`
  - surface stable browser capability wording using the same runtime-derived semantics
- Modify: `desktop/app_test.go`
  - lock fallback/browser status behavior and ensure new browser-policy summary does not drift in degraded mode
- Modify: `cmd/cli/main.go`
  - keep `tools list` stable and, if already appropriate, improve `runtime status` or `tools list` wording around browser scope
- Modify: `cmd/cli/main_test.go`
  - assert the updated browser scope wording and non-drifting browser tool inventory

### Acceptance fixtures and verifier

- Modify: `desktop/frontend/src/App.tsx`
  - add one deterministic authenticated acceptance pane with stable selectors and no product-level IA expansion
- Modify: `scripts/verify-desktop-live-refresh.py`
  - add authenticated-session browser scenario and public-web read-only browser scenario
- Modify: `scripts/run-desktop-live-refresh-check.ps1`
  - keep wrapper output aligned with the expanded browser acceptance lanes

### Documentation

- Modify: `docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md`
  - upgrade browser notes from local-only to authenticated plus constrained public-web baseline
- Modify: `docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md`
  - add the new browser acceptance evidence requirements

## Execution Sequence

Implementation order is fixed:

1. browser host policy and validation
2. authenticated session bootstrap
3. authenticated deterministic acceptance lane
4. constrained public-web read-only lane
5. CLI / desktop wording sync
6. docs and matrix closeout

## Task 1: Browser Host Policy And URL Validation

**Files:**
- Create: `internal/core/browser/policy.go`
- Create: `internal/core/browser/policy_test.go`
- Modify: `internal/core/browser/core.go`
- Modify: `internal/core/browser/core_test.go`

- [ ] **Step 1: Write failing policy tests for the new browser scope**

Cover these cases:

- localhost remains allowed over `http`
- `https://example.com`-style host is rejected when not allowlisted
- an explicitly allowlisted HTTPS host is accepted
- `http` remains rejected for non-local hosts
- malformed allowlist entries fail closed

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser -run "Policy|NormalizeURL|AllowedURL" -v
```

Expected:

- fails because the new policy-backed path does not exist yet

- [ ] **Step 2: Implement internal browser policy parsing**

Add a focused policy unit instead of expanding `core.go` further:

- localhost defaults stay built in
- non-local hosts require `https`
- public hosts come from a deterministic config source
- unknown hosts stay rejected with `ErrURLNotAllowed`

Suggested config inputs:

- `GEN_CODE_BROWSER_ALLOWED_HOSTS`
- optional JSON file path for richer test fixtures if string env parsing becomes brittle

- [ ] **Step 3: Replace hard-coded localhost checks in `core.go`**

Keep the public tool contract unchanged:

- `browser.open` still accepts only `url`
- `browser.navigate` still accepts only `tabId` and `url`

Only the internal validation path changes:

- `normalizeURL(raw string)` becomes `normalizeURL(raw string, policy Policy)` or equivalent
- existing call sites must keep stable error categories

- [ ] **Step 4: Re-run focused browser tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser -run "Policy|NormalizeURL|AllowedURL" -v
```

Expected:

- PASS

- [ ] **Step 5: Commit**

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
git add internal/core/browser/policy.go internal/core/browser/policy_test.go internal/core/browser/core.go internal/core/browser/core_test.go
git commit -m "feat: add browser host policy"
```

## Task 2: Authenticated Session Bootstrap

**Files:**
- Modify: `internal/core/browser/chromedp_driver.go`
- Modify: `internal/core/browser/types.go` only if an internal-only helper type is clearly cleaner there; otherwise keep new types in `policy.go`
- Modify: `internal/core/browser/core_test.go`
- Create or Modify: adjacent browser driver tests if missing

- [ ] **Step 1: Write failing tests for authenticated-session bootstrap**

Cover these cases:

- matching host applies configured cookies before first navigation
- non-matching host does not receive auth bootstrap
- invalid cookie config returns stable setup failure
- missing profile for an auth-marked lane fails closed rather than silently downgrading

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser -run "Auth|Cookie|Session" -v
```

Expected:

- fails because session bootstrap does not exist yet

- [ ] **Step 2: Implement minimal per-host session bootstrap**

Constraints:

- no new task input fields
- no browser-specific approval model
- cookie-based bootstrap only for this phase
- local storage, credential entry, and OAuth remain out of scope

Recommended shape:

- policy maps host -> session profile
- profile contains deterministic cookie definitions
- driver applies cookies before the first navigation on a fresh tab for that host

- [ ] **Step 3: Preserve stable summaries and errors**

Ensure the existing snapshot/result model still works:

- success summaries stay identity-first
- session failures surface as stable browser errors
- `browser.extract` and `browser.screenshot` remain artifact-friendly

- [ ] **Step 4: Re-run browser package tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser -v
```

Expected:

- PASS

- [ ] **Step 5: Commit**

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
git add internal/core/browser/chromedp_driver.go internal/core/browser/core_test.go
git commit -m "feat: add browser session bootstrap"
```

## Task 3: Deterministic Authenticated Browser Acceptance Lane

**Files:**
- Modify: `desktop/frontend/src/App.tsx`
- Modify: `scripts/verify-desktop-live-refresh.py`
- Modify: `desktop/app.go`
- Modify: `desktop/app_test.go`

- [ ] **Step 1: Add a stable authenticated acceptance fixture to the desktop UI**

The fixture must stay deterministic and internal:

- use a dedicated acceptance pane
- expose a cookie-gated authenticated state
- expose stable selectors for authenticated heading, session badge, and extracted text
- avoid introducing a new standalone browser page in the product IA

Suggested selectors:

- `[data-testid='authenticated-browser-heading']`
- `[data-testid='authenticated-browser-session']`
- `[data-testid='authenticated-browser-result']`

- [ ] **Step 2: Add a direct-task authenticated browser scenario**

Extend the verifier with a scenario that:

- opens the authenticated acceptance URL
- verifies the session badge is present after bootstrap
- extracts authenticated text
- captures one screenshot artifact
- records `taskId`, `kind`, `resultSummary`, `artifactPath`, and visibility evidence

- [ ] **Step 3: Lock desktop fallback and runtime status behavior**

Fallback does not become a browser acceptance lane, but tests must prove:

- browser status remains readable
- new capability wording does not imply canonical auth acceptance in fallback mode
- existing source/trust semantics do not drift

- [ ] **Step 4: Run desktop regression plus acceptance wrapper**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected:

- desktop tests PASS
- acceptance output includes explicit authenticated browser evidence

- [ ] **Step 5: Commit**

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
git add desktop/frontend/src/App.tsx desktop/app.go desktop/app_test.go scripts/verify-desktop-live-refresh.py
git commit -m "test: add authenticated browser acceptance lane"
```

## Task 4: Constrained Public-Web Read-Only Lane

**Files:**
- Modify: `internal/core/browser/policy.go`
- Modify: `scripts/verify-desktop-live-refresh.py`
- Modify: `scripts/run-desktop-live-refresh-check.ps1`
- Modify: `docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md`

- [ ] **Step 1: Fix the public-web target contract**

Use one deterministic HTTPS target only. The lane must be:

- read-only
- extraction and screenshot only
- no click or type requirements
- allowlisted explicitly, not by wildcard

Default recommendation:

- `GEN_CODE_BROWSER_PUBLIC_BASE_URL=https://example.com/`

If the team wants stricter determinism later, swap this to a repo-owned remote fixture without changing the runtime tool contract.

- [ ] **Step 2: Add verifier preflight and stable skip/fail semantics**

Verifier behavior must be explicit:

- if the public URL is configured and reachable, the lane is required
- if the release process intentionally disables public-web validation, the report must say so clearly
- network failure must be classified separately from selector/assertion failure

- [ ] **Step 3: Add the public-web scenario**

Scenario requirements:

- `browser.open` on the allowlisted HTTPS target
- `browser.extract` from body or a stable selector
- `browser.screenshot`
- result JSON records `transportScope=public-web-read-only`, `targetUrl`, `taskId`, `resultSummary`, and `artifactPath`

- [ ] **Step 4: Re-run browser and acceptance gates**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser ./internal/core/runtime ./internal/core/runner ./cmd/cli
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected:

- package tests PASS
- acceptance output includes a public-web browser evidence section or an explicit, policy-backed skip reason

- [ ] **Step 5: Commit**

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
git add internal/core/browser/policy.go scripts/verify-desktop-live-refresh.py scripts/run-desktop-live-refresh-check.ps1 docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md
git commit -m "test: add public web browser baseline"
```

## Task 5: CLI, Matrix, And Release Wording Closeout

**Files:**
- Modify: `cmd/cli/main.go`
- Modify: `cmd/cli/main_test.go`
- Modify: `desktop/app.go`
- Modify: `docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md`
- Modify: `docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md`

- [ ] **Step 1: Update human-readable browser scope wording**

Required message shape:

- current verified baseline is no longer “controlled local URLs only”
- verified scope is “authenticated controlled session lane + constrained public-web read-only lane”
- explicitly state that arbitrary public-web compatibility is still not claimed

- [ ] **Step 2: Add CLI tests for non-drifting browser wording**

Lock:

- browser tools remain listed with existing kinds
- scope wording reflects the new baseline
- no surface claims blanket website compatibility

- [ ] **Step 3: Update capability matrix and release checklist**

Matrix notes must make the boundary explicit:

- browser tools remain individually verified
- baseline now covers localhost, authenticated controlled session, and one allowlisted HTTPS read-only lane
- arbitrary auth flows and arbitrary third-party sites remain out of scope

- [ ] **Step 4: Run final regression bundle**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser ./internal/core/runtime ./internal/core/runner ./cmd/cli
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected:

- all commands exit `0`
- acceptance report records authenticated and public-web browser evidence
- docs, checklist, and surface copy agree on the new browser baseline

- [ ] **Step 5: Commit**

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
git add cmd/cli/main.go cmd/cli/main_test.go desktop/app.go docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md
git commit -m "docs: close browser verified baseline expansion"
```

## Acceptance Standard

This phase is done when all of the following are true:

- browser URL validation is policy-backed instead of localhost-only hard coding
- authenticated session bootstrap works without changing browser task input shapes
- canonical acceptance includes one deterministic authenticated browser lane
- canonical acceptance includes one constrained public-web read-only lane
- CLI, desktop wording, checklist, and matrix all describe the same verified boundary
- the product still does not claim blanket arbitrary website compatibility

## Risks And Default Decisions

- Public internet in the release gate is the main instability risk.
  - Default decision: keep the public-web lane constrained to one explicit HTTPS target and classify reachability failures separately.
- Auth bootstrap can grow into a generic login framework if left loose.
  - Default decision: cookie-only bootstrap for one host profile in this phase.
- Browser contract churn would create downstream noise.
  - Default decision: do not add browser task kinds or new top-level runtime contract resources.

## Out Of Scope Follow-Ups

- arbitrary authenticated website support
- cookie import from the user’s real browser
- uploads, downloads, popup windows, and iframe-heavy flows
- browser write-approval semantics
- desktop workflow-visibility polish outside browser scope

## Self-Review

- Spec coverage:
  - authenticated lane: Task 2 + Task 3
  - public-web lane: Task 1 + Task 4
  - stable surfaces and docs: Task 5
- Placeholder scan:
  - no `TODO` or `TBD` placeholders remain
- Type consistency:
  - existing browser task kinds remain unchanged; new behavior is internal policy and acceptance expansion only

Plan complete and saved to `docs/superpowers/plans/2026-05-18-browser-authenticated-public-web-baseline-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
