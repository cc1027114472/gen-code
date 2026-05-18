package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

const browserActionTimeout = 20 * time.Second

type tabSession struct {
	id                string
	ctx               context.Context
	cancel            context.CancelFunc
	url               string
	title             string
	loading           bool
	canGoBack         bool
	canGoForward      bool
	history           []string
	historyIndex      int
	bootstrappedHosts map[string]bool
}

// Driver is the default CDP-backed controlled local browser core.
type Driver struct {
	rootCtx     context.Context
	allocCtx    context.Context
	allocCancel context.CancelFunc

	mu                  sync.Mutex
	tabs                map[string]*tabSession
	activeID            string
	nextTabNum          int
	latestActionSummary string
	latestActionError   string
	policy              Policy
	applySessionCookies func(context.Context, *tabSession, *url.URL, SessionProfile) error
}

// NewDriver constructs a shared controlled browser core.
func NewDriver() *Driver {
	return newDriverWithPolicy(defaultPolicy())
}

func newDriverWithPolicy(policy Policy) *Driver {
	driver := &Driver{
		rootCtx:    context.Background(),
		tabs:       map[string]*tabSession{},
		nextTabNum: 1,
		activeID:   "",
		policy:     policy,
	}
	driver.applySessionCookies = driver.applySessionCookiesForProfile
	return driver
}

func (d *Driver) State(_ context.Context) (Snapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.snapshotLocked(), nil
}

