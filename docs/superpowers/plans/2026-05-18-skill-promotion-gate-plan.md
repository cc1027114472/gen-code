# Skill Promotion Gate Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote selected project-internal staged skills into the runtime-visible `catalog/{codex,cc}` baseline without breaking source isolation, localization governance, or static capability verification.

**Architecture:** The `internal/core/skill/imports/{codex,cc}` tree remains the staging area, while `internal/core/skill/catalog/{codex,cc}` remains the only runtime-visible truth for copied skills. Promotion happens in gated batches: first classify staged imports, then trim oversized or over-bundled assets, then copy only approved skills into the active catalog and re-run governance verification across runtime, CLI, and desktop fallback.

**Tech Stack:** Go runtime discovery, CLI skills inventory, desktop fallback inventory, Markdown skill catalogs, governance and capability audit tests.

---

## File Structure

### Runtime and governance surfaces

- `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/imports/codex`
  - Project-internal staging area for Codex-sourced skills that are not yet runtime-visible.
- `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/imports/cc`
  - Project-internal staging area for CC-sourced skills that are not yet runtime-visible.
- `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/catalog/codex`
  - Runtime-visible Codex skill truth after promotion.
- `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/catalog/cc`
  - Runtime-visible CC skill truth after promotion.
- `D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/discovery.go`
  - Existing runtime discovery truth; should keep reading only `catalog/{codex,cc}` plus built-in `common.browser`.
- `D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main.go`
  - Existing CLI inventory surface for grouped skills and governance fields.
- `D:/GOWorks/gen-code-heji/gen-code/desktop/app.go`
  - Existing desktop fallback inventory aggregation.

### Tests and verification

- `D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/*skill*test*.go`
  - Runtime discovery and governance truth verification.
- `D:/GOWorks/gen-code-heji/gen-code/internal/router/*skill*test*.go`
  - `/api/skills` governance exposure verification.
- `D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main_test.go`
  - `skills list` grouped summary and inventory output verification.
- `D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go`
  - Desktop fallback inventory consistency verification.

### Planning and governance docs

- `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-18-skill-import-staging-inventory.md`
  - Current staging-only inventory truth.
- `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-skill-governance-baseline.md`
  - Governance baseline doc that must stay aligned with runtime truth.
- `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md`
  - Capability matrix that must stay aligned with promotion outcome.

## Promotion Policy

### Isolation model

- `common` stays shared and built-in.
- `codex` runtime view must stay `common + codex`.
- `cc` runtime view must stay `common + cc`.
- same-name skills may exist in both `codex` and `cc`, but promotion and runtime visibility must remain group-scoped.
- Do not create extra runtime groups for source roots such as `.system`, bundled plugins, or `.agents`.

### Translation hygiene rule

- Every translated skill must pass a mojibake / garbled-text check before promotion.
- A skill with corrupted Chinese text, broken UTF-8 carryover, or visibly garbled copied prose must remain blocked even if structure and references are otherwise valid.
- The mojibake check applies to:
  - primary `SKILL.md`
  - referenced local markdown content that ships with the promoted copy
  - user-visible templates or guidance text that is part of the skill package
- The mojibake check is per-skill and mandatory for every translated or promoted item, including items already classified as `ready`.

### Promotion gate states

- `ready`
  - Small or moderate footprint.
  - Clear skill structure.
  - No oversized vendored dependency tree.
  - No mojibake or garbled translated text in the promoted copy.
  - Suitable for direct copy into `catalog/{group}` after audit.
- `needs-trim`
  - Correct source/group, but bundled with extra scripts, fonts, plugin payloads, or vendored dependencies that should not become runtime-visible as-is.
  - Still requires a per-skill mojibake / garbled-text check after trimming and before promotion.
- `defer`
  - Better treated as a tool suite, plugin bundle, or ecosystem package than as a normal single governed skill in the current baseline.

### Current classification truth

#### Promoted: Codex

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

#### Needs-Trim: Codex

- none

#### Ready: CC

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

Current first promoted CC batch:

- `breakdown-epic-arch`
- `breakdown-epic-pm`
- `breakdown-feature-prd`
- `create-implementation-plan`
- `tailwindcss`
- `vite`

Current second promoted CC batch:

- `architecture-blueprint-generator`
- `find-skills`
- `frontend-design`
- `go-backend-clean-architecture`
- `kb-audit-flow-prototype`
- `ralph-loop`
- `web-design-guidelines`

Current first needs-trim CC batch promoted after trim + localization:

- `agent-browser`
- `canvas-design`
- `planning-with-files`
- `ui-ux-pro-max`
- `use-my-browser`

Current ready-localization CC promotion result:

- `react-vite-expert`
- `vercel-react-best-practices`
- `skill-creator`

#### Needs-Trim: CC

- none

#### Defer: CC

- `gstack`

### Evidence behind current classification

