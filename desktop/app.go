package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultRuntimeBaseURL = "http://127.0.0.1:10008"
	defaultTaskTitle      = "New Task"
	defaultThreadName     = "New Thread"
)

var localIDCounter atomic.Uint64

type App struct {
	ctx     context.Context
	store   *localRuntimeStore
	browser *browserWorkspace
}

type apiEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type RuntimeStatus struct {
	AppName               string              `json:"appName"`
	AppEnv                string              `json:"appEnv"`
	Port                  int                 `json:"port"`
	Debug                 bool                `json:"debug"`
	ShutdownTimeout       string              `json:"shutdownTimeout"`
	TrustedProxies        []string            `json:"trustedProxies"`
	LogLevel              string              `json:"logLevel"`
	HTTPAccessLog         bool                `json:"httpAccessLog"`
	WorkspaceRoot         string              `json:"workspaceRoot"`
	WorkspaceID           string              `json:"workspaceId"`
	ProjectRoot           string              `json:"projectRoot"`
	ThreadCount           int                 `json:"threadCount"`
	ActiveThreadID        string              `json:"activeThreadId"`
	Threads               []ThreadSummary     `json:"threads"`
	Tasks                 []TaskSummary       `json:"tasks"`
	Approvals             []ApprovalSummary   `json:"approvals"`
	Messages              []MessageSummary    `json:"messages"`
	ToolCalls             []ToolCallSummary   `json:"toolCalls"`
	Artifacts             []ArtifactSummary   `json:"artifacts"`
	Events                []EventSummary      `json:"events"`
	DesktopReady          bool                `json:"desktopReady"`
	RuntimeState          string              `json:"runtimeState"`
	RuntimeReady          bool                `json:"runtimeReady"`
	RuntimeMessage        string              `json:"runtimeMessage"`
	RuntimeSource         string              `json:"runtimeSource"`
	SupportsSSE           bool                `json:"supportsSSE"`
	SSEEndpoint           string              `json:"sseEndpoint"`
	LastSyncAt            string              `json:"lastSyncAt"`
	SkillsByGroup         map[string][]string `json:"skillsByGroup"`
	ToolsByGroup          map[string][]string `json:"toolsByGroup"`
	MCPByGroup            map[string][]string `json:"mcpByGroup"`
	Providers             []ProviderSummary   `json:"providers"`
	MissingPaths          []string            `json:"missingPaths"`
	StateStore            string              `json:"stateStore"`
	StatePath             string              `json:"statePath"`
	UsesProjectLocalStore bool                `json:"usesProjectLocalStore"`
	RecoverySummary       string              `json:"recoverySummary"`
	UpdatedAt             string              `json:"updatedAt"`
}

type ThreadSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	ActiveModel    string `json:"activeModel"`
	PermissionMode string `json:"permissionMode"`
	IsActive       bool   `json:"isActive"`
}

type TaskSummary struct {
	ID             string `json:"id"`
	ThreadID       string `json:"threadId"`
	Title          string `json:"title"`
	Kind           string `json:"kind"`
	Input          string `json:"input"`
	Status         string `json:"status"`
	ResultSummary  string `json:"resultSummary"`
	ApprovalStatus string `json:"approvalStatus"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type ApprovalSummary struct {
	ID          string   `json:"id"`
	ThreadID    string   `json:"threadId"`
	TaskID      string   `json:"taskId"`
	ToolKind    string   `json:"toolKind"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	TargetPaths []string `json:"targetPaths"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type MessageSummary struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type ToolCallSummary struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	ToolID    string `json:"toolId"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"createdAt"`
}

type ArtifactSummary struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"createdAt"`
}

type EventSummary struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type BridgeCheckResult struct {
	OK          bool   `json:"ok"`
	Message     string `json:"message"`
	CheckedAt   string `json:"checkedAt"`
	RuntimeHint string `json:"runtimeHint"`
}

type BrowserTab struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	IsActive     bool   `json:"isActive"`
	CanGoBack    bool   `json:"canGoBack"`
	CanGoForward bool   `json:"canGoForward"`
}

type BrowserWorkspaceState struct {
	IsOpen      bool         `json:"isOpen"`
	Tabs        []BrowserTab `json:"tabs"`
	ActiveTabID string       `json:"activeTabId"`
}

type apiStatus struct {
	State          string `json:"state"`
	Ready          bool   `json:"ready"`
	Message        string `json:"message"`
	RuntimeSource  string `json:"runtimeSource"`
	StateStore     string `json:"stateStore"`
	StatePath      string `json:"statePath"`
	WorkspaceID    string `json:"workspaceId"`
	ProjectRoot    string `json:"projectRoot"`
	ThreadCount    int    `json:"threadCount"`
	ActiveThreadID string `json:"activeThreadId"`
	TaskCount      int    `json:"taskCount"`
	EventCount     int    `json:"eventCount"`
}

type apiThread struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	ActiveModel    string `json:"activeModel"`
	PermissionMode string `json:"permissionMode"`
	IsActive       bool   `json:"isActive"`
}

type apiTask struct {
	ID             string `json:"id"`
	ThreadID       string `json:"threadId"`
	Title          string `json:"title"`
	Kind           string `json:"kind"`
	Input          string `json:"input"`
	Status         string `json:"status"`
	ResultSummary  string `json:"resultSummary"`
	ApprovalStatus string `json:"approvalStatus"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}

type apiApproval struct {
	ID          string   `json:"id"`
	ThreadID    string   `json:"threadId"`
	TaskID      string   `json:"taskId"`
	ToolKind    string   `json:"toolKind"`
	Status      string   `json:"status"`
	Summary     string   `json:"summary"`
	TargetPaths []string `json:"targetPaths"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type apiMessage struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

type apiToolCall struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	ToolID    string `json:"toolId"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"createdAt"`
}

type apiArtifact struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"createdAt"`
}

type apiEvent struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

type apiSkill struct {
	ID    string `json:"id"`
	Group string `json:"group"`
}

type apiTool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Permission  string `json:"permissionMode"`
	Source      string `json:"source"`
	Kind        string `json:"kind"`
	ReadOnly    bool   `json:"readOnly"`
	Executable  bool   `json:"executable"`
}

type apiProvider struct {
	Kind              string `json:"kind"`
	Enabled           bool   `json:"enabled"`
	BaseURL           string `json:"baseUrl"`
	DefaultModel      string `json:"defaultModel"`
	HasAuthToken      bool   `json:"hasAuthToken"`
	SupportsChat      bool   `json:"supportsChat"`
	SupportsResponses bool   `json:"supportsResponses"`
	PreferredAPIStyle string `json:"preferredApiStyle"`
	Recommended       bool   `json:"recommended"`
	RecommendedReason string `json:"recommendedReason"`
}

type apiMCPServer struct {
	ID            string `json:"id"`
	Source        string `json:"source"`
	ToolCount     int    `json:"toolCount"`
	ResourceCount int    `json:"resourceCount"`
}

type apiBridgeCheck struct {
	OK      bool           `json:"ok"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type runtimeClient struct {
	baseURL string
	client  http.Client
}

type browserWorkspace struct {
	mu         sync.Mutex
	isOpen     bool
	tabs       []BrowserTab
	activeTab  string
	nextTabNum int
}

type localRuntimeStore struct {
	mu sync.Mutex
	db *sql.DB
}

type persistedThread struct {
	ID             string
	Name           string
	Status         string
	ActiveModel    string
	PermissionMode string
	IsActive       bool
}

type persistedTask struct {
	ID             string
	ThreadID       string
	Title          string
	Kind           string
	Input          string
	Status         string
	ResultSummary  string
	ApprovalStatus string
	CreatedAt      string
	UpdatedAt      string
}

type persistedApproval struct {
	ID          string
	ThreadID    string
	TaskID      string
	ToolKind    string
	Status      string
	Summary     string
	TargetPaths []string
	CreatedAt   string
	UpdatedAt   string
}

type TaskCreateInput struct {
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Input string `json:"input"`
}

type ProviderSummary struct {
	Kind              string `json:"kind"`
	Enabled           bool   `json:"enabled"`
	BaseURL           string `json:"baseUrl"`
	DefaultModel      string `json:"defaultModel"`
	HasAuthToken      bool   `json:"hasAuthToken"`
	SupportsChat      bool   `json:"supportsChat"`
	SupportsResponses bool   `json:"supportsResponses"`
	PreferredAPIStyle string `json:"preferredApiStyle"`
	Recommended       bool   `json:"recommended"`
	RecommendedReason string `json:"recommendedReason"`
}

type persistedMessage struct {
	ID        string
	ThreadID  string
	Role      string
	Content   string
	CreatedAt string
}

type persistedToolCall struct {
	ID        string
	ThreadID  string
	ToolID    string
	Status    string
	Summary   string
	CreatedAt string
}

type persistedArtifact struct {
	ID        string
	ThreadID  string
	Path      string
	Kind      string
	CreatedAt string
}

type persistedEvent struct {
	ID        string
	ThreadID  string
	Type      string
	Message   string
	CreatedAt string
}

func NewApp() *App {
	return &App{
		store:   newLocalRuntimeStore(),
		browser: newBrowserWorkspace(),
	}
}

