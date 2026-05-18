# Runtime Entry Release Checklist

## Canonical Targets

- UI: `http://127.0.0.1:5174/`
- API: `http://127.0.0.1:10008`
- Canonical lane: `remote-app-server` with `canonical` trust

## Release Gate

### 1. Desktop Go regression

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
```

Pass standard:

- exits `0`
- fallback persistence, manual refresh, and restart coverage stay green

### 2. Desktop frontend build regression

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

Pass standard:

- exits `0`
- TypeScript and Vite build succeed

### 3. Canonical live acceptance regression

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Pass standard:

- exits `0`
- wrapper must either use an already-healthy canonical instance or bootstrap the current repo code before verification
- `runtimeSource = remote-app-server`
- `runtimeTrust = canonical`
- `refreshMode` and `fallbackEvidenceMode` are recorded in the verifier output
- `fallback` is supporting evidence only, not a second browser gate
- `canonicalRuntimeUrl`, when present, matches the API target used for the run
- all direct read-tool, MCP execution, agent, apply, and rollback checks pass
- canonical browser acceptance now explicitly includes a controlled browser interaction lane for `browser.open`, `browser.type`, `browser.click`, `browser.extract`, and `browser.screenshot`, plus one managed authenticated session fixture lane and multiple constrained verified HTTPS read-only lanes
- canonical full acceptance also includes one UI-first `agent.run` browser scenario on a controlled local page, with child `browser.open -> browser.type -> browser.click -> browser.extract -> browser.screenshot -> respond`
- MCP execution canonical lane now includes fixture regression, official SDK external, and third-party time server scenarios
- canonical direct read-tool coverage explicitly includes `workspace.read_file`, `workspace.list_files`, `workspace.stat_file`, `workspace.read_files_batch`, `workspace.list_files_filtered`, and `workspace.search_text_detailed`
- canonical lane explicitly includes parent/child waiting visibility and one resumed-to-completion recovery continuation scenario
- current MCP verification is a multi-server stdio external baseline: fixture regression, official SDK external, and one third-party time server; it is still not blanket compatibility for arbitrary third-party MCP servers

## Fallback Interpretation Rules

- `local-fallback` supports restart and degraded desktop evidence
- `local-fallback` supports source/trust/manual-refresh evidence
- `local-fallback` does not satisfy the canonical live acceptance gate
- fallback evidence must stay explicitly labeled as non-browser acceptance

## Required Evidence To Record

- exact UI and API URLs
- `runtimeSource`
- `runtimeTrust`
- `refreshMode`
- `fallbackEvidenceMode`
- thread ID
- apply write execution ID
- rollback write execution ID
- MCP execution task ID and summary
- browser interaction task IDs, summaries, and screenshot artifact path
- UI-first browser-agent parent task ID, child browser task IDs/kinds, extract summary, assistant message, and screenshot artifact path
- authenticated browser fixture task IDs, session evidence, profile evidence, and screenshot artifact path
- public-web read-only browser task IDs, per-target URL evidence, and screenshot artifact path
- any intentional port override
- any skipped MCP or skill-governance assertion, with reason
- for the first real GitHub smoke run: workflow name, run id, artifact download status, and failure category when the run is not green
