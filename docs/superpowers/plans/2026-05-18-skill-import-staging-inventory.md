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

This staging inventory exists so later phases can decide which imported skills should be promoted into:

- `internal/core/skill/catalog/codex`
- `internal/core/skill/catalog/cc`

without mixing `codex` and `cc` provenance or silently expanding the runtime-visible skill set.

## Codex Staging Imports

Project-internal staging path:
`internal/core/skill/imports/codex`

Imported items:

- `architecture-blueprint-generator`
- `browser-use`
- `chrome`
- `design-consultation`
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

Notes:

- `browser-use` and `chrome` were copied from plugin-bundled version directories and kept in staging only.
- `imagegen`, `openai-docs`, `plugin-creator`, `skill-creator`, and `skill-installer` came from the Codex `.system` skill set.
- Several imports include helper assets, scripts, references, or plugin metadata beyond `SKILL.md`.

## CC Staging Imports

Project-internal staging path:
`internal/core/skill/imports/cc`

Imported items:

- `agent-browser`
- `architecture-blueprint-generator`
- `breakdown-epic-arch`
- `breakdown-epic-pm`
- `breakdown-feature-prd`
- `canvas-design`
- `create-implementation-plan`
- `find-skills`
- `frontend-design`
- `go-backend-clean-architecture`
- `gstack`
- `kb-audit-flow-prototype`
- `planning-with-files`
- `ralph-loop`
- `react-vite-expert`
- `skill-creator`
- `tailwindcss`
- `ui-ux-pro-max`
- `use-my-browser`
- `vercel-react-best-practices`
- `vite`
- `web-design-guidelines`

Notes:

- `gstack` is the largest staged import and includes many nested sub-skills and support assets.
- Several imports overlap by name with Codex-visible skills, such as `architecture-blueprint-generator`, `frontend-design`, and `skill-creator`.
- These same-name imports are intentionally kept separate in staging so later promotion can preserve source isolation.

## Current Boundaries

- These staged imports are **not** read by `discoverSiblingRuntimeContent(...)`.
- These staged imports are **not** surfaced by `skills list`, `/api/skills`, or desktop fallback skill inventory.
- The current runtime-visible skill truth remains:
  - built-in `common.browser`
  - project-local `catalog/codex`
  - project-local `catalog/cc`

## Next Promotion Gate

Before any staged import is promoted into the runtime-visible catalog, it should go through:

1. source-to-group confirmation (`codex` or `cc`)
2. project-local copy integrity check
3. localization audit decision
4. static capability verification
5. runtime / CLI / desktop inventory update
6. capability matrix and governance baseline sync

Until that work is complete, this staging inventory is the only supported truth for the newly copied project-internal imports.