func (a *App) shutdown(context.Context) {
	if a == nil || a.store == nil {
		return
	}
	_ = a.store.Close()
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetAppInfo() string {
	return "gen-code desktop shell ready"
}

func (a *App) BrowserState() BrowserWorkspaceState {
	if a == nil || a.browser == nil {
		return BrowserWorkspaceState{}
	}
	return a.browser.State()
}

func (a *App) BrowserOpen(rawURL string) BrowserWorkspaceState {
	if a == nil || a.browser == nil {
		return BrowserWorkspaceState{}
	}
	return a.browser.Open(rawURL)
}

func (a *App) BrowserNavigate(tabID string, rawURL string) BrowserWorkspaceState {
	if a == nil || a.browser == nil {
		return BrowserWorkspaceState{}
	}
	return a.browser.Navigate(tabID, rawURL)
}

func (a *App) BrowserBack(tabID string) BrowserWorkspaceState {
	if a == nil || a.browser == nil {
		return BrowserWorkspaceState{}
	}
	return a.browser.Back(tabID)
}

func (a *App) BrowserForward(tabID string) BrowserWorkspaceState {
	if a == nil || a.browser == nil {
		return BrowserWorkspaceState{}
	}
	return a.browser.Forward(tabID)
}

func (a *App) BrowserReload(tabID string) BrowserWorkspaceState {
	if a == nil || a.browser == nil {
		return BrowserWorkspaceState{}
	}
	return a.browser.Reload(tabID)
}

func (a *App) BrowserCloseTab(tabID string) BrowserWorkspaceState {
	if a == nil || a.browser == nil {
		return BrowserWorkspaceState{}
	}
	return a.browser.CloseTab(tabID)
}

func (a *App) BrowserActivateTab(tabID string) BrowserWorkspaceState {
	if a == nil || a.browser == nil {
		return BrowserWorkspaceState{}
	}
	return a.browser.ActivateTab(tabID)
}

func (a *App) GetRuntimeStatus() RuntimeStatus {
	status, err := a.collectRuntimeStatus()
	if err != nil {
		return RuntimeStatus{
			AppName:        "gen-code",
			AppEnv:         "local",
			Port:           10008,
			DesktopReady:   true,
			RuntimeState:   "degraded",
			RuntimeReady:   false,
			RuntimeMessage: err.Error(),
			RuntimeSource:  "desktop-local",
			StateStore:     "sqlite",
			StatePath:      "",
			SkillsByGroup:  map[string][]string{},
			ToolsByGroup:   map[string][]string{},
			MCPByGroup:     map[string][]string{},
			MissingPaths:   []string{},
			UpdatedAt:      time.Now().Format(time.RFC3339),
		}
	}
	return status
}

func (a *App) CheckBridge() BridgeCheckResult {
	now := time.Now().Format(time.RFC3339)
	if bridge, err := fetchBridgeCheck(newRuntimeClient()); err == nil {
		bridge.CheckedAt = now
		if bridge.RuntimeHint == "" {
			bridge.RuntimeHint = "runtime bridge online"
		}
		return bridge
	}

	status := a.GetRuntimeStatus()
	return BridgeCheckResult{
		OK:          true,
		Message:     "Go bridge is available, using local desktop runtime fallback.",
		CheckedAt:   now,
		RuntimeHint: fmt.Sprintf("%s / %s / %s", status.RuntimeSource, status.RuntimeState, fallbackText(status.StatePath, "no-state-path")),
	}
}

func (a *App) CreateThread(name string) RuntimeStatus {
	if status, err := createThread(newRuntimeClient(), name); err == nil {
		return status
	}
	return a.store.CreateThread(name)
}

func (a *App) ActivateThread(id string) RuntimeStatus {
	if status, err := activateThread(newRuntimeClient(), id); err == nil {
		return status
	}
	return a.store.ActivateThread(id)
}

func (a *App) CreateTask(threadID string, payload string) RuntimeStatus {
	input := parseTaskCreateInput(payload)
	if status, err := createTask(newRuntimeClient(), threadID, input); err == nil {
		return status
	}
	return a.store.CreateTask(threadID, input)
}

func (a *App) AdvanceTask(taskID string) RuntimeStatus {
	current := a.GetRuntimeStatus()
	if status, err := runTask(newRuntimeClient(), current.ActiveThreadID, taskID, current.Tasks); err == nil {
		return status
	}
	return a.store.RunTask(taskID)
}

func (a *App) ApproveTask(threadID string, taskID string) RuntimeStatus {
	if status, err := approveTask(newRuntimeClient(), threadID, taskID); err == nil {
		return status
	}
	return a.store.ApproveTask(threadID, taskID)
}

func (a *App) RejectTask(threadID string, taskID string) RuntimeStatus {
	if status, err := rejectTask(newRuntimeClient(), threadID, taskID); err == nil {
		return status
	}
	return a.store.RejectTask(threadID, taskID)
}

func (a *App) collectRuntimeStatus() (RuntimeStatus, error) {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return RuntimeStatus{}, err
	}

	baseStatus, err := buildBaseStatus(workspaceRoot)
	if err != nil {
		return RuntimeStatus{}, err
	}

	client := newRuntimeClient()
	liveStatus, liveErr := collectRemoteRuntimeStatus(client, workspaceRoot, baseStatus)
	if liveErr == nil {
		return liveStatus, nil
	}

	localStatus, err := a.store.Snapshot(*baseStatus)
	if err != nil {
		return RuntimeStatus{}, err
	}
	localStatus.RuntimeMessage = "External runtime is unavailable, switched to project-local SQLite fallback."
	localStatus.RuntimeState = "fallback"
	localStatus.RuntimeReady = true
	localStatus.RuntimeSource = "desktop-local"
	localStatus.SupportsSSE = false
	localStatus.SSEEndpoint = ""
	localStatus.UpdatedAt = time.Now().Format(time.RFC3339)
	localStatus.LastSyncAt = ""
	return localStatus, nil
}

func buildBaseStatus(workspaceRoot string) (*RuntimeStatus, error) {
	missingPaths := []string{}
	desktopModule := filepath.Join(workspaceRoot, "desktop", "go.mod")
	if _, err := os.Stat(desktopModule); err != nil {
		missingPaths = append(missingPaths, desktopModule)
	}
	desktopFrontend := filepath.Join(workspaceRoot, "desktop", "frontend", "package.json")
	if _, err := os.Stat(desktopFrontend); err != nil {
		missingPaths = append(missingPaths, desktopFrontend)
	}
	sort.Strings(missingPaths)

	statePath := defaultStateStorePath(workspaceRoot)
	return &RuntimeStatus{
		AppName:               "gen-code",
		AppEnv:                "local",
		Port:                  10008,
		Debug:                 false,
		ShutdownTimeout:       "10s",
		TrustedProxies:        []string{"127.0.0.1"},
		LogLevel:              "info",
		HTTPAccessLog:         true,
		WorkspaceRoot:         workspaceRoot,
		DesktopReady:          true,
		SkillsByGroup:         map[string][]string{},
		ToolsByGroup:          map[string][]string{},
		MCPByGroup:            map[string][]string{},
		Providers:             localProviderCatalog(),
		MissingPaths:          missingPaths,
		StateStore:            "sqlite",
		StatePath:             statePath,
		UsesProjectLocalStore: true,
		UpdatedAt:             time.Now().Format(time.RFC3339),
	}, nil
}

func collectRemoteRuntimeStatus(client runtimeClient, workspaceRoot string, base *RuntimeStatus) (RuntimeStatus, error) {
	runtimeStatus := apiStatus{}
	if err := client.fetchEnvelope("/api/runtime/status", &runtimeStatus); err != nil {
		return RuntimeStatus{}, err
	}

	var skillsPayload struct {
		Items []apiSkill `json:"items"`
	}
	if err := client.fetchEnvelope("/api/skills", &skillsPayload); err != nil {
		return RuntimeStatus{}, err
	}

	var toolsPayload struct {
		Items []apiTool `json:"items"`
	}
	if err := client.fetchEnvelope("/api/tools", &toolsPayload); err != nil {
		return RuntimeStatus{}, err
	}

	var providerPayload struct {
		Items []apiProvider `json:"items"`
	}
	if err := client.fetchEnvelope("/api/providers", &providerPayload); err != nil {
		return RuntimeStatus{}, err
	}

	var mcpPayload struct {
		Items []apiMCPServer `json:"items"`
	}
	if err := client.fetchEnvelope("/api/mcp/servers", &mcpPayload); err != nil {
		return RuntimeStatus{}, err
	}

	var threadPayload struct {
		Items []apiThread `json:"items"`
	}
	if err := client.fetchEnvelope("/api/threads", &threadPayload); err != nil {
		return RuntimeStatus{}, err
	}

	tasksPayload := struct {
		Items []apiTask `json:"items"`
	}{}
	approvalsPayload := struct {
		Items []apiApproval `json:"items"`
	}{}
	messagesPayload := struct {
		Items []apiMessage `json:"items"`
	}{}
	toolCallsPayload := struct {
		Items []apiToolCall `json:"items"`
	}{}
	artifactsPayload := struct {
		Items []apiArtifact `json:"items"`
	}{}
	eventsPayload := struct {
		Items []apiEvent `json:"items"`
	}{}
	if runtimeStatus.ActiveThreadID != "" {
		threadID := url.PathEscape(runtimeStatus.ActiveThreadID)
		if err := client.fetchEnvelope("/api/threads/"+threadID+"/tasks", &tasksPayload); err != nil {
			return RuntimeStatus{}, err
		}
		if err := client.fetchEnvelope("/api/threads/"+threadID+"/approvals", &approvalsPayload); err != nil {
			return RuntimeStatus{}, err
		}
		if err := client.fetchEnvelope("/api/threads/"+threadID+"/messages", &messagesPayload); err != nil {
			return RuntimeStatus{}, err
		}
		if err := client.fetchEnvelope("/api/threads/"+threadID+"/tool-calls", &toolCallsPayload); err != nil {
			return RuntimeStatus{}, err
		}
		if err := client.fetchEnvelope("/api/threads/"+threadID+"/artifacts", &artifactsPayload); err != nil {
			return RuntimeStatus{}, err
		}
		if err := client.fetchEnvelope("/api/threads/"+threadID+"/events", &eventsPayload); err != nil {
			return RuntimeStatus{}, err
		}
	}

	status := *base
	status.WorkspaceRoot = workspaceRoot
	status.WorkspaceID = runtimeStatus.WorkspaceID
	status.ProjectRoot = runtimeStatus.ProjectRoot
	status.ThreadCount = runtimeStatus.ThreadCount
	status.ActiveThreadID = runtimeStatus.ActiveThreadID
	status.Threads = mapThreads(threadPayload.Items)
	status.Tasks = mapTasks(tasksPayload.Items)
	status.Approvals = mapApprovals(approvalsPayload.Items)
	status.Messages = mapMessages(messagesPayload.Items)
	status.ToolCalls = mapToolCalls(toolCallsPayload.Items)
	status.Artifacts = mapArtifacts(artifactsPayload.Items)
	status.Events = mapEvents(eventsPayload.Items)
	status.RuntimeState = runtimeStatus.State
	status.RuntimeReady = runtimeStatus.Ready
	status.RuntimeMessage = runtimeStatus.Message
	status.RuntimeSource = fallbackText(runtimeStatus.RuntimeSource, "runtime-http")
	status.StateStore = fallbackText(runtimeStatus.StateStore, "sqlite")
	status.StatePath = fallbackText(runtimeStatus.StatePath, defaultStateStorePath(workspaceRoot))
	status.UsesProjectLocalStore = strings.EqualFold(status.StateStore, "sqlite")
	status.SupportsSSE = true
	if runtimeStatus.ActiveThreadID != "" {
		status.SSEEndpoint = strings.TrimRight(runtimeBaseURL(), "/") + "/api/threads/" + url.PathEscape(runtimeStatus.ActiveThreadID) + "/events/stream"
	}
	status.LastSyncAt = time.Now().Format(time.RFC3339)
	status.SkillsByGroup = groupSkills(skillsPayload.Items)
	status.ToolsByGroup = groupTools(toolsPayload.Items)
	status.MCPByGroup = groupMCPServers(mcpPayload.Items)
	status.Providers = mapProviders(providerPayload.Items)
	status.RecoverySummary = fmt.Sprintf("Live runtime connected. Active thread: %s, tasks: %d, messages: %d, tool calls: %d, artifacts: %d.", fallbackText(runtimeStatus.ActiveThreadID, "none"), len(status.Tasks), len(status.Messages), len(status.ToolCalls), len(status.Artifacts))
	status.UpdatedAt = time.Now().Format(time.RFC3339)
	return status, nil
}

