package main

import (
	"bytes"
	"context"
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
	AppName         string              `json:"appName"`
	AppEnv          string              `json:"appEnv"`
	Port            int                 `json:"port"`
	Debug           bool                `json:"debug"`
	ShutdownTimeout string              `json:"shutdownTimeout"`
	TrustedProxies  []string            `json:"trustedProxies"`
	LogLevel        string              `json:"logLevel"`
	HTTPAccessLog   bool                `json:"httpAccessLog"`
	WorkspaceRoot   string              `json:"workspaceRoot"`
	WorkspaceID     string              `json:"workspaceId"`
	ProjectRoot     string              `json:"projectRoot"`
	ThreadCount     int                 `json:"threadCount"`
	ActiveThreadID  string              `json:"activeThreadId"`
	Threads         []ThreadSummary     `json:"threads"`
	Tasks           []TaskSummary       `json:"tasks"`
	Events          []EventSummary      `json:"events"`
	DesktopReady    bool                `json:"desktopReady"`
	RuntimeState    string              `json:"runtimeState"`
	RuntimeReady    bool                `json:"runtimeReady"`
	RuntimeMessage  string              `json:"runtimeMessage"`
	RuntimeSource   string              `json:"runtimeSource"`
	SupportsSSE     bool                `json:"supportsSSE"`
	SSEEndpoint     string              `json:"sseEndpoint"`
	LastSyncAt      string              `json:"lastSyncAt"`
	SkillsByGroup   map[string][]string `json:"skillsByGroup"`
	ToolsByGroup    map[string][]string `json:"toolsByGroup"`
	MCPByGroup      map[string][]string `json:"mcpByGroup"`
	MissingPaths    []string            `json:"missingPaths"`
	UpdatedAt       string              `json:"updatedAt"`
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
	mu     sync.Mutex
	status localRuntimeState
}

type localRuntimeState struct {
	workspaceID    string
	projectRoot    string
	activeThreadID string
	threadSeq      int
	taskSeq        int
	eventSeq       int
	threads        []ThreadSummary
	tasks          []TaskSummary
	events         []EventSummary
}

func NewApp() *App {
	return &App{
		store: newLocalRuntimeStore(),
	}
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
		RuntimeHint: fmt.Sprintf("%s / %s", status.RuntimeSource, status.RuntimeState),
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

	localStatus := a.store.Snapshot(*baseStatus)
	localStatus.RuntimeMessage = "External runtime is unavailable, switched to desktop-local fallback."
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

	return &RuntimeStatus{
		AppName:         "gen-code",
		AppEnv:          "local",
		Port:            10008,
		Debug:           false,
		ShutdownTimeout: "10s",
		TrustedProxies:  []string{"127.0.0.1"},
		LogLevel:        "info",
		HTTPAccessLog:   true,
		WorkspaceRoot:   workspaceRoot,
		DesktopReady:    true,
		SkillsByGroup:   map[string][]string{},
		ToolsByGroup:    map[string][]string{},
		MCPByGroup:      map[string][]string{},
		MissingPaths:    missingPaths,
		UpdatedAt:       time.Now().Format(time.RFC3339),
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
	status.SupportsSSE = true
	if runtimeStatus.ActiveThreadID != "" {
		status.SSEEndpoint = strings.TrimRight(runtimeBaseURL(), "/") + "/api/threads/" + url.PathEscape(runtimeStatus.ActiveThreadID) + "/events/stream"
	}
	status.LastSyncAt = time.Now().Format(time.RFC3339)
	status.SkillsByGroup = groupSkills(skillsPayload.Items)
	status.ToolsByGroup = groupTools(toolsPayload.Items)
	status.MCPByGroup = groupMCPServers(mcpPayload.Items)
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
	return &localRuntimeStore{
		status: localRuntimeState{
			workspaceID: "desktop-local",
		},
	}
}

func (s *localRuntimeStore) Snapshot(base RuntimeStatus) RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ensureProjectRoot(base.WorkspaceRoot)
	return s.snapshotLocked(base)
}

func (s *localRuntimeStore) CreateThread(name string) RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.status.threadSeq++
	threadID := fmt.Sprintf("local-thread-%d", s.status.threadSeq)
	threadName := fallbackText(strings.TrimSpace(name), fmt.Sprintf("%s %d", defaultThreadName, s.status.threadSeq))
	now := time.Now().Format(time.RFC3339)

	for i := range s.status.threads {
		s.status.threads[i].IsActive = false
	}

	thread := ThreadSummary{
		ID:             threadID,
		Name:           threadName,
		Status:         "idle",
		PermissionMode: "workspace-write",
		IsActive:       true,
	}

	s.status.threads = append([]ThreadSummary{thread}, s.status.threads...)
	s.status.activeThreadID = threadID
	s.appendEventLocked(threadID, "thread.created", fmt.Sprintf("Created thread %s", threadName), now)

	base := s.baseStatusLocked()
	return s.snapshotLocked(base)
}

func (s *localRuntimeStore) ActivateThread(id string) RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedID := strings.TrimSpace(id)
	for i := range s.status.threads {
		s.status.threads[i].IsActive = s.status.threads[i].ID == trimmedID
		if s.status.threads[i].IsActive {
			s.status.activeThreadID = trimmedID
			s.appendEventLocked(trimmedID, "thread.activated", fmt.Sprintf("Activated thread %s", s.status.threads[i].Name), time.Now().Format(time.RFC3339))
		}
	}

	base := s.baseStatusLocked()
	return s.snapshotLocked(base)
}

