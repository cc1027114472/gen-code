# Browser Capability Strengthening Design

## Summary

This design strengthens browser capability in `gen-code` as a productized runtime surface instead of leaving it as a desktop-only workspace affordance plus external acceptance helper behavior.

The scope for this phase is intentionally constrained:

- support **local / controlled pages only**
- strengthen **runtime built-in browser tools**
- strengthen **desktop browser workspace UX**
- let **agent.run** use the same browser tools
- use **real page interaction**, not tab-state-only simulation

This phase does **not** attempt to become a general public-web browser automation platform. It does not cover arbitrary external websites, auth flows, file upload/download, iframe-heavy pages, multi-window browsing, or a broad network-transport compatibility story.

## Goals

1. Introduce a shared browser core used by both runtime and desktop.
2. Expose browser behavior through stable built-in runtime tools.
3. Upgrade desktop browser workspace from local tab state management into a UI surface backed by the same shared core.
4. Allow `agent.run` to perform minimal real browser workflows on controlled pages.
5. Add one stable canonical acceptance lane for browser interaction on a controlled local page.

## Non-Goals

- arbitrary public-web browsing compatibility
- OAuth, login/session import, cookie portability, extension integration
- iframe/shadow-root/platform-wide selector completeness
- visual AI clicking or image-based target detection
- browser resource marketplace/platform work
- broad browser-specific desktop redesign unrelated to the task flow

## Design Principles

- **One core, two surfaces**: runtime tools and desktop UX must use the same browser core.
- **Agent goes through tools**: agents do not call desktop browser methods directly.
- **Controlled-scope first**: local and explicitly controlled URLs only.
- **Stable evidence over fancy prose**: task summaries, tool-call summaries, artifact records, and acceptance assertions must be based on stable identity signals.
- **Minimal selector contract first**: start with reliable selectors before expanding expressiveness.

## Target Architecture

### 1. Shared Browser Core

Introduce a shared browser core responsible for all real browser/session/page interaction.

Responsibilities:

- browser/session lifecycle
- tab/page lifecycle
- opening and navigating tabs
- back/forward/reload
- current page state capture
- element lookup by approved selector types
- click and type actions
- screenshot generation
- text extraction
- controlled-URL allowlisting
- normalized browser errors

The shared core is the only layer allowed to directly talk to the underlying browser execution engine.

### 2. Runtime Browser Tools

Expose browser capability as built-in runtime tools that fit the existing `tasks create/run` and tool-call model.

The runtime surface is the contract layer used by:

- CLI
- task runner
- agent loop
- canonical acceptance

The runtime layer does not duplicate browser logic. It adapts requests/responses to the shared browser core.

### 3. Desktop Browser Workspace

Desktop remains the human-facing browser workspace, but it should stop being the source of truth for browser behavior.

Desktop responsibilities:

- show tabs and active tab
- show current URL/title/loading state
- show recent action result / error state
- provide manual browser actions for users
- reflect runtime and agent actions performed through the shared core

Desktop should not invent a second browser behavior model. It should render and invoke the same browser capabilities exposed through the shared core/runtime contract.

### 4. Agent Integration

`agent.run` uses browser capability through runtime tools only. Agent planning and execution should treat browser actions the same way it treats workspace actions today: through task/tool boundaries with stable summaries and observable child actions.

## Browser Tool Contract

This phase introduces two groups of browser tools.

### A. Tab / Navigation Tools

- `browser.state`
- `browser.open`
- `browser.navigate`
- `browser.back`
- `browser.forward`
- `browser.reload`
- `browser.close_tab`
- `browser.activate_tab`

#### `browser.state`

Purpose:

- return the current browser workspace snapshot

Minimum output shape should support:

- active tab id
- tab list
- per-tab:
  - `id`
  - `url`
  - `title`
  - `loading`

#### `browser.open`

Purpose:

- open a new tab for a controlled URL

Minimum input:

- `url`

#### `browser.navigate`

Purpose:

- navigate an existing tab to a controlled URL

Minimum input:

- `tabId`
- `url`

#### `browser.back` / `browser.forward` / `browser.reload`

Purpose:

- standard navigation controls for a tab

Minimum input:

- `tabId`

#### `browser.close_tab`

Purpose:

- close a tab by id

Minimum input:

- `tabId`

#### `browser.activate_tab`

Purpose:

- switch the active tab in the shared browser workspace

Minimum input:

- `tabId`

### B. Page Interaction Tools

- `browser.click`
- `browser.type`
- `browser.screenshot`
- `browser.extract`

#### `browser.click`

Purpose:

- click a real page element in a controlled page

Minimum input:

- `tabId`
- `selector`

#### `browser.type`

Purpose:

- type into a real page element

Minimum input:

- `tabId`
- `selector`
- `text`

#### `browser.screenshot`

Purpose:

- capture a real page screenshot from a tab

Minimum input:

- `tabId`

Expected output/result integration:

- stable result summary
- artifact record or equivalent durable path/reference

#### `browser.extract`

Purpose:

- extract stable text or page content from a real page element or whole page

Minimum input:

- `tabId`
- optional `selector`

Expected output/result integration:

- stable text/structured result
- summary suitable for task/result/tool-call assertions

## Selector Scope

This phase intentionally limits selector support to a minimal reliable set:

- `data-testid`
- `id`
- simple CSS selectors

