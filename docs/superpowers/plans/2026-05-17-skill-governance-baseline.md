# Skill Governance Baseline

This note defines the minimum governance boundary for grouped skills in `gen-code` without claiming new runtime behavior.

- governed groups: `common`, `codex`, `cc`
- minimum inventory fields: `skill id`, `group`, `source`, `verification status`, `localization checked`, `capability verified`, `isolation status`
- grouped summary fields: `implemented`, `verified`, `localization-pending`, `capability-pending`
- allowed baseline states: `implemented`, `verified`, `partial`, `blocked`
- `skill discovered` does not mean `skill accepted`
- current verified scope is the grouped governance baseline only, not per-skill business capability acceptance
- copied-skill capability verification is now closed as a per-skill static baseline: main document exists, frontmatter is valid, local relative references resolve, and each copied skill retains a minimal reusable capability structure inside `internal/core/skill/catalog`
- Phase B acceptance closes runtime, desktop, approval, and built-in tool coverage first
- grouped skill verification remains separate from per-skill business capability acceptance, but the 1:1 Chinese localization audit baseline is now explicitly closed
- MCP metadata and skill governance are tracked separately to avoid mixing inventory status with runtime acceptance status
- `localization checked = true` only means the copied skill under `internal/core/skill/catalog` has passed the current 1:1 Chinese-localization audit baseline
- `isolation status` only means the skill still sits within the expected `common` / `codex` / `cc` grouping boundary
- the project-local copied catalog under `gen-code/internal/core/skill/catalog` is the only governance truth for copied `codex` / `cc` skills; sibling source directories are reference-only

## CLI Baseline

`gen-code skills list` should expose:

- source / source trust / source detail
- grouped summary lines for `common`, `codex`, `cc`
- per-skill inventory rows with:
  - `group`
  - `source`
  - `verification`
  - `localization`
  - `capability`
  - `isolation`

## Audit Interpretation

- `skill discovered != skill verified`
- `localization checked = false` is an honest audit result, not a temporary display default
- `localization checked = true` means the copied skill in `internal/core/skill/catalog` passed the current 1:1 Chinese-localization audit, not that the skill capability itself has been accepted
- `capability verified = true` means the copied skill in `internal/core/skill/catalog` passed the current per-skill static capability baseline, not that a skill execution engine or per-skill business acceptance exists
- `isolation status = isolated` means a non-common skill stays in its own governed group
- `isolation status = shared-common` is only valid for reusable common skills
- `isolation status = blocked` means the skill could not be cleanly classified and must remain a governance exception

Expected grouped wording:

- `common: implemented=<n> verified=<n> localization-pending=<n>`
- `codex: implemented=<n> verified=<n> localization-pending=<n> capability-pending=<n>`
- `cc: implemented=<n> verified=<n> localization-pending=<n> capability-pending=<n>`
