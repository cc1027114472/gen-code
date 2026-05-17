# Fallback MCP Skill Governance Closeout Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the highest-priority remaining gaps after the Phase 3 default workflow by making `desktop local-fallback` easier to trust, turning the current release notes into one stable regression baseline, and establishing a minimal but explicit MCP and skill-governance verification surface.

**Architecture:** Keep the current `thread -> task -> approval -> write execution -> runtime status -> desktop workbench` model unchanged. This phase does not add tools or new execution routes. Instead, it tightens the product and release surface around three existing boundaries: `remote-app-server` versus `local-fallback`, MCP metadata health visibility, and grouped-skill governance status. The implementation should prefer derivation from the existing runtime contract and desktop bridge instead of adding parallel status logic.

**Tech Stack:** Go, SQLite, Wails desktop shell, React + TypeScript, existing runtime contract, existing desktop runtime bridge, Python/PowerShell verification scripts

---

## File Structure

- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\app.go`
  - Keep fallback runtime status, source/trust text, and bridge-exposed state consistent with the app-server contract.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\app_test.go`
  - Lock fallback runtime visibility, approval/write-execution visibility, and manual-refresh wording in tests.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\runtimeBridge.ts`
  - Keep the desktop runtime payload as the only front-end source for runtime source, trust, refresh mode, MCP health, and skill-governance summaries.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
  - Make fallback status, manual refresh, MCP health, and skill-governance baseline readable in the existing three-column workbench.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\style.css`
  - Add only the styles needed to support the new summary chips, warning states, and governance cards.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\status.go`
  - Normalize runtime status derivation for MCP health summaries and grouped skill baseline summaries without creating a second contract.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\discovery.go`
  - Surface stable grouped-skill inventory and MCP health labels through the existing runtime discovery path.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\discovery_test.go`
  - Verify grouped-skill and MCP discovery summaries stay stable.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\mcp\manager.go`
  - Tighten health labeling semantics for `enabled`, `disabled`, `degraded`, and `unreachable`.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\appserver\runtimecontract\contract.go`
  - Only if needed to expose stable summary fields already implied by current runtime payloads.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
  - Make `runtime status`, `tools list`, and any existing discovery output show the same source/trust/MCP/skill-governance semantics as desktop.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`
  - Lock help and output formatting for fallback, MCP health, and governance summaries.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\run-desktop-live-refresh-check.ps1`
  - Keep the canonical `5174 + 10008` wrapper aligned with the new release baseline wording.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\verify-desktop-live-refresh.py`
  - Emit stable release-baseline evidence and explicitly distinguish canonical remote checks from fallback exclusions.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-runtime-entry-checklist.md`
  - Convert it from a local note into the single maintained release baseline for this phase.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-skill-governance-baseline.md`
  - Add the exact baseline fields, pass criteria, and non-goals for grouped skill governance.
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-capability-matrix-and-tool-coverage.md`
  - Update matrix rows to reflect what becomes verified after this phase.
- Add: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-fallback-mcp-skill-governance-acceptance-report.md`
  - Record the final two-lane fallback/remote verification outcome plus MCP and skill-governance baseline evidence.

### Task 1: Close the desktop local-fallback visibility gap

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\app.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\app_test.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\runtimeBridge.ts`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\App.tsx`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\desktop\frontend\src\style.css`

- [ ] **Step 1: Add failing desktop tests for fallback wording and visibility**

Add focused assertions in `desktop/app_test.go` that check:

- fallback runtime source remains `local-fallback`
- fallback trust remains `degraded`
- manual refresh wording is present when SSE is unavailable
- latest approval and latest write execution remain visible after fallback approve/reject flows

Test shape to add:

```go
func TestDesktopFallbackRuntimeStatusShowsManualRefreshMode(t *testing.T) {
	t.Setenv("GENCODE_RUNTIME_BASE_URL", "http://127.0.0.1:1")
	t.Setenv("GENCODE_DESKTOP_STATE_PATH", filepath.Join(t.TempDir(), "fallback-refresh.sqlite"))

	app := NewApp()
	defer app.shutdown(nil)

	status := app.GetRuntimeStatus()
	if status.RuntimeSource != "local-fallback" {
		t.Fatalf("expected local-fallback runtime source, got %q", status.RuntimeSource)
	}
	if status.RuntimeTrust != "degraded" {
		t.Fatalf("expected degraded runtime trust, got %q", status.RuntimeTrust)
	}
	if status.SupportsSSE {
		t.Fatal("expected fallback runtime to disable SSE")
	}
	if !strings.Contains(status.RuntimeMessage, "manual refresh") {
		t.Fatalf("expected manual refresh wording, got %q", status.RuntimeMessage)
	}
}
```

- [ ] **Step 2: Run the desktop fallback tests to expose current gaps**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./... -run "Fallback|ManualRefresh" -v
```

Expected: FAIL if wording, fallback summary fields, or latest approval/write execution visibility is still inconsistent.

- [ ] **Step 3: Tighten fallback runtime and bridge derivation**

In `desktop/app.go` and `runtimeBridge.ts`, normalize one set of derived labels for:

