# Browser Real Auth And Multi-Site Baseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the current browser verified baseline from allowlisted local pages plus one fixture-backed authenticated lane into a more realistic browser baseline with managed authenticated session profiles and multiple stable public-web evidence lanes.

**Architecture:** Keep the existing `browser.*` task family, policy file loading, chromedp driver, desktop browser workspace, and canonical Playwright verifier as the only mainline. Add a project-controlled authenticated session profile layer plus a multi-target public-web verification matrix, then route CLI, runtime discovery, desktop wording, and acceptance output through the same baseline semantics.

**Tech Stack:** Go, chromedp/CDP, Playwright Python verifier, PowerShell wrapper, React desktop frontend, existing runtime contract and desktop bridge.

---

## File Structure

- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\policy.go`
  Purpose: Expand browser policy parsing from host allowlist plus inline cookies into reusable authenticated session profile semantics and explicit verified-lane policy metadata.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\chromedp_driver.go`
  Purpose: Apply authenticated session profiles deterministically before first navigation, support profile-aware bootstrap, and keep tab/session transitions stable.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\types.go`
  Purpose: If needed, add minimal internal request metadata for session-profile selection without changing the public task contract shape.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\core_test.go`
  Purpose: Keep browser URL normalization and baseline constraints covered after policy expansion.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\chromedp_driver_test.go`
  Purpose: Add deterministic tests for authenticated profile bootstrap and stable tab/session behavior.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\policy_test.go`
  Purpose: Lock policy parsing, profile resolution, BOM compatibility, and malformed-profile failure behavior.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\discovery.go`
  Purpose: Update built-in browser tool descriptions to reflect the expanded verified baseline wording.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
  Purpose: Keep `tools list` and related browser wording aligned with the new verified baseline semantics.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`
  Purpose: Lock CLI wording and baseline summary output.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\app.go`
  Purpose: Keep desktop browser summary and fallback wording aligned with the new runtime/browser baseline.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\app_test.go`
  Purpose: Prove desktop fallback/browser summary semantics stay readable and stable.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
  Purpose: Extend embedded preview fixtures so canonical acceptance can exercise richer authenticated states without inventing a separate product surface.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\verify-desktop-live-refresh.py`
  Purpose: Add real-auth profile scenarios and multiple public-web evidence lanes to the canonical acceptance matrix.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\run-desktop-live-refresh-check.ps1`
  Purpose: Generate deterministic policy/session fixtures and public-web target config for the expanded browser acceptance lane.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-capability-matrix-and-tool-coverage.md`
  Purpose: Update browser verified-baseline wording and remaining-gap statements.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-runtime-entry-checklist.md`
  Purpose: Record the new browser canonical gate requirements and evidence outputs.

## Task 1: Define The Expanded Browser Baseline Contract

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\policy.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\policy_test.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\core_test.go`

- [ ] **Step 1: Write failing policy tests for reusable authenticated session profiles and explicit multi-site allowlist behavior**

Add tests that prove:
- one host can require a named authenticated profile instead of repeating raw inline cookies
- one profile can safely serve multiple hosts
- malformed profile references fail closed
- BOM-prefixed policy files still parse
- non-HTTPS public hosts remain rejected unless they are local

Expected test additions:

```go
func TestPolicyResolvesSessionProfileForNamedAuthenticatedLane(t *testing.T) {
	policy := newPolicyFromSources("", writePolicyFile(t, `{
  "profiles": {
    "acceptance-session": {
      "cookies": [
        { "name": "gc_auth", "value": "acceptance-session", "path": "/" }
      ]
    }
  },
  "hosts": {
    "127.0.0.1": { "sessionRequired": true, "profile": "acceptance-session" },
    "localhost": { "sessionRequired": true, "profile": "acceptance-session" }
  }
}`))

	profile, needsSession, err := policy.sessionProfileForHost("127.0.0.1")
	require.NoError(t, err)
	require.True(t, needsSession)
	require.Len(t, profile.Cookies, 1)
	require.Equal(t, "gc_auth", profile.Cookies[0].Name)
}
```

- [ ] **Step 2: Run the focused browser policy tests to verify they fail first**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser -run "Policy|NormalizeURL" -v
```

