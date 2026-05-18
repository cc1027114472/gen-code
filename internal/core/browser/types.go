package browser

// Snapshot describes the current controlled browser workspace state.
type Snapshot struct {
	ActiveTabID         string `json:"activeTabId"`
	Tabs                []Tab  `json:"tabs"`
	LatestActionSummary string `json:"latestActionSummary,omitempty"`
	LatestActionError   string `json:"latestActionError,omitempty"`
}

// Tab describes a single controlled browser tab.
type Tab struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	Loading      bool   `json:"loading"`
	CanGoBack    bool   `json:"canGoBack"`
	CanGoForward bool   `json:"canGoForward"`
}

// OpenRequest opens a new controlled tab.
type OpenRequest struct {
	URL string `json:"url"`
}

// NavigateRequest navigates an existing controlled tab.
type NavigateRequest struct {
	TabID string `json:"tabId"`
	URL   string `json:"url"`
}

// TabRequest targets a single controlled tab.
type TabRequest struct {
	TabID string `json:"tabId"`
}

// ClickRequest clicks a selector in a controlled tab.
type ClickRequest struct {
	TabID    string `json:"tabId"`
	Selector string `json:"selector"`
}

// TypeRequest types text into a selector in a controlled tab.
type TypeRequest struct {
	TabID    string `json:"tabId"`
	Selector string `json:"selector"`
	Text     string `json:"text"`
}

// ExtractRequest extracts text from a selector or the page body.
type ExtractRequest struct {
	TabID    string `json:"tabId"`
	Selector string `json:"selector,omitempty"`
}

// ScreenshotRequest captures a screenshot for a controlled tab.
type ScreenshotRequest struct {
	TabID string `json:"tabId"`
}

// ExtractResult returns extracted text plus the latest browser snapshot.
type ExtractResult struct {
	Snapshot Snapshot `json:"snapshot"`
	Text     string   `json:"text"`
}

// ScreenshotResult returns screenshot bytes plus the latest browser snapshot.
type ScreenshotResult struct {
	Snapshot Snapshot `json:"snapshot"`
	Bytes    []byte   `json:"-"`
}