- runtime source
- runtime trust
- refresh mode
- degraded/canonical detail

Implementation shape:

```go
func fallbackRuntimeMessage(status runtimecontract.RuntimeStatus) string {
	if status.RuntimeSource == "local-fallback" && !status.SupportsSSE {
		return "desktop local-fallback active; manual refresh mode"
	}
	return strings.TrimSpace(status.RuntimeMessage)
}
```

```ts
export function formatRefreshMode(status: RuntimeStatus): string {
  if (!status.supportsSSE) return "手动刷新模式";
  if (status.sseConnected) return "SSE 实时刷新";
  return "SSE 重连中";
}
```

- [ ] **Step 4: Update the desktop cards without changing the information architecture**

In `App.tsx` and `style.css`, keep the existing three-column layout and add only:

- a stable runtime source chip
- a stable refresh-mode chip
- a fallback note that explains the degraded lane honestly
- a clearer latest approval/latest write execution area for fallback snapshots

UI helper shape:

```tsx
function formatRuntimeLaneLabel(status: RuntimeStatus): string {
  if (status.runtimeSource === "remote-app-server") return "远端运行时 / remote-app-server";
  if (status.runtimeSource === "local-fallback") return "本地回退 / desktop local-fallback";
  return "运行时来源未知";
}
```

- [ ] **Step 5: Re-run the desktop regressions**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...

Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add desktop/app.go desktop/app_test.go desktop/frontend/src/runtimeBridge.ts desktop/frontend/src/App.tsx desktop/frontend/src/style.css
git commit -m "feat: clarify desktop fallback runtime lane"
```

### Task 2: Turn the current notes into one maintained release baseline

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\run-desktop-live-refresh-check.ps1`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\scripts\verify-desktop-live-refresh.py`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-runtime-entry-checklist.md`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-capability-matrix-and-tool-coverage.md`
- Add: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-fallback-mcp-skill-governance-acceptance-report.md`

- [ ] **Step 1: Add a failing verification assertion for release-baseline evidence**

In `verify-desktop-live-refresh.py`, add a structured requirement that the acceptance result contains:

- `runtimeSource`
- `runtimeTrust`
- `uiBaseUrl`
- `apiBaseUrl`
- `refreshMode`
- `fallbackEvidenceMode`

Implementation shape to add near the result validation:

```python
required_release_keys = [
    "runtimeSource",
    "runtimeTrust",
    "uiBaseUrl",
    "apiBaseUrl",
    "acceptanceReport",
]
for key in required_release_keys:
    if key not in result:
        raise AssertionError(f"missing release-baseline key: {key}")
```

- [ ] **Step 2: Run the canonical live verification**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected: PASS on the default `5174 + 10008` lane with `runtimeSource = remote-app-server`.

- [ ] **Step 3: Update the release checklist to become the one maintained gate**

Tighten `2026-05-17-runtime-entry-checklist.md` so it explicitly lists:

- canonical targets
- exact commands
- pass standards
- what counts as fallback evidence
- what does not count as release-verified

Checklist structure to keep:

```markdown
## Release Gate

### 1. Desktop Go regression
### 2. Desktop frontend build regression
### 3. Canonical live acceptance regression

## Fallback Interpretation Rules
## Required Evidence To Record
```

- [ ] **Step 4: Write the combined acceptance report**

Create `2026-05-17-fallback-mcp-skill-governance-acceptance-report.md` with:

- remote lane evidence
- fallback lane evidence
- MCP baseline evidence
- skill-governance baseline evidence
- remaining exclusions

Document skeleton:

```markdown
# 2026-05-17 Fallback MCP Skill Governance Acceptance Report

## Remote Canonical Lane
- source:
- trust:
- refresh mode:

## Local Fallback Lane
- source:
- trust:
- evidence mode:

## MCP Health Baseline
- listed:
- health labels:

## Skill Governance Baseline
- groups:
- inventory fields:
- localization audit status:
```

- [ ] **Step 5: Re-run the release commands**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...

Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build

Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add scripts/run-desktop-live-refresh-check.ps1 scripts/verify-desktop-live-refresh.py docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md docs/superpowers/plans/2026-05-17-fallback-mcp-skill-governance-acceptance-report.md
git commit -m "docs: finalize release baseline and acceptance evidence"
```

### Task 3: Close the metadata-level MCP health gap honestly

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\mcp\manager.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\status.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\discovery.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\discovery_test.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`

- [ ] **Step 1: Add failing tests for MCP health labels**

Add tests in `discovery_test.go` and `main_test.go` that assert:

- MCP servers expose stable labels from `enabled`, `disabled`, `degraded`, `unreachable`
- CLI prints the same labels without inventing new wording

Test shape:

```go
func TestDiscoveryIncludesStableMCPHealthLabels(t *testing.T) {
	status := RuntimeStatus{
		MCPServers: []runtimecontract.MCPServer{
			{Name: "browser", Enabled: true, Health: "degraded"},
		},
	}
	if got := summarizeMCPHealth(status.MCPServers[0]); !strings.Contains(got, "degraded") {
		t.Fatalf("expected degraded MCP health label, got %q", got)
	}
}
```

