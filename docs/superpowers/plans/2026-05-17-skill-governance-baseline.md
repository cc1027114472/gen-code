# Skill Governance Baseline

This note defines the minimum governance boundary for grouped skills in `gen-code` without claiming new runtime behavior.

- governed groups: `common`, `codex`, `cc`
- minimum inventory fields: `skill id`, `group`, `source`, `verification status`, `localization checked`
- grouped summary fields: `implemented`, `verified`, `localization-pending`
- allowed baseline states: `implemented`, `verified`, `partial`, `blocked`
- `skill discovered` does not mean `skill accepted`
- current verified scope is the grouped governance baseline only, not per-skill business capability acceptance
- Phase B acceptance closes runtime, desktop, approval, and built-in tool coverage first
- grouped skill verification and Chinese localization audit remain separate follow-up work
- MCP metadata and skill governance are tracked separately to avoid mixing inventory status with runtime acceptance status

## CLI Baseline

`gen-code skills list` should expose:

- source / source trust / source detail
- grouped summary lines for `common`, `codex`, `cc`
- per-skill inventory rows with:
  - `group`
  - `source`
  - `verification`
  - `localization`

Expected grouped wording:

- `common: implemented=<n> verified=<n> localization-pending=<n>`
- `codex: implemented=<n> verified=<n> localization-pending=<n>`
- `cc: implemented=<n> verified=<n> localization-pending=<n>`
