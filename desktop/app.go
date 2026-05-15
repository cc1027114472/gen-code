package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type App struct {
	ctx context.Context
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
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
	Title    string `json:"title"`
	Status   string `json:"status"`
}

type EventSummary struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
	Type     string `json:"type"`
	Message  string `json:"message"`
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
	WorkspaceID    string `json:"workspaceId"`
	ProjectRoot    string `json:"projectRoot"`
	ThreadCount    int    `json:"threadCount"`
	ActiveThreadID string `json:"activeThreadId"`
}

type apiThread struct {
	ID                  string `json:"id"`
	WorkspaceID         string `json:"workspaceId"`
	Name                string `json:"name"`
	Status              string `json:"status"`
	ActiveModel         string `json:"activeModel"`
	PermissionMode      string `json:"permissionMode"`
	MessageHistoryCount int    `json:"messageHistoryCount"`
	ToolCallCount       int    `json:"toolCallCount"`
	ArtifactCount       int    `json:"artifactCount"`
	CreatedAt           string `json:"createdAt"`
	IsActive            bool   `json:"isActive"`
}

type apiTask struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Title     string `json:"title"`
	Status    string `json:"status"`
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
	ID          string `json:"id"`
	Group       string `json:"group"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source"`
}

type apiTool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Permission  string `json:"permissionMode"`
	Source      string `json:"source"`
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

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) GetAppInfo() string {
	return "gen-code desktop shell ready"
}

func (a *App) GetRuntimeStatus() RuntimeStatus {
	status, err := collectRuntimeStatus()
	if err != nil {
		return RuntimeStatus{
			AppName:        "gen-code",
			AppEnv:         "local",
			Port:           10008,
			DesktopReady:   true,
			RuntimeState:   "unavailable",
			RuntimeReady:   false,
			RuntimeMessage: err.Error(),
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
	bridge, err := fetchBridgeCheck()
	if err != nil {
		return BridgeCheckResult{
			OK:          false,
			Message:     err.Error(),
			CheckedAt:   time.Now().Format(time.RFC3339),
			RuntimeHint: "gen-code / local",
		}
	}
	return bridge
}

func (a *App) CreateThread(name string) RuntimeStatus {
	_ = createThread(name)
	return a.GetRuntimeStatus()
}

func (a *App) ActivateThread(id string) RuntimeStatus {
	_ = activateThread(id)
	return a.GetRuntimeStatus()
}

func (a *App) CreateTask(threadID string, title string) RuntimeStatus {
	_ = createTask(threadID, title)
	return a.GetRuntimeStatus()
}

func collectRuntimeStatus() (RuntimeStatus, error) {
	workspaceRoot, err := findWorkspaceRoot()
	if err != nil {
		return RuntimeStatus{}, err
	}

	const port = 10008
	statusEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/runtime/status", port)
	skillsEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/skills", port)
	toolsEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/tools", port)
	mcpEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/mcp/servers", port)
	threadsEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/threads", port)

	runtimeStatus := apiStatus{}
	if err := fetchEnvelope(statusEndpoint, &runtimeStatus); err != nil {
		return RuntimeStatus{}, err
	}

	var skillsPayload struct {
		Items []apiSkill `json:"items"`
	}
	if err := fetchEnvelope(skillsEndpoint, &skillsPayload); err != nil {
		return RuntimeStatus{}, err
	}

	var toolsPayload struct {
		Items []apiTool `json:"items"`
	}
	if err := fetchEnvelope(toolsEndpoint, &toolsPayload); err != nil {
		return RuntimeStatus{}, err
	}

	var mcpPayload struct {
		Items []apiMCPServer `json:"items"`
	}
	if err := fetchEnvelope(mcpEndpoint, &mcpPayload); err != nil {
		return RuntimeStatus{}, err
	}

	var threadPayload struct {
		Items []apiThread `json:"items"`
	}
	if err := fetchEnvelope(threadsEndpoint, &threadPayload); err != nil {
		return RuntimeStatus{}, err
	}

	tasksPayload := struct {
		Items []apiTask `json:"items"`
	}{}
	eventsPayload := struct {
		Items []apiEvent `json:"items"`
	}{}
	if runtimeStatus.ActiveThreadID != "" {
		tasksEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/threads/%s/tasks", port, runtimeStatus.ActiveThreadID)
		eventsEndpoint := fmt.Sprintf("http://127.0.0.1:%d/api/threads/%s/events", port, runtimeStatus.ActiveThreadID)
		if err := fetchEnvelope(tasksEndpoint, &tasksPayload); err != nil {
			return RuntimeStatus{}, err
		}
		if err := fetchEnvelope(eventsEndpoint, &eventsPayload); err != nil {
			return RuntimeStatus{}, err
		}
	}

	missingPaths := []string{}
	desktopModule := filepath.Join(workspaceRoot, "desktop", "go.mod")
	if _, err := os.Stat(desktopModule); err != nil {
		missingPaths = append(missingPaths, desktopModule)
	}
	desktopFrontend := filepath.Join(workspaceRoot, "desktop", "frontend", "package.json")
	if _, err := os.Stat(desktopFrontend); err != nil {
		missingPaths = append(missingPaths, desktopFrontend)
	}

	return RuntimeStatus{
		AppName:         "gen-code",
		AppEnv:          "local",
		Port:            port,
		Debug:           false,
		ShutdownTimeout: "10s",
		TrustedProxies:  []string{"127.0.0.1"},
		LogLevel:        "info",
		HTTPAccessLog:   true,
		WorkspaceRoot:   workspaceRoot,
		WorkspaceID:     runtimeStatus.WorkspaceID,
		ProjectRoot:     runtimeStatus.ProjectRoot,
		ThreadCount:     runtimeStatus.ThreadCount,
		ActiveThreadID:  runtimeStatus.ActiveThreadID,
		Threads:         mapThreads(threadPayload.Items),
		Tasks:           mapTasks(tasksPayload.Items),
		Events:          mapEvents(eventsPayload.Items),
		DesktopReady:    true,
		RuntimeState:    runtimeStatus.State,
		RuntimeReady:    runtimeStatus.Ready,
		RuntimeMessage:  runtimeStatus.Message,
		SkillsByGroup:   groupSkills(skillsPayload.Items),
		ToolsByGroup:    groupTools(toolsPayload.Items),
		MCPByGroup:      groupMCPServers(mcpPayload.Items),
		MissingPaths:    missingPaths,
		UpdatedAt:       time.Now().Format(time.RFC3339),
	}, nil
}

func fetchBridgeCheck() (BridgeCheckResult, error) {
	const port = 10008
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/api/bridge/check", port)
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(`{}`))
	if err != nil {
		return BridgeCheckResult{}, err
	}
	request.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return BridgeCheckResult{}, err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return BridgeCheckResult{}, fmt.Errorf("bridge check failed: %s", response.Status)
	}

	var envelope apiEnvelope[apiBridgeCheck]
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return BridgeCheckResult{}, err
	}
	if envelope.Code != 0 {
		return BridgeCheckResult{}, fmt.Errorf("bridge check failed: %s", envelope.Message)
	}

	return BridgeCheckResult{
		OK:          envelope.Data.OK,
		Message:     envelope.Data.Message,
		CheckedAt:   time.Now().Format(time.RFC3339),
		RuntimeHint: "gen-code / local",
	}, nil
}

