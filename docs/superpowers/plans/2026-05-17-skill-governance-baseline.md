# Skill Governance Baseline

This note defines the minimum governance boundary for grouped skills in `gen-code` without claiming new runtime behavior.

- governed groups: `common`, `codex`, `cc`
- minimum inventory fields: `skill id`, `group`, `source`, `verification status`, `localization checked`, `isolation status`
- grouped summary fields: `implemented`, `verified`, `localization-pending`
- allowed baseline states: `implemented`, `verified`, `partial`, `blocked`
- `skill discovered` does not mean `skill accepted`
- current verified scope is the grouped governance baseline only, not per-skill business capability acceptance
- Phase B acceptance closes runtime, desktop, approval, and built-in tool coverage first
- grouped skill verification and Chinese localization audit remain separate follow-up work
- MCP metadata and skill governance are tracked separately to avoid mixing inventory status with runtime acceptance status
- `localization checked = true` only means the copied skill has passed the current 1:1 Chinese-localization audit baseline
- `isolation status` only means the skill still sits within the expected `common` / `codex` / `cc` grouping boundary

## CLI Baseline

`gen-code skills list` should expose:

- source / source trust / source detail
- grouped summary lines for `common`, `codex`, `cc`
- per-skill inventory rows with:
  - `group`
  - `source`
  - `verification`
  - `localization`
  - `isolation`

## Audit Interpretation

- `skill discovered != skill verified`
- `localization checked = false` is an honest audit result, not a temporary display default
- `isolation status = isolated` means a non-common skill stays in its own governed group
- `isolation status = shared-common` is only valid for reusable common skills
- `isolation status = blocked` means the skill could not be cleanly classified and must remain a governance exception

Expected grouped wording:

- `common: implemented=<n> verified=<n> localization-pending=<n>`
- `codex: implemented=<n> verified=<n> localization-pending=<n>`
- `cc: implemented=<n> verified=<n> localization-pending=<n>`
