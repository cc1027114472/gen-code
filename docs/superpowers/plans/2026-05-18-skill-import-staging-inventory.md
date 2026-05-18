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
- `browser-use`
- `chrome`
- `frontend-design`
- `golang-backend-development`
- `imagegen`
- `kb-audit-governance-sync`
- `kb-audit-page-docs`
- `kb-audit-product-blueprint`
- `kb-audit-report-pack`
- `openai-docs`
- `plugin-creator`
- `design-consultation`
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
- `browser-use` and `chrome` were rebuilt from plugin-bundled staging shells into canonical project-local skill packages rooted at `<id>/SKILL.md`, with only the minimal retained scripts and docs needed for the current governed baseline
- `design-consultation` was rebuilt from a garbled gstack-generated staging copy into a standalone canonical copied skill that keeps the design-system workflow while dropping gstack-only preamble, macros, telemetry hooks, and external path dependencies

### Needs-Trim: Codex

- none

Why:

- no Codex staged imports remain in the `needs-trim` lane after `design-consultation` was rebuilt as a standalone canonical copied skill and promoted into `catalog/codex`

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

- none

Why:

- all remaining promotable CC skills have either already been promoted or deliberately remain deferred as non-standard bundles

### Trimmed And Promoted: CC

- `agent-browser`
- `canvas-design`
- `careful`
- `connect-chrome`
- `freeze`
- `guard`
- `land-and-deploy`
- `planning-with-files`
- `setup-browser-cookies`
- `setup-deploy`
- `qa`
- `review`
- `ship`
- `ui-ux-pro-max`
- `unfreeze`
- `use-my-browser`

Why:

- each package was reduced to the smallest machine-usable project-local copy kept in `imports/cc`
- retained user-visible markdown, templates, and script-facing guidance were completed with 1:1 Chinese localization
- each promoted copy passed mojibake review, localization audit, and static capability verification before entering `catalog/cc`
- `ui-ux-pro-max` was first rebuilt from a placeholder shell into a self-contained project-local package with `scripts/search.py` plus the minimal retained data files needed for `--design-system`, `--domain`, and `--stack` flows
- `careful`, `freeze`, and `unfreeze` were extracted from the deferred `gstack` suite into standalone project-local copied packages so they could enter the normal `cc` promotion lane without making the full suite runtime-visible
- `guard` followed as a sibling-aware copied skill that intentionally reuses the already promoted `careful` and `freeze` hook assets instead of duplicating them
- `setup-browser-cookies`, `connect-chrome`, and `setup-deploy` were promoted as a gstack-heavy preamble-retained lane: the project-local copied truth is the cleaned and localized `SKILL.md`, while external browser, extension, user-state, and deploy-environment references remain documented as part of the workflow
- `qa` and `review` now extend that same gstack-heavy preamble-retained lane with localized project-local `SKILL.md` copies plus their retained local markdown references and templates
- `ship` and `land-and-deploy` now complete the release lane in that same gstack-heavy preamble-retained model, keeping only localized project-local `SKILL.md` truth while preserving external PR / CI / deploy / canary workflow semantics

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

- `gstack` is no longer tracked as a generic oversize defer; it is explicitly treated as a suite-governance defer
- the staged package combines copied skill candidates with suite infrastructure, bundled tooling, browser runtime assets, build scripts, tests, and vendored dependencies
- it behaves more like a governed suite or bundled tool ecosystem than a normal single promoted skill, so it cannot enter `catalog/cc` through the ordinary per-skill promotion lane
- the lightweight split lane is now complete for `careful`, `freeze`, `unfreeze`, and the sibling-aware `guard`

Promote target:

- none in the current phase

Blocking checks:

- suite-to-subskill governance design
- explicit sub-skill visibility policy
- runtime exposure policy
- asset-retention policy
- project-local standalone truth per future sub-skill
- per-sub-skill localization, mojibake, and capability gates

