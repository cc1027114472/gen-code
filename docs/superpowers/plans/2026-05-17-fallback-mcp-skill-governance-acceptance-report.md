# 2026-05-17 Fallback MCP Skill Governance Acceptance Report

## Remote Canonical Lane
- source: `remote-app-server`
- trust: `canonical`
- refresh mode: `SSE 实时刷新`
- thread id: `thread-186`
- apply write execution id: `writeexec-66`
- rollback write execution id: `writeexec-67`
- ui base url: `http://127.0.0.1:5174/`
- api base url: `http://127.0.0.1:10008`

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
- `canonicalRuntimeUrl` was empty in the current acceptance payload and was therefore not used as a pass condition