func newRuntimeClient() runtimeClient {
	return runtimeClient{
		baseURL: strings.TrimRight(runtimeBaseURL(), "/"),
		client: http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func newBrowserWorkspace() *browserWorkspace {
	workspace := &browserWorkspace{
		isOpen:     true,
		tabs:       []BrowserTab{},
		nextTabNum: 1,
	}
	workspace.openLocked(defaultBrowserURL())
	return workspace
}

func defaultBrowserURL() string {
	base := strings.TrimSpace(os.Getenv("GENCODE_DESKTOP_BROWSER_URL"))
	if base != "" {
		return normalizeBrowserURL(base)
	}
	return "http://127.0.0.1:5174/"
}

func normalizeBrowserURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return defaultRuntimeBaseURL
	}
	if strings.HasPrefix(trimmed, "localhost:") || strings.HasPrefix(trimmed, "127.0.0.1:") {
		return "http://" + trimmed
	}
	if !strings.Contains(trimmed, "://") {
		return "http://" + trimmed
	}
	return trimmed
}

func browserTabTitle(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "本地预览"
	}
	if parsed.Host != "" {
		return parsed.Host
	}
	if parsed.Path != "" {
		return filepath.Base(parsed.Path)
	}
	return "本地预览"
}

func cloneBrowserTabs(items []BrowserTab, activeID string) []BrowserTab {
	cloned := make([]BrowserTab, 0, len(items))
	for _, item := range items {
		item.IsActive = item.ID == activeID
		cloned = append(cloned, item)
	}
	return cloned
}

func (b *browserWorkspace) State() BrowserWorkspaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return BrowserWorkspaceState{
		IsOpen:      b.isOpen,
		Tabs:        cloneBrowserTabs(b.tabs, b.activeTab),
		ActiveTabID: b.activeTab,
	}
}

func (b *browserWorkspace) Open(rawURL string) BrowserWorkspaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.isOpen = true
	b.openLocked(rawURL)
	return BrowserWorkspaceState{
		IsOpen:      b.isOpen,
		Tabs:        cloneBrowserTabs(b.tabs, b.activeTab),
		ActiveTabID: b.activeTab,
	}
}

func (b *browserWorkspace) Navigate(tabID string, rawURL string) BrowserWorkspaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	targetID := strings.TrimSpace(tabID)
	targetURL := normalizeBrowserURL(rawURL)
	if targetID == "" {
		b.openLocked(targetURL)
	} else {
		for index, item := range b.tabs {
			if item.ID != targetID {
				continue
			}
			item.URL = targetURL
			item.Title = browserTabTitle(targetURL)
			item.Status = "ready"
			item.CanGoBack = false
			item.CanGoForward = false
			b.tabs[index] = item
			b.activeTab = item.ID
			break
		}
	}
	return BrowserWorkspaceState{
		IsOpen:      true,
		Tabs:        cloneBrowserTabs(b.tabs, b.activeTab),
		ActiveTabID: b.activeTab,
	}
}

func (b *browserWorkspace) Reload(tabID string) BrowserWorkspaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	targetID := strings.TrimSpace(tabID)
	if targetID == "" {
		targetID = b.activeTab
	}
	for index, item := range b.tabs {
		if item.ID != targetID {
			continue
		}
		item.Status = "ready"
		b.tabs[index] = item
		b.activeTab = item.ID
		break
	}
	return BrowserWorkspaceState{
		IsOpen:      b.isOpen,
		Tabs:        cloneBrowserTabs(b.tabs, b.activeTab),
		ActiveTabID: b.activeTab,
	}
}

func (b *browserWorkspace) Back(tabID string) BrowserWorkspaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	targetID := strings.TrimSpace(tabID)
	if targetID == "" {
		targetID = b.activeTab
	}
	for index, item := range b.tabs {
		if item.ID != targetID {
			continue
		}
		item.CanGoBack = false
		b.tabs[index] = item
		b.activeTab = item.ID
		break
	}
	return BrowserWorkspaceState{
		IsOpen:      b.isOpen,
		Tabs:        cloneBrowserTabs(b.tabs, b.activeTab),
		ActiveTabID: b.activeTab,
	}
}

func (b *browserWorkspace) Forward(tabID string) BrowserWorkspaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	targetID := strings.TrimSpace(tabID)
	if targetID == "" {
		targetID = b.activeTab
	}
	for index, item := range b.tabs {
		if item.ID != targetID {
			continue
		}
		item.CanGoForward = false
		b.tabs[index] = item
		b.activeTab = item.ID
		break
	}
	return BrowserWorkspaceState{
		IsOpen:      b.isOpen,
		Tabs:        cloneBrowserTabs(b.tabs, b.activeTab),
		ActiveTabID: b.activeTab,
	}
}

func (b *browserWorkspace) CloseTab(tabID string) BrowserWorkspaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	targetID := strings.TrimSpace(tabID)
	if targetID == "" {
		targetID = b.activeTab
	}
	filtered := make([]BrowserTab, 0, len(b.tabs))
	for _, item := range b.tabs {
		if item.ID == targetID {
			continue
		}
		filtered = append(filtered, item)
	}
	b.tabs = filtered
	if len(b.tabs) == 0 {
		b.openLocked(defaultBrowserURL())
	}
	if !browserTabExists(b.tabs, b.activeTab) {
		b.activeTab = b.tabs[len(b.tabs)-1].ID
	}
	return BrowserWorkspaceState{
		IsOpen:      b.isOpen,
		Tabs:        cloneBrowserTabs(b.tabs, b.activeTab),
		ActiveTabID: b.activeTab,
	}
}

func (b *browserWorkspace) ActivateTab(tabID string) BrowserWorkspaceState {
	b.mu.Lock()
	defer b.mu.Unlock()
	targetID := strings.TrimSpace(tabID)
	if browserTabExists(b.tabs, targetID) {
		b.activeTab = targetID
		b.isOpen = true
	}
	return BrowserWorkspaceState{
		IsOpen:      b.isOpen,
		Tabs:        cloneBrowserTabs(b.tabs, b.activeTab),
		ActiveTabID: b.activeTab,
	}
}

func (b *browserWorkspace) openLocked(rawURL string) {
	targetURL := normalizeBrowserURL(rawURL)
	tabID := fmt.Sprintf("browser-tab-%d", b.nextTabNum)
	b.nextTabNum++
	tab := BrowserTab{
		ID:           tabID,
		Title:        browserTabTitle(targetURL),
		URL:          targetURL,
		Status:       "ready",
		IsActive:     true,
		CanGoBack:    false,
		CanGoForward: false,
	}
	b.tabs = append(b.tabs, tab)
	b.activeTab = tabID
}

func browserTabExists(items []BrowserTab, targetID string) bool {
	for _, item := range items {
		if item.ID == targetID {
			return true
		}
	}
	return false
}

func runtimeBaseURL() string {
	if value := strings.TrimSpace(os.Getenv("GENCODE_RUNTIME_BASE_URL")); value != "" {
		return value
	}
	return defaultRuntimeBaseURL
}

func fetchBridgeCheck(client runtimeClient) (BridgeCheckResult, error) {
	var payload apiBridgeCheck
	if err := client.postEnvelope("/api/bridge/check", map[string]any{}, &payload); err != nil {
		return BridgeCheckResult{}, err
	}

	return BridgeCheckResult{
		OK:          payload.OK,
		Message:     fallbackText(payload.Message, "runtime bridge online"),
		RuntimeHint: "runtime-http",
	}, nil
}

func createThread(client runtimeClient, name string) (RuntimeStatus, error) {
	threadName := fallbackText(strings.TrimSpace(name), defaultThreadName)
	var created map[string]any
	if err := client.postEnvelope("/api/threads", map[string]string{"name": threadName}, &created); err != nil {
		return RuntimeStatus{}, err
	}
	return NewApp().GetRuntimeStatus(), nil
}

func activateThread(client runtimeClient, id string) (RuntimeStatus, error) {
	threadID := strings.TrimSpace(id)
	if threadID == "" {
		return RuntimeStatus{}, fmt.Errorf("thread id is required")
	}
	var activated map[string]any
	if err := client.postEnvelope("/api/threads/"+url.PathEscape(threadID)+"/activate", map[string]any{}, &activated); err != nil {
		return RuntimeStatus{}, err
	}
	return NewApp().GetRuntimeStatus(), nil
}

func createTask(client runtimeClient, threadID string, input TaskCreateInput) (RuntimeStatus, error) {
	trimmedThreadID := strings.TrimSpace(threadID)
	if trimmedThreadID == "" {
		return RuntimeStatus{}, fmt.Errorf("thread id is required")
	}
	normalized := normalizeTaskCreateInput(input)
	var created map[string]any
	requestBody := map[string]string{
		"title": normalized.Title,
		"kind":  normalized.Kind,
		"input": normalized.Input,
	}
	if err := client.postEnvelope("/api/threads/"+url.PathEscape(trimmedThreadID)+"/tasks", requestBody, &created); err != nil {
		legacyBody := map[string]string{"title": normalized.Title}
		if legacyErr := client.postEnvelope("/api/threads/"+url.PathEscape(trimmedThreadID)+"/tasks", legacyBody, &created); legacyErr != nil {
			return RuntimeStatus{}, err
		}
	}
	return NewApp().GetRuntimeStatus(), nil
}

