package main

import (
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
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultRuntimeBaseURL = "http://127.0.0.1:10008"
	defaultTaskTitle      = "New Task"
	defaultThreadName     = "New Thread"
)

type App struct {
	ctx   context.Context
	store *localRuntimeStore
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
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
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
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updatedAt"`
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
	ID         string `json:"id"`
	Permission string `json:"permissionMode"`
	Source     string `json:"source"`
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
	ID        string
	ThreadID  string
	Title     string
	Status    string
	UpdatedAt string
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
		store: newLocalRuntimeStore(),
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

func (a *App) CreateTask(threadID string, title string) RuntimeStatus {
	if status, err := createTask(newRuntimeClient(), threadID, title); err == nil {
		return status
	}
	return a.store.CreateTask(threadID, title)
}

func (a *App) AdvanceTask(taskID string) RuntimeStatus {
	if status, err := advanceTask(newRuntimeClient(), a.GetRuntimeStatus().ActiveThreadID, taskID, a.GetRuntimeStatus().Tasks); err == nil {
		return status
	}
	return a.store.AdvanceTask(taskID)
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
	eventsPayload := struct {
		Items []apiEvent `json:"items"`
	}{}
	if runtimeStatus.ActiveThreadID != "" {
		threadID := url.PathEscape(runtimeStatus.ActiveThreadID)
		if err := client.fetchEnvelope("/api/threads/"+threadID+"/tasks", &tasksPayload); err != nil {
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
	status.RecoverySummary = fmt.Sprintf("Live runtime connected. Active thread: %s, tasks: %d.", fallbackText(runtimeStatus.ActiveThreadID, "none"), len(status.Tasks))
	status.UpdatedAt = time.Now().Format(time.RFC3339)
	return status, nil
}

func newRuntimeClient() runtimeClient {
	return runtimeClient{
		baseURL: strings.TrimRight(runtimeBaseURL(), "/"),
		client: http.Client{
			Timeout: 1500 * time.Millisecond,
		},
	}
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

func createTask(client runtimeClient, threadID string, title string) (RuntimeStatus, error) {
	trimmedThreadID := strings.TrimSpace(threadID)
	if trimmedThreadID == "" {
		return RuntimeStatus{}, fmt.Errorf("thread id is required")
	}
	taskTitle := fallbackText(strings.TrimSpace(title), defaultTaskTitle)
	var created map[string]any
	if err := client.postEnvelope("/api/threads/"+url.PathEscape(trimmedThreadID)+"/tasks", map[string]string{"title": taskTitle}, &created); err != nil {
		return RuntimeStatus{}, err
	}
	return NewApp().GetRuntimeStatus(), nil
}

func advanceTask(client runtimeClient, threadID string, taskID string, tasks []TaskSummary) (RuntimeStatus, error) {
	trimmedThreadID := strings.TrimSpace(threadID)
	trimmedTaskID := strings.TrimSpace(taskID)
	if trimmedThreadID == "" || trimmedTaskID == "" {
		return RuntimeStatus{}, fmt.Errorf("thread id and task id are required")
	}

	nextStatus := "running"
	for _, task := range tasks {
		if task.ID != trimmedTaskID {
			continue
		}
		nextStatus = nextTaskStatus(task.Status)
		break
	}

	var updated map[string]any
	if err := client.postEnvelope("/api/threads/"+url.PathEscape(trimmedThreadID)+"/tasks/"+url.PathEscape(trimmedTaskID)+"/status", map[string]string{"status": nextStatus}, &updated); err != nil {
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
		label := item.ID
		if permission := strings.TrimSpace(item.Permission); permission != "" {
			label = fmt.Sprintf("%s (%s)", item.ID, permission)
		}
		groups[group] = append(groups[group], label)
	}
	return normalizeGroups(groups)
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
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Title:     item.Title,
			Status:    item.Status,
			UpdatedAt: item.UpdatedAt,
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
	`, threadID, workspaceID, threadName, "idle", "", "workspace-write", now); err != nil {
		return runtimeErrorStatus(err)
	}
	if err := s.saveWorkspace(tx, base.WorkspaceRoot, threadID); err != nil {
		return runtimeErrorStatus(err)
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, fmt.Sprintf("local-event-%d", time.Now().UnixNano()), threadID, "thread.created", fmt.Sprintf("Created thread %s", threadName), now); err != nil {
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
	`, fmt.Sprintf("local-event-%d", time.Now().UnixNano()), threadID, "thread.activated", fmt.Sprintf("Activated thread %s", threadID), now); err != nil {
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

func (s *localRuntimeStore) CreateTask(threadID string, title string) RuntimeStatus {
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

	taskTitle := fallbackText(strings.TrimSpace(title), defaultTaskTitle)
	now := time.Now().Format(time.RFC3339)
	taskID := fmt.Sprintf("local-task-%d", time.Now().UnixNano())

	tx, err := s.db.Begin()
	if err != nil {
		return runtimeErrorStatus(err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		INSERT INTO tasks(id, thread_id, title, status, updated_at, created_at)
		VALUES(?, ?, ?, ?, ?, ?)
	`, taskID, trimmedThreadID, taskTitle, "queued", now, now); err != nil {
		return runtimeErrorStatus(err)
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, fmt.Sprintf("local-event-%d", time.Now().UnixNano()), trimmedThreadID, "task.created", fmt.Sprintf("Queued task %s", taskTitle), now); err != nil {
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

func (s *localRuntimeStore) AdvanceTask(taskID string) RuntimeStatus {
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

	row := s.db.QueryRow(`SELECT thread_id, title, status FROM tasks WHERE id = ?`, trimmedTaskID)
	var threadID string
	var title string
	var currentStatus string
	if err := row.Scan(&threadID, &title, &currentStatus); err != nil {
		status, snapshotErr := s.snapshotLocked(*base)
		if snapshotErr != nil {
			return runtimeErrorStatus(snapshotErr)
		}
		return status
	}

	nextStatus := nextTaskStatus(currentStatus)
	now := time.Now().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return runtimeErrorStatus(err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE tasks SET status = ?, updated_at = ? WHERE id = ?`, nextStatus, now, trimmedTaskID); err != nil {
		return runtimeErrorStatus(err)
	}
	if _, err := tx.Exec(`
		INSERT INTO events(id, thread_id, type, message, created_at)
		VALUES(?, ?, ?, ?, ?)
	`, fmt.Sprintf("local-event-%d", time.Now().UnixNano()), threadID, "task.updated", fmt.Sprintf("Task %s moved to %s", title, nextStatus), now); err != nil {
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
			status TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			type TEXT NOT NULL,
			message TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
	}

	for _, statement := range schema {
		if _, err := db.Exec(statement); err != nil {
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
	events, err := s.readEvents(workspaceRecord.ActiveThreadID)
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
	status.Events = toEventSummaries(events)
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
	status.RecoverySummary = fmt.Sprintf("Recovered %d thread(s), %d task(s), %d event(s) from project-local state store.", len(threads), len(tasks), len(events))
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
		SELECT id, thread_id, title, status, updated_at
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
		if err := rows.Scan(&item.ID, &item.ThreadID, &item.Title, &item.Status, &item.UpdatedAt); err != nil {
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
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Title:     item.Title,
			Status:    item.Status,
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

func nextTaskStatus(current string) string {
	switch current {
	case "queued":
		return "running"
	case "running":
		return "completed"
	case "completed":
		return "failed"
	case "failed":
		return "failed"
	default:
		return "queued"
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