Expected: FAIL because profile-based policy support is not implemented yet.

- [ ] **Step 3: Implement minimal profile-aware policy parsing without changing the public browser task contract**

Implement:
- internal `profiles` section support in the browser policy file
- host-to-profile references
- explicit failure when a host references a missing profile
- reuse of the current `SessionProfile` / `SessionCookie` structures so no new public runtime resource is needed

Keep the public task input contract unchanged:

```go
type SessionProfile struct {
	Cookies []SessionCookie
}
```

Add only internal policy-file parsing types such as:

```go
type policyFileConfig struct {
	AllowedHosts []string                       `json:"allowedHosts"`
	Profiles     map[string]profilePolicyConfig `json:"profiles"`
	Hosts        map[string]hostPolicyConfig    `json:"hosts"`
}
```

- [ ] **Step 4: Re-run the focused browser policy tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser -run "Policy|NormalizeURL" -v
```

Expected: PASS with the new profile-aware policy behavior covered.

- [ ] **Step 5: Commit**

```bash
git add internal/core/browser/policy.go internal/core/browser/policy_test.go internal/core/browser/core_test.go
git commit -m "feat: add profile-aware browser policy baseline"
```

## Task 2: Make Browser Session Bootstrap Stable For Real Auth Profiles

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\chromedp_driver.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\browser\chromedp_driver_test.go`

- [ ] **Step 1: Write failing driver tests for deterministic authenticated bootstrap across first navigation and multi-host reuse**

Add tests that prove:
- first navigation to an authenticated lane receives the session profile before page extraction
- the same tab does not redundantly re-bootstrap the same host
- switching between two hosts that share one named profile stays deterministic

Target test shape:

```go
func TestEnsureSessionBootstrapAppliesNamedProfilePerHostOnce(t *testing.T) {
	// configure profile-backed policy
	// invoke ensureSessionBootstrap twice for the same host
	// assert applySessionCookies called once
}
```

- [ ] **Step 2: Run the focused driver tests to verify they fail first**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser -run "EnsureSessionBootstrap|Driver" -v
```

Expected: FAIL until the bootstrap flow understands the expanded policy/profile model.

- [ ] **Step 3: Implement the minimal driver changes**

Keep the current lifecycle:
- `browser.open` / `browser.navigate` remain the only entry points
- `ensureSessionBootstrap(...)` remains the single pre-navigation hook

Implement:
- profile-aware host bootstrap
- deterministic cookie application before first navigation
- stable per-host bootstrap memoization
- any small retry or sync needed so first extract sees authenticated state without adding a new public task kind

- [ ] **Step 4: Re-run the focused driver tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser -run "EnsureSessionBootstrap|Driver" -v
```

Expected: PASS with stable bootstrap behavior.

- [ ] **Step 5: Commit**

```bash
git add internal/core/browser/chromedp_driver.go internal/core/browser/chromedp_driver_test.go
git commit -m "feat: stabilize browser auth session bootstrap"
```

## Task 3: Expand Canonical Browser Acceptance To Realer Auth And Multi-Site Public Lanes

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\verify-desktop-live-refresh.py`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\run-desktop-live-refresh-check.ps1`

- [ ] **Step 1: Write the acceptance scenarios first in the verifier**

Add scenario definitions for:
- one richer authenticated lane that exercises at least two authenticated selectors or state tokens
- at least two public-web read-only targets, one remaining `example.com`, one second stable HTTPS target controlled by config (current wrapper default: `https://www.iana.org/domains/reserved`)
- result JSON that distinguishes:
  - `authenticatedControlledSessionResults`
  - `publicWebReadOnlyResults`
  - per-target preflight and per-target visibility evidence

Expected Python structures:

```python
PUBLIC_BROWSER_TARGETS = [
    {"id": "example-domain", "url": "...", "selector": "h1", "expected_text": "Example Domain"},
    {"id": "secondary-public", "url": "...", "selector": "...", "expected_text": "..."},
]
```

- [ ] **Step 2: Run Python syntax validation and the wrapper once to confirm the new scenarios fail before implementation is finished**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
python -m py_compile .\scripts\verify-desktop-live-refresh.py
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected: Python syntax passes, but acceptance fails because the richer auth/public matrix is not fully wired yet.