func runTask(client runtimeClient, threadID string, taskID string, tasks []TaskSummary) (RuntimeStatus, error) {
	trimmedThreadID := strings.TrimSpace(threadID)
	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedThreadID == "" || trimmedTaskID == "" {
		return RuntimeStatus{}, fmt.Errorf("thread id and task id are required")
	}

	var updated map[string]any
	runBody := map[string]any{
		"status": "running",
	}
	for _, task := range tasks {
		if task.ID != trimmedTaskID {
			continue
		}
		runBody["kind"] = task.Kind
		runBody["input"] = task.Input
		runBody["title"] = task.Title
		break
	}
	if err := client.postEnvelope("/api/threads/"+url.PathEscape(trimmedThreadID)+"/tasks/"+url.PathEscape(trimmedTaskID)+"/run", runBody, &updated); err != nil {
		if legacyErr := client.postEnvelope("/api/threads/"+url.PathEscape(trimmedThreadID)+"/tasks/"+url.PathEscape(trimmedTaskID)+"/status", map[string]string{"status": "running"}, &updated); legacyErr != nil {
			return RuntimeStatus{}, err
		}
	}
	return NewApp().GetRuntimeStatus(), nil
}

func approveTask(client runtimeClient, threadID string, taskID string) (RuntimeStatus, error) {
	trimmedThreadID := strings.TrimSpace(threadID)
	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedThreadID == "" || trimmedTaskID == "" {
		return RuntimeStatus{}, fmt.Errorf("thread id and task id are required")
	}
	var updated map[string]any
	if err := client.postEnvelope("/api/threads/"+url.PathEscape(trimmedThreadID)+"/tasks/"+url.PathEscape(trimmedTaskID)+"/approve", map[string]any{}, &updated); err != nil {
		return RuntimeStatus{}, err
	}
	return NewApp().GetRuntimeStatus(), nil
}

func rejectTask(client runtimeClient, threadID string, taskID string) (RuntimeStatus, error) {
	trimmedThreadID := strings.TrimSpace(threadID)
	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedThreadID == "" || trimmedTaskID == "" {
		return RuntimeStatus{}, fmt.Errorf("thread id and task id are required")
	}
	var updated map[string]any
	if err := client.postEnvelope("/api/threads/"+url.PathEscape(trimmedThreadID)+"/tasks/"+url.PathEscape(trimmedTaskID)+"/reject", map[string]any{}, &updated); err != nil {
		return RuntimeStatus{}, err
	}
	return NewApp().GetRuntimeStatus(), nil
}

func (c runtimeClient) fetchEnvelope(path string, target any) error {
	response, err := c.client.Get(c.baseURL + path)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return decodeEnvelope(response, target)
}

func (c runtimeClient) postEnvelope(path string, body any, target any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return decodeEnvelope(response, target)
}

func decodeEnvelope(response *http.Response, target any) error {
	if response.StatusCode >= http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		if len(body) == 0 {
			return fmt.Errorf("request failed: %s", response.Status)
		}
		return fmt.Errorf("request failed: %s %s", response.Status, strings.TrimSpace(string(body)))
	}

	switch typed := target.(type) {
	case *apiStatus:
		var envelope apiEnvelope[apiStatus]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiSkill `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiSkill `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiTool `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiTool `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiMCPServer `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiMCPServer `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiThread `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiThread `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiTask `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiTask `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiEvent `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiEvent `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiMessage `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiMessage `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiToolCall `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiToolCall `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiProvider `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiProvider `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *struct {
		Items []apiArtifact `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiArtifact `json:"items"`
		}]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *apiBridgeCheck:
		var envelope apiEnvelope[apiBridgeCheck]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	case *map[string]any:
		var envelope apiEnvelope[map[string]any]
		if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
			return err
		}
		if envelope.Code != 0 {
			return fmt.Errorf("request failed: %s", envelope.Message)
		}
		*typed = envelope.Data
	default:
		return fmt.Errorf("unsupported envelope target")
	}

	return nil
}

func groupSkills(items []apiSkill) map[string][]string {
	groups := map[string][]string{}
	for _, item := range items {
		group := fallbackText(strings.TrimSpace(item.Group), "common")
		groups[group] = append(groups[group], item.ID)
	}
	return normalizeGroups(groups)
}

func groupTools(items []apiTool) map[string][]string {
	groups := map[string][]string{}
	for _, item := range items {
		group := fallbackText(strings.TrimSpace(item.Source), "runtime")
		label := formatToolLabel(item.ID, item.Kind, item.Permission, item.Executable, item.ReadOnly)
		groups[group] = append(groups[group], label)
	}
	return normalizeGroups(groups)
}

func formatToolLabel(id string, kind string, permission string, executable bool, readOnly bool) string {
	parts := make([]string, 0, 4)
	if trimmedKind := strings.TrimSpace(kind); trimmedKind != "" {
		parts = append(parts, trimmedKind)
	}
	if trimmedPermission := strings.TrimSpace(permission); trimmedPermission != "" {
		parts = append(parts, trimmedPermission)
	}
	if executable {
		parts = append(parts, "executable")
	} else {
		parts = append(parts, "descriptor")
	}
	if readOnly {
		parts = append(parts, "read-only")
	}
	if len(parts) == 0 {
		return id
	}
	return fmt.Sprintf("%s (%s)", id, strings.Join(parts, ", "))
}

func mapProviders(items []apiProvider) []ProviderSummary {
	result := make([]ProviderSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ProviderSummary{
			Kind:              item.Kind,
			Enabled:           item.Enabled,
			BaseURL:           item.BaseURL,
			DefaultModel:      item.DefaultModel,
			HasAuthToken:      item.HasAuthToken,
			SupportsChat:      item.SupportsChat,
			SupportsResponses: item.SupportsResponses,
			PreferredAPIStyle: item.PreferredAPIStyle,
			Recommended:       item.Recommended,
			RecommendedReason: item.RecommendedReason,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Recommended != result[j].Recommended {
			return result[i].Recommended
		}
		return result[i].Kind < result[j].Kind
	})
	return result
}

func localProviderCatalog() []ProviderSummary {
	return []ProviderSummary{
		{
			Kind:              "anthropic",
			Enabled:           strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")) != "" || strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN")) != "",
			BaseURL:           strings.TrimSpace(os.Getenv("ANTHROPIC_BASE_URL")),
			DefaultModel:      strings.TrimSpace(os.Getenv("ANTHROPIC_MODEL")),
			HasAuthToken:      strings.TrimSpace(os.Getenv("ANTHROPIC_AUTH_TOKEN")) != "",
			SupportsChat:      false,
			SupportsResponses: true,
			PreferredAPIStyle: "openai-responses",
			Recommended:       true,
			RecommendedReason: "Current gateway is wired through OpenAI Responses compatibility.",
		},
	}
}

func localToolCatalog() []apiTool {
	return []apiTool{
		{ID: "workspace.read_file", Name: "Read File", Description: "Read a file from workspace", Permission: "read-only", Source: "runtime", Kind: "workspace.read_file", ReadOnly: true, Executable: true},
		{ID: "workspace.list_files", Name: "List Files", Description: "List files from workspace", Permission: "read-only", Source: "runtime", Kind: "workspace.list_files", ReadOnly: true, Executable: true},
		{ID: "workspace.search_text", Name: "Search Text", Description: "Search text in workspace", Permission: "read-only", Source: "runtime", Kind: "workspace.search_text", ReadOnly: true, Executable: true},
		{ID: "workspace.apply_patch", Name: "Apply Patch", Description: "Apply an approved text patch inside the workspace", Permission: "ask-user", Source: "runtime", Kind: "workspace.apply_patch", ReadOnly: false, Executable: true},
		{ID: "thread.message.append", Name: "Append Message", Description: "Append a thread-local message", Permission: "workspace-write", Source: "runtime", Kind: "thread.message.append", ReadOnly: false, Executable: true},
	}
}

func groupMCPServers(items []apiMCPServer) map[string][]string {
	groups := map[string][]string{}
	for _, item := range items {
		group := fallbackText(strings.TrimSpace(item.Source), "runtime")
		label := fmt.Sprintf("%s (tools:%d resources:%d)", item.ID, item.ToolCount, item.ResourceCount)
		groups[group] = append(groups[group], label)
	}
	return normalizeGroups(groups)
}

func normalizeGroups(groups map[string][]string) map[string][]string {
	for key, items := range groups {
		sort.Strings(items)
		deduped := make([]string, 0, len(items))
		last := ""
		for _, item := range items {
			if item == "" || item == last {
				continue
			}
			deduped = append(deduped, item)
			last = item
		}
		groups[key] = deduped
	}
	return groups
}

func mapThreads(items []apiThread) []ThreadSummary {
	result := make([]ThreadSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ThreadSummary{
			ID:             item.ID,
			Name:           item.Name,
			Status:         item.Status,
			ActiveModel:    item.ActiveModel,
			PermissionMode: item.PermissionMode,
			IsActive:       item.IsActive,
		})
	}
	return result
}

func mapTasks(items []apiTask) []TaskSummary {
	result := make([]TaskSummary, 0, len(items))
	for _, item := range items {
		result = append(result, TaskSummary{
			ID:             item.ID,
			ThreadID:       item.ThreadID,
			Title:          item.Title,
			Kind:           item.Kind,
			Input:          item.Input,
			Status:         item.Status,
			ResultSummary:  item.ResultSummary,
			ApprovalStatus: item.ApprovalStatus,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		})
	}
	return result
}

func mapApprovals(items []apiApproval) []ApprovalSummary {
	result := make([]ApprovalSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ApprovalSummary{
			ID:          item.ID,
			ThreadID:    item.ThreadID,
			TaskID:      item.TaskID,
			ToolKind:    item.ToolKind,
			Status:      item.Status,
			Summary:     item.Summary,
			TargetPaths: append([]string(nil), item.TargetPaths...),
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return result
}

func mapMessages(items []apiMessage) []MessageSummary {
	result := make([]MessageSummary, 0, len(items))
	for _, item := range items {
		result = append(result, MessageSummary{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Role:      item.Role,
			Content:   item.Content,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func mapToolCalls(items []apiToolCall) []ToolCallSummary {
	result := make([]ToolCallSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ToolCallSummary{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			ToolID:    item.ToolID,
			Status:    item.Status,
			Summary:   item.Summary,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func mapArtifacts(items []apiArtifact) []ArtifactSummary {
	result := make([]ArtifactSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ArtifactSummary{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Path:      item.Path,
			Kind:      item.Kind,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func mapEvents(items []apiEvent) []EventSummary {
	result := make([]EventSummary, 0, len(items))
	for _, item := range items {
		result = append(result, EventSummary{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Type:      item.Type,
			Message:   item.Message,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func newLocalRuntimeStore() *localRuntimeStore {
	return &localRuntimeStore{}
}

func (s *localRuntimeStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *localRuntimeStore) Snapshot(base RuntimeStatus) (RuntimeStatus, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDB(base.WorkspaceRoot); err != nil {
		return RuntimeStatus{}, err
	}
	return s.snapshotLocked(base)
}

func (s *localRuntimeStore) CreateThread(name string) RuntimeStatus {
	base, err := buildBaseStatusFromStore()
	if err != nil {
		return runtimeErrorStatus(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDB(base.WorkspaceRoot); err != nil {
		return runtimeErrorStatus(err)
	}

	threadID := fmt.Sprintf("local-thread-%d", time.Now().UnixNano())
	threadName := fallbackText(strings.TrimSpace(name), defaultThreadName)
	workspaceID := fallbackText(strings.TrimSpace(base.WorkspaceID), filepath.Base(base.WorkspaceRoot))
	now := time.Now().Format(time.RFC3339)

	tx, err := s.db.Begin()
	if err != nil {
		return runtimeErrorStatus(err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO threads(id, workspace_id, name, status, active_model, permission_mode, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
	`, threadID, workspaceID, threadName, "idle", "", "ask-user", now); err != nil {
		return runtimeErrorStatus(err)
	}
	if err := s.saveWorkspace(tx, base.WorkspaceRoot, threadID); err != nil {
		return runtimeErrorStatus(err)
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-event"), threadID, "thread.created", fmt.Sprintf("Created thread %s", threadName), now); err != nil {
		return runtimeErrorStatus(err)
	}
	if err := tx.Commit(); err != nil {
		return runtimeErrorStatus(err)
	}

	status, err := s.snapshotLocked(*base)
	if err != nil {
		return runtimeErrorStatus(err)
	}
	return status
}

