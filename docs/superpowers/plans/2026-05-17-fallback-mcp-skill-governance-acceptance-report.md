# 2026-05-17 Fallback MCP Skill Governance Acceptance Report

## Remote Canonical Lane
- source: `remote-app-server`
- trust: `canonical`
- refresh mode: recorded by `scripts/verify-desktop-live-refresh.py`

## Local Fallback Lane
- source: `local-fallback`
- trust: `degraded`
- evidence mode: `go-test-evidence`

## MCP Health Baseline
- listed: metadata-only server listing
- health labels: `enabled`, `disabled`, `degraded`, `unreachable`

## Skill Governance Baseline
- groups: `common`, `codex`, `cc`
- inventory fields: group, source, verification status, localization checked
- localization audit status: not completed

## Remaining Exclusions
- fallback is not a second browser acceptance gate
- end-to-end MCP execution is still out of scope
- per-skill governance acceptance remains separate from grouped inventory
