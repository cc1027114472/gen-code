# Skill Import Staging Inventory

**Date:** May 18, 2026

**Purpose:** Record the additional project-internal skills copied into `gen-code` staging without changing the current runtime discovery truth. These imports are staged for later governance and rollout work; they are **not** part of the currently verified `catalog/{codex,cc}` baseline.

## Summary

- staging root:
  `gen-code/internal/core/skill/imports`
- rollout mode:
  copied into project-internal staging only
- current discovery impact:
  none
- current verified governance baseline:
  unchanged
- intended isolation model:
  `common` shared, `codex` isolated, `cc` isolated
- runtime truth rule:
  `imports is staging evidence, catalog is runtime truth`
- same-name rule:
  same-name skills may exist in both `codex` and `cc`, but promotion and runtime visibility must remain group-scoped
- translation hygiene rule:
  every translated skill must be checked for mojibake or other garbled text before it can be promoted

This staging inventory exists so later phases can decide which imported skills should be promoted into:

- `internal/core/skill/catalog/codex`
- `internal/core/skill/catalog/cc`

without mixing `codex` and `cc` provenance or silently expanding the runtime-visible skill set.

## Gate-State Summary Schema

Each staged skill should be tracked with at least:

- `skill`
- `source_group`
- `gate_state`
- `why`
- `promote_target`
- `blocking_checks`
- `notes`

## Codex Staging Imports

Project-internal staging path:
`internal/core/skill/imports/codex`

### Promoted: Codex

- `architecture-blueprint-generator`
- `frontend-design`
- `golang-backend-development`
- `imagegen`
- `kb-audit-governance-sync`
- `kb-audit-page-docs`
- `kb-audit-product-blueprint`
- `kb-audit-report-pack`
- `openai-docs`
- `plugin-creator`
- `skill-creator`
- `skill-installer`

Why:

- each project-local copied package now has a clean project-internal truth under `imports/codex`
- promoted copies passed the 1:1 Chinese localization audit
- each promoted skill passed mojibake review on the retained project-local copy
- static capability verification stayed `true` after promotion into `catalog/codex`

Notes:

- `imagegen`, `openai-docs`, `plugin-creator`, `skill-creator`, and `skill-installer` came from the Codex `.system` skill set, but runtime-visible `source` still remains `codex`
- several promoted packages include helper assets, scripts, references, or plugin metadata beyond `SKILL.md`

### Needs-Trim: Codex

- `browser-use`
- `chrome`
- `design-consultation`

Why:

- `browser-use` and `chrome` were copied from plugin-bundled version directories
- `design-consultation` still depends on surrounding `gstack`-style commands and paths that do not fit the current standalone catalog baseline
- the remaining blocked imports still carry a much larger or looser bundle boundary than a normal governed skill
- the remaining blocked imports must be trimmed to the smallest machine-usable copy before any promotion

Promote target:

- `internal/core/skill/catalog/codex`

Blocking checks:

- source-group confirmation
- dependency and asset trimming
- 1:1 Chinese localization audit
- mojibake / garbled-text check per translated skill
- static capability verification

## CC Staging Imports

Project-internal staging path:
`internal/core/skill/imports/cc`

### Ready: CC

- `architecture-blueprint-generator`
- `breakdown-epic-arch`
- `breakdown-epic-pm`
- `breakdown-feature-prd`
- `create-implementation-plan`
- `frontend-design`
- `go-backend-clean-architecture`
- `kb-audit-flow-prototype`
- `ralph-loop`
- `tailwindcss`
- `vercel-react-best-practices`
- `vite`

Promoted this round:

- `react-vite-expert`
- `vercel-react-best-practices`
- `skill-creator`

Why:

- the project-local copied package now passes the 1:1 Chinese localization audit
- the copied package continues to pass static capability verification after localization
- it is now runtime-visible from `internal/core/skill/catalog/cc`

Notes:

- Several imports overlap by name with Codex-visible skills, such as `architecture-blueprint-generator`, `frontend-design`, and `skill-creator`.
- These same-name imports are intentionally kept separate in staging so later promotion can preserve source isolation.

Blocking checks before promotion:

- project-local copy integrity
- 1:1 Chinese localization audit
- mojibake / garbled-text check per translated skill
- static capability verification

### Needs-Trim: CC

- `ui-ux-pro-max`

Why:

- this import is a valid CC-sourced staging item, but it still points at project-external placeholders instead of a self-contained project-local copied package
- `ui-ux-pro-max` currently points at project-external `src/ui-ux-pro-max` placeholders rather than a self-contained project-local copied package

### Trimmed And Promoted: CC

- `agent-browser`
- `canvas-design`
- `planning-with-files`
- `use-my-browser`

Why:

- each package was reduced to the smallest machine-usable project-local copy kept in `imports/cc`
- retained user-visible markdown, templates, and script-facing guidance were completed with 1:1 Chinese localization
- each promoted copy passed mojibake review, localization audit, and static capability verification before entering `catalog/cc`

Promote target:

- `internal/core/skill/catalog/cc`

Blocking checks:

- source-group confirmation
- dependency and asset trimming
- 1:1 Chinese localization audit
- mojibake / garbled-text check per translated skill
- static capability verification

### Defer: CC

- `gstack`

Why:

- `gstack` is the largest staged import and includes many nested sub-skills and support assets
- it behaves more like a governed suite or bundled tool ecosystem than a normal single promoted skill

Promote target:

- none in the current phase

Blocking checks:

- separate bundle-governance design
- explicit sub-skill visibility policy
- runtime exposure policy
- asset-retention policy

## Current Boundaries

- These staged imports are **not** read by `discoverSiblingRuntimeContent(...)`.
- These staged imports are **not** surfaced by `skills list`, `/api/skills`, or desktop fallback skill inventory.
- These staged imports are staging evidence only; `catalog/{codex,cc}` remains the runtime-visible truth.
- The current runtime-visible skill truth remains:
  - built-in `common.browser`
  - project-local `catalog/codex`
  - project-local `catalog/cc`

## Next Promotion Gate

Before any staged import is promoted into the runtime-visible catalog, it should go through:

1. source-to-group confirmation (`codex` or `cc`)
2. project-local copy integrity check
3. 1:1 Chinese localization audit decision
4. mojibake / garbled-text check for the translated skill body and references
5. static capability verification
6. runtime / CLI / desktop inventory update
7. capability matrix and governance baseline sync

### Mandatory per-skill mojibake check

This is a hard gate for every translated or promoted skill, not a spot check:

- inspect the project-local copy that will be used inside `gen-code`
- inspect `SKILL.md` and any copied local Markdown or text references shipped with that skill
- reject mojibake, replacement glyphs, broken punctuation, mixed-encoding artifacts, or visibly garbled Chinese text
- fix the project-local copy before the skill can be treated as promotion-ready
- preserve the existing isolation model while doing this review:
  - `codex` runtime view = `common + codex`
  - `cc` runtime view = `common + cc`
  - same-name `codex` and `cc` skills must stay isolated, not merged

`ready` only means structurally promotable. It does not waive the mandatory mojibake / garbled-text check.

Until that work is complete, this staging inventory is the only supported truth for the newly copied project-internal imports.