Explicitly out of scope for this phase:

- XPath
- full accessibility-query coverage
- iframe traversal
- shadow-root generalization
- image-based or coordinate-based targeting

## Controlled URL Boundary

Browser execution is limited to local or controlled pages.

Allowed examples:

- `http://127.0.0.1:*`
- `http://localhost:*`
- desktop preview / project-local controlled URLs

The shared browser core must reject out-of-scope URLs with a stable permission/boundary error.

## Error Semantics

Browser tools should normalize to stable error categories, at minimum:

- tab not found
- URL not allowed
- page not ready
- selector not found
- element not interactable
- browser session unavailable
- screenshot failed
- extraction failed

These errors should propagate into:

- task status/result summary
- tool-call summary
- acceptance assertions

## Desktop UX Design

Desktop keeps the existing browser area structure where possible, but upgrades it with stable signals.

### Required UI Signals

Per active browser workspace:

- current active tab
- tab title
- current URL
- loading state
- latest action summary
- latest action error, if any

### Required Manual Actions

- open
- navigate
- back
- forward
- reload
- close tab
- activate tab

This phase should also surface the result of runtime/agent-driven browser actions so that a user can tell whether:

- a click happened
- text was typed
- extraction succeeded
- a screenshot was captured

### UX Constraint

Desktop must not create a separate browser-state explanation model. The state displayed in the UI should be derived from the same shared browser core/runtime-facing state used by tools.

## Agent Integration Design

`agent.run` should only use browser capability through runtime browser tools.

### Supported Browser Action Sequence for This Phase

- `browser.open` or `browser.navigate`
- `browser.state`
- `browser.click`
- `browser.type`
- `browser.extract`
- `browser.screenshot`

### Example Minimal Workflow

1. Open a controlled local page.
2. Check browser state and confirm active tab / URL.
3. Type into an input.
4. Click a control.
5. Extract resulting text.
6. Optionally capture a screenshot for evidence.
7. Respond.

### Explicit Agent Constraints

This phase should not allow agent plans to depend on:

- arbitrary external websites
- multi-window flows
- auth-heavy flows
- uploads/downloads
- iframe-heavy flows
- visual target guessing

## Testing Strategy

### 1. Browser Core Tests

Cover:

- session startup
- tab lifecycle
- URL allowlist enforcement
- selector lookup failure
- element interaction failure
- extraction success/failure
- screenshot success/failure
- session unavailable behavior

### 2. Runtime / Runner Tests

Cover:

- browser tools appear in discovery / tools list
- direct task execution for:
  - `browser.open`
  - `browser.navigate`
  - `browser.click`
  - `browser.type`
  - `browser.extract`
  - `browser.screenshot`
- result summaries remain stable
- screenshot output yields durable evidence
- agent child task chain uses browser tools in sequence

### 3. Desktop Tests

Cover:

- browser workspace state readability
- tab activation and URL updates
- recent action summary propagation
- runtime-triggered and desktop-triggered state consistency

### 4. Canonical Acceptance

Add one controlled-page browser lane that proves:

- page open/navigation
- type into input
- click a control
- extract a resulting value
- capture a screenshot
- browser task visibility in desktop UI

## Incremental Implementation Plan

### Phase A: Shared Core + Navigation Baseline

Build:

- shared browser core
- desktop integration with the core
- navigation tool group:
  - `browser.state`
  - `browser.open`
  - `browser.navigate`
  - `browser.back`
  - `browser.forward`
  - `browser.reload`
  - `browser.close_tab`
  - `browser.activate_tab`

### Phase B: Real Page Interaction Tools

Build:

- `browser.click`
- `browser.type`
- `browser.extract`
- `browser.screenshot`

Also add:

- task/result/tool-call summary integration
- artifact/evidence plumbing for screenshots

### Phase C: Agent + Acceptance + Closeout

Build:

- minimal `agent.run` browser action planning/execution
- canonical browser acceptance lane
- checklist / matrix / documentation closeout

## Acceptance Criteria

This phase is complete when all of the following are true:

1. Browser capability is available as built-in runtime tools for controlled local pages.
2. Desktop and runtime both use the same shared browser core.
3. `agent.run` can complete a minimal controlled-page browser workflow through browser tools.
4. Canonical acceptance explicitly proves one browser interaction lane end-to-end.
5. Browser errors, task summaries, and evidence remain stable enough for regression tests.

## Risks And Mitigations

### Risk: Desktop and runtime drift again

Mitigation:

- keep all real interaction logic in the shared core
- keep desktop as UI, not truth source

### Risk: Browser scope explodes into a general web platform

Mitigation:

- restrict URLs to controlled local pages
- keep selector support intentionally narrow
- defer uploads/auth/iframes/external web

### Risk: Acceptance becomes flaky

Mitigation:

- use controlled local pages only
- assert stable identity signals
- avoid brittle prose checks
- avoid uncontrolled timing assumptions

### Risk: Agent workflows become underconstrained

Mitigation:

- gate browser usage through explicit runtime tools
- add sequence-based agent verification for minimal workflows

## Recommendation

Proceed with the shared-core design and implement in three phases:

- Phase A: core + navigation
- Phase B: real page interaction
- Phase C: agent + canonical acceptance closeout

This gives `gen-code` a real browser capability baseline without prematurely turning it into a broad browser automation platform.