func (d *Driver) Open(ctx context.Context, request OpenRequest) (Snapshot, error) {
	normalizedURL, err := normalizeURLWithPolicy(request.URL, d.policy)
	if err != nil {
		return d.fail(err)
	}
	tab, err := d.newTab(ctx)
	if err != nil {
		return d.fail(err)
	}
	if err := d.navigateTab(ctx, tab, normalizedURL, true); err != nil {
		d.mu.Lock()
		d.closeTabLocked(tab.id)
		d.mu.Unlock()
		return d.fail(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser tab opened: %s", tab.id)
	d.latestActionError = ""
	return d.snapshotLocked(), nil
}

func (d *Driver) Navigate(ctx context.Context, request NavigateRequest) (Snapshot, error) {
	normalizedURL, err := normalizeURLWithPolicy(request.URL, d.policy)
	if err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	tab, ok := d.tabs[strings.TrimSpace(request.TabID)]
	if !ok {
		d.mu.Unlock()
		return d.fail(ErrTabNotFound)
	}
	d.mu.Unlock()
	if err := d.navigateTab(ctx, tab, normalizedURL, true); err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser tab navigated: %s", tab.id)
	d.latestActionError = ""
	return d.snapshotLocked(), nil
}

func (d *Driver) Back(ctx context.Context, request TabRequest) (Snapshot, error) {
	tab, err := d.lookupTab(request.TabID)
	if err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	if tab.historyIndex <= 0 {
		d.activeID = tab.id
		d.latestActionSummary = fmt.Sprintf("browser tab went back: %s", tab.id)
		d.latestActionError = ""
		snapshot := d.snapshotLocked()
		d.mu.Unlock()
		return snapshot, nil
	}
	targetURL := tab.history[tab.historyIndex-1]
	d.mu.Unlock()
	if err := d.navigateTab(ctx, tab, targetURL, false); err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	tab.historyIndex--
	tab.canGoBack = tab.historyIndex > 0
	tab.canGoForward = tab.historyIndex < len(tab.history)-1
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser tab went back: %s", tab.id)
	d.latestActionError = ""
	snapshot := d.snapshotLocked()
	d.mu.Unlock()
	return snapshot, nil
}

func (d *Driver) Forward(ctx context.Context, request TabRequest) (Snapshot, error) {
	tab, err := d.lookupTab(request.TabID)
	if err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	if tab.historyIndex >= len(tab.history)-1 {
		d.activeID = tab.id
		d.latestActionSummary = fmt.Sprintf("browser tab went forward: %s", tab.id)
		d.latestActionError = ""
		snapshot := d.snapshotLocked()
		d.mu.Unlock()
		return snapshot, nil
	}
	targetURL := tab.history[tab.historyIndex+1]
	d.mu.Unlock()
	if err := d.navigateTab(ctx, tab, targetURL, false); err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	tab.historyIndex++
	tab.canGoBack = tab.historyIndex > 0
	tab.canGoForward = tab.historyIndex < len(tab.history)-1
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser tab went forward: %s", tab.id)
	d.latestActionError = ""
	snapshot := d.snapshotLocked()
	d.mu.Unlock()
	return snapshot, nil
}

func (d *Driver) Reload(ctx context.Context, request TabRequest) (Snapshot, error) {
	tab, err := d.lookupTab(request.TabID)
	if err != nil {
		return d.fail(err)
	}
	if err := d.runOnTab(ctx, tab, chromedp.Reload()); err != nil {
		return d.fail(fmt.Errorf("%w: %v", ErrSessionUnavailable, err))
	}
	if err := d.syncTab(ctx, tab); err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser tab reloaded: %s", tab.id)
	d.latestActionError = ""
	return d.snapshotLocked(), nil
}

func (d *Driver) CloseTab(_ context.Context, request TabRequest) (Snapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	tabID := strings.TrimSpace(request.TabID)
	if tabID == "" {
		tabID = d.activeID
	}
	if _, ok := d.tabs[tabID]; !ok {
		return d.failLocked(ErrTabNotFound)
	}
	d.closeTabLocked(tabID)
	d.latestActionSummary = fmt.Sprintf("browser tab closed: %s", tabID)
	d.latestActionError = ""
	return d.snapshotLocked(), nil
}

func (d *Driver) ActivateTab(_ context.Context, request TabRequest) (Snapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	tabID := strings.TrimSpace(request.TabID)
	if _, ok := d.tabs[tabID]; !ok {
		return d.failLocked(ErrTabNotFound)
	}
	d.activeID = tabID
	d.latestActionSummary = fmt.Sprintf("browser tab activated: %s", tabID)
	d.latestActionError = ""
	return d.snapshotLocked(), nil
}

func (d *Driver) Click(ctx context.Context, request ClickRequest) (Snapshot, error) {
	tab, selector, err := d.lookupSelectorRequest(request.TabID, request.Selector)
	if err != nil {
		return d.fail(err)
	}
	if err := d.runOnTab(ctx, tab,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Click(selector, chromedp.ByQuery),
	); err != nil {
		if isSelectorError(err) {
			return d.fail(fmt.Errorf("%w: %s", ErrSelectorNotFound, selector))
		}
		return d.fail(fmt.Errorf("%w: %v", ErrElementNotInteractable, err))
	}
	if err := d.syncTab(ctx, tab); err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser click executed: %s", tab.id)
	d.latestActionError = ""
	return d.snapshotLocked(), nil
}

func (d *Driver) Type(ctx context.Context, request TypeRequest) (Snapshot, error) {
	tab, selector, err := d.lookupSelectorRequest(request.TabID, request.Selector)
	if err != nil {
		return d.fail(err)
	}
	if strings.TrimSpace(request.Text) == "" {
		return d.fail(fmt.Errorf("%w: empty text", ErrElementNotInteractable))
	}
	quotedText, marshalErr := json.Marshal(request.Text)
	if marshalErr != nil {
		return d.fail(fmt.Errorf("%w: %v", ErrElementNotInteractable, marshalErr))
	}
	script := fmt.Sprintf(`(() => {
		const element = document.querySelector(%q);
		if (!element) {
			return "missing";
		}
		element.focus();
		element.value = %s;
		element.dispatchEvent(new Event("input", { bubbles: true }));
		element.dispatchEvent(new Event("change", { bubbles: true }));
		return "ok";
	})()`, selector, string(quotedText))
	var typeResult string
	if err := d.runOnTab(ctx, tab,
		chromedp.WaitVisible(selector, chromedp.ByQuery),
		chromedp.Evaluate(script, &typeResult),
	); err != nil {
		if isSelectorError(err) {
			return d.fail(fmt.Errorf("%w: %s", ErrSelectorNotFound, selector))
		}
		return d.fail(fmt.Errorf("%w: %v", ErrElementNotInteractable, err))
	}
	if typeResult == "missing" {
		return d.fail(fmt.Errorf("%w: %s", ErrSelectorNotFound, selector))
	}
	if err := d.syncTab(ctx, tab); err != nil {
		return d.fail(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser type executed: %s", tab.id)
	d.latestActionError = ""
	return d.snapshotLocked(), nil
}

func (d *Driver) Extract(ctx context.Context, request ExtractRequest) (ExtractResult, error) {
	tab, err := d.lookupTab(request.TabID)
	if err != nil {
		return d.failExtract(err)
	}
	selector := strings.TrimSpace(request.Selector)
	var text string
	if selector == "" {
		if err := d.runOnTab(ctx, tab, chromedp.Text("body", &text, chromedp.ByQuery)); err != nil {
			return d.failExtract(fmt.Errorf("%w: %v", ErrExtractionFailed, err))
		}
	} else {
		if err := d.runOnTab(ctx, tab,
			chromedp.WaitVisible(selector, chromedp.ByQuery),
			chromedp.Text(selector, &text, chromedp.ByQuery),
		); err != nil {
			if isSelectorError(err) {
				return d.failExtract(fmt.Errorf("%w: %s", ErrSelectorNotFound, selector))
			}
			return d.failExtract(fmt.Errorf("%w: %v", ErrExtractionFailed, err))
		}
	}
	if err := d.syncTab(ctx, tab); err != nil {
		return d.failExtract(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser extract completed: %s", tab.id)
	d.latestActionError = ""
	return ExtractResult{
		Snapshot: d.snapshotLocked(),
		Text:     strings.TrimSpace(text),
	}, nil
}

func (d *Driver) Screenshot(ctx context.Context, request ScreenshotRequest) (ScreenshotResult, error) {
	tab, err := d.lookupTab(request.TabID)
	if err != nil {
		return d.failScreenshot(err)
	}
	var bytes []byte
	if err := d.runOnTab(ctx, tab, chromedp.FullScreenshot(&bytes, 90)); err != nil {
		return d.failScreenshot(fmt.Errorf("%w: %v", ErrScreenshotFailed, err))
	}
	if err := d.syncTab(ctx, tab); err != nil {
		return d.failScreenshot(err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.activeID = tab.id
	d.latestActionSummary = fmt.Sprintf("browser screenshot captured: %s", tab.id)
	d.latestActionError = ""
	return ScreenshotResult{
		Snapshot: d.snapshotLocked(),
		Bytes:    append([]byte(nil), bytes...),
	}, nil
}

func (d *Driver) lookupTab(tabID string) (*tabSession, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	targetID := strings.TrimSpace(tabID)
	if targetID == "" {
		targetID = d.activeID
	}
	tab, ok := d.tabs[targetID]
	if !ok {
		return nil, ErrTabNotFound
	}
	return tab, nil
}

func (d *Driver) lookupSelectorRequest(tabID string, selector string) (*tabSession, string, error) {
	tab, err := d.lookupTab(tabID)
	if err != nil {
		return nil, "", err
	}
	value := strings.TrimSpace(selector)
	if value == "" {
		return nil, "", ErrSelectorNotFound
	}
	return tab, value, nil
}

func (d *Driver) ensureAllocator() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.allocCtx != nil {
		return nil
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.Headless,
		chromedp.DisableGPU,
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-background-networking", true),
	)
	if path := detectBrowserBinary(); path != "" {
		opts = append(opts, chromedp.ExecPath(path))
	}
	allocCtx, cancel := chromedp.NewExecAllocator(d.rootCtx, opts...)
	d.allocCtx = allocCtx
	d.allocCancel = cancel
	return nil
}

func (d *Driver) newTab(ctx context.Context) (*tabSession, error) {
	if err := d.ensureAllocator(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	tabCtx, cancel := chromedp.NewContext(d.allocCtx)
	if err := chromedp.Run(tabCtx); err != nil {
		cancel()
		return nil, fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	tab := &tabSession{
		id:                fmt.Sprintf("browser-tab-%d", d.nextTabNum),
		ctx:               tabCtx,
		cancel:            cancel,
		history:           []string{},
		historyIndex:      -1,
		bootstrappedHosts: map[string]bool{},
	}
	d.nextTabNum++
	d.tabs[tab.id] = tab
	d.activeID = tab.id
	if ctx.Err() != nil {
		cancel()
		delete(d.tabs, tab.id)
		return nil, fmt.Errorf("%w: %v", ErrSessionUnavailable, ctx.Err())
	}
	return tab, nil
}

func (d *Driver) navigateTab(ctx context.Context, tab *tabSession, targetURL string, recordHistory bool) error {
	if err := d.ensureSessionBootstrap(ctx, tab, targetURL); err != nil {
		return err
	}
	if err := d.runOnTab(ctx, tab, chromedp.Navigate(targetURL)); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	if err := d.syncTab(ctx, tab); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	tab.url = targetURL
	if recordHistory {
		if tab.historyIndex < len(tab.history)-1 {
			tab.history = append([]string(nil), tab.history[:tab.historyIndex+1]...)
		}
		tab.history = append(tab.history, targetURL)
		tab.historyIndex = len(tab.history) - 1
	}
	tab.canGoBack = tab.historyIndex > 0
	tab.canGoForward = tab.historyIndex >= 0 && tab.historyIndex < len(tab.history)-1
	return nil
}

func (d *Driver) ensureSessionBootstrap(ctx context.Context, tab *tabSession, targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	profile, needsSession, err := d.policy.sessionProfileForHost(parsed.Hostname())
	if err != nil {
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	if !needsSession {
		return nil
	}
	host := strings.ToLower(parsed.Hostname())
	d.mu.Lock()
	if tab.bootstrappedHosts == nil {
		tab.bootstrappedHosts = map[string]bool{}
	}
	if tab.bootstrappedHosts[host] {
		d.mu.Unlock()
		return nil
	}
	d.mu.Unlock()
	if err := d.applySessionCookies(ctx, tab, parsed, profile); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}
	d.mu.Lock()
	tab.bootstrappedHosts[host] = true
	d.mu.Unlock()
	return nil
}

func (d *Driver) applySessionCookiesForProfile(ctx context.Context, tab *tabSession, target *url.URL, profile SessionProfile) error {
	if target == nil {
		return fmt.Errorf("missing navigation target")
	}
	if len(profile.Cookies) == 0 {
		return fmt.Errorf("missing session cookies")
	}
	targetURL := target.Scheme + "://" + target.Host
	actions := []chromedp.Action{
		network.Enable(),
		chromedp.ActionFunc(func(actionCtx context.Context) error {
			for _, cookie := range profile.Cookies {
				params := network.SetCookie(cookie.Name, cookie.Value).
					WithURL(targetURL).
					WithPath(cookie.Path).
					WithSecure(cookie.Secure).
					WithHTTPOnly(cookie.HTTPOnly)
				if cookie.Domain != "" {
					params = params.WithDomain(cookie.Domain)
				}
				if cookie.SameSite != "" {
					params = params.WithSameSite(network.CookieSameSite(cookie.SameSite))
				}
				if cookie.ExpiresUnix > 0 {
					expires := cdp.TimeSinceEpoch(time.Unix(cookie.ExpiresUnix, 0).UTC())
					params = params.WithExpires(&expires)
				}
				if err := params.Do(actionCtx); err != nil {
					return err
				}
			}
			return nil
		}),
	}
	return d.runOnTab(ctx, tab, actions...)
}

func (d *Driver) syncTab(ctx context.Context, tab *tabSession) error {
	if ctx.Err() != nil {
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, ctx.Err())
	}
	taskCtx, taskCancel := context.WithTimeout(tab.ctx, browserActionTimeout)
	defer taskCancel()
	var title string
	var currentURL string
	if err := chromedp.Run(taskCtx,
		chromedp.Title(&title),
		chromedp.Location(&currentURL),
	); err != nil {
		return fmt.Errorf("%w: %v", ErrSessionUnavailable, err)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	tab.title = strings.TrimSpace(title)
	if strings.TrimSpace(currentURL) != "" {
		tab.url = strings.TrimSpace(currentURL)
	}
	tab.loading = false
	return nil
}

func (d *Driver) runOnTab(ctx context.Context, tab *tabSession, actions ...chromedp.Action) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	taskCtx, cancel := context.WithTimeout(tab.ctx, browserActionTimeout)
	defer cancel()
	return chromedp.Run(taskCtx, actions...)
}

func (d *Driver) closeTabLocked(tabID string) {
	tab, ok := d.tabs[tabID]
	if !ok {
		return
	}
	tab.cancel()
	delete(d.tabs, tabID)
	if d.activeID == tabID {
		d.activeID = ""
		for id := range d.tabs {
			d.activeID = id
		}
	}
}

func (d *Driver) snapshotLocked() Snapshot {
	items := make([]Tab, 0, len(d.tabs))
	for _, tab := range d.tabs {
		items = append(items, Tab{
			ID:           tab.id,
			URL:          tab.url,
			Title:        fallbackBrowserTitle(tab.title, tab.url),
			Loading:      tab.loading,
			CanGoBack:    tab.canGoBack,
			CanGoForward: tab.canGoForward,
		})
	}
	sort.Slice(items, func(i int, j int) bool {
		return items[i].ID < items[j].ID
	})
	return Snapshot{
		ActiveTabID:         d.activeID,
		Tabs:                items,
		LatestActionSummary: d.latestActionSummary,
		LatestActionError:   d.latestActionError,
	}
}

func (d *Driver) fail(err error) (Snapshot, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.failLocked(err)
}

func (d *Driver) failLocked(err error) (Snapshot, error) {
	d.latestActionError = err.Error()
	return d.snapshotLocked(), err
}

func (d *Driver) failExtract(err error) (ExtractResult, error) {
	snapshot, failErr := d.fail(err)
	return ExtractResult{Snapshot: snapshot}, failErr
}

func (d *Driver) failScreenshot(err error) (ScreenshotResult, error) {
	snapshot, failErr := d.fail(err)
	return ScreenshotResult{Snapshot: snapshot}, failErr
}

func fallbackBrowserTitle(title string, rawURL string) string {
	if strings.TrimSpace(title) != "" {
		return strings.TrimSpace(title)
	}
	if strings.TrimSpace(rawURL) == "" {
		return "Controlled page"
	}
	return rawURL
}

func isSelectorError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "could not find node") ||
		strings.Contains(message, "not visible") ||
		strings.Contains(message, "context deadline exceeded")
}

func detectBrowserBinary() string {
	candidates := []string{}
	if envPath := strings.TrimSpace(os.Getenv("GENCODE_BROWSER_EXECUTABLE")); envPath != "" {
		candidates = append(candidates, envPath)
	}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		)
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