#### `gstack` Suite Governance Classification

This section records the stable classification truth for `internal/core/skill/imports/cc/gstack`. It is governance evidence only and does not change runtime discovery.

Suite infrastructure:

- top-level `package.json` and bundled workspace metadata
- `.gstack/`, `bin/`, `browse/`, `lib/`, `scripts/`, `supabase/`, `extension/`
- `node_modules/`, `test/`, and other build/test/runtime support trees

Candidate sub-skills:

- directory skills with their own `SKILL.md`, such as `autoplan`, `benchmark`, `browse`, `canary`, `codex`, `connect-chrome`, `cso`, `design-consultation`, `design-html`, `design-review`, `design-shotgun`, `document-release`, `gstack-upgrade`, `guard`, `investigate`, `land-and-deploy`, `learn`, `office-hours`, `plan-ceo-review`, `plan-design-review`, `plan-eng-review`, `qa`, `qa-only`, `review`, `setup-browser-cookies`, `setup-deploy`, `ship`, and `unfreeze`
- `careful`, `freeze`, `unfreeze`, and `guard` have already been extracted into standalone `cc` copied packages and promoted through the normal skill lane
- `qa` and `review` have now also been extracted into standalone `cc` copied packages with their retained local markdown assets promoted alongside them

Promoted and closed:

- `careful`
- `freeze`
- `unfreeze`
- `guard`
- `setup-browser-cookies`
- `connect-chrome`
- `setup-deploy`
- `qa`
- `review`
- `ship`
- `land-and-deploy`

Non-promotable suite-only surfaces:

- telemetry, upgrade, routing, global-discover, repo-mode, learn/logging, and browser-daemon style flows that coordinate multiple skills or the broader suite runtime
- shared build, compile, browser server, and extension-host flows that support the suite as a product rather than a single governed copied skill

Suite-only / long-term defer surfaces:

- `autoplan`
- `benchmark`
- `canary`
- `codex`
- `cso`
- `design-consultation`
- `design-html`
- `design-review`
- `design-shotgun`
- `document-release`
- `gstack-upgrade`
- `investigate`
- `learn`
- `office-hours`
- `plan-ceo-review`
- `plan-design-review`
- `plan-eng-review`
- `qa-only`
- `retro`

Why these remain suite-only:

- each one is a workflow-heavy orchestration surface rather than a minimal copied skill package
- they continue to lean on shared gstack preamble, suite routing, learnings, review loops, browser runtime, or broader orchestration semantics
- they are no longer tracked as pending promotion backlog in the current product baseline

Stable blocker labels for future `gstack` split work:

- `suite-only dependency`
- `missing standalone project-local truth`
- `bundled runtime/tooling dependency`
- `oversized asset bundle`
- `localization pending`
- `capability structure pending`

#### `browse` blocked truth

- `browse` was evaluated as the final remaining runtime-heavy `gstack` split candidate and is now explicitly blocked, not merely “next up”
- the staged package can build `browse.exe` and `find-browse.exe`, but the compiled retained set does not collapse into a safe minimal copied package:
  - `find-browse` still resolves external install locations instead of treating the copied package itself as the primary truth
  - the CLI / daemon flow still carries suite-coupled runtime semantics such as `sidebar-agent` and adjacent source-tree expectations
  - the compiled binaries balloon to runtime-heavy artifacts that are materially larger than the staged source slice and do not satisfy the intended minimal governed baseline
- current blocker labels for `browse` are:
  - `missing standalone project-local truth`
  - `bundled runtime/tooling dependency`
  - `oversized asset bundle`
- as a result, `browse` does not enter `internal/core/skill/catalog/cc` in the current phase

Future split entry rule:

- no further single-skill promotion work remains in the current `gstack` lane
- future work, if any, starts from a brand-new redesign decision for `browse` or keeps the remaining suite surfaces in long-term defer

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
