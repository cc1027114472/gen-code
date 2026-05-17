# Runtime Entry Release Checklist

## Canonical Targets

- UI: `http://127.0.0.1:5174/`
- Canonical remote runtime: `http://127.0.0.1:10008`
- Override rule: set `GEN_CODE_API_BASE_URL` only when the trusted current-code runtime is intentionally running on another clean port, and record that port in the acceptance notes

## Release Gate

This is the short release checklist for the currently real and runnable regression lane. A release-candidate pass requires all 3 commands below to pass without local edits between runs.

This checklist is intentionally limited to the runtime and desktop acceptance gate. It does not by itself mark skill governance inventory or MCP capability expansion as release-verified.

### 1. Desktop Go regression

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
```

Pass standard:

- command exits with code `0`
- no package in `desktop` fails
- fallback persistence coverage remains green, including restart and waiting-state tests

### 2. Desktop frontend build regression

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

Pass standard:

- command exits with code `0`
- TypeScript compilation succeeds
- Vite production build completes

### 3. Canonical live acceptance regression

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Pass standard:

- command exits with code `0`
- wrapper resolves:
  - UI base URL to `http://127.0.0.1:5174/` unless intentionally overridden
  - API base URL to `http://127.0.0.1:10008` unless intentionally overridden
- `/api/runtime/status` reports:
  - `runtimeSource = remote-app-server`
  - `runtimeTrust = canonical`
- if `canonicalRuntimeUrl` is present, it matches the API base URL used for the run
- Python verification completes the direct second-batch read tools
- Python verification completes all 3 `agent.run` scenarios:
  - `filter_then_read`
  - `search_then_detailed`
  - `stat_then_read`
- `workspace.apply_patch` enters approval, executes after approval, and records a write execution
- `workspace.apply_patch.rollback` enters approval, executes after approval, and records a rollback write execution

## Fallback Interpretation Rules

- `local-fallback` is valid supporting evidence for persistence and restart correctness
- `local-fallback` is not a substitute for the canonical remote acceptance lane
- if the default `10008` runtime is stale or intentionally replaced, use a clean port override and record the exact port in the acceptance report

## Required Evidence To Record

For the acceptance notes or closeout doc, record at least:

- exact UI and API target URLs used
- runtime source and runtime trust
- thread ID from the live acceptance run
- apply write execution ID
- rollback write execution ID
- any intentional port override or degraded-path exception
- any skipped skill-governance or MCP-health assertion, with the reason recorded explicitly
