package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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
	AppName               string                  `json:"appName"`
	AppEnv                string                  `json:"appEnv"`
	Port                  int                     `json:"port"`
	Debug                 bool                    `json:"debug"`
	ShutdownTimeout       string                  `json:"shutdownTimeout"`
	TrustedProxies        []string                `json:"trustedProxies"`
	LogLevel              string                  `json:"logLevel"`
	HTTPAccessLog         bool                    `json:"httpAccessLog"`
	WorkspaceRoot         string                  `json:"workspaceRoot"`
	WorkspaceID           string                  `json:"workspaceId"`
	WorkspaceSummary      WorkspaceSummary        `json:"workspaceSummary"`
	ProjectRoot           string                  `json:"projectRoot"`
	ThreadCount           int                     `json:"threadCount"`
	ActiveThreadID        string                  `json:"activeThreadId"`
	ActiveThreadSummary   ThreadWorkflowSummary   `json:"activeThreadSummary"`
	Threads               []ThreadSummary         `json:"threads"`
	Tasks                 []TaskSummary           `json:"tasks"`
	Approvals             []ApprovalSummary       `json:"approvals"`
	WriteExecutions       []WriteExecutionSummary `json:"writeExecutions"`
	Messages              []MessageSummary        `json:"messages"`
	ToolCalls             []ToolCallSummary       `json:"toolCalls"`
	Artifacts             []ArtifactSummary       `json:"artifacts"`
	RuntimeFlags          []RuntimeFlagSummary    `json:"runtimeFlags"`
	Events                []EventSummary          `json:"events"`
	DesktopReady          bool                    `json:"desktopReady"`
	RuntimeState          string                  `json:"runtimeState"`
	RuntimeReady          bool                    `json:"runtimeReady"`
	RuntimeMessage        string                  `json:"runtimeMessage"`
	RuntimeSource         string                  `json:"runtimeSource"`
	RuntimeSourceDetail   string                  `json:"runtimeSourceDetail"`
	RuntimeTrust          string                  `json:"runtimeTrust"`
	CanonicalRuntimeURL   string                  `json:"canonicalRuntimeUrl"`
	SupportsSSE           bool                    `json:"supportsSSE"`
	SSEEndpoint           string                  `json:"sseEndpoint"`
	LastSyncAt            string                  `json:"lastSyncAt"`
	SkillsByGroup         map[string][]string     `json:"skillsByGroup"`
	ToolsByGroup          map[string][]string     `json:"toolsByGroup"`
	MCPByGroup            map[string][]string     `json:"mcpByGroup"`
	Providers             []ProviderSummary       `json:"providers"`
	MissingPaths          []string                `json:"missingPaths"`
	StateStore            string                  `json:"stateStore"`
	StatePath             string                  `json:"statePath"`
	UsesProjectLocalStore bool                    `json:"usesProjectLocalStore"`
	RecoverySummary       string                  `json:"recoverySummary"`
	UpdatedAt             string                  `json:"updatedAt"`
}

type WorkspaceSummary struct {
	ID                    string `json:"id"`
	Root                  string `json:"root"`
	ProjectRoot           string `json:"projectRoot"`
	ActiveThreadID        string `json:"activeThreadId"`
	ActiveThreadName      string `json:"activeThreadName"`
	ThreadCount           int    `json:"threadCount"`
	TaskCount             int    `json:"taskCount"`
	WaitingTaskCount      int    `json:"waitingTaskCount"`
	ApprovalRequiredCount int    `json:"approvalRequiredCount"`
	PendingApprovalCount  int    `json:"pendingApprovalCount"`
	CompletedTaskCount    int    `json:"completedTaskCount"`
	FailedTaskCount       int    `json:"failedTaskCount"`
	WriteExecutionCount   int    `json:"writeExecutionCount"`
	Summary               string `json:"summary"`
}

