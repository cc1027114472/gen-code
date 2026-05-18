# Browser Capability Strengthening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a shared browser core, expose controlled local browser behavior as built-in runtime tools, and close the desktop plus agent acceptance lane for real page interaction.

**Architecture:** This plan moves browser behavior out of the desktop-only `browserWorkspace` mock and into a shared Go browser core under `internal/core/browser`. Runtime tasks, desktop Wails methods, and `agent.run` all call the same core, while the desktop frontend stays a thin state-and-controls surface. Canonical verification stays on the existing desktop live-refresh lane and proves one controlled local browser workflow end to end.

**Tech Stack:** Go, Wails, React, TypeScript, Python Playwright, PowerShell, chromedp or an equivalent Go CDP driver for controlled local pages only.

---

## Scope

This plan implements the approved browser design in three phases:

- Phase A: shared core plus navigation baseline
- Phase B: real page interaction plus screenshot and extract evidence
- Phase C: `agent.run`, desktop acceptance, and docs closeout

This plan does not include:

- arbitrary public-web browsing
- auth, cookie import, OAuth, uploads, downloads, or iframe-heavy flows
- visual clicking or image-based targeting
- a new browser-specific top-level runtime API
- a desktop information architecture redesign

## File Map

### Shared browser core

- Create: [internal/core/browser/types.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/types.go)
  - shared request, response, selector, tab, and snapshot types used by runtime and desktop
- Create: [internal/core/browser/errors.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/errors.go)
  - stable browser error categories such as `tab not found`, `url not allowed`, and `selector not found`
- Create: [internal/core/browser/core.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/core.go)
  - browser core interface plus session lifecycle, allowlist checks, and state normalization
- Create: [internal/core/browser/chromedp_driver.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/chromedp_driver.go)
  - concrete controlled-page implementation that opens tabs, navigates, clicks, types, extracts, and captures screenshots
- Create: [internal/core/browser/core_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/core_test.go)
  - unit tests for tab lifecycle, allowlist enforcement, selectors, extract, and screenshot failures

### Runtime task and tool surface

- Modify: [internal/core/runner/runner.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/runner.go)
  - add browser task kinds, input parsing, result summaries, tool-call summaries, and screenshot artifact writes
- Modify: [internal/core/runtime/status.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status.go)
  - expose browser built-in tools in `Tools()` and keep runtime discovery aligned with the shared core rollout
- Modify: [internal/appserver/runtimecontract/contract.go](/D:/GOWorks/gen-code-heji/gen-code/internal/appserver/runtimecontract/contract.go)
  - add only the minimum browser snapshot or artifact contract fields if runtime and desktop need shared JSON shape
- Modify: [cmd/cli/main.go](/D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main.go)
  - make `tools list` and generic `tasks create/run` examples show browser tool usage for controlled local pages
- Test: [internal/core/runtime/status_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status_test.go)
- Test: [cmd/cli/main_test.go](/D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main_test.go)

### Desktop browser surface