func (s *localRuntimeStore) ActivateThread(id string) RuntimeStatus {
	base, err := buildBaseStatusFromStore()
	if err != nil {
		return runtimeErrorStatus(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDB(base.WorkspaceRoot); err != nil {
		return runtimeErrorStatus(err)
	}

	threadID := strings.TrimSpace(id)
	if threadID == "" {
		status, snapshotErr := s.snapshotLocked(*base)
		if snapshotErr != nil {
			return runtimeErrorStatus(snapshotErr)
		}
		return status
	}

	now := time.Now().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return runtimeErrorStatus(err)
	}
	defer tx.Rollback()

	if err := s.saveWorkspace(tx, base.WorkspaceRoot, threadID); err != nil {
		return runtimeErrorStatus(err)
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-event"), threadID, "thread.activated", fmt.Sprintf("Activated thread %s", threadID), now); err != nil {
		return runtimeErrorStatus(err)
	}
	if err := tx.Commit(); err != nil {
		return runtimeErrorStatus(err)
	}

	status, err := s.snapshotLocked(*base)
	if err != nil {
		return runtimeErrorStatus(err)
	}
	return status
}

func (s *localRuntimeStore) CreateTask(threadID string, input TaskCreateInput) RuntimeStatus {
	base, err := buildBaseStatusFromStore()
	if err != nil {
		return runtimeErrorStatus(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDB(base.WorkspaceRoot); err != nil {
		return runtimeErrorStatus(err)
	}

	activeThreadID, err := s.readActiveThreadID()
	if err != nil {
		return runtimeErrorStatus(err)
	}
	trimmedThreadID := fallbackText(strings.TrimSpace(threadID), activeThreadID)
	if trimmedThreadID == "" {
		status, snapshotErr := s.snapshotLocked(*base)
		if snapshotErr != nil {
			return runtimeErrorStatus(snapshotErr)
		}
		return status
	}

	normalized := normalizeTaskCreateInput(input)
	now := time.Now().Format(time.RFC3339)
	taskID := nextLocalID("local-task")
	permissionMode, err := s.readThreadPermissionMode(trimmedThreadID)
	if err != nil {
		return runtimeErrorStatus(err)
	}

	tx, err := s.db.Begin()
	if err != nil {
		return runtimeErrorStatus(err)
	}
	defer tx.Rollback()

	statusValue := "queued"
	resultSummary := ""
	approvalStatus := ""
	eventType := "task.created"
	eventMessage := fmt.Sprintf("Created task %s (%s)", normalized.Title, normalized.Kind)
	if normalized.Kind == "workspace.apply_patch" {
		pathValue, patchValue, parseErr := parseLocalPatchInput(normalized.Input)
		if parseErr != nil {
			return runtimeErrorStatus(parseErr)
		}
		targets, targetErr := extractLocalPatchTargets(patchValue)
		if targetErr != nil {
			return runtimeErrorStatus(targetErr)
		}
		summary := fmt.Sprintf("%s; %s", localApprovalSummary(normalized.Kind, targets), localTruncatedPatchSummary(patchValue, 120))
		switch permissionMode {
		case "read-only":
			statusValue = "failed"
			resultSummary = "permission denied: read-only mode does not allow workspace writes"
			approvalStatus = "rejected"
			eventType = "task.failed"
			eventMessage = resultSummary
		case "", "ask-user":
			statusValue = "needs_approval"
			resultSummary = summary
			approvalStatus = "pending"
			eventType = "task.approval_required"
			eventMessage = summary
		default:
			statusValue = "queued"
			approvalStatus = "direct"
		}
		_ = pathValue
	}

	if _, err := tx.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, taskID, trimmedThreadID, normalized.Title, normalized.Kind, normalized.Input, statusValue, resultSummary, approvalStatus, now, now); err != nil {
		return runtimeErrorStatus(err)
	}
	if _, err := tx.Exec(`
		INSERT INTO thread_messages(id, thread_id, role, content, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-message"), trimmedThreadID, "user", normalized.Input, now); err != nil {
		return runtimeErrorStatus(err)
	}
	if normalized.Kind == "workspace.apply_patch" && approvalStatus == "pending" {
		pathValue, patchValue, _ := parseLocalPatchInput(normalized.Input)
		targets, _ := extractLocalPatchTargets(patchValue)
		targetJSON := encodeTargetPaths(targets)
		if len(targets) == 0 {
			targetJSON = encodeTargetPaths([]string{pathValue})
		}
		if _, err := tx.Exec(`
			INSERT INTO thread_approvals(id, thread_id, task_id, tool_kind, status, summary, target_paths, created_at, updated_at)
			VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, nextLocalID("local-approval"), trimmedThreadID, taskID, normalized.Kind, "pending", resultSummary, targetJSON, now, now); err != nil {
			return runtimeErrorStatus(err)
		}
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-event"), trimmedThreadID, eventType, eventMessage, now); err != nil {
		return runtimeErrorStatus(err)
	}
	if err := tx.Commit(); err != nil {
		return runtimeErrorStatus(err)
	}

	status, err := s.snapshotLocked(*base)
	if err != nil {
		return runtimeErrorStatus(err)
	}
	return status
}

func (s *localRuntimeStore) RunTask(taskID string) RuntimeStatus {
	base, err := buildBaseStatusFromStore()
	if err != nil {
		return runtimeErrorStatus(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDB(base.WorkspaceRoot); err != nil {
		return runtimeErrorStatus(err)
	}

	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedTaskID == "" {
		status, snapshotErr := s.snapshotLocked(*base)
		if snapshotErr != nil {
			return runtimeErrorStatus(snapshotErr)
		}
		return status
	}

	row := s.db.QueryRow(`SELECT thread_id, title, kind, input, status, approval_status FROM tasks WHERE id = ?`, trimmedTaskID)
	var threadID string
	var title string
	var kind string
	var input string
	var currentStatus string
	var approvalStatus string
	if err := row.Scan(&threadID, &title, &kind, &input, &currentStatus, &approvalStatus); err != nil {
		status, snapshotErr := s.snapshotLocked(*base)
		if snapshotErr != nil {
			return runtimeErrorStatus(snapshotErr)
		}
		return status
	}

	now := time.Now().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return runtimeErrorStatus(err)
	}
	defer tx.Rollback()

	if kind == "workspace.apply_patch" {
		if currentStatus == "needs_approval" {
			return runtimeErrorStatus(fmt.Errorf("task approval required"))
		}
		if err := s.executePatchTask(tx, base.WorkspaceRoot, trimmedTaskID, threadID, title, input, approvalStatus, now); err != nil {
			return runtimeErrorStatus(err)
		}
	} else {
		resultSummary := buildTaskResultSummary(kind, input, currentStatus)
		if _, err := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, updated_at = ? WHERE id = ?`, "completed", resultSummary, now, trimmedTaskID); err != nil {
			return runtimeErrorStatus(err)
		}
		if _, err := tx.Exec(`
			INSERT INTO thread_tool_calls(id, thread_id, tool_id, status, summary, created_at)
			VALUES(?, ?, ?, ?, ?, ?)
		`, nextLocalID("local-tool-call"), threadID, "task.run", "completed", resultSummary, now); err != nil {
			return runtimeErrorStatus(err)
		}
		if _, err := tx.Exec(`
			INSERT INTO thread_messages(id, thread_id, role, content, created_at)
			VALUES(?, ?, ?, ?, ?)
		`, nextLocalID("local-message"), threadID, "assistant", resultSummary, now); err != nil {
			return runtimeErrorStatus(err)
		}
		if _, err := tx.Exec(`
			INSERT INTO events(id, thread_id, type, message, created_at)
			VALUES(?, ?, ?, ?, ?)
		`, nextLocalID("local-event"), threadID, "task.run.completed", fmt.Sprintf("Ran task %s and completed it", title), now); err != nil {
			return runtimeErrorStatus(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return runtimeErrorStatus(err)
	}

	status, err := s.snapshotLocked(*base)
	if err != nil {
		return runtimeErrorStatus(err)
	}
	return status
}

func (s *localRuntimeStore) ensureDB(workspaceRoot string) error {
	if s.db != nil {
		return nil
	}

	statePath := defaultStateStorePath(workspaceRoot)
	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return err
	}

	db, err := sql.Open("sqlite", statePath)
	if err != nil {
		return err
	}

	schema := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS workspace (
			id TEXT PRIMARY KEY,
			project_root TEXT NOT NULL,
			shared_docs_root TEXT NOT NULL,
			created_at TEXT NOT NULL,
			active_thread_id TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS threads (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			active_model TEXT NOT NULL DEFAULT '',
			permission_mode TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			title TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT 'prompt',
			input TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			result_summary TEXT NOT NULL DEFAULT '',
			approval_status TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS thread_messages (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS thread_tool_calls (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			tool_id TEXT NOT NULL,
			status TEXT NOT NULL,
			summary TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS thread_artifacts (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			path TEXT NOT NULL,
			kind TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS thread_runtime_flags (
			thread_id TEXT NOT NULL,
			key TEXT NOT NULL,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(thread_id, key)
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			type TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS thread_approvals (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			task_id TEXT NOT NULL,
			tool_kind TEXT NOT NULL,
			status TEXT NOT NULL,
			summary TEXT NOT NULL,
			target_paths TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			return err
		}
	}
	migrations := []string{
		`ALTER TABLE tasks ADD COLUMN kind TEXT NOT NULL DEFAULT 'prompt'`,
		`ALTER TABLE tasks ADD COLUMN input TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN result_summary TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN approval_status TEXT NOT NULL DEFAULT ''`,
	}
	for _, statement := range migrations {
		if _, err := db.Exec(statement); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			db.Close()
			return err
		}
	}

	now := time.Now().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO schema_version(version) SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM schema_version)`); err != nil {
		db.Close()
		return err
	}
	if _, err := db.Exec(`
		INSERT INTO workspace(id, project_root, shared_docs_root, created_at, active_thread_id)
		VALUES(?, ?, ?, ?, '')
		ON CONFLICT(id) DO UPDATE SET
			project_root=excluded.project_root,
			shared_docs_root=excluded.shared_docs_root
	`, filepath.Base(workspaceRoot), workspaceRoot, filepath.Join(workspaceRoot, "docs"), now); err != nil {
		db.Close()
		return err
	}

	s.db = db
	return nil
}

func (s *localRuntimeStore) snapshotLocked(base RuntimeStatus) (RuntimeStatus, error) {
	workspaceRecord, err := s.readWorkspace()
	if err != nil {
		return RuntimeStatus{}, err
	}

	threads, err := s.readThreads(workspaceRecord.ID, workspaceRecord.ActiveThreadID)
	if err != nil {
		return RuntimeStatus{}, err
	}
	tasks, err := s.readTasks(workspaceRecord.ActiveThreadID)
	if err != nil {
		return RuntimeStatus{}, err
	}
	messages, err := s.readMessages(workspaceRecord.ActiveThreadID)
	if err != nil {
		return RuntimeStatus{}, err
	}
	toolCalls, err := s.readToolCalls(workspaceRecord.ActiveThreadID)
	if err != nil {
		return RuntimeStatus{}, err
	}
	artifacts, err := s.readArtifacts(workspaceRecord.ActiveThreadID)
	if err != nil {
		return RuntimeStatus{}, err
	}
	events, err := s.readEvents(workspaceRecord.ActiveThreadID)
	if err != nil {
		return RuntimeStatus{}, err
	}
	approvals, err := s.readApprovals(workspaceRecord.ActiveThreadID)
	if err != nil {
		return RuntimeStatus{}, err
	}

	status := base
	status.WorkspaceID = fallbackText(workspaceRecord.ID, filepath.Base(base.WorkspaceRoot))
	status.ProjectRoot = fallbackText(workspaceRecord.ProjectRoot, base.WorkspaceRoot)
	status.ThreadCount = len(threads)
	status.ActiveThreadID = workspaceRecord.ActiveThreadID
	status.Threads = toThreadSummaries(threads)
	status.Tasks = toTaskSummaries(tasks)
	status.Messages = toMessageSummaries(messages)
	status.ToolCalls = toToolCallSummaries(toolCalls)
	status.Artifacts = toArtifactSummaries(artifacts)
	status.Events = toEventSummaries(events)
	status.Approvals = toApprovalSummaries(approvals)
	status.ToolsByGroup = groupTools(localToolCatalog())
	status.Providers = localProviderCatalog()
	status.RuntimeState = "fallback"
	status.RuntimeReady = true
	status.RuntimeMessage = "Using project-local SQLite runtime fallback because no external runtime is connected."
	status.RuntimeSource = "desktop-local"
	status.SupportsSSE = false
	status.SSEEndpoint = ""
	status.LastSyncAt = ""
	status.StateStore = "sqlite"
	status.StatePath = defaultStateStorePath(base.WorkspaceRoot)
	status.UsesProjectLocalStore = true
	status.RecoverySummary = fmt.Sprintf("Recovered %d thread(s), %d task(s), %d approval(s), %d message(s), %d tool call(s), %d artifact(s), %d event(s) from project-local state store.", len(threads), len(tasks), len(approvals), len(messages), len(toolCalls), len(artifacts), len(events))
	status.UpdatedAt = time.Now().Format(time.RFC3339)
	return status, nil
}

func (s *localRuntimeStore) readThreads(workspaceID string, activeThreadID string) ([]persistedThread, error) {
	rows, err := s.db.Query(`
		SELECT id, name, status, active_model, permission_mode
		FROM threads
		WHERE workspace_id = ?
		ORDER BY datetime(created_at) DESC, id DESC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedThread{}
	for rows.Next() {
		var item persistedThread
		if err := rows.Scan(&item.ID, &item.Name, &item.Status, &item.ActiveModel, &item.PermissionMode); err != nil {
			return nil, err
		}
		item.IsActive = item.ID == activeThreadID
		result = append(result, item)
	}
	return result, rows.Err()
}

type persistedWorkspace struct {
	ID             string
	ProjectRoot    string
	ActiveThreadID string
}

func (s *localRuntimeStore) readWorkspace() (persistedWorkspace, error) {
	row := s.db.QueryRow(`
		SELECT id, project_root, active_thread_id
		FROM workspace
		ORDER BY created_at ASC
		LIMIT 1
	`)
	var item persistedWorkspace
	if err := row.Scan(&item.ID, &item.ProjectRoot, &item.ActiveThreadID); err != nil {
		if err == sql.ErrNoRows {
			return persistedWorkspace{}, nil
		}
		return persistedWorkspace{}, err
	}
	return item, nil
}

func (s *localRuntimeStore) readActiveThreadID() (string, error) {
	workspaceRecord, err := s.readWorkspace()
	if err != nil {
		return "", err
	}
	return workspaceRecord.ActiveThreadID, nil
}

func (s *localRuntimeStore) saveWorkspace(tx *sql.Tx, workspaceRoot string, activeThreadID string) error {
	workspaceID := filepath.Base(workspaceRoot)
	_, err := tx.Exec(`
		INSERT INTO workspace(id, project_root, shared_docs_root, created_at, active_thread_id)
		VALUES(?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			project_root=excluded.project_root,
			shared_docs_root=excluded.shared_docs_root,
			active_thread_id=excluded.active_thread_id
	`, workspaceID, workspaceRoot, filepath.Join(workspaceRoot, "docs"), time.Now().Format(time.RFC3339), activeThreadID)
	return err
}

func (s *localRuntimeStore) readTasks(threadID string) ([]persistedTask, error) {
	query := `
		SELECT id, thread_id, title, kind, input, status, result_summary, approval_status, created_at, updated_at
		FROM tasks
	`
	args := []any{}
	if strings.TrimSpace(threadID) != "" {
		query += ` WHERE thread_id = ?`
		args = append(args, threadID)
	}
	query += ` ORDER BY datetime(updated_at) DESC, id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedTask{}
	for rows.Next() {
		var item persistedTask
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.Title, &item.Kind, &item.Input, &item.Status, &item.ResultSummary, &item.ApprovalStatus, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *localRuntimeStore) readApprovals(threadID string) ([]persistedApproval, error) {
	query := `
		SELECT id, thread_id, task_id, tool_kind, status, summary, target_paths, created_at, updated_at
		FROM thread_approvals
	`
	args := []any{}
	if strings.TrimSpace(threadID) != "" {
		query += ` WHERE thread_id = ?`
		args = append(args, threadID)
	}
	query += ` ORDER BY datetime(updated_at) DESC, id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedApproval{}
	for rows.Next() {
		var item persistedApproval
		var targetPaths string
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.TaskID, &item.ToolKind, &item.Status, &item.Summary, &targetPaths, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.TargetPaths = decodeTargetPaths(targetPaths)
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *localRuntimeStore) readMessages(threadID string) ([]persistedMessage, error) {
	query := `
		SELECT id, thread_id, role, content, created_at
		FROM thread_messages
	`
	args := []any{}
	if strings.TrimSpace(threadID) != "" {
		query += ` WHERE thread_id = ?`
		args = append(args, threadID)
	}
	query += ` ORDER BY datetime(created_at) DESC, id DESC LIMIT 12`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedMessage{}
	for rows.Next() {
		var item persistedMessage
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.Role, &item.Content, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *localRuntimeStore) readToolCalls(threadID string) ([]persistedToolCall, error) {
	query := `
		SELECT id, thread_id, tool_id, status, summary, created_at
		FROM thread_tool_calls
	`
	args := []any{}
	if strings.TrimSpace(threadID) != "" {
		query += ` WHERE thread_id = ?`
		args = append(args, threadID)
	}
	query += ` ORDER BY datetime(created_at) DESC, id DESC LIMIT 12`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedToolCall{}
	for rows.Next() {
		var item persistedToolCall
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.ToolID, &item.Status, &item.Summary, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *localRuntimeStore) readArtifacts(threadID string) ([]persistedArtifact, error) {
	query := `
		SELECT id, thread_id, path, kind, created_at
		FROM thread_artifacts
	`
	args := []any{}
	if strings.TrimSpace(threadID) != "" {
		query += ` WHERE thread_id = ?`
		args = append(args, threadID)
	}
	query += ` ORDER BY datetime(created_at) DESC, id DESC LIMIT 12`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedArtifact{}
	for rows.Next() {
		var item persistedArtifact
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.Path, &item.Kind, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (s *localRuntimeStore) readEvents(threadID string) ([]persistedEvent, error) {
	query := `
		SELECT id, thread_id, type, message, created_at
		FROM events
	`
	args := []any{}
	if strings.TrimSpace(threadID) != "" {
		query += ` WHERE thread_id = ?`
		args = append(args, threadID)
	}
	query += ` ORDER BY datetime(created_at) DESC, id DESC LIMIT 24`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedEvent{}
	for rows.Next() {
		var item persistedEvent
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.Type, &item.Message, &item.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func toThreadSummaries(items []persistedThread) []ThreadSummary {
	result := make([]ThreadSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ThreadSummary{
			ID:             item.ID,
			Name:           item.Name,
			Status:         item.Status,
			ActiveModel:    item.ActiveModel,
			PermissionMode: item.PermissionMode,
			IsActive:       item.IsActive,
		})
	}
	return result
}

func toTaskSummaries(items []persistedTask) []TaskSummary {
	result := make([]TaskSummary, 0, len(items))
	for _, item := range items {
		result = append(result, TaskSummary{
			ID:             item.ID,
			ThreadID:       item.ThreadID,
			Title:          item.Title,
			Kind:           item.Kind,
			Input:          item.Input,
			Status:         item.Status,
			ResultSummary:  item.ResultSummary,
			ApprovalStatus: item.ApprovalStatus,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		})
	}
	return result
}

func toApprovalSummaries(items []persistedApproval) []ApprovalSummary {
	result := make([]ApprovalSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ApprovalSummary{
			ID:          item.ID,
			ThreadID:    item.ThreadID,
			TaskID:      item.TaskID,
			ToolKind:    item.ToolKind,
			Status:      item.Status,
			Summary:     item.Summary,
			TargetPaths: append([]string(nil), item.TargetPaths...),
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return result
}

func (s *localRuntimeStore) ApproveTask(threadID string, taskID string) RuntimeStatus {
	return s.handleApprovalAction(threadID, taskID, true)
}

func (s *localRuntimeStore) RejectTask(threadID string, taskID string) RuntimeStatus {
	return s.handleApprovalAction(threadID, taskID, false)
}

func (s *localRuntimeStore) handleApprovalAction(threadID string, taskID string, approved bool) RuntimeStatus {
	base, err := buildBaseStatusFromStore()
	if err != nil {
		return runtimeErrorStatus(err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureDB(base.WorkspaceRoot); err != nil {
		return runtimeErrorStatus(err)
	}

	trimmedThreadID := strings.TrimSpace(threadID)
	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedThreadID == "" || trimmedTaskID == "" {
		return runtimeErrorStatus(fmt.Errorf("thread id and task id are required"))
	}

	task, err := s.readTaskByID(trimmedTaskID)
	if err != nil {
		return runtimeErrorStatus(err)
	}
	if task.ThreadID != trimmedThreadID {
		return runtimeErrorStatus(fmt.Errorf("task does not belong to thread"))
	}
	approval, err := s.readApprovalByTask(trimmedThreadID, trimmedTaskID)
	if err != nil {
		return runtimeErrorStatus(err)
	}

	now := time.Now().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return runtimeErrorStatus(err)
	}
	defer tx.Rollback()

	if approved {
		if _, err := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, approval_status = ?, updated_at = ? WHERE id = ?`, "queued", approval.Summary, "approved", now, trimmedTaskID); err != nil {
			return runtimeErrorStatus(err)
		}
		if _, err := tx.Exec(`UPDATE thread_approvals SET status = ?, summary = ?, updated_at = ? WHERE task_id = ?`, "approved", approval.Summary, now, trimmedTaskID); err != nil {
			return runtimeErrorStatus(err)
		}
		if err := insertEventTx(tx, trimmedThreadID, "task.approved", approval.Summary, now); err != nil {
			return runtimeErrorStatus(err)
		}
		if err := insertEventTx(tx, trimmedThreadID, "toolcall.approved", approval.Summary, now); err != nil {
			return runtimeErrorStatus(err)
		}
		if task.Kind == "workspace.apply_patch" {
			if err := s.executePatchTask(tx, base.WorkspaceRoot, trimmedTaskID, trimmedThreadID, task.Title, task.Input, "approved", now); err != nil {
				return runtimeErrorStatus(err)
			}
			if _, err := tx.Exec(`UPDATE thread_approvals SET status = ?, summary = ?, updated_at = ? WHERE task_id = ?`, "executed", readTaskResultSummaryTx(tx, trimmedTaskID), now, trimmedTaskID); err != nil {
				return runtimeErrorStatus(err)
			}
			if _, err := tx.Exec(`UPDATE tasks SET approval_status = ?, updated_at = ? WHERE id = ?`, "executed", now, trimmedTaskID); err != nil {
				return runtimeErrorStatus(err)
			}
		}
	} else {
		rejectedSummary := approval.Summary
		if rejectedSummary == "" {
			rejectedSummary = "approval rejected"
		}
		rejectedSummary = "approval rejected: " + rejectedSummary
		if _, err := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, approval_status = ?, updated_at = ? WHERE id = ?`, "failed", rejectedSummary, "rejected", now, trimmedTaskID); err != nil {
			return runtimeErrorStatus(err)
		}
		if _, err := tx.Exec(`UPDATE thread_approvals SET status = ?, summary = ?, updated_at = ? WHERE task_id = ?`, "rejected", rejectedSummary, now, trimmedTaskID); err != nil {
			return runtimeErrorStatus(err)
		}
		if err := insertEventTx(tx, trimmedThreadID, "task.rejected", rejectedSummary, now); err != nil {
			return runtimeErrorStatus(err)
		}
		if err := insertEventTx(tx, trimmedThreadID, "toolcall.rejected", rejectedSummary, now); err != nil {
			return runtimeErrorStatus(err)
		}
	}

	if err := tx.Commit(); err != nil {
		return runtimeErrorStatus(err)
	}

	status, err := s.snapshotLocked(*base)
	if err != nil {
		return runtimeErrorStatus(err)
	}
	return status
}

func (s *localRuntimeStore) readThreadPermissionMode(threadID string) (string, error) {
	row := s.db.QueryRow(`SELECT permission_mode FROM threads WHERE id = ?`, threadID)
	var mode string
	if err := row.Scan(&mode); err != nil {
		return "", err
	}
	return strings.TrimSpace(mode), nil
}

func (s *localRuntimeStore) readTaskByID(taskID string) (persistedTask, error) {
	row := s.db.QueryRow(`
		SELECT id, thread_id, title, kind, input, status, result_summary, approval_status, created_at, updated_at
		FROM tasks
		WHERE id = ?
	`, taskID)
	var item persistedTask
	if err := row.Scan(&item.ID, &item.ThreadID, &item.Title, &item.Kind, &item.Input, &item.Status, &item.ResultSummary, &item.ApprovalStatus, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return persistedTask{}, err
	}
	return item, nil
}

func (s *localRuntimeStore) readApprovalByTask(threadID string, taskID string) (persistedApproval, error) {
	row := s.db.QueryRow(`
		SELECT id, thread_id, task_id, tool_kind, status, summary, target_paths, created_at, updated_at
		FROM thread_approvals
		WHERE thread_id = ? AND task_id = ?
		ORDER BY datetime(updated_at) DESC
		LIMIT 1
	`, threadID, taskID)
	var item persistedApproval
	var targetPaths string
	if err := row.Scan(&item.ID, &item.ThreadID, &item.TaskID, &item.ToolKind, &item.Status, &item.Summary, &targetPaths, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return persistedApproval{}, err
	}
	item.TargetPaths = decodeTargetPaths(targetPaths)
	return item, nil
}

func insertEventTx(tx *sql.Tx, threadID string, eventType string, message string, createdAt string) error {
	_, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-event"), threadID, eventType, message, createdAt)
	return err
}

func readTaskResultSummaryTx(tx *sql.Tx, taskID string) string {
	row := tx.QueryRow(`SELECT result_summary FROM tasks WHERE id = ?`, taskID)
	var summary string
	if err := row.Scan(&summary); err != nil {
		return ""
	}
	return summary
}

func (s *localRuntimeStore) executePatchTask(tx *sql.Tx, workspaceRoot string, taskID string, threadID string, title string, rawInput string, approvalStatus string, now string) error {
	pathValue, patchValue, err := parseLocalPatchInput(rawInput)
	if err != nil {
		return err
	}
	resolvedPath, err := resolveLocalWorkspacePath(workspaceRoot, pathValue)
	if err != nil {
		return err
	}
	if approvalStatus != "approved" && approvalStatus != "executed" && approvalStatus != "direct" {
		return fmt.Errorf("task approval required")
	}

	resultSummary, err := applyLocalWorkspacePatch(resolvedPath, patchValue, workspaceRoot)
	if err != nil {
		if _, updateErr := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, updated_at = ? WHERE id = ?`, "failed", err.Error(), now, taskID); updateErr != nil {
			return updateErr
		}
		if _, toolErr := tx.Exec(`
			INSERT INTO thread_tool_calls(id, thread_id, tool_id, status, summary, created_at)
			VALUES(?, ?, ?, ?, ?, ?)
		`, nextLocalID("local-tool-call"), threadID, "workspace.apply_patch", "failed", err.Error(), now); toolErr != nil {
			return toolErr
		}
		if eventErr := insertEventTx(tx, threadID, "toolcall.failed", err.Error(), now); eventErr != nil {
			return eventErr
		}
		return insertEventTx(tx, threadID, "task.failed", err.Error(), now)
	}

	if _, err := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, updated_at = ? WHERE id = ?`, "completed", resultSummary, now, taskID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO thread_tool_calls(id, thread_id, tool_id, status, summary, created_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, nextLocalID("local-tool-call"), threadID, "workspace.apply_patch", "completed", resultSummary, now); err != nil {
		return err
	}
	if err := insertEventTx(tx, threadID, "toolcall.completed", resultSummary, now); err != nil {
		return err
	}
	if err := insertEventTx(tx, threadID, "task.completed", fmt.Sprintf("Ran task %s and completed it", title), now); err != nil {
		return err
	}
	return nil
}

func toMessageSummaries(items []persistedMessage) []MessageSummary {
	result := make([]MessageSummary, 0, len(items))
	for _, item := range items {
		result = append(result, MessageSummary{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Role:      item.Role,
			Content:   item.Content,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func toToolCallSummaries(items []persistedToolCall) []ToolCallSummary {
	result := make([]ToolCallSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ToolCallSummary{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			ToolID:    item.ToolID,
			Status:    item.Status,
			Summary:   item.Summary,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func toArtifactSummaries(items []persistedArtifact) []ArtifactSummary {
	result := make([]ArtifactSummary, 0, len(items))
	for _, item := range items {
		result = append(result, ArtifactSummary{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Path:      item.Path,
			Kind:      item.Kind,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func toEventSummaries(items []persistedEvent) []EventSummary {
	result := make([]EventSummary, 0, len(items))
	for _, item := range items {
		result = append(result, EventSummary{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Type:      item.Type,
			Message:   item.Message,
			CreatedAt: item.CreatedAt,
		})
	}
	return result
}

func buildBaseStatusFromStore() (*RuntimeStatus, error) {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return nil, err
	}
	return buildBaseStatus(workspaceRoot)
}

func parseTaskCreateInput(payload string) TaskCreateInput {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return normalizeTaskCreateInput(TaskCreateInput{})
	}

	var parsed TaskCreateInput
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		if err := json.Unmarshal([]byte(trimmed), &parsed); err == nil {
			return normalizeTaskCreateInput(parsed)
		}
	}

	return normalizeTaskCreateInput(TaskCreateInput{
		Title: trimmed,
		Input: trimmed,
	})
}

func normalizeTaskCreateInput(input TaskCreateInput) TaskCreateInput {
	kind := fallbackText(strings.TrimSpace(input.Kind), "prompt")
	taskInput := strings.TrimSpace(input.Input)
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = defaultTaskTitle
		if taskInput != "" {
			title = compactText(taskInput, 48)
		}
	}
	return TaskCreateInput{
		Title: title,
		Kind:  kind,
		Input: taskInput,
	}
}

func buildTaskResultSummary(kind string, input string, previousStatus string) string {
	kindLabel := fallbackText(strings.TrimSpace(kind), "prompt")
	inputLabel := compactText(input, 96)
	if inputLabel == "" {
		inputLabel = "no input"
	}
	if previousStatus == "completed" {
		return fmt.Sprintf("Task rerun completed for %s with %s.", kindLabel, inputLabel)
	}
	return fmt.Sprintf("Task completed for %s with %s.", kindLabel, inputLabel)
}

func parseLocalPatchInput(raw string) (string, string, error) {
	var input struct {
		Path  string `json:"path"`
		Patch string `json:"patch"`
	}
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(input.Path) == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if strings.TrimSpace(input.Patch) == "" {
		return "", "", fmt.Errorf("patch is required")
	}
	return strings.TrimSpace(input.Path), strings.TrimSpace(input.Patch), nil
}

func extractLocalPatchTargets(raw string) ([]string, error) {
	lines := splitLocalPatchLines(strings.TrimSpace(raw))
	targets := make([]string, 0)
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "*** Update File: "):
			targets = append(targets, strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: ")))
		case strings.HasPrefix(line, "*** Add File: "):
			targets = append(targets, strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: ")))
		case strings.HasPrefix(line, "*** Delete File: "):
			return nil, fmt.Errorf("delete patch is not allowed")
		}
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("patch does not declare target files")
	}
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(targets))
	for _, target := range targets {
		key := strings.ToLower(filepath.ToSlash(filepath.Clean(target)))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, target)
	}
	return unique, nil
}

func localApprovalSummary(kind string, targets []string) string {
	label := strings.TrimSpace(kind)
	if label == "" {
		label = "write"
	}
	if len(targets) == 0 {
		return fmt.Sprintf("approval required for %s", label)
	}
	return fmt.Sprintf("approval required for %s on %s", label, strings.Join(targets, ", "))
}

func localTruncatedPatchSummary(raw string, max int) string {
	lines := splitLocalPatchLines(strings.TrimSpace(raw))
	delta := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			delta++
		}
	}
	summary := fmt.Sprintf("%d patch line(s)", delta)
	return compactText(summary, max)
}

func splitLocalPatchLines(value string) []string {
	scanner := bufio.NewScanner(strings.NewReader(value))
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func splitLocalTextLines(value string) []string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	if normalized == "" {
		return []string{}
	}
	return strings.Split(normalized, "\n")
}

func resolveLocalWorkspacePath(workspaceRoot string, provided string) (string, error) {
	root, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", err
	}
	target := strings.TrimSpace(provided)
	if target == "" {
		target = "."
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	resolved, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside workspace")
	}
	return resolved, nil
}

func localWorkspaceRelative(workspaceRoot string, target string) string {
	relative, err := filepath.Rel(workspaceRoot, target)
	if err != nil || relative == "." {
		return "."
	}
	return filepath.ToSlash(relative)
}

func applyLocalWorkspacePatch(targetPath string, patch string, workspaceRoot string) (string, error) {
	trimmed := strings.TrimSpace(patch)
	if trimmed == "" {
		return "", fmt.Errorf("patch is required")
	}
	lines := splitLocalPatchLines(trimmed)
	if len(lines) < 2 || lines[0] != "*** Begin Patch" {
		return "", fmt.Errorf("unsupported patch format")
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "*** Delete File: ") {
			return "", fmt.Errorf("delete patch is not allowed")
		}
	}

	var op string
	var patchPath string
	start := -1
	for index, line := range lines {
		switch {
		case strings.HasPrefix(line, "*** Update File: "):
			op = "update"
			patchPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: "))
			start = index + 1
		case strings.HasPrefix(line, "*** Add File: "):
			op = "add"
			patchPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: "))
			start = index + 1
		}
		if start != -1 {
			break
		}
	}
	if op == "" || patchPath == "" {
		return "", fmt.Errorf("patch must contain a file operation")
	}
	if !sameLocalNormalizedPath(filepath.Clean(filepath.FromSlash(patchPath)), filepath.Clean(targetPath)) &&
		filepath.Base(filepath.Clean(filepath.FromSlash(patchPath))) != filepath.Base(targetPath) {
		return "", fmt.Errorf("patch path does not match target path")
	}

	switch op {
	case "add":
		return applyLocalAddFilePatch(targetPath, lines[start:], workspaceRoot)
	case "update":
		return applyLocalUpdateFilePatch(targetPath, lines[start:], workspaceRoot)
	default:
		return "", fmt.Errorf("unsupported patch operation")
	}
}

func sameLocalNormalizedPath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil && rightErr == nil {
		return strings.EqualFold(filepath.Clean(leftAbs), filepath.Clean(rightAbs))
	}
	return strings.EqualFold(filepath.ToSlash(filepath.Clean(left)), filepath.ToSlash(filepath.Clean(right)))
}

func applyLocalAddFilePatch(targetPath string, lines []string, workspaceRoot string) (string, error) {
	if _, err := os.Stat(targetPath); err == nil {
		return "", fmt.Errorf("target file already exists")
	}
	content := make([]string, 0)
	for _, line := range lines {
		if line == "*** End Patch" {
			break
		}
		if !strings.HasPrefix(line, "+") {
			if strings.TrimSpace(line) == "" {
				content = append(content, "")
				continue
			}
			return "", fmt.Errorf("add patch must contain only added lines")
		}
		content = append(content, strings.TrimPrefix(line, "+"))
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(targetPath, []byte(strings.Join(content, "\n")), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("applied patch to %s: created %d line(s)", localWorkspaceRelative(workspaceRoot, targetPath), len(content)), nil
}

func applyLocalUpdateFilePatch(targetPath string, lines []string, workspaceRoot string) (string, error) {
	originalBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return "", err
	}
	original := splitLocalTextLines(string(originalBytes))
	result := make([]string, 0, len(original))
	sourceIndex := 0
	appliedLines := 0

	for _, line := range lines {
		if line == "*** End Patch" {
			break
		}
		if strings.HasPrefix(line, "*** Move to: ") {
			return "", fmt.Errorf("move patch is not allowed")
		}
		if strings.HasPrefix(line, "@@") || line == "*** End of File" {
			continue
		}
		if line == "" {
			return "", fmt.Errorf("unexpected blank patch line")
		}
		marker := line[:1]
		text := line[1:]
		switch marker {
		case " ":
			found := findLocalNextLine(original, sourceIndex, text)
			if found < 0 {
				return "", fmt.Errorf("patch context not found: %s", text)
			}
			result = append(result, original[sourceIndex:found+1]...)
			sourceIndex = found + 1
		case "-":
			found := findLocalNextLine(original, sourceIndex, text)
			if found < 0 {
				return "", fmt.Errorf("patch removal not found: %s", text)
			}
			result = append(result, original[sourceIndex:found]...)
			sourceIndex = found + 1
			appliedLines++
		case "+":
			result = append(result, text)
			appliedLines++
		default:
			return "", fmt.Errorf("unsupported patch line: %s", line)
		}
	}
	result = append(result, original[sourceIndex:]...)
	if err := os.WriteFile(targetPath, []byte(strings.Join(result, "\n")), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("applied patch to %s: updated %d line(s)", localWorkspaceRelative(workspaceRoot, targetPath), appliedLines), nil
}

func findLocalNextLine(lines []string, start int, want string) int {
	for index := start; index < len(lines); index++ {
		if lines[index] == want {
			return index
		}
	}
	return -1
}

func encodeTargetPaths(paths []string) string {
	if len(paths) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(paths)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func decodeTargetPaths(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var paths []string
	if err := json.Unmarshal([]byte(raw), &paths); err != nil {
		return nil
	}
	return paths
}

func compactText(value string, max int) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if normalized == "" || max <= 0 {
		return normalized
	}
	runes := []rune(normalized)
	if len(runes) <= max {
		return normalized
	}
	return string(runes[:max]) + "..."
}

func nextLocalID(prefix string) string {
	serial := localIDCounter.Add(1)
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), serial)
}

func defaultStateStorePath(workspaceRoot string) string {
	if override := strings.TrimSpace(os.Getenv("GENCODE_DESKTOP_STATE_PATH")); override != "" {
		return override
	}
	return filepath.Join(workspaceRoot, ".gen-code", "state.db")
}

func runtimeErrorStatus(err error) RuntimeStatus {
	return RuntimeStatus{
		AppName:        "gen-code",
		AppEnv:         "local",
		Port:           10008,
		DesktopReady:   true,
		RuntimeState:   "degraded",
		RuntimeReady:   false,
		RuntimeMessage: err.Error(),
		RuntimeSource:  "desktop-local",
		StateStore:     "sqlite",
		SkillsByGroup:  map[string][]string{},
		ToolsByGroup:   map[string][]string{},
		MCPByGroup:     map[string][]string{},
		MissingPaths:   []string{},
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}
}

func findWorkspaceRoot() (string, error) {
	current, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if pathLooksLikeWorkspaceRoot(current) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not locate gen-code workspace root")
		}
		current = parent
	}
}

func pathLooksLikeWorkspaceRoot(path string) bool {
	required := []string{
		filepath.Join(path, "go.mod"),
		filepath.Join(path, "desktop"),
		filepath.Join(path, "cmd"),
	}
	for _, item := range required {
		if _, err := os.Stat(item); err != nil {
			return false
		}
	}
	return true
}

func fallbackText(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