func createThread(name string) error {
	const port = 10008
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/api/threads", port)
	payload := map[string]string{"name": name}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	return doMutation(request)
}

func activateThread(id string) error {
	const port = 10008
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/api/threads/%s/activate", port, id)
	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(`{}`))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	return doMutation(request)
}

func createTask(threadID string, title string) error {
	const port = 10008
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/api/threads/%s/tasks", port, threadID)
	payload := map[string]string{"title": title}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	return doMutation(request)
}

func doMutation(request *http.Request) error {
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("request failed: %s", response.Status)
	}

	var envelope apiEnvelope[map[string]any]
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		return fmt.Errorf("request failed: %s", envelope.Message)
	}
	return nil
}

func fetchEnvelope[T any](url string, target *T) error {
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("request failed: %s", response.Status)
	}

	var envelope apiEnvelope[T]
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		return err
	}
	if envelope.Code != 0 {
		return fmt.Errorf("request failed: %s", envelope.Message)
	}

	*target = envelope.Data
	return nil
}

func groupSkills(items []apiSkill) map[string][]string {
	groups := map[string][]string{
		"common": {},
		"codex":  {},
		"cc":     {},
	}
	for _, item := range items {
		group := item.Group
		if group == "" {
			group = "common"
		}
		groups[group] = append(groups[group], item.ID)
	}
	return normalizeGroups(groups)
}

func groupTools(items []apiTool) map[string][]string {
	groups := map[string][]string{}
	for _, item := range items {
		group := item.Source
		if group == "" {
			group = "runtime"
		}
		label := item.ID
		if item.Permission != "" {
			label = fmt.Sprintf("%s (%s)", item.ID, item.Permission)
		}
		groups[group] = append(groups[group], label)
	}
	return normalizeGroups(groups)
}

func groupMCPServers(items []apiMCPServer) map[string][]string {
	groups := map[string][]string{}
	for _, item := range items {
		group := item.Source
		if group == "" {
			group = "unspecified"
		}
		label := fmt.Sprintf("%s (tools:%d resources:%d)", item.ID, item.ToolCount, item.ResourceCount)
		groups[group] = append(groups[group], label)
	}
	return normalizeGroups(groups)
}

func normalizeGroups(groups map[string][]string) map[string][]string {
	for key, items := range groups {
		deduped := make([]string, 0, len(items))
		seen := map[string]struct{}{}
		for _, item := range items {
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			deduped = append(deduped, item)
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
			ID:       item.ID,
			ThreadID: item.ThreadID,
			Title:    item.Title,
			Status:   item.Status,
		})
	}
	return result
}

func mapEvents(items []apiEvent) []EventSummary {
	result := make([]EventSummary, 0, len(items))
	for _, item := range items {
		result = append(result, EventSummary{
			ID:       item.ID,
			ThreadID: item.ThreadID,
			Type:     item.Type,
			Message:  item.Message,
		})
	}
	return result
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