- `codex/browser-use`: previously about `340` files, about `7.08 MB` before trim and canonical promotion
- `codex/chrome`: previously about `345` files, about `7.78 MB` before trim and canonical promotion
- `codex/design-consultation`: blocked earlier by a garbled generated copy plus gstack-only preamble and macro dependencies, then rebuilt as a standalone single-skill copied package before promotion
- `cc/canvas-design`: about `83` files, about `5.3 MB`
- `cc/gstack`: about `6426` files, about `119.54 MB`

These measurements explain why the large staged imports originally required trim or defer decisions before they could fit the governed skill baseline.

## Task 1: Freeze the promotion gate inventory

**Files:**
- Modify: `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-18-skill-import-staging-inventory.md`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-18-skill-promotion-gate-plan.md`

- [ ] **Step 1: Add explicit gate state sections to the staging inventory**

Update the staging inventory so it does not only list imported skills, but also records the current gate state for each imported item: `ready`, `needs-trim`, or `defer`.

- [ ] **Step 2: Record the isolation rule in both planning docs**

Ensure both docs state that promotion must preserve:

```text
codex runtime view = common + codex
cc runtime view = common + cc
```

and must not mix same-name skills across `codex` and `cc`.

- [ ] **Step 3: Record the mandatory mojibake gate**

Ensure both docs explicitly say that each translated skill must be checked for garbled text before it can move from staging into `catalog/{codex,cc}`.

- [ ] **Step 4: Record the inspection scope**

Ensure both docs explicitly lock the per-skill mojibake inspection scope to:

```text
SKILL.md
copied local markdown or text references
user-visible templates or guidance text shipped with the promoted copy
```

- [ ] **Step 5: Commit**

```bash
git add docs/superpowers/plans/2026-05-18-skill-import-staging-inventory.md docs/superpowers/plans/2026-05-18-skill-promotion-gate-plan.md
git commit -m "docs: define skill promotion gate"
```

## Task 2: Promote the first Ready Codex batch

**Files:**
- Create: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/catalog/codex/<skill>/...`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/discovery.go` only if inventory assumptions require adjacent updates, not new source roots
- Test: `D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/*skill*test*.go`
- Test: `D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main_test.go`
- Test: `D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go`

- [ ] **Step 1: Copy one small Codex-ready slice into active catalog**

Start with the safest subset:

```text
frontend-design
```

Do not promote `golang-backend-development` in the same batch unless its copied skill package also passes the full 1:1 Chinese localization gate; its staged `SKILL.md` is usable, but the broader copied package still needs separate localization follow-up if runtime-visible supporting docs are included.

Copy these from `imports/codex` into `catalog/codex` using project-local copies only.

- [ ] **Step 2: Run localization and static capability checks on the copied slice**

Verify each promoted skill still passes:

```text
localizationChecked = true
capabilityVerified = true
mojibake check = clean
```

before touching discovery assertions.

- [ ] **Step 3: Update tests if grouped counts change**

Only update tests whose assertions depend on catalog inventory counts or enumerated skill IDs. Do not widen runtime discovery beyond `catalog/codex`.

- [ ] **Step 4: Manually inspect translated text for mojibake before promotion lands**

For each copied skill, check:

```text
section headings
bullet lists
Chinese prose paragraphs
quoted examples
referenced markdown files included in the promoted copy
```

Expected: no replacement glyphs, no broken encoding sequences, no visibly corrupted copied Chinese text.

This check is mandatory for every promoted skill in the batch, even when the skill remains in the `ready` classification.

- [ ] **Step 5: Run the focused governance regression**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runtime ./internal/router ./cmd/cli -run "Skill|Governance|Discovery|SkillsList|Capability" -v
```

Expected: all skill governance tests still pass with the larger `catalog/codex` set.

- [ ] **Step 6: Commit**

```bash
git add internal/core/skill/catalog/codex internal/core/runtime cmd/cli desktop/app_test.go docs/superpowers/plans
git commit -m "feat: promote first codex skill batch"
```

## Task 3: Promote the first Ready CC batch

**Files:**
- Create: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/catalog/cc/<skill>/...`
- Test: `D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/*skill*test*.go`
- Test: `D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main_test.go`
- Test: `D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go`

- [ ] **Step 1: Copy one small CC-ready slice into active catalog**

Start with the safest subset:

```text
breakdown-epic-arch
breakdown-epic-pm
breakdown-feature-prd
create-implementation-plan
tailwindcss
vite
```

- [ ] **Step 2: Re-check group isolation**

Confirm runtime-visible results still imply:

```text
codex skills do not appear in cc inventory
cc skills do not appear in codex inventory
common.browser remains shared
```

- [ ] **Step 3: Manually inspect translated text for mojibake before promotion lands**

For each copied skill, check:

```text
section headings
bullet lists
Chinese prose paragraphs
quoted examples
referenced markdown files included in the promoted copy
```

Expected: no replacement glyphs, no broken encoding sequences, no visibly corrupted copied Chinese text.

This check is mandatory for every promoted skill in the batch, even when the skill remains in the `ready` classification.

- [ ] **Step 4: Run the same focused governance regression**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runtime ./internal/router ./cmd/cli -run "Skill|Governance|Discovery|SkillsList|Capability" -v
```

Expected: grouped summary and per-skill fields remain aligned.

- [ ] **Step 5: Commit**

```bash
git add internal/core/skill/catalog/cc internal/core/runtime cmd/cli desktop/app_test.go docs/superpowers/plans
git commit -m "feat: promote first cc skill batch"
```

## Task 4: Trim oversized staged imports before any promotion

**Files:**
- Modify: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/imports/codex/browser-use/...`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/imports/codex/chrome/...`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/imports/cc/agent-browser/...`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/imports/cc/canvas-design/...`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/imports/cc/skill-creator/...`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/imports/cc/ui-ux-pro-max/...`
- Test: `D:/GOWorks/gen-code-heji/gen-code/internal/core/skill/*capability*test*.go`

- [ ] **Step 1: Define a minimal-retention rule**

For each `needs-trim` skill, define what is truly required to keep the skill machine-usable:

```text
SKILL.md
referenced local docs
referenced templates
required scripts
small supporting assets only when referenced
```

Everything else stays out of the future promoted copy.

- [ ] **Step 2: Remove vendored dependencies and non-essential bulk assets from staging where appropriate**

Examples to target:

```text
node_modules
compiled plugin payloads
large font packs
example bundles not referenced by SKILL.md
```

Do not remove assets blindly; keep only what the skill text or scripts actually depend on.

- [ ] **Step 3: Re-run static capability checks on trimmed staging copies**

Expected: trimming should reduce footprint without causing:

```text
missing referenced file
missing primary skill document
missing capability structure
```

- [ ] **Step 4: Re-run mojibake checks on all retained translated text**

Expected: trimming does not leave behind:

```text
garbled headings
broken Chinese bullets
corrupted copied guidance text
half-trimmed references with unreadable prose
```

This re-check is required per skill before any `needs-trim` item can be reclassified as promotion-ready.

- [ ] **Step 5: Commit**

```bash
git add internal/core/skill/imports internal/core/skill docs/superpowers/plans
git commit -m "chore: trim oversized staged skill imports"
```

## Task 5: Keep `gstack` deferred and document why

**Files:**
- Modify: `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-18-skill-import-staging-inventory.md`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-18-skill-promotion-gate-plan.md`

- [ ] **Step 1: Mark `gstack` as deferred by policy**

Document that `gstack` is not a normal single governed skill in the current model because it behaves more like a suite or bundled tool ecosystem.

- [ ] **Step 2: Record the future route**

Document that any future `gstack` adoption must go through a separate design covering:

```text
bundle boundaries
sub-skill treatment
asset policy
runtime visibility semantics
```

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/plans/2026-05-18-skill-import-staging-inventory.md docs/superpowers/plans/2026-05-18-skill-promotion-gate-plan.md
git commit -m "docs: defer gstack promotion"
```

## Task 6: Close out matrix and baseline docs after each promoted batch

**Files:**
- Modify: `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-skill-governance-baseline.md`
- Modify: `D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md`

- [ ] **Step 1: Update runtime truth language**

Whenever a staged skill is promoted, update docs to say it is now governed through project-local `catalog/{codex,cc}`, not only through staging.

- [ ] **Step 2: Keep governance language bounded**

Do not claim:

```text
skill execution engine exists
per-skill business behavior is end-to-end accepted
```

Only claim:

```text
project-local discovery
localization audit
static capability verification
group isolation
```

- [ ] **Step 3: Commit**

```bash
git add docs/superpowers/plans/2026-05-17-skill-governance-baseline.md docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md
git commit -m "docs: sync promoted skill governance baseline"
```

## Regression Gate

Run after every promotion batch:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/runtime ./internal/router ./cmd/cli ./internal/core/skill -run "Skill|Governance|Discovery|SkillsList|Capability" -v
```

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
```

If desktop/frontend skill rendering is touched:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

## Acceptance Criteria

- Only `catalog/{codex,cc}` remains runtime-visible for copied skills.
- Staged imports remain a promotion workspace, not a second discovery truth.
- `codex` and `cc` remain isolated while `common` remains shared.
- Promoted skills keep `localizationChecked=true` and `capabilityVerified=true`.
- Every promoted translated skill passes a mojibake / garbled-text inspection.
- Every promoted translated skill passes a mandatory per-skill mojibake / garbled-text inspection across the retained project-local copy.
- `needs-trim` skills are not promoted until bulk assets are reduced to the smallest machine-usable footprint.
- `gstack` remains deferred unless a separate bundle-governance design is approved.

## Self-Review

- Spec coverage:
  - Covers classification, ready-vs-trim-vs-defer policy, promotion order, testing, documentation sync, and isolation preservation.
- Placeholder scan:
  - No `TBD`, `TODO`, or unspecified “handle later” implementation steps are left in the plan body.
- Type consistency:
  - Uses the existing terms `catalog`, `imports`, `localizationChecked`, `capabilityVerified`, and grouped `codex` / `cc` / `common` consistently.

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-05-18-skill-promotion-gate-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