- [ ] **Step 3: Implement the minimal fixture and wrapper support**

Implement:
- richer authenticated fixture content in `App.tsx` using the same embedded preview surface
- deterministic policy/profile file generation in the wrapper
- support for multiple configured public-web targets in the wrapper and verifier
- summary-first assertions so acceptance stays stable and does not depend on fragile prose

Do not add a new desktop page or new public API.

- [ ] **Step 4: Re-run syntax validation and full canonical acceptance**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
python -m py_compile .\scripts\verify-desktop-live-refresh.py
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected: PASS with successful controlled, richer-authenticated, and multi-target public-web browser lanes.

- [ ] **Step 5: Commit**

```bash
git add desktop/frontend/src/App.tsx scripts/verify-desktop-live-refresh.py scripts/run-desktop-live-refresh-check.ps1
git commit -m "feat: expand browser acceptance to real auth and multi-site lanes"
```

## Task 4: Align Runtime, CLI, And Desktop Baseline Wording

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\discovery.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\app.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\app_test.go`

- [ ] **Step 1: Add or update tests that lock the new browser baseline wording**

Cover:
- CLI `tools list` browser baseline line
- desktop browser capability summary for remote canonical and fallback evidence-only modes
- runtime discovery descriptions for browser tools

Representative assertions:

```go
require.Contains(t, output, "browser verified baseline")
require.Contains(t, output, "managed authenticated session lane")
require.Contains(t, output, "multi-target verified HTTPS read-only lanes")
```

- [ ] **Step 2: Run the focused CLI/desktop tests first**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./cmd/cli -run "Tools|Browser" -v
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./... -run "Browser|Fallback" -v
```

Expected: FAIL until wording and summaries are updated.

- [ ] **Step 3: Implement the wording alignment**

Update:
- browser tool descriptions in runtime discovery
- CLI browser baseline summary
- desktop `browserCapabilitySummary(...)`

Keep the message precise:
- verified baseline expanded
- still allowlist-backed
- still not blanket arbitrary-site/action compatibility

- [ ] **Step 4: Re-run the focused tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./cmd/cli -run "Tools|Browser" -v
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./... -run "Browser|Fallback" -v
```

Expected: PASS with aligned wording.

- [ ] **Step 5: Commit**

```bash
git add internal/core/runtime/discovery.go cmd/cli/main.go cmd/cli/main_test.go desktop/app.go desktop/app_test.go
git commit -m "docs: align browser baseline wording across surfaces"
```

## Task 5: Full Regression And Documentation Closeout

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-capability-matrix-and-tool-coverage.md`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-runtime-entry-checklist.md`

- [ ] **Step 1: Update the capability matrix and checklist language**

Update the matrix so browser notes clearly say:
- verified baseline now includes managed authenticated session profiles
- verified baseline now includes more than one constrained public-web read-only target
- remaining gap is still arbitrary-site / arbitrary-auth blanket compatibility

Update the checklist so the canonical gate explicitly records:
- auth profile evidence
- multiple public-web target evidence
- any skipped optional target with reason

- [ ] **Step 2: Run the full required regression suite**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser ./internal/core/runtime ./internal/core/runner ./cmd/cli
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected: all commands exit `0`.

- [ ] **Step 3: Verify the acceptance report contains the new evidence**

Check the final verifier output for:
- controlled local browser lane
- richer authenticated session lane
- multi-target public-web lane
- screenshot artifact paths
- stable summary identity tokens

- [ ] **Step 4: Commit the docs and final closeout**

```bash
git add docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md
git commit -m "docs: record expanded browser verified baseline"
```

## Self-Review

- Spec coverage: the plan covers internal policy/bootstrap, canonical acceptance, CLI/desktop/runtime wording, and docs closeout for the browser baseline gap currently listed in the capability matrix.
- Placeholder scan: no `TODO`, `TBD`, or unnamed files remain.
- Type consistency: the plan keeps the existing `browser.*` task family, `SessionProfile`/`SessionCookie` structures, runtime discovery surfaces, and acceptance wrapper/verifier flow instead of inventing a second browser contract.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-18-browser-real-auth-and-multi-site-baseline-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
