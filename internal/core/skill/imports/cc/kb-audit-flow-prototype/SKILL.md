---
name: kb-audit-flow-prototype
description: Use when the user wants to build, extend, or verify staged flow prototypes from `F:\codex\artifacts\kb-audit`, including HTML demo prototypes and business-flow alignment checks across procurement, labor, revenue-side, machinery, and approval flows.
---

# KB Audit Flow Prototype

## Overview

Use this skill for this project when the task is to turn the `kb-audit` document set into staged flow prototypes, extend those prototypes by business chain, or verify that the prototype still matches the documented business flow.

This skill covers the full working loop:

- rebuild context from `kb-audit` docs
- identify the target chain and current phase
- update the staged prototype
- validate prototype-versus-document alignment

Read `references/source-of-truth.md` before changing anything.

## When To Use

Use this skill when the user asks for any of these:

- continue the staged prototype for this project
- add labor, revenue-side, machinery, approval, or other documented chains
- check whether the current prototype is consistent with the business flow docs
- correct mismatches between menus, pages, node order, and documented business meaning
- build or revise HTML demo prototypes derived from `kb-audit`

Do not use this skill for:

- generic frontend work outside this project
- Axure-native implementation only
- unrelated landing pages or marketing sites

## Core Rules

### 1. Documents are the truth source

Do not start from the current prototype alone.

Always rebuild context from the relevant docs first:

- the business-flow summary doc for the target chain
- the site menu handbook or product module understanding doc
- the approval center design doc whenever approval is involved

### 2. Build by chain and by phase

Default expansion order:

1. home and navigation
2. procurement plus approval
3. labor
4. revenue-side
5. machinery

For each chain:

- preserve the full documented chain
- make the current phase interactive
- keep later or lighter nodes visible when needed
- explicitly mark what is fully interactive versus context-only

### 3. Keep prototype and menu aligned

If a node has a page, it should also have a discoverable entry point unless there is a deliberate reason not to expose it.

Check three layers together:

- business chain nodes
- left navigation or page entry points
- actual rendered page implementations

Do not leave "has page but no menu entry" or "has node label but no page" gaps unless the user explicitly wants placeholders.

### 4. Approval must behave like a control layer

When the target chain includes approval:

- show approval as the control layer, not as an isolated ornament
- keep approval actions conceptually attached to the business document
- make strong-binding rules visible, such as "cannot generate order before approval passes"

### 5. Validate after each substantial change

Always do these checks:

- syntax-check JS
- confirm relevant files exist
- compare the prototype chain text against the documented chain
- note any deliberate scope compression or incomplete interaction depth

## Output Defaults

Default prototype output directory:

- `F:\codex\artifacts\kb-audit\html-demo-prototype`

Default files:

- `index.html`
- `styles.css`
- `app.js`
- usage notes markdown file

Update the existing staged prototype in place unless the user explicitly asks for a separate variant.

## Reference

Use `references/source-of-truth.md` for the file map, phase baseline, and validation checklist.