func (s *localRuntimeStore) CreateTask(threadID string, title string) RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedThreadID := strings.TrimSpace(threadID)
	if trimmedThreadID == "" {
		trimmedThreadID = s.status.activeThreadID
	}
	if trimmedThreadID == "" {
		base := s.baseStatusLocked()
		return s.snapshotLocked(base)
	}

	s.status.taskSeq++
	now := time.Now().Format(time.RFC3339)
	task := TaskSummary{
		ID:        fmt.Sprintf("local-task-%d", s.status.taskSeq),
		ThreadID:  trimmedThreadID,
		Title:     fallbackText(strings.TrimSpace(title), fmt.Sprintf("%s %d", defaultTaskTitle, s.status.taskSeq)),
		Status:    "queued",
		UpdatedAt: now,
	}
	s.status.tasks = append([]TaskSummary{task}, s.status.tasks...)
	s.appendEventLocked(trimmedThreadID, "task.created", fmt.Sprintf("Queued task %s", task.Title), now)

	base := s.baseStatusLocked()
	return s.snapshotLocked(base)
}

func (s *localRuntimeStore) AdvanceTask(taskID string) RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	trimmedID := strings.TrimSpace(taskID)
	now := time.Now().Format(time.RFC3339)
	for i := range s.status.tasks {
		if s.status.tasks[i].ID != trimmedID {
			continue
		}
		s.status.tasks[i].Status = nextTaskStatus(s.status.tasks[i].Status)
		s.status.tasks[i].UpdatedAt = now
		s.appendEventLocked(s.status.tasks[i].ThreadID, "task.updated", fmt.Sprintf("Task %s moved to %s", s.status.tasks[i].Title, s.status.tasks[i].Status), now)
		break
	}

	base := s.baseStatusLocked()
	return s.snapshotLocked(base)
}

func (s *localRuntimeStore) appendEventLocked(threadID string, eventType string, message string, createdAt string) {
	s.status.eventSeq++
	event := EventSummary{
		ID:        fmt.Sprintf("local-event-%d", s.status.eventSeq),
		ThreadID:  threadID,
		Type:      eventType,
		Message:   message,
		CreatedAt: createdAt,
	}
	s.status.events = append([]EventSummary{event}, s.status.events...)
	if len(s.status.events) > 24 {
		s.status.events = s.status.events[:24]
	}
}

func (s *localRuntimeStore) baseStatusLocked() RuntimeStatus {
	return RuntimeStatus{
		AppName:         "gen-code",
		AppEnv:          "local",
		Port:            10008,
		Debug:           false,
		ShutdownTimeout: "10s",
		TrustedProxies:  []string{"127.0.0.1"},
		LogLevel:        "info",
		HTTPAccessLog:   true,
		WorkspaceRoot:   s.status.projectRoot,
		ProjectRoot:     s.status.projectRoot,
		DesktopReady:    true,
		SkillsByGroup:   map[string][]string{},
		ToolsByGroup:    map[string][]string{},
		MCPByGroup:      map[string][]string{},
	}
}

func (s *localRuntimeStore) snapshotLocked(base RuntimeStatus) RuntimeStatus {
	status := base
	status.WorkspaceID = fallbackText(s.status.workspaceID, "desktop-local")
	status.ProjectRoot = fallbackText(s.status.projectRoot, base.ProjectRoot)
	status.ThreadCount = len(s.status.threads)
	status.ActiveThreadID = s.status.activeThreadID
	status.Threads = append([]ThreadSummary(nil), s.status.threads...)
	status.Tasks = filterTasksByThread(s.status.tasks, s.status.activeThreadID)
	status.Events = filterEventsByThread(s.status.events, s.status.activeThreadID)
	status.RuntimeState = "fallback"
	status.RuntimeReady = true
	status.RuntimeMessage = "Using desktop-local runtime fallback because no external runtime is connected."
	status.RuntimeSource = "desktop-local"
	status.SupportsSSE = false
	status.SSEEndpoint = ""
	status.LastSyncAt = ""
	status.UpdatedAt = time.Now().Format(time.RFC3339)
	return status
}

func (s *localRuntimeStore) ensureProjectRoot(workspaceRoot string) {
	if workspaceRoot == "" {
		return
	}
	if s.status.projectRoot == "" {
		s.status.projectRoot = workspaceRoot
	}
}

func filterTasksByThread(tasks []TaskSummary, threadID string) []TaskSummary {
	if threadID == "" {
		return append([]TaskSummary(nil), tasks...)
	}
	result := make([]TaskSummary, 0, len(tasks))
	for _, task := range tasks {
		if task.ThreadID == threadID {
			result = append(result, task)
		}
	}
	return result
}

func filterEventsByThread(events []EventSummary, threadID string) []EventSummary {
	if threadID == "" {
		return append([]EventSummary(nil), events...)
	}
	result := make([]EventSummary, 0, len(events))
	for _, event := range events {
		if event.ThreadID == threadID {
			result = append(result, event)
		}
	}
	return result
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