- Modify: [desktop/app.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
  - replace `browserWorkspace` truth with shared core wiring and expose Wails methods for the full browser tool set
- Modify: [desktop/app_test.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)
  - extend browser workspace tests from tab-state smoke checks to real shared-core assertions
- Modify: [desktop/frontend/src/runtimeBridge.ts](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/runtimeBridge.ts)
  - add bridge methods and fallback shims for click, type, extract, and screenshot
- Modify: [desktop/frontend/src/App.tsx](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx)
  - show stable browser state, latest action summary, latest error summary, and minimal manual controls

### Agent and acceptance

- Modify: [internal/core/runner/agent_loop.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/agent_loop.go)
  - add browser action types, browser child-task creation, and minimal sequence validation
- Modify: [internal/core/runner/agent_loop_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/agent_loop_test.go)
  - lock browser action planning and child-task sequencing
- Modify: [scripts/verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
  - add one controlled local browser lane with open, type, click, extract, and screenshot checks
- Modify: [scripts/run-desktop-live-refresh-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
  - keep wrapper output stable when browser acceptance artifacts are produced

### Docs and closeout

- Modify: [docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md)
  - mark browser built-ins and browser acceptance evidence to the correct verified state
- Modify: [docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md)
  - record the canonical browser lane and controlled-page boundary

## Task 1: Introduce The Shared Browser Core

**Files:**
- Create: [internal/core/browser/types.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/types.go)
- Create: [internal/core/browser/errors.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/errors.go)
- Create: [internal/core/browser/core.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/core.go)
- Create: [internal/core/browser/chromedp_driver.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/chromedp_driver.go)
- Test: [internal/core/browser/core_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/core_test.go)

- [ ] **Step 1: Write the failing browser-core tests**

Add tests that prove the new package owns real browser behavior:

```go
func TestCoreRejectsDisallowedURL(t *testing.T) {
	core := newTestCore(t)
	_, err := core.Open(context.Background(), OpenRequest{URL: "https://example.com"})
	require.ErrorIs(t, err, ErrURLNotAllowed)
}

func TestCoreOpenNavigateAndState(t *testing.T) {
	serverURL := startBrowserFixtureServer(t)
	core := newTestCore(t)

	state, err := core.Open(context.Background(), OpenRequest{URL: serverURL})
	require.NoError(t, err)
	require.NotEmpty(t, state.ActiveTabID)
	require.Len(t, state.Tabs, 1)
	require.Equal(t, serverURL, state.Tabs[0].URL)
}
```

- [ ] **Step 2: Run the package tests to verify they fail**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
go test ./internal/core/browser -run "Core|Browser" -v
```

Expected:

- build or test failure because the `internal/core/browser` package and its exported error surface do not exist yet

- [ ] **Step 3: Create the shared browser types and errors**

Create the minimal shared shapes first so runtime and desktop can agree on names:

```go
type Snapshot struct {
	ActiveTabID string `json:"activeTabId"`
	Tabs        []Tab  `json:"tabs"`
}

type Tab struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	Loading      bool   `json:"loading"`
	CanGoBack    bool   `json:"canGoBack"`
	CanGoForward bool   `json:"canGoForward"`
}

var (
	ErrTabNotFound            = errors.New("browser: tab not found")
	ErrURLNotAllowed          = errors.New("browser: url not allowed")
	ErrSelectorNotFound       = errors.New("browser: selector not found")
	ErrElementNotInteractable = errors.New("browser: element not interactable")
)
```

- [ ] **Step 4: Implement the core interface and controlled URL policy**

Add a core interface and make allowlisting explicit:

```go
type Core interface {
	State(context.Context) (Snapshot, error)
	Open(context.Context, OpenRequest) (Snapshot, error)
	Navigate(context.Context, NavigateRequest) (Snapshot, error)
	Back(context.Context, TabRequest) (Snapshot, error)
	Forward(context.Context, TabRequest) (Snapshot, error)
	Reload(context.Context, TabRequest) (Snapshot, error)
	CloseTab(context.Context, TabRequest) (Snapshot, error)
	ActivateTab(context.Context, TabRequest) (Snapshot, error)
}

func allowedURL(raw string) bool {
	return strings.HasPrefix(raw, "http://127.0.0.1:") || strings.HasPrefix(raw, "http://localhost:")
}
```

- [ ] **Step 5: Implement the concrete CDP-backed driver**

Keep the first driver intentionally narrow:

```go
type Driver struct {
	allocCtx context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
	tabs     map[string]*tabSession
	activeID string
}

func NewDriver() *Driver {
	return &Driver{tabs: map[string]*tabSession{}}
}
```

Implementation requirements:

- one browser allocation context shared across controlled local tabs
- one stable in-memory tab registry
- title, URL, and loading state normalized into `Snapshot`
- no public-web URLs accepted

- [ ] **Step 6: Re-run the browser-core tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
go test ./internal/core/browser -run "Core|Browser" -v
```

Expected:

- PASS for open, navigate, state, and allowlist tests

- [ ] **Step 7: Commit**

```bash
git add internal/core/browser/types.go internal/core/browser/errors.go internal/core/browser/core.go internal/core/browser/chromedp_driver.go internal/core/browser/core_test.go go.mod go.sum
git commit -m "feat: add shared browser core"
```

## Task 2: Expose Browser Navigation As Built-In Runtime Tools

**Files:**
- Modify: [internal/core/runner/runner.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/runner.go)
- Modify: [internal/core/runtime/status.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status.go)
- Modify: [cmd/cli/main.go](/D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main.go)
- Test: [internal/core/runtime/status_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runtime/status_test.go)
- Test: [cmd/cli/main_test.go](/D:/GOWorks/gen-code-heji/gen-code/cmd/cli/main_test.go)
- Test: [internal/core/runner/runner_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/runner_test.go)

- [ ] **Step 1: Write failing runner and discovery tests for browser navigation tools**

Add tests for:

```go
func TestToolsIncludesBrowserNavigationKinds(t *testing.T) {
	tools, err := service.Tools(context.Background())
	require.NoError(t, err)
	require.Contains(t, collectToolIDs(tools), "browser.open")
	require.Contains(t, collectToolIDs(tools), "browser.navigate")
	require.Contains(t, collectToolIDs(tools), "browser.state")
}

func TestRunTaskBrowserOpenCompletes(t *testing.T) {
	task := createTask(t, runner.KindBrowserOpen, `{"url":"http://127.0.0.1:40123/"}`)
	result, err := rt.RunTask(context.Background(), threadID, task.ID)
	require.NoError(t, err)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "browser tab opened")
}
```

- [ ] **Step 2: Run targeted tests to verify they fail**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
go test ./internal/core/runtime ./internal/core/runner ./cmd/cli -run "Browser|Tools|Task" -v
```

Expected:

- failures because browser built-in kinds are not registered and runner does not understand them

- [ ] **Step 3: Add browser task kind constants and input parsing**

Extend the runner constants and parsing helpers:

```go
const (
	KindBrowserState       = "browser.state"
	KindBrowserOpen        = "browser.open"
	KindBrowserNavigate    = "browser.navigate"
	KindBrowserBack        = "browser.back"
	KindBrowserForward     = "browser.forward"
	KindBrowserReload      = "browser.reload"
	KindBrowserCloseTab    = "browser.close_tab"
	KindBrowserActivateTab = "browser.activate_tab"
)
```

Add minimal request decoding:

```go
type browserOpenInput struct {
	URL string `json:"url"`
}

type browserTabInput struct {
	TabID string `json:"tabId"`
}
```

- [ ] **Step 4: Route navigation kinds through the shared core**

Inside `RunTask`, add one switch branch per navigation kind and keep summaries stable:

```go
case KindBrowserOpen:
	state, err := r.browser.Open(ctx, browser.OpenRequest{URL: input.URL})
	if err != nil {
		return failTask(err)
	}
	return completeTask(fmt.Sprintf("browser tab opened: %s", state.ActiveTabID))
```

Stable summary rules:

- `browser.state`: `browser state captured`
- `browser.open`: `browser tab opened: <tabId>`
- `browser.navigate`: `browser tab navigated: <tabId>`
- `browser.back`: `browser tab went back: <tabId>`
- `browser.forward`: `browser tab went forward: <tabId>`
- `browser.reload`: `browser tab reloaded: <tabId>`
- `browser.close_tab`: `browser tab closed: <tabId>`
- `browser.activate_tab`: `browser tab activated: <tabId>`

- [ ] **Step 5: Register the browser navigation tools in runtime discovery and CLI output**

Add browser descriptors to the runtime tool list with the same IDs used by the runner:

```go
runtimecontract.Tool{
	ID:          "browser.open",
	Name:        "Browser Open",
	Description: "Open a controlled local URL in the shared browser workspace",
	Permission:  "read-only",
	Source:      "runtime",
	Kind:        "browser.open",
	ReadOnly:    true,
	Executable:  true,
}
```

CLI expectations:

- `gen-code tools list` shows all browser navigation tools
- `gen-code tasks create --kind=browser.open --input='{"url":"http://127.0.0.1:5174/"}'` remains the primary CLI path

- [ ] **Step 6: Re-run the targeted runtime, runner, and CLI tests**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
go test ./internal/core/runtime ./internal/core/runner ./cmd/cli -run "Browser|Tools|Task" -v
```

Expected:

- PASS for browser tool discovery and navigation task execution

- [ ] **Step 7: Commit**

```bash
git add internal/core/runner/runner.go internal/core/runner/runner_test.go internal/core/runtime/status.go internal/core/runtime/status_test.go cmd/cli/main.go cmd/cli/main_test.go
git commit -m "feat: expose browser navigation tools"
```

## Task 3: Replace Desktop Mock State With The Shared Core

**Files:**
- Modify: [desktop/app.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
- Modify: [desktop/app_test.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)
- Modify: [desktop/frontend/src/runtimeBridge.ts](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/runtimeBridge.ts)
- Modify: [desktop/frontend/src/App.tsx](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx)

- [ ] **Step 1: Write failing desktop tests for shared-core-backed browser state**

Replace pure tab-mutation assertions with shared-core expectations:

```go
func TestBrowserWorkspaceFlowUsesSharedCore(t *testing.T) {
	app := NewApp()
	state := app.BrowserOpen("http://127.0.0.1:40123/")
	require.True(t, state.IsOpen)
	require.NotEmpty(t, state.ActiveTabID)
	require.Equal(t, "http://127.0.0.1:40123/", state.Tabs[0].URL)
}
```

Add one negative-path test:

```go
func TestBrowserOpenRejectsExternalURL(t *testing.T) {
	app := NewApp()
	state := app.BrowserOpen("https://example.com")
	require.Contains(t, latestBrowserSummary(state), "url not allowed")
}
```

- [ ] **Step 2: Run desktop tests to verify they fail**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
go test ./... -run "BrowserWorkspace|BrowserOpen" -v
```

Expected:

- failures because desktop still uses the local `browserWorkspace` mock instead of the shared core and does not surface stable error summaries

- [ ] **Step 3: Replace `browserWorkspace` storage with a shared-core adapter**

In [desktop/app.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app.go), replace the mutable mock with a thin adapter:

```go
type browserAdapter struct {
	core browser.Core
}

func (a *App) BrowserState() BrowserWorkspaceState {
	snapshot, err := a.browser.State(a.ctx)
	return mapBrowserSnapshot(snapshot, err)
}
```

Rules for the migration:

- keep existing Wails method names for backward compatibility
- do not keep a second tab registry in desktop memory
- map shared-core errors into stable summary and error fields once

- [ ] **Step 4: Extend the bridge and UI for stable browser status**

Add bridge functions for the new interaction methods and keep fallback behavior local-only:

```ts
export async function BrowserClick(tabId: string, selector: string): Promise<BrowserWorkspaceState> {
  return WailsBrowserClick(tabId, selector);
}
```

Update [App.tsx](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx) to show:

- active tab title
- current URL
- loading state
- latest action summary
- latest action error

and minimal manual controls for:

- open
- navigate
- back
- forward
- reload
- close tab
- activate tab

- [ ] **Step 5: Re-run desktop tests and the frontend build**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
go test ./...
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

Expected:

- desktop Go tests PASS
- frontend build PASS with no bridge type drift

- [ ] **Step 6: Commit**

```bash
git add desktop/app.go desktop/app_test.go desktop/frontend/src/runtimeBridge.ts desktop/frontend/src/App.tsx
git commit -m "feat: connect desktop browser workspace to shared core"
```

## Task 4: Add Real Page Interaction Tools And Screenshot Evidence

**Files:**
- Modify: [internal/core/browser/types.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/types.go)
- Modify: [internal/core/browser/core.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/core.go)
- Modify: [internal/core/browser/chromedp_driver.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/chromedp_driver.go)
- Modify: [internal/core/runner/runner.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/runner.go)
- Modify: [desktop/app.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app.go)
- Modify: [desktop/frontend/src/runtimeBridge.ts](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/runtimeBridge.ts)
- Modify: [desktop/frontend/src/App.tsx](/D:/GOWorks/gen-code-heji/gen-code/desktop/frontend/src/App.tsx)
- Test: [internal/core/browser/core_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/browser/core_test.go)
- Test: [internal/core/runner/runner_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/runner_test.go)
- Test: [desktop/app_test.go](/D:/GOWorks/gen-code-heji/gen-code/desktop/app_test.go)

- [ ] **Step 1: Write failing tests for click, type, extract, and screenshot**

Add core tests:

```go
func TestCoreTypeClickAndExtract(t *testing.T) {
	serverURL := startBrowserFixtureServer(t)
	core := newTestCore(t)
	state, _ := core.Open(context.Background(), OpenRequest{URL: serverURL})

	_, err := core.Type(context.Background(), TypeRequest{TabID: state.ActiveTabID, Selector: "[data-testid='name']", Text: "browser"})
	require.NoError(t, err)

	result, err := core.Extract(context.Background(), ExtractRequest{TabID: state.ActiveTabID, Selector: "[data-testid='result']"})
	require.NoError(t, err)
	require.Equal(t, "browser", result.Text)
}
```

Add runner tests:

```go
func TestRunTaskBrowserScreenshotCreatesArtifact(t *testing.T) {
	result := runBrowserTask(t, runner.KindBrowserScreenshot, `{"tabId":"browser-tab-1"}`)
	require.Equal(t, "completed", result.Status)
	require.Contains(t, result.ResultSummary, "browser screenshot captured")
	require.Len(t, artifactsForThread(t), 1)
}
```

- [ ] **Step 2: Run the targeted tests to verify they fail**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
go test ./internal/core/browser ./internal/core/runner -run "Click|Type|Extract|Screenshot" -v
```

Expected:

- failures because the interaction requests and screenshot artifact plumbing are not implemented yet

- [ ] **Step 3: Add interaction request types and driver methods**

Extend the shared package with the minimum input shapes:

```go
type ClickRequest struct {
	TabID    string `json:"tabId"`
	Selector string `json:"selector"`
}

type TypeRequest struct {
	TabID    string `json:"tabId"`
	Selector string `json:"selector"`
	Text     string `json:"text"`
}

type ExtractRequest struct {
	TabID    string `json:"tabId"`
	Selector string `json:"selector,omitempty"`
}
```

Driver requirements:

- selectors allowed: `data-testid`, `id`, and simple CSS
- errors normalized to the shared error values
- screenshot returns a deterministic file name and byte payload

- [ ] **Step 4: Add browser interaction task kinds and artifact integration**

Extend the runner with:

```go
const (
	KindBrowserClick      = "browser.click"
	KindBrowserType       = "browser.type"
	KindBrowserExtract    = "browser.extract"
	KindBrowserScreenshot = "browser.screenshot"
)
```

Stable summary rules:

- `browser.click`: `browser click executed: <tabId>`
- `browser.type`: `browser type executed: <tabId>`
- `browser.extract`: `browser extract completed: <tabId>`
- `browser.screenshot`: `browser screenshot captured: <path>`

When screenshot succeeds:

- persist one artifact entry with `kind=browser.screenshot`
- keep result summary stable enough for CLI and acceptance assertions

- [ ] **Step 5: Surface the new actions in desktop Wails and React**

Add Wails methods:

```go
func (a *App) BrowserClick(tabID string, selector string) BrowserWorkspaceState
func (a *App) BrowserType(tabID string, selector string, text string) BrowserWorkspaceState
func (a *App) BrowserExtract(tabID string, selector string) BrowserWorkspaceState
func (a *App) BrowserScreenshot(tabID string) BrowserWorkspaceState
```

UI additions must stay minimal:

- selector input
- text input for type
- action buttons for click, type, extract, screenshot
- latest result summary and latest error summary visible beside the active tab

- [ ] **Step 6: Re-run browser, runner, desktop, and frontend checks**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
go test ./internal/core/browser ./internal/core/runner
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
go test ./...
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

Expected:

- PASS for interaction tools, screenshot evidence, and desktop wiring

- [ ] **Step 7: Commit**

```bash
git add internal/core/browser/types.go internal/core/browser/core.go internal/core/browser/chromedp_driver.go internal/core/browser/core_test.go internal/core/runner/runner.go internal/core/runner/runner_test.go desktop/app.go desktop/app_test.go desktop/frontend/src/runtimeBridge.ts desktop/frontend/src/App.tsx
git commit -m "feat: add browser interaction tools and screenshot evidence"
```

## Task 5: Add Agent Browser Actions And Canonical Acceptance

**Files:**
- Modify: [internal/core/runner/agent_loop.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/agent_loop.go)
- Modify: [internal/core/runner/agent_loop_test.go](/D:/GOWorks/gen-code-heji/gen-code/internal/core/runner/agent_loop_test.go)
- Modify: [scripts/verify-desktop-live-refresh.py](/D:/GOWorks/gen-code-heji/gen-code/scripts/verify-desktop-live-refresh.py)
- Modify: [scripts/run-desktop-live-refresh-check.ps1](/D:/GOWorks/gen-code-heji/gen-code/scripts/run-desktop-live-refresh-check.ps1)
- Modify: [docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md)
- Modify: [docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md](/D:/GOWorks/gen-code-heji/gen-code/docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md)

- [ ] **Step 1: Write failing agent and acceptance tests**

Add agent loop tests that prove browser actions become child tasks:

```go
func TestAgentPlanBrowserSequenceCreatesBrowserChildTasks(t *testing.T) {
	action := AgentAction{Type: "browser_open", URL: "http://127.0.0.1:40123/"}
	err := validateAgentActionSequence(state, action)
	require.NoError(t, err)
}
```

Add verifier expectations for one browser lane result block:

```python
assert browser_result["kind"] == "browser.screenshot"
assert browser_result["resultSummary"].startswith("browser screenshot captured")
assert browser_result["toolCallsVisible"] is True
```

- [ ] **Step 2: Run the targeted tests to verify they fail**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
go test ./internal/core/runner -run "Agent.*Browser|Browser.*Agent" -v
python .\scripts\verify-desktop-live-refresh.py
```

Expected:

- agent tests fail because browser actions are unknown
- verifier fails because no canonical browser scenario is present yet

- [ ] **Step 3: Add minimal browser action support to `agent.run`**

Extend `AgentAction` only as far as needed:

```go
type AgentAction struct {
	Type     string `json:"type"`
	URL      string `json:"url,omitempty"`
	TabID    string `json:"tabId,omitempty"`
	Selector string `json:"selector,omitempty"`
	Text     string `json:"text,omitempty"`
}
```

Supported browser action types for this phase:

- `browser_open`
- `browser_state`
- `browser_click`
- `browser_type`
- `browser_extract`
- `browser_screenshot`
- `respond`

Each action must create exactly one child task using the built-in browser kinds from Tasks 2 and 4.

- [ ] **Step 4: Add one controlled-page canonical browser acceptance lane**

Re-use the existing verifier structure and add a dedicated scenario after the direct read tools and before MCP or write lanes.

Required flow:

1. open a stable local fixture page
2. type into an input with `data-testid`
3. click a control with `data-testid`
4. extract a stable result node
5. capture a screenshot
6. assert desktop UI visibility for task card, tool call, and result summary

The acceptance JSON output must include:

```json
{
  "browserExecutionResult": {
    "taskIds": [],
    "toolKinds": [],
    "activeTabId": "",
    "extractSummary": "",
    "screenshotArtifactPath": "",
    "toolCallsVisible": true
  }
}
```

- [ ] **Step 5: Update the capability matrix and runtime checklist**

Document only the verified boundary:

- browser built-ins are verified for controlled local pages
- desktop and runtime share one browser core
- `agent.run` uses browser tools through child tasks
- canonical acceptance covers one controlled local browser lane

Do not claim:

- arbitrary public web
- auth flows
- iframe-heavy flows

- [ ] **Step 6: Run the full verification suite**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser ./internal/core/runtime ./internal/core/runner ./cmd/cli
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected:

- all targeted Go tests PASS
- desktop frontend build PASS
- canonical desktop live-refresh acceptance PASS with an explicit browser result section

- [ ] **Step 7: Commit**

```bash
git add internal/core/runner/agent_loop.go internal/core/runner/agent_loop_test.go scripts/verify-desktop-live-refresh.py scripts/run-desktop-live-refresh-check.ps1 docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md
git commit -m "feat: verify browser capability end to end"
```

## Task 6: Final Regression And Closeout

**Files:**
- Verify only, no new source files

- [ ] **Step 1: Run the browser-focused Go suite from the repository root**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
$env:GOTOOLCHAIN='auto'
go test ./internal/core/browser ./internal/core/runtime ./internal/core/runner ./cmd/cli
```

Expected:

- PASS with browser core, runtime tools, and agent coverage all green

- [ ] **Step 2: Run the desktop Go suite**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop
$env:GOTOOLCHAIN='auto'
go test ./...
```

Expected:

- PASS with shared-core-backed browser desktop coverage

- [ ] **Step 3: Run the desktop frontend build**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code\desktop\frontend
npm run build
```

Expected:

- PASS with no runtime bridge type mismatch

- [ ] **Step 4: Run the canonical acceptance wrapper**

Run:

```powershell
Set-Location D:\GOWorks\gen-code-heji\gen-code
powershell -ExecutionPolicy Bypass -File .\scripts\run-desktop-live-refresh-check.ps1
```

Expected:

- PASS with the browser lane recorded beside the existing remote canonical acceptance evidence

- [ ] **Step 5: Create the final integration commit**

```bash
git status --short
git add internal/core/browser internal/core/runner internal/core/runtime cmd/cli desktop/app.go desktop/app_test.go desktop/frontend/src/runtimeBridge.ts desktop/frontend/src/App.tsx scripts/verify-desktop-live-refresh.py scripts/run-desktop-live-refresh-check.ps1 docs/superpowers/plans/2026-05-17-capability-matrix-and-tool-coverage.md docs/superpowers/plans/2026-05-17-runtime-entry-checklist.md
git commit -m "feat: strengthen controlled browser capability"
```

## Self-Review

### Spec coverage

- Shared browser core: covered by Task 1 and Task 3
- Runtime built-in browser tools: covered by Task 2 and Task 4
- Desktop browser workspace UX: covered by Task 3 and Task 4
- `agent.run` browser calling chain: covered by Task 5
- Canonical controlled-page acceptance lane: covered by Task 5 and Task 6
- Controlled URL boundary and stable error semantics: covered by Task 1 and Task 4

### Placeholder scan

The plan intentionally avoids `TBD`, `TODO`, and "similar to Task N" shorthand. Each task names exact files, exact commands, and the stable summary strings or interfaces that must exist after implementation.

### Type consistency

The same browser tool IDs are used throughout the plan:

- `browser.state`
- `browser.open`
- `browser.navigate`
- `browser.back`
- `browser.forward`
- `browser.reload`
- `browser.close_tab`
- `browser.activate_tab`
- `browser.click`
- `browser.type`
- `browser.extract`
- `browser.screenshot`

The same agent action names are used throughout the plan:

- `browser_open`
- `browser_state`
- `browser_click`
- `browser_type`
- `browser_extract`
- `browser_screenshot`

## Execution Choice

After saving this plan, execute it with one of these modes:

1. Subagent-Driven
2. Inline Execution