type ThreadWorkflowSummary struct {
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	Status                string `json:"status"`
	PermissionMode        string `json:"permissionMode"`
	ActiveModel           string `json:"activeModel"`
	TaskCount             int    `json:"taskCount"`
	WaitingTaskCount      int    `json:"waitingTaskCount"`
	WaitingForTaskCount   int    `json:"waitingForTaskCount"`
	WaitingForApproval    int    `json:"waitingForApprovalCount"`
	ApprovalRequiredCount int    `json:"approvalRequiredCount"`
	PendingApprovalCount  int    `json:"pendingApprovalCount"`
	CompletedTaskCount    int    `json:"completedTaskCount"`
	FailedTaskCount       int    `json:"failedTaskCount"`
	ChildTaskCount        int    `json:"childTaskCount"`
	WriteExecutionCount   int    `json:"writeExecutionCount"`
	LatestTaskID          string `json:"latestTaskId"`
	LatestApprovalTaskID  string `json:"latestApprovalTaskId"`
	LatestWriteTaskID     string `json:"latestWriteTaskId"`
	Summary               string `json:"summary"`
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
	ID                    string   `json:"id"`
	ThreadID              string   `json:"threadId"`
	Title                 string   `json:"title"`
	Kind                  string   `json:"kind"`
	Input                 string   `json:"input"`
	Status                string   `json:"status"`
	ResultSummary         string   `json:"resultSummary"`
	ApprovalStatus        string   `json:"approvalStatus"`
	ParentTaskID          string   `json:"parentTaskId"`
	WaitingStatus         string   `json:"waitingStatus"`
	WaitingTaskID         string   `json:"waitingTaskId"`
	WaitingSummary        string   `json:"waitingSummary"`
	WorkflowLabel         string   `json:"workflowLabel"`
	ChildTaskIDs          []string `json:"childTaskIds"`
	LatestChildTaskID     string   `json:"latestChildTaskId"`
	ApprovalID            string   `json:"approvalId"`
	ApprovalSummary       string   `json:"approvalSummary"`
	WriteExecutionID      string   `json:"writeExecutionId"`
	WriteExecutionSummary string   `json:"writeExecutionSummary"`
	AgentStep             int      `json:"agentStep"`
	AgentMaxSteps         int      `json:"agentMaxSteps"`
	AgentPlanSummary      string   `json:"agentPlanSummary"`
	AgentCurrentStepTitle string   `json:"agentCurrentStepTitle"`
	AgentLastReasoning    string   `json:"agentLastReasoning"`
	CreatedAt             string   `json:"createdAt"`
	UpdatedAt             string   `json:"updatedAt"`
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

type WriteExecutionSummary struct {
	ID                 string   `json:"id"`
	ThreadID           string   `json:"threadId"`
	TaskID             string   `json:"taskId"`
	ApprovalID         string   `json:"approvalId"`
	ToolKind           string   `json:"toolKind"`
	Operation          string   `json:"operation"`
	RelatedExecutionID string   `json:"relatedExecutionId"`
	Status             string   `json:"status"`
	TargetPaths        []string `json:"targetPaths"`
	PatchSummary       string   `json:"patchSummary"`
	BeforeSummary      string   `json:"beforeSummary"`
	AfterSummary       string   `json:"afterSummary"`
	ResultSummary      string   `json:"resultSummary"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
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

type RuntimeFlagSummary struct {
	ThreadID  string `json:"threadId"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updatedAt"`
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
	State               string `json:"state"`
	Ready               bool   `json:"ready"`
	Message             string `json:"message"`
	RuntimeSource       string `json:"runtimeSource"`
	RuntimeSourceDetail string `json:"runtimeSourceDetail"`
	RuntimeTrust        string `json:"runtimeTrust"`
	CanonicalRuntimeURL string `json:"canonicalRuntimeUrl"`
	StateStore          string `json:"stateStore"`
	StatePath           string `json:"statePath"`
	WorkspaceID         string `json:"workspaceId"`
	ProjectRoot         string `json:"projectRoot"`
	ThreadCount         int    `json:"threadCount"`
	ActiveThreadID      string `json:"activeThreadId"`
	TaskCount           int    `json:"taskCount"`
	EventCount          int    `json:"eventCount"`
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
	ID                    string   `json:"id"`
	ThreadID              string   `json:"threadId"`
	Title                 string   `json:"title"`
	Kind                  string   `json:"kind"`
	Input                 string   `json:"input"`
	Status                string   `json:"status"`
	ResultSummary         string   `json:"resultSummary"`
	ApprovalStatus        string   `json:"approvalStatus"`
	ParentTaskID          string   `json:"parentTaskId"`
	WaitingStatus         string   `json:"waitingStatus"`
	WaitingTaskID         string   `json:"waitingTaskId"`
	WaitingSummary        string   `json:"waitingSummary"`
	WorkflowLabel         string   `json:"workflowLabel"`
	ChildTaskIDs          []string `json:"childTaskIds"`
	LatestChildTaskID     string   `json:"latestChildTaskId"`
	ApprovalID            string   `json:"approvalId"`
	ApprovalSummary       string   `json:"approvalSummary"`
	WriteExecutionID      string   `json:"writeExecutionId"`
	WriteExecutionSummary string   `json:"writeExecutionSummary"`
	AgentStep             int      `json:"agentStep"`
	AgentMaxSteps         int      `json:"agentMaxSteps"`
	AgentPlanSummary      string   `json:"agentPlanSummary"`
	AgentCurrentStepTitle string   `json:"agentCurrentStepTitle"`
	AgentLastReasoning    string   `json:"agentLastReasoning"`
	CreatedAt             string   `json:"createdAt"`
	UpdatedAt             string   `json:"updatedAt"`
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

type apiWriteExecution struct {
	ID                 string   `json:"id"`
	ThreadID           string   `json:"threadId"`
	TaskID             string   `json:"taskId"`
	ApprovalID         string   `json:"approvalId"`
	ToolKind           string   `json:"toolKind"`
	Operation          string   `json:"operation"`
	RelatedExecutionID string   `json:"relatedExecutionId"`
	Status             string   `json:"status"`
	TargetPaths        []string `json:"targetPaths"`
	PatchSummary       string   `json:"patchSummary"`
	BeforeSummary      string   `json:"beforeSummary"`
	AfterSummary       string   `json:"afterSummary"`
	ResultSummary      string   `json:"resultSummary"`
	CreatedAt          string   `json:"createdAt"`
	UpdatedAt          string   `json:"updatedAt"`
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
	Enabled       bool   `json:"enabled"`
	ToolCount     int    `json:"toolCount"`
	ResourceCount int    `json:"resourceCount"`
	Status        string `json:"status"`
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
	ParentTaskID   string
	WaitingStatus  string
	AgentState     string
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

type persistedWriteExecution struct {
	ID                 string
	ThreadID           string
	TaskID             string
	ApprovalID         string
	ToolKind           string
	Operation          string
	RelatedExecutionID string
	Status             string
	TargetPaths        []string
	PatchHash          string
	PatchSummary       string
	BeforeSummary      string
	AfterSummary       string
	RollbackPayload    string
	ResultSummary      string
	CreatedAt          string
	UpdatedAt          string
}

type localWriteExecutionFileSnapshot struct {
	Path          string `json:"path"`
	BeforeExists  bool   `json:"beforeExists"`
	BeforeContent string `json:"beforeContent"`
	BeforeHash    string `json:"beforeHash"`
	AfterExists   bool   `json:"afterExists"`
	AfterHash     string `json:"afterHash"`
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

type persistedRuntimeFlag struct {
	ThreadID  string
	Key       string
	Value     string
	UpdatedAt string
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
			AppName:             "gen-code",
			AppEnv:              "local",
			Port:                10008,
			DesktopReady:        true,
			RuntimeState:        "degraded",
			RuntimeReady:        false,
			RuntimeMessage:      err.Error(),
			RuntimeSource:       "local-fallback",
			RuntimeSourceDetail: "project-local SQLite fallback because the canonical app-server runtime is unavailable",
			RuntimeTrust:        "degraded",
			StateStore:          "sqlite",
			StatePath:           "",
			SkillsByGroup:       map[string][]string{},
			ToolsByGroup:        map[string][]string{},
			MCPByGroup:          map[string][]string{},
			MissingPaths:        []string{},
			UpdatedAt:           time.Now().Format(time.RFC3339),
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
		RuntimeHint: fmt.Sprintf("%s / %s / %s / %s", status.RuntimeSource, fallbackText(status.RuntimeTrust, "unknown"), status.RuntimeState, fallbackText(status.StatePath, "no-state-path")),
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
	localStatus.RuntimeSource = "local-fallback"
	localStatus.RuntimeSourceDetail = "project-local SQLite fallback because the canonical app-server runtime is unavailable"
	localStatus.RuntimeTrust = "degraded"
	localStatus.CanonicalRuntimeURL = ""
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
	writeExecutionsPayload := struct {
		Items []apiWriteExecution `json:"items"`
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
		if _, err := client.fetchEnvelopeOptional("/api/threads/"+threadID+"/write-executions", &writeExecutionsPayload); err != nil {
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
	status.WriteExecutions = mapWriteExecutions(writeExecutionsPayload.Items)
	status.Messages = mapMessages(messagesPayload.Items)
	status.ToolCalls = mapToolCalls(toolCallsPayload.Items)
	status.Artifacts = mapArtifacts(artifactsPayload.Items)
	status.Events = mapEvents(eventsPayload.Items)
	status.RuntimeState = runtimeStatus.State
	status.RuntimeReady = runtimeStatus.Ready
	status.RuntimeMessage = runtimeStatus.Message
	status.RuntimeSource = fallbackText(runtimeStatus.RuntimeSource, "remote-app-server")
	status.RuntimeSourceDetail = fallbackText(runtimeStatus.RuntimeSourceDetail, "canonical shared runtime served by the app-server entry")
	status.RuntimeTrust = fallbackText(runtimeStatus.RuntimeTrust, "canonical")
	status.CanonicalRuntimeURL = fallbackText(runtimeStatus.CanonicalRuntimeURL, strings.TrimRight(runtimeBaseURL(), "/"))
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
	normalizeWorkflowSummaries(&status)
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
		RuntimeHint: "remote-app-server",
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

func (c runtimeClient) fetchEnvelopeOptional(path string, target any) (bool, error) {
	response, err := c.client.Get(c.baseURL + path)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err := decodeEnvelope(response, target); err != nil {
		return false, err
	}
	return true, nil
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
		Items []apiWriteExecution `json:"items"`
	}:
		var envelope apiEnvelope[struct {
			Items []apiWriteExecution `json:"items"`
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
		status := fallbackText(strings.TrimSpace(item.Status), "unknown")
		label := fmt.Sprintf("%s (%s, tools:%d resources:%d)", item.ID, status, item.ToolCount, item.ResourceCount)
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
			ID:                    item.ID,
			ThreadID:              item.ThreadID,
			Title:                 item.Title,
			Kind:                  item.Kind,
			Input:                 item.Input,
			Status:                item.Status,
			ResultSummary:         item.ResultSummary,
			ApprovalStatus:        item.ApprovalStatus,
			ParentTaskID:          item.ParentTaskID,
			WaitingStatus:         item.WaitingStatus,
			WaitingTaskID:         item.WaitingTaskID,
			WaitingSummary:        item.WaitingSummary,
			WorkflowLabel:         item.WorkflowLabel,
			ChildTaskIDs:          append([]string(nil), item.ChildTaskIDs...),
			LatestChildTaskID:     item.LatestChildTaskID,
			ApprovalID:            item.ApprovalID,
			ApprovalSummary:       item.ApprovalSummary,
			WriteExecutionID:      item.WriteExecutionID,
			WriteExecutionSummary: item.WriteExecutionSummary,
			AgentStep:             item.AgentStep,
			AgentMaxSteps:         item.AgentMaxSteps,
			AgentPlanSummary:      item.AgentPlanSummary,
			AgentCurrentStepTitle: item.AgentCurrentStepTitle,
			AgentLastReasoning:    item.AgentLastReasoning,
			CreatedAt:             item.CreatedAt,
			UpdatedAt:             item.UpdatedAt,
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

func mapWriteExecutions(items []apiWriteExecution) []WriteExecutionSummary {
	result := make([]WriteExecutionSummary, 0, len(items))
	for _, item := range items {
		result = append(result, WriteExecutionSummary{
			ID:                 item.ID,
			ThreadID:           item.ThreadID,
			TaskID:             item.TaskID,
			ApprovalID:         item.ApprovalID,
			ToolKind:           item.ToolKind,
			Operation:          item.Operation,
			RelatedExecutionID: item.RelatedExecutionID,
			Status:             item.Status,
			TargetPaths:        append([]string(nil), item.TargetPaths...),
			PatchSummary:       item.PatchSummary,
			BeforeSummary:      item.BeforeSummary,
			AfterSummary:       item.AfterSummary,
			ResultSummary:      item.ResultSummary,
			CreatedAt:          item.CreatedAt,
			UpdatedAt:          item.UpdatedAt,
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
	} else if normalized.Kind == "workspace.apply_patch.rollback" {
		writeExecutionID, parseErr := parseLocalRollbackInput(normalized.Input)
		if parseErr != nil {
			return runtimeErrorStatus(parseErr)
		}
		source, sourceErr := readWriteExecutionByIDTx(tx, trimmedThreadID, writeExecutionID)
		if sourceErr != nil {
			return runtimeErrorStatus(sourceErr)
		}
		summary := localRollbackApprovalSummary(source.TargetPaths)
		switch permissionMode {
		case "read-only":
			statusValue = "failed"
			resultSummary = "rollback failed: permission denied: read-only mode does not allow workspace writes"
			approvalStatus = "rejected"
			eventType = "task.failed"
			eventMessage = resultSummary
		case "", "ask-user":
			statusValue = "needs_approval"
			resultSummary = summary
			approvalStatus = "pending"
			eventType = "task.rollback_required"
			eventMessage = summary
		default:
			statusValue = "queued"
			approvalStatus = "direct"
		}
	}

	if _, err := tx.Exec(`
		INSERT INTO tasks(id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, taskID, trimmedThreadID, normalized.Title, normalized.Kind, normalized.Input, statusValue, resultSummary, approvalStatus, "", "", "", now, now); err != nil {
		return runtimeErrorStatus(err)
	}
	if _, err := tx.Exec(`
		INSERT INTO thread_messages(id, thread_id, role, content, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-message"), trimmedThreadID, "user", normalized.Input, now); err != nil {
		return runtimeErrorStatus(err)
	}
	if (normalized.Kind == "workspace.apply_patch" || normalized.Kind == "workspace.apply_patch.rollback") && approvalStatus == "pending" {
		targetJSON := ""
		switch normalized.Kind {
		case "workspace.apply_patch":
			pathValue, patchValue, _ := parseLocalPatchInput(normalized.Input)
			targets, _ := extractLocalPatchTargets(patchValue)
			targetJSON = encodeTargetPaths(targets)
			if len(targets) == 0 {
				targetJSON = encodeTargetPaths([]string{pathValue})
			}
		case "workspace.apply_patch.rollback":
			writeExecutionID, _ := parseLocalRollbackInput(normalized.Input)
			source, _ := readWriteExecutionByIDTx(tx, trimmedThreadID, writeExecutionID)
			targetJSON = encodeTargetPaths(source.TargetPaths)
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
		if err := s.executePatchTask(tx, base.WorkspaceRoot, trimmedTaskID, threadID, "", title, input, approvalStatus, now); err != nil {
			return runtimeErrorStatus(err)
		}
	} else if kind == "workspace.apply_patch.rollback" {
		if currentStatus == "needs_approval" {
			return runtimeErrorStatus(fmt.Errorf("task approval required"))
		}
		if err := s.executeRollbackTask(tx, base.WorkspaceRoot, trimmedTaskID, threadID, "", title, input, approvalStatus, now); err != nil {
			return runtimeErrorStatus(err)
		}
	} else if kind == "thread.toolcall.append" {
		if err := s.executeToolCallAppendTask(tx, trimmedTaskID, threadID, input, now); err != nil {
			return runtimeErrorStatus(err)
		}
	} else if kind == "thread.artifact.append" {
		if err := s.executeArtifactAppendTask(tx, trimmedTaskID, threadID, input, now); err != nil {
			return runtimeErrorStatus(err)
		}
	} else if kind == "thread.runtimeflag.set" {
		if err := s.executeRuntimeFlagSetTask(tx, trimmedTaskID, threadID, input, now); err != nil {
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
			parent_task_id TEXT NOT NULL DEFAULT '',
			waiting_status TEXT NOT NULL DEFAULT '',
			agent_state TEXT NOT NULL DEFAULT '',
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
		`CREATE TABLE IF NOT EXISTS thread_write_executions (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			task_id TEXT NOT NULL,
			approval_id TEXT NOT NULL DEFAULT '',
			tool_kind TEXT NOT NULL,
			operation TEXT NOT NULL DEFAULT 'apply',
			related_execution_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			target_paths TEXT NOT NULL DEFAULT '',
			patch_hash TEXT NOT NULL DEFAULT '',
			patch_summary TEXT NOT NULL DEFAULT '',
			before_snapshot_summary TEXT NOT NULL DEFAULT '',
			after_snapshot_summary TEXT NOT NULL DEFAULT '',
			rollback_payload TEXT NOT NULL DEFAULT '',
			result_summary TEXT NOT NULL DEFAULT '',
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
		`ALTER TABLE tasks ADD COLUMN parent_task_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN waiting_status TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tasks ADD COLUMN agent_state TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE thread_write_executions ADD COLUMN operation TEXT NOT NULL DEFAULT 'apply'`,
		`ALTER TABLE thread_write_executions ADD COLUMN related_execution_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE thread_write_executions ADD COLUMN rollback_payload TEXT NOT NULL DEFAULT ''`,
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
	runtimeFlags, err := s.readRuntimeFlags(workspaceRecord.ActiveThreadID)
	if err != nil {
		return RuntimeStatus{}, err
	}
	writeExecutions, err := s.readWriteExecutions(workspaceRecord.ActiveThreadID)
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
	status.RuntimeFlags = toRuntimeFlagSummaries(runtimeFlags)
	status.Events = toEventSummaries(events)
	status.Approvals = toApprovalSummaries(approvals)
	status.WriteExecutions = toWriteExecutionSummaries(writeExecutions)
	status.ToolsByGroup = groupTools(localToolCatalog())
	status.Providers = localProviderCatalog()
	status.RuntimeState = "fallback"
	status.RuntimeReady = true
	status.RuntimeMessage = "Using project-local SQLite runtime fallback because no external runtime is connected."
	status.RuntimeSource = "local-fallback"
	status.RuntimeSourceDetail = "project-local SQLite fallback because the canonical app-server runtime is unavailable"
	status.RuntimeTrust = "degraded"
	status.CanonicalRuntimeURL = ""
	status.SupportsSSE = false
	status.SSEEndpoint = ""
	status.LastSyncAt = ""
	status.StateStore = "sqlite"
	status.StatePath = defaultStateStorePath(base.WorkspaceRoot)
	status.UsesProjectLocalStore = true
	status.RecoverySummary = fmt.Sprintf("Recovered %d thread(s), %d task(s), %d approval(s), %d write execution(s), %d message(s), %d tool call(s), %d artifact(s), %d event(s) from project-local state store.", len(threads), len(tasks), len(approvals), len(writeExecutions), len(messages), len(toolCalls), len(artifacts), len(events))
	status.UpdatedAt = time.Now().Format(time.RFC3339)
	normalizeWorkflowSummaries(&status)
	return status, nil
}

type agentStateSummary struct {
	Step              int
	MaxSteps          int
	WaitingTaskID     string
	WaitingSummary    string
	WorkflowLabel     string
	ChildTaskIDs      []string
	LatestChildTaskID string
	PlanSummary       string
	CurrentStepTitle  string
	LastReasoning     string
}

func parseAgentStateSummary(raw string) agentStateSummary {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return agentStateSummary{}
	}
	var payload struct {
		StepIndex          int      `json:"stepIndex"`
		MaxSteps           int      `json:"maxSteps"`
		WaitingChildTaskID string   `json:"waitingChildTaskId"`
		LatestChildTaskID  string   `json:"latestChildTaskId"`
		Status             string   `json:"status"`
		Goal               string   `json:"goal"`
		PlanSummary        string   `json:"planSummary"`
		CurrentStepTitle   string   `json:"currentStepTitle"`
		LastReasoning      string   `json:"lastReasoning"`
		ChildTaskIDs       []string `json:"childTaskIds"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return agentStateSummary{}
	}
	waitingTaskID := fallbackText(strings.TrimSpace(payload.WaitingChildTaskID), strings.TrimSpace(payload.LatestChildTaskID))
	waitingSummary := ""
	switch strings.TrimSpace(payload.Status) {
	case "waiting_for_task":
		waitingSummary = fmt.Sprintf("waiting for child task %s", fallbackText(waitingTaskID, "to start"))
	case "waiting_for_approval":
		waitingSummary = fmt.Sprintf("waiting for approval before resuming%s", withOptionalLabel(waitingTaskID, " via child task "))
	}
	workflowLabel := ""
	statusLabel := strings.TrimSpace(payload.Status)
	if payload.StepIndex > 0 && payload.MaxSteps > 0 {
		workflowLabel = fmt.Sprintf("step %d/%d", payload.StepIndex, payload.MaxSteps)
	}
	if waitingSummary != "" {
		if workflowLabel == "" {
			workflowLabel = waitingSummary
		} else {
			workflowLabel = waitingSummary + " / " + workflowLabel
		}
	}
	if statusLabel != "" {
		if workflowLabel == "" {
			workflowLabel = statusLabel
		} else {
			workflowLabel = statusLabel + " / " + workflowLabel
		}
	}
	return agentStateSummary{
		Step:              payload.StepIndex,
		MaxSteps:          payload.MaxSteps,
		WaitingTaskID:     waitingTaskID,
		WaitingSummary:    waitingSummary,
		WorkflowLabel:     workflowLabel,
		ChildTaskIDs:      append([]string(nil), payload.ChildTaskIDs...),
		LatestChildTaskID: fallbackText(strings.TrimSpace(payload.LatestChildTaskID), waitingTaskID),
		PlanSummary:       fallbackText(strings.TrimSpace(payload.PlanSummary), compactText(payload.Goal, 96)),
		CurrentStepTitle:  strings.TrimSpace(payload.CurrentStepTitle),
		LastReasoning:     strings.TrimSpace(payload.LastReasoning),
	}
}

func normalizeWorkflowSummaries(status *RuntimeStatus) {
	if status == nil {
		return
	}

	taskIndex := make(map[string]*TaskSummary, len(status.Tasks))
	threadIndex := make(map[string]ThreadSummary, len(status.Threads))
	for _, thread := range status.Threads {
		threadIndex[thread.ID] = thread
	}
	for i := range status.Tasks {
		taskIndex[status.Tasks[i].ID] = &status.Tasks[i]
	}

	childTaskIDsByParent := map[string][]string{}
	for i := range status.Tasks {
		task := &status.Tasks[i]
		if task.ParentTaskID != "" {
			childTaskIDsByParent[task.ParentTaskID] = append(childTaskIDsByParent[task.ParentTaskID], task.ID)
		}
	}
	for i := range status.Approvals {
		approval := status.Approvals[i]
		if task := taskIndex[approval.TaskID]; task != nil {
			task.ApprovalID = approval.ID
			task.ApprovalSummary = localApprovalSummary(approval.ToolKind, approval.TargetPaths)
		}
	}
	for i := range status.WriteExecutions {
		execution := status.WriteExecutions[i]
		if task := taskIndex[execution.TaskID]; task != nil {
			task.WriteExecutionID = execution.ID
			if task.WriteExecutionSummary == "" {
				task.WriteExecutionSummary = fallbackText(execution.ResultSummary, execution.PatchSummary)
			}
			if task.ApprovalID == "" {
				task.ApprovalID = execution.ApprovalID
			}
		}
	}

	summaryByThread := map[string]*ThreadWorkflowSummary{}
	for _, thread := range status.Threads {
		copy := ThreadWorkflowSummary{
			ID:             thread.ID,
			Name:           thread.Name,
			Status:         thread.Status,
			PermissionMode: thread.PermissionMode,
			ActiveModel:    thread.ActiveModel,
		}
		summaryByThread[thread.ID] = &copy
	}

	for i := range status.Tasks {
		task := &status.Tasks[i]
		if len(task.ChildTaskIDs) == 0 {
			task.ChildTaskIDs = append([]string(nil), childTaskIDsByParent[task.ID]...)
		}
		if task.LatestChildTaskID == "" && len(task.ChildTaskIDs) > 0 {
			task.LatestChildTaskID = task.ChildTaskIDs[0]
		}
		if task.WorkflowLabel == "" {
			task.WorkflowLabel = buildTaskWorkflowLabel(*task)
		}
		if task.WaitingSummary == "" {
			task.WaitingSummary = buildWaitingSummary(*task)
		}
		threadSummary := summaryByThread[task.ThreadID]
		if threadSummary == nil {
			continue
		}
		threadSummary.TaskCount++
		if task.ParentTaskID != "" {
			threadSummary.ChildTaskCount++
		}
		switch task.Status {
		case "completed":
			threadSummary.CompletedTaskCount++
		case "failed":
			threadSummary.FailedTaskCount++
		case "needs_approval":
			threadSummary.ApprovalRequiredCount++
		case "waiting_for_task":
			threadSummary.WaitingTaskCount++
			threadSummary.WaitingForTaskCount++
		case "waiting_for_approval":
			threadSummary.WaitingTaskCount++
			threadSummary.WaitingForApproval++
		}
		if task.WaitingStatus == "waiting_for_task" && task.Status != "waiting_for_task" {
			threadSummary.WaitingTaskCount++
			threadSummary.WaitingForTaskCount++
		}
		if task.WaitingStatus == "waiting_for_approval" && task.Status != "waiting_for_approval" {
			threadSummary.WaitingTaskCount++
			threadSummary.WaitingForApproval++
		}
		if task.ApprovalStatus == "pending" {
			threadSummary.PendingApprovalCount++
		}
		if task.ApprovalID != "" && threadSummary.LatestApprovalTaskID == "" {
			threadSummary.LatestApprovalTaskID = task.ID
		}
		if task.WriteExecutionID != "" {
			threadSummary.WriteExecutionCount++
			if threadSummary.LatestWriteTaskID == "" {
				threadSummary.LatestWriteTaskID = task.ID
			}
		}
		if threadSummary.LatestTaskID == "" {
			threadSummary.LatestTaskID = task.ID
		}
	}

	activeThreadName := ""
	activeSummary := ThreadWorkflowSummary{}
	if activeThread, ok := threadIndex[status.ActiveThreadID]; ok {
		activeThreadName = activeThread.Name
	}
	if summary := summaryByThread[status.ActiveThreadID]; summary != nil {
		summary.Summary = buildThreadWorkflowSummary(*summary)
		activeSummary = *summary
	}
	for _, summary := range summaryByThread {
		if summary.Summary == "" {
			summary.Summary = buildThreadWorkflowSummary(*summary)
		}
	}

	waitingTaskCount := 0
	approvalRequiredCount := 0
	pendingApprovalCount := 0
	completedTaskCount := 0
	failedTaskCount := 0
	for _, task := range status.Tasks {
		if task.WaitingStatus != "" || task.Status == "waiting_for_task" || task.Status == "waiting_for_approval" {
			waitingTaskCount++
		}
		if task.Status == "needs_approval" {
			approvalRequiredCount++
		}
		if task.ApprovalStatus == "pending" {
			pendingApprovalCount++
		}
		if task.Status == "completed" {
			completedTaskCount++
		}
		if task.Status == "failed" {
			failedTaskCount++
		}
	}
	status.ActiveThreadSummary = activeSummary
	status.WorkspaceSummary = WorkspaceSummary{
		ID:                    status.WorkspaceID,
		Root:                  status.WorkspaceRoot,
		ProjectRoot:           fallbackText(status.ProjectRoot, status.WorkspaceRoot),
		ActiveThreadID:        status.ActiveThreadID,
		ActiveThreadName:      activeThreadName,
		ThreadCount:           len(status.Threads),
		TaskCount:             len(status.Tasks),
		WaitingTaskCount:      waitingTaskCount,
		ApprovalRequiredCount: approvalRequiredCount,
		PendingApprovalCount:  pendingApprovalCount,
		CompletedTaskCount:    completedTaskCount,
		FailedTaskCount:       failedTaskCount,
		WriteExecutionCount:   len(status.WriteExecutions),
		Summary:               buildWorkspaceSummary(status, activeThreadName, waitingTaskCount, approvalRequiredCount, pendingApprovalCount),
	}
}

func buildTaskWorkflowLabel(task TaskSummary) string {
	parts := make([]string, 0, 8)
	if task.ParentTaskID != "" {
		parts = append(parts, "child task")
	}
	if task.Status != "" {
		parts = append(parts, task.Status)
	}
	if task.WaitingStatus != "" {
		parts = append(parts, task.WaitingStatus)
	}
	if task.ApprovalStatus == "pending" || task.Status == "needs_approval" {
		parts = append(parts, "approval required")
	}
	if task.WriteExecutionID != "" {
		parts = append(parts, "write execution linked")
	}
	if task.WaitingSummary != "" {
		parts = append(parts, task.WaitingSummary)
	}
	if task.AgentStep > 0 && task.AgentMaxSteps > 0 {
		parts = append(parts, fmt.Sprintf("step %d/%d", task.AgentStep, task.AgentMaxSteps))
	}
	if task.AgentLastReasoning != "" {
		parts = append(parts, "reasoning: "+task.AgentLastReasoning)
	}
	if len(parts) == 0 {
		return fallbackText(task.Kind, "task")
	}
	return strings.Join(parts, " / ")
}

func buildWaitingSummary(task TaskSummary) string {
	switch task.WaitingStatus {
	case "waiting_for_task":
		if task.WaitingTaskID != "" {
			return fmt.Sprintf("waiting for child task %s", task.WaitingTaskID)
		}
		if task.LatestChildTaskID != "" {
			return fmt.Sprintf("waiting for child task %s", task.LatestChildTaskID)
		}
		return "waiting for child task"
	case "waiting_for_approval":
		if task.ApprovalID != "" {
			return fmt.Sprintf("waiting for approval %s", task.ApprovalID)
		}
		if task.WaitingTaskID != "" {
			return fmt.Sprintf("waiting for approval via child task %s", task.WaitingTaskID)
		}
		return "waiting for approval"
	default:
		return ""
	}
}

func buildThreadWorkflowSummary(summary ThreadWorkflowSummary) string {
	parts := []string{
		fmt.Sprintf("%d task(s)", summary.TaskCount),
	}
	if summary.ChildTaskCount > 0 {
		parts = append(parts, fmt.Sprintf("%d child task(s)", summary.ChildTaskCount))
	}
	if summary.WaitingTaskCount > 0 {
		parts = append(parts, fmt.Sprintf("%d waiting", summary.WaitingTaskCount))
	}
	if summary.WaitingForTaskCount > 0 {
		parts = append(parts, fmt.Sprintf("%d waiting for task", summary.WaitingForTaskCount))
	}
	if summary.WaitingForApproval > 0 {
		parts = append(parts, fmt.Sprintf("%d waiting for approval", summary.WaitingForApproval))
	}
	if summary.PendingApprovalCount > 0 {
		parts = append(parts, fmt.Sprintf("%d pending approval", summary.PendingApprovalCount))
	}
	if summary.WriteExecutionCount > 0 {
		parts = append(parts, fmt.Sprintf("%d write execution(s)", summary.WriteExecutionCount))
	}
	return strings.Join(parts, " / ")
}

func buildWorkspaceSummary(status *RuntimeStatus, activeThreadName string, waitingTaskCount int, approvalRequiredCount int, pendingApprovalCount int) string {
	if status == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("%d thread(s)", len(status.Threads))}
	if strings.TrimSpace(activeThreadName) != "" {
		parts = append(parts, fmt.Sprintf("active thread %s", activeThreadName))
	}
	if waitingTaskCount > 0 {
		parts = append(parts, fmt.Sprintf("%d waiting task(s)", waitingTaskCount))
	}
	if approvalRequiredCount > 0 || pendingApprovalCount > 0 {
		parts = append(parts, fmt.Sprintf("%d approval-required / %d pending", approvalRequiredCount, pendingApprovalCount))
	}
	if len(status.WriteExecutions) > 0 {
		parts = append(parts, fmt.Sprintf("%d write execution(s)", len(status.WriteExecutions)))
	}
	return strings.Join(parts, " / ")
}

func withOptionalLabel(value string, prefix string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return prefix + strings.TrimSpace(value)
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
		SELECT id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, created_at, updated_at
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
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.Title, &item.Kind, &item.Input, &item.Status, &item.ResultSummary, &item.ApprovalStatus, &item.ParentTaskID, &item.WaitingStatus, &item.AgentState, &item.CreatedAt, &item.UpdatedAt); err != nil {
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

func (s *localRuntimeStore) readWriteExecutions(threadID string) ([]persistedWriteExecution, error) {
	query := `
		SELECT id, thread_id, task_id, approval_id, tool_kind, operation, related_execution_id, status, target_paths, patch_hash, patch_summary, before_snapshot_summary, after_snapshot_summary, rollback_payload, result_summary, created_at, updated_at
		FROM thread_write_executions
	`
	args := []any{}
	if strings.TrimSpace(threadID) != "" {
		query += ` WHERE thread_id = ?`
		args = append(args, threadID)
	}
	query += ` ORDER BY datetime(updated_at) DESC, id DESC LIMIT 12`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedWriteExecution{}
	for rows.Next() {
		var item persistedWriteExecution
		var targetPaths string
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.TaskID, &item.ApprovalID, &item.ToolKind, &item.Operation, &item.RelatedExecutionID, &item.Status, &targetPaths, &item.PatchHash, &item.PatchSummary, &item.BeforeSummary, &item.AfterSummary, &item.RollbackPayload, &item.ResultSummary, &item.CreatedAt, &item.UpdatedAt); err != nil {
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

func (s *localRuntimeStore) readRuntimeFlags(threadID string) ([]persistedRuntimeFlag, error) {
	query := `
		SELECT thread_id, key, value, updated_at
		FROM thread_runtime_flags
	`
	args := []any{}
	if strings.TrimSpace(threadID) != "" {
		query += ` WHERE thread_id = ?`
		args = append(args, threadID)
	}
	query += ` ORDER BY datetime(updated_at) DESC, key ASC LIMIT 24`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []persistedRuntimeFlag{}
	for rows.Next() {
		var item persistedRuntimeFlag
		if err := rows.Scan(&item.ThreadID, &item.Key, &item.Value, &item.UpdatedAt); err != nil {
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
		agent := parseAgentStateSummary(item.AgentState)
		result = append(result, TaskSummary{
			ID:                    item.ID,
			ThreadID:              item.ThreadID,
			Title:                 item.Title,
			Kind:                  item.Kind,
			Input:                 item.Input,
			Status:                item.Status,
			ResultSummary:         item.ResultSummary,
			ApprovalStatus:        item.ApprovalStatus,
			ParentTaskID:          item.ParentTaskID,
			WaitingStatus:         item.WaitingStatus,
			WaitingTaskID:         agent.WaitingTaskID,
			WaitingSummary:        agent.WaitingSummary,
			WorkflowLabel:         agent.WorkflowLabel,
			ChildTaskIDs:          append([]string(nil), agent.ChildTaskIDs...),
			LatestChildTaskID:     agent.LatestChildTaskID,
			AgentStep:             agent.Step,
			AgentMaxSteps:         agent.MaxSteps,
			AgentPlanSummary:      agent.PlanSummary,
			AgentCurrentStepTitle: agent.CurrentStepTitle,
			AgentLastReasoning:    agent.LastReasoning,
			CreatedAt:             item.CreatedAt,
			UpdatedAt:             item.UpdatedAt,
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

func toWriteExecutionSummaries(items []persistedWriteExecution) []WriteExecutionSummary {
	result := make([]WriteExecutionSummary, 0, len(items))
	for _, item := range items {
		result = append(result, WriteExecutionSummary{
			ID:                 item.ID,
			ThreadID:           item.ThreadID,
			TaskID:             item.TaskID,
			ApprovalID:         item.ApprovalID,
			ToolKind:           item.ToolKind,
			Operation:          item.Operation,
			RelatedExecutionID: item.RelatedExecutionID,
			Status:             item.Status,
			TargetPaths:        append([]string(nil), item.TargetPaths...),
			PatchSummary:       item.PatchSummary,
			BeforeSummary:      item.BeforeSummary,
			AfterSummary:       item.AfterSummary,
			ResultSummary:      item.ResultSummary,
			CreatedAt:          item.CreatedAt,
			UpdatedAt:          item.UpdatedAt,
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
		if task.Kind == "workspace.apply_patch" || task.Kind == "workspace.apply_patch.rollback" {
			var execErr error
			switch task.Kind {
			case "workspace.apply_patch":
				execErr = s.executePatchTask(tx, base.WorkspaceRoot, trimmedTaskID, trimmedThreadID, approval.ID, task.Title, task.Input, "approved", now)
			case "workspace.apply_patch.rollback":
				execErr = s.executeRollbackTask(tx, base.WorkspaceRoot, trimmedTaskID, trimmedThreadID, approval.ID, task.Title, task.Input, "approved", now)
			}
			if execErr != nil {
				return runtimeErrorStatus(execErr)
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
		SELECT id, thread_id, title, kind, input, status, result_summary, approval_status, parent_task_id, waiting_status, agent_state, created_at, updated_at
		FROM tasks
		WHERE id = ?
	`, taskID)
	var item persistedTask
	if err := row.Scan(&item.ID, &item.ThreadID, &item.Title, &item.Kind, &item.Input, &item.Status, &item.ResultSummary, &item.ApprovalStatus, &item.ParentTaskID, &item.WaitingStatus, &item.AgentState, &item.CreatedAt, &item.UpdatedAt); err != nil {
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

func readWriteExecutionByIDTx(tx *sql.Tx, threadID string, executionID string) (persistedWriteExecution, error) {
	row := tx.QueryRow(`
		SELECT id, thread_id, task_id, approval_id, tool_kind, operation, related_execution_id, status, target_paths, patch_hash, patch_summary, before_snapshot_summary, after_snapshot_summary, rollback_payload, result_summary, created_at, updated_at
		FROM thread_write_executions
		WHERE thread_id = ? AND id = ?
		LIMIT 1
	`, threadID, executionID)
	var item persistedWriteExecution
	var targetPaths string
	if err := row.Scan(&item.ID, &item.ThreadID, &item.TaskID, &item.ApprovalID, &item.ToolKind, &item.Operation, &item.RelatedExecutionID, &item.Status, &targetPaths, &item.PatchHash, &item.PatchSummary, &item.BeforeSummary, &item.AfterSummary, &item.RollbackPayload, &item.ResultSummary, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return persistedWriteExecution{}, fmt.Errorf("write execution not found")
		}
		return persistedWriteExecution{}, err
	}
	item.TargetPaths = decodeTargetPaths(targetPaths)
	return item, nil
}

func readLatestCompletedApplyExecutionTx(tx *sql.Tx, threadID string) (persistedWriteExecution, error) {
	row := tx.QueryRow(`
		SELECT id, thread_id, task_id, approval_id, tool_kind, operation, related_execution_id, status, target_paths, patch_hash, patch_summary, before_snapshot_summary, after_snapshot_summary, rollback_payload, result_summary, created_at, updated_at
		FROM thread_write_executions
		WHERE thread_id = ? AND operation = 'apply' AND status = 'completed'
		ORDER BY datetime(updated_at) DESC, id DESC
		LIMIT 1
	`, threadID)
	var item persistedWriteExecution
	var targetPaths string
	if err := row.Scan(&item.ID, &item.ThreadID, &item.TaskID, &item.ApprovalID, &item.ToolKind, &item.Operation, &item.RelatedExecutionID, &item.Status, &targetPaths, &item.PatchHash, &item.PatchSummary, &item.BeforeSummary, &item.AfterSummary, &item.RollbackPayload, &item.ResultSummary, &item.CreatedAt, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return persistedWriteExecution{}, fmt.Errorf("rollback failed: no completed apply execution found")
		}
		return persistedWriteExecution{}, err
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

func (s *localRuntimeStore) executePatchTask(tx *sql.Tx, workspaceRoot string, taskID string, threadID string, approvalID string, title string, rawInput string, approvalStatus string, now string) error {
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

	targetPaths := collectLocalWriteExecutionTargets(pathValue, patchValue)
	rollbackSnapshot, err := captureLocalRollbackSnapshot(resolvedPath, targetPaths[0])
	if err != nil {
		return err
	}
	patchHash := hashLocalText(patchValue)
	patchSummary := localPatchExecutionSummary(targetPaths, patchValue)
	beforeSummary := localFileSnapshotSummary(resolvedPath)

	resultSummary, err := applyLocalWorkspacePatch(resolvedPath, patchValue, workspaceRoot)
	rollbackSnapshot.AfterExists, rollbackSnapshot.AfterHash = readLocalFilePresenceAndHash(resolvedPath)
	if err != nil {
		afterSummary := localFileSnapshotSummary(resolvedPath)
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
		if err := insertLocalWriteExecutionTx(tx, persistedWriteExecution{
			ID:              nextLocalID("local-write-execution"),
			ThreadID:        threadID,
			TaskID:          taskID,
			ApprovalID:      approvalID,
			ToolKind:        "workspace.apply_patch",
			Operation:       "apply",
			Status:          "failed",
			TargetPaths:     targetPaths,
			PatchHash:       patchHash,
			PatchSummary:    patchSummary,
			BeforeSummary:   beforeSummary,
			AfterSummary:    afterSummary,
			RollbackPayload: encodeLocalRollbackPayload([]localWriteExecutionFileSnapshot{rollbackSnapshot}),
			ResultSummary:   err.Error(),
			CreatedAt:       now,
			UpdatedAt:       now,
		}); err != nil {
			return err
		}
		return insertEventTx(tx, threadID, "task.failed", err.Error(), now)
	}

	afterSummary := localFileSnapshotSummary(resolvedPath)
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
	if err := insertLocalWriteExecutionTx(tx, persistedWriteExecution{
		ID:              nextLocalID("local-write-execution"),
		ThreadID:        threadID,
		TaskID:          taskID,
		ApprovalID:      approvalID,
		ToolKind:        "workspace.apply_patch",
		Operation:       "apply",
		Status:          "completed",
		TargetPaths:     targetPaths,
		PatchHash:       patchHash,
		PatchSummary:    patchSummary,
		BeforeSummary:   beforeSummary,
		AfterSummary:    afterSummary,
		RollbackPayload: encodeLocalRollbackPayload([]localWriteExecutionFileSnapshot{rollbackSnapshot}),
		ResultSummary:   resultSummary,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		return err
	}
	if err := insertEventTx(tx, threadID, "task.completed", fmt.Sprintf("Ran task %s and completed it", title), now); err != nil {
		return err
	}
	return nil
}

func (s *localRuntimeStore) executeRollbackTask(tx *sql.Tx, workspaceRoot string, taskID string, threadID string, approvalID string, title string, rawInput string, approvalStatus string, now string) error {
	writeExecutionID, err := parseLocalRollbackInput(rawInput)
	if err != nil {
		return err
	}
	if approvalStatus != "approved" && approvalStatus != "executed" && approvalStatus != "direct" {
		return fmt.Errorf("task approval required")
	}

	source, err := readWriteExecutionByIDTx(tx, threadID, writeExecutionID)
	if err != nil {
		return err
	}
	targetPaths := append([]string(nil), source.TargetPaths...)
	patchSummary := localRollbackPatchSummary(source)
	recordFailure := func(message string, beforeSummary string, afterSummary string) error {
		if afterSummary == "" {
			afterSummary = beforeSummary
		}
		if _, updateErr := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, updated_at = ? WHERE id = ?`, "failed", message, now, taskID); updateErr != nil {
			return updateErr
		}
		if _, toolErr := tx.Exec(`
			INSERT INTO thread_tool_calls(id, thread_id, tool_id, status, summary, created_at)
			VALUES(?, ?, ?, ?, ?, ?)
		`, nextLocalID("local-tool-call"), threadID, "workspace.apply_patch.rollback", "failed", message, now); toolErr != nil {
			return toolErr
		}
		if eventErr := insertEventTx(tx, threadID, "toolcall.failed", message, now); eventErr != nil {
			return eventErr
		}
		if err := insertLocalWriteExecutionTx(tx, persistedWriteExecution{
			ID:                 nextLocalID("local-write-execution"),
			ThreadID:           threadID,
			TaskID:             taskID,
			ApprovalID:         approvalID,
			ToolKind:           "workspace.apply_patch.rollback",
			Operation:          "rollback",
			RelatedExecutionID: source.ID,
			Status:             "failed",
			TargetPaths:        targetPaths,
			PatchSummary:       patchSummary,
			BeforeSummary:      beforeSummary,
			AfterSummary:       afterSummary,
			ResultSummary:      message,
			CreatedAt:          now,
			UpdatedAt:          now,
		}); err != nil {
			return err
		}
		return insertEventTx(tx, threadID, "task.failed", message, now)
	}

	if source.Operation != "apply" {
		return recordFailure(fmt.Sprintf("rollback failed: write execution %s is not an apply execution", source.ID), "", "")
	}
	if source.Status != "completed" {
		return recordFailure(fmt.Sprintf("rollback failed: write execution %s is not completed", source.ID), "", "")
	}
	latestApply, latestErr := readLatestCompletedApplyExecutionTx(tx, threadID)
	if latestErr != nil {
		return latestErr
	}
	if latestApply.ID != source.ID {
		return recordFailure("rollback failed: only the latest completed apply execution can be rolled back", "", "")
	}
	payload, err := decodeLocalRollbackPayload(source.RollbackPayload)
	if err != nil {
		return recordFailure(fmt.Sprintf("rollback failed: %s", err.Error()), "", "")
	}
	if len(payload) == 0 {
		return recordFailure(fmt.Sprintf("rollback failed: write execution %s has no rollback payload", source.ID), "", "")
	}

	beforeSummary := localFileSnapshotSummary(localResolveRollbackPrimaryPath(workspaceRoot, payload))
	changeSummary, afterSummary, rollbackErr := applyLocalRollbackPayload(workspaceRoot, payload)
	if rollbackErr != nil {
		return recordFailure(fmt.Sprintf("rollback failed: %s", rollbackErr.Error()), beforeSummary, beforeSummary)
	}

	resultSummary := localRollbackExecutionSummary(targetPaths, changeSummary)
	if _, err := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, updated_at = ? WHERE id = ?`, "completed", resultSummary, now, taskID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO thread_tool_calls(id, thread_id, tool_id, status, summary, created_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, nextLocalID("local-tool-call"), threadID, "workspace.apply_patch.rollback", "completed", resultSummary, now); err != nil {
		return err
	}
	if err := insertEventTx(tx, threadID, "toolcall.completed", resultSummary, now); err != nil {
		return err
	}
	if err := insertLocalWriteExecutionTx(tx, persistedWriteExecution{
		ID:                 nextLocalID("local-write-execution"),
		ThreadID:           threadID,
		TaskID:             taskID,
		ApprovalID:         approvalID,
		ToolKind:           "workspace.apply_patch.rollback",
		Operation:          "rollback",
		RelatedExecutionID: source.ID,
		Status:             "completed",
		TargetPaths:        targetPaths,
		PatchSummary:       patchSummary,
		BeforeSummary:      beforeSummary,
		AfterSummary:       afterSummary,
		ResultSummary:      resultSummary,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		return err
	}
	if err := insertEventTx(tx, threadID, "task.completed", fmt.Sprintf("Rolled back task %s", title), now); err != nil {
		return err
	}
	return nil
}

func insertLocalWriteExecutionTx(tx *sql.Tx, item persistedWriteExecution) error {
	_, err := tx.Exec(`
		INSERT INTO thread_write_executions(id, thread_id, task_id, approval_id, tool_kind, operation, related_execution_id, status, target_paths, patch_hash, patch_summary, before_snapshot_summary, after_snapshot_summary, rollback_payload, result_summary, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.ThreadID, item.TaskID, item.ApprovalID, item.ToolKind, fallbackText(strings.TrimSpace(item.Operation), "apply"), strings.TrimSpace(item.RelatedExecutionID), item.Status, encodeTargetPaths(item.TargetPaths), item.PatchHash, item.PatchSummary, item.BeforeSummary, item.AfterSummary, fallbackText(item.RollbackPayload, "[]"), item.ResultSummary, item.CreatedAt, item.UpdatedAt)
	return err
}

func collectLocalWriteExecutionTargets(pathValue string, patchValue string) []string {
	targets, err := extractLocalPatchTargets(patchValue)
	if err != nil || len(targets) == 0 {
		targets = []string{strings.TrimSpace(pathValue)}
	}
	return targets
}

func localPatchExecutionSummary(targets []string, patch string) string {
	added, removed := localPatchChangeStats(patch)
	changeSummary := fmt.Sprintf("%d added / %d removed line(s)", added, removed)
	if len(targets) == 0 {
		return changeSummary
	}
	return fmt.Sprintf("applied patch to %s: %s", strings.Join(targets, ", "), changeSummary)
}

func localPatchChangeStats(patch string) (int, int) {
	added := 0
	removed := 0
	for _, line := range splitLocalPatchLines(strings.TrimSpace(patch)) {
		if strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			added++
		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			removed++
		}
	}
	return added, removed
}

func localFileSnapshotSummary(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "missing file"
		}
		return fmt.Sprintf("snapshot unavailable: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("unreadable file / %d byte(s): %v", info.Size(), err)
	}
	lineCount := len(splitLocalTextLines(string(content)))
	return fmt.Sprintf("exists / %d line(s) / %d byte(s) / sha256:%s", lineCount, len(content), shortLocalHashBytes(content))
}

func hashLocalText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func encodeLocalRollbackPayload(items []localWriteExecutionFileSnapshot) string {
	if len(items) == 0 {
		return "[]"
	}
	payload, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(payload)
}

func decodeLocalRollbackPayload(raw string) ([]localWriteExecutionFileSnapshot, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	var items []localWriteExecutionFileSnapshot
	if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
		return nil, fmt.Errorf("invalid rollback payload")
	}
	return items, nil
}

func captureLocalRollbackSnapshot(targetPath string, pathLabel string) (localWriteExecutionFileSnapshot, error) {
	content, exists, err := readLocalOptionalFile(targetPath)
	if err != nil {
		return localWriteExecutionFileSnapshot{}, err
	}
	snapshot := localWriteExecutionFileSnapshot{
		Path:          pathLabel,
		BeforeExists:  exists,
		BeforeContent: content,
	}
	if exists {
		snapshot.BeforeHash = hashLocalText(content)
	}
	return snapshot, nil
}

func readLocalOptionalFile(targetPath string) (string, bool, error) {
	content, err := os.ReadFile(targetPath)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(content), true, nil
}

func readLocalFilePresenceAndHash(targetPath string) (bool, string) {
	content, exists, err := readLocalOptionalFile(targetPath)
	if err != nil || !exists {
		return exists, ""
	}
	return true, hashLocalText(content)
}

func localResolveRollbackPrimaryPath(workspaceRoot string, payload []localWriteExecutionFileSnapshot) string {
	if len(payload) == 0 {
		return workspaceRoot
	}
	resolved, err := resolveLocalWorkspacePath(workspaceRoot, payload[0].Path)
	if err != nil {
		return workspaceRoot
	}
	return resolved
}

func applyLocalRollbackPayload(workspaceRoot string, payload []localWriteExecutionFileSnapshot) (string, string, error) {
	changes := make([]string, 0, len(payload))
	afterSummaries := make([]string, 0, len(payload))
	for _, item := range payload {
		resolvedPath, err := resolveLocalWorkspacePath(workspaceRoot, item.Path)
		if err != nil {
			return "", "", err
		}
		currentContent, currentExists, err := readLocalOptionalFile(resolvedPath)
		if err != nil {
			return "", "", err
		}
		currentHash := ""
		if currentExists {
			currentHash = hashLocalText(currentContent)
		}
		if currentExists != item.AfterExists {
			return "", "", fmt.Errorf("file drift detected for %s", item.Path)
		}
		if currentExists && currentHash != item.AfterHash {
			return "", "", fmt.Errorf("file drift detected for %s", item.Path)
		}
		if item.BeforeExists {
			if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
				return "", "", err
			}
			if err := os.WriteFile(resolvedPath, []byte(item.BeforeContent), 0o644); err != nil {
				return "", "", err
			}
			changes = append(changes, fmt.Sprintf("restored %s", item.Path))
		} else {
			if currentExists {
				if err := os.Remove(resolvedPath); err != nil {
					return "", "", err
				}
			}
			changes = append(changes, fmt.Sprintf("removed %s", item.Path))
		}
		afterSummaries = append(afterSummaries, fmt.Sprintf("%s => %s", item.Path, localFileSnapshotSummary(resolvedPath)))
	}
	return strings.Join(changes, "; "), strings.Join(afterSummaries, " | "), nil
}

func shortLocalHashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:8])
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

func toRuntimeFlagSummaries(items []persistedRuntimeFlag) []RuntimeFlagSummary {
	result := make([]RuntimeFlagSummary, 0, len(items))
	for _, item := range items {
		result = append(result, RuntimeFlagSummary{
			ThreadID:  item.ThreadID,
			Key:       item.Key,
			Value:     item.Value,
			UpdatedAt: item.UpdatedAt,
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

func parseLocalToolCallAppendInput(raw string) (string, string, string, error) {
	var input struct {
		ToolID  string `json:"toolId"`
		Status  string `json:"status"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return "", "", "", err
	}
	if strings.TrimSpace(input.ToolID) == "" {
		return "", "", "", fmt.Errorf("toolId is required")
	}
	if strings.TrimSpace(input.Status) == "" {
		return "", "", "", fmt.Errorf("status is required")
	}
	if strings.TrimSpace(input.Summary) == "" {
		return "", "", "", fmt.Errorf("summary is required")
	}
	return strings.TrimSpace(input.ToolID), strings.TrimSpace(input.Status), strings.TrimSpace(input.Summary), nil
}

func parseLocalArtifactAppendInput(raw string) (string, string, error) {
	var input struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(input.Path) == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if strings.TrimSpace(input.Kind) == "" {
		return "", "", fmt.Errorf("kind is required")
	}
	return strings.TrimSpace(input.Path), strings.TrimSpace(input.Kind), nil
}

func parseLocalRuntimeFlagSetInput(raw string) (string, string, error) {
	var input struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(input.Key) == "" {
		return "", "", fmt.Errorf("key is required")
	}
	return strings.TrimSpace(input.Key), input.Value, nil
}

func (s *localRuntimeStore) executeToolCallAppendTask(tx *sql.Tx, taskID string, threadID string, input string, now string) error {
	toolID, statusValue, summary, err := parseLocalToolCallAppendInput(input)
	if err != nil {
		return err
	}
	resultSummary := fmt.Sprintf("tool call %s appended", toolID)
	if _, err := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, updated_at = ? WHERE id = ?`, "completed", resultSummary, now, taskID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO thread_tool_calls(id, thread_id, tool_id, status, summary, created_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, nextLocalID("local-tool-call"), threadID, toolID, statusValue, summary, now); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-event"), threadID, "toolcall.appended", resultSummary, now); err != nil {
		return err
	}
	return nil
}

func (s *localRuntimeStore) executeArtifactAppendTask(tx *sql.Tx, taskID string, threadID string, input string, now string) error {
	pathValue, kindValue, err := parseLocalArtifactAppendInput(input)
	if err != nil {
		return err
	}
	resultSummary := fmt.Sprintf("artifact %s appended", kindValue)
	if _, err := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, updated_at = ? WHERE id = ?`, "completed", resultSummary, now, taskID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO thread_artifacts(id, thread_id, path, kind, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-artifact"), threadID, pathValue, kindValue, now); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-event"), threadID, "artifact.appended", resultSummary, now); err != nil {
		return err
	}
	return nil
}

func (s *localRuntimeStore) executeRuntimeFlagSetTask(tx *sql.Tx, taskID string, threadID string, input string, now string) error {
	keyValue, flagValue, err := parseLocalRuntimeFlagSetInput(input)
	if err != nil {
		return err
	}
	resultSummary := fmt.Sprintf("runtime flag %s updated", keyValue)
	if _, err := tx.Exec(`UPDATE tasks SET status = ?, result_summary = ?, updated_at = ? WHERE id = ?`, "completed", resultSummary, now, taskID); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO thread_runtime_flags(thread_id, key, value, updated_at)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(thread_id, key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, threadID, keyValue, flagValue, now); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, nextLocalID("local-event"), threadID, "runtimeflag.updated", resultSummary, now); err != nil {
		return err
	}
	return nil
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

func parseLocalRollbackInput(raw string) (string, error) {
	var input struct {
		WriteExecutionID string `json:"writeExecutionId"`
	}
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.WriteExecutionID) == "" {
		return "", fmt.Errorf("writeExecutionId is required")
	}
	return strings.TrimSpace(input.WriteExecutionID), nil
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

func localRollbackApprovalSummary(targets []string) string {
	label := strings.Join(targets, ", ")
	if strings.TrimSpace(label) == "" {
		label = "workspace"
	}
	return fmt.Sprintf("approval required for rollback of %s", label)
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

func localRollbackExecutionSummary(targets []string, changeSummary string) string {
	label := strings.Join(targets, ", ")
	if strings.TrimSpace(label) == "" {
		label = "workspace"
	}
	if strings.TrimSpace(changeSummary) == "" {
		return fmt.Sprintf("rolled back patch on %s", label)
	}
	return fmt.Sprintf("rolled back patch on %s: %s", label, changeSummary)
}

func localRollbackPatchSummary(source persistedWriteExecution) string {
	if strings.TrimSpace(source.PatchSummary) == "" {
		return fmt.Sprintf("rollback of %s", source.ID)
	}
	return fmt.Sprintf("rollback of %s", source.PatchSummary)
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
		AppName:             "gen-code",
		AppEnv:              "local",
		Port:                10008,
		DesktopReady:        true,
		RuntimeState:        "degraded",
		RuntimeReady:        false,
		RuntimeMessage:      err.Error(),
		RuntimeSource:       "local-fallback",
		RuntimeSourceDetail: "project-local SQLite fallback because the canonical app-server runtime is unavailable",
		RuntimeTrust:        "degraded",
		StateStore:          "sqlite",
		SkillsByGroup:       map[string][]string{},
		ToolsByGroup:        map[string][]string{},
		MCPByGroup:          map[string][]string{},
		MissingPaths:        []string{},
		UpdatedAt:           time.Now().Format(time.RFC3339),
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