- [ ] **Step 2: Run the targeted runtime and CLI tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runtime ./cmd/cli -run "MCP|Discovery" -v
```

Expected: FAIL if label derivation is still inconsistent.

- [ ] **Step 3: Normalize one MCP health summary path**

In `manager.go`, `status.go`, and `discovery.go`, derive a single readable summary from the current metadata surface without pretending full MCP execution coverage exists.

Implementation shape:

```go
func summarizeMCPHealth(server runtimecontract.MCPServer) string {
	health := strings.TrimSpace(server.Health)
	if health == "" {
		health = "unknown"
	}
	if server.Enabled {
		return fmt.Sprintf("%s (%s)", server.Name, health)
	}
	return fmt.Sprintf("%s (disabled)", server.Name)
}
```

- [ ] **Step 4: Surface the same summary in CLI and runtime**

Update `main.go` so `runtime status` or the existing discovery output shows:

- MCP server count
- per-server health label
- explicit note that this is metadata health, not end-to-end MCP execution

Output shape:

```text
MCP servers:
- browser (enabled, degraded)
- docs (enabled, reachable)
Note: this lane verifies metadata health only.
```

- [ ] **Step 5: Re-run the targeted tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runtime ./cmd/cli -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/core/mcp/manager.go internal/core/runtime/status.go internal/core/runtime/discovery.go internal/core/runtime/discovery_test.go cmd/cli/main.go cmd/cli/main_test.go
git commit -m "feat: stabilize mcp metadata health summaries"
```

### Task 4: Establish the minimum grouped-skill governance baseline

**Files:**
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\core\runtime\discovery.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\internal\appserver\runtimecontract\contract.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\cmd\cli\main_test.go`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-skill-governance-baseline.md`
- Modify: `D:\GOWorks\gen-code-heji\gen-code\docs\superpowers\plans\2026-05-17-capability-matrix-and-tool-coverage.md`

- [ ] **Step 1: Add failing tests for grouped skill baseline summaries**

Add tests that assert the grouped skill baseline can expose at least:

- group name
- source
- verification status
- localization checked state

Test shape:

```go
func TestDiscoveryIncludesSkillGovernanceBaseline(t *testing.T) {
	skills := []runtimecontract.SkillDescriptor{
		{ID: "browser", Group: "common", Source: "bundled"},
	}
	summary := summarizeSkillGovernance(skills)
	if !strings.Contains(summary, "common") {
		t.Fatalf("expected group name in summary, got %q", summary)
	}
}
```

- [ ] **Step 2: Run the targeted discovery and CLI tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runtime ./cmd/cli -run "Skill|Governance|Discovery" -v
```

Expected: FAIL if grouped skill summaries are still too implicit.

- [ ] **Step 3: Surface the minimum governance fields through existing discovery**

Do not create a second protocol. Reuse the current discovery/runtime surfaces to derive:

- governed groups: `common`, `codex`, `cc`
- inventory fields:
  - skill id
  - group
  - source
  - verification status
  - localization checked

Implementation shape:

```go
type SkillGovernanceSummary struct {
	Group               string
	ImplementedCount    int
	VerifiedCount       int
	LocalizationPending int
}
```

- [ ] **Step 4: Make the CLI and docs consume the same baseline**

Update `main.go` and `2026-05-17-skill-governance-baseline.md` so the baseline clearly states:

- `skill discovered` is not `skill accepted`
- grouped skill inventory is separate from runtime release acceptance
- localization audit is still a tracked status, not silently assumed complete

CLI output shape:

```text
Skill governance:
- common: implemented=12 verified=4 localization-pending=8
- codex: implemented=3 verified=0 localization-pending=3
- cc: implemented=2 verified=0 localization-pending=2
```

- [ ] **Step 5: Re-run the discovery and CLI tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runtime ./cmd/cli -v
```

Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/core/runtime/discovery.go internal/appserver/runtimecontract/contract.go cmd/cli/main.go cmd/cli/main_test.go docs/superpowers/plans/2026-05-17-skill-governance-baseline.md docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md
git commit -m "docs: establish grouped skill governance baseline"
```

## Self-Review

- Spec coverage:
  - fallback visibility and manual-refresh semantics: covered by Task 1
  - release baseline and exact commands: covered by Task 2
  - metadata-level MCP health acceptance: covered by Task 3
  - grouped skill governance baseline: covered by Task 4
- Placeholder scan:
  - all tasks include exact files and commands
  - all test steps include a concrete command and expected signal
  - no `TODO`, `TBD`, or hand-wavy “add validation later” placeholders remain
- Type consistency:
  - plan consistently uses `remote-app-server`, `local-fallback`, `RuntimeSource`, `RuntimeTrust`, `supportsSSE`, grouped skill governance, and MCP metadata health terminology

## Execution Handoff

**Plan complete and saved to `docs/superpowers/plans/2026-05-17-fallback-mcp-skill-governance-closeout-plan.md`. Two execution options:**

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
