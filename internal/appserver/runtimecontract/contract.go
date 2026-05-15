package runtimecontract

import "context"

// Status describes the current runtime availability exposed by the app server.
type Status struct {
	State          string `json:"state"`
	Ready          bool   `json:"ready"`
	Message        string `json:"message,omitempty"`
	RuntimeSource  string `json:"runtimeSource,omitempty"`
	WorkspaceID    string `json:"workspaceId,omitempty"`
	ProjectRoot    string `json:"projectRoot,omitempty"`
	ThreadCount    int    `json:"threadCount,omitempty"`
	ActiveThreadID string `json:"activeThreadId,omitempty"`
	TaskCount      int    `json:"taskCount,omitempty"`
	EventCount     int    `json:"eventCount,omitempty"`
}

// WorkspaceDescriptor describes the current workspace container.
type WorkspaceDescriptor struct {
	ID                string `json:"id"`
	ProjectRoot       string `json:"projectRoot"`
	SharedDocsRoot    string `json:"sharedDocsRoot"`
	CreatedAt         string `json:"createdAt"`
	ActiveThreadCount int    `json:"activeThreadCount"`
}

// ThreadDescriptor describes a single workspace thread.
type ThreadDescriptor struct {
	ID                  string `json:"id"`
	WorkspaceID         string `json:"workspaceId"`
	Name                string `json:"name"`
	Status              string `json:"status"`
	ActiveModel         string `json:"activeModel,omitempty"`
	PermissionMode      string `json:"permissionMode"`
	MessageHistoryCount int    `json:"messageHistoryCount"`
	ToolCallCount       int    `json:"toolCallCount"`
	ArtifactCount       int    `json:"artifactCount"`
	CreatedAt           string `json:"createdAt"`
	IsActive            bool   `json:"isActive"`
}

// CreateThreadRequest defines the minimum request body for creating a thread.
type CreateThreadRequest struct {
	Name           string `json:"name"`
	ActiveModel    string `json:"activeModel"`
	PermissionMode string `json:"permissionMode"`
}

// TaskDescriptor describes a single thread-local task.
type TaskDescriptor struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt,omitempty"`
}

// EventDescriptor describes a single thread event for logs/activity views.
type EventDescriptor struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

// CreateTaskRequest defines the minimum request body for creating a thread task.
type CreateTaskRequest struct {
	Title string `json:"title"`
}

// UpdateTaskStatusRequest defines the minimum request body for updating a task status.
type UpdateTaskStatusRequest struct {
	Status string `json:"status"`
}

// Skill describes an available runtime skill.
type Skill struct {
	ID          string `json:"id"`
	Group       string `json:"group"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
}

// Tool describes an available runtime tool.
type Tool struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Permission  string `json:"permissionMode,omitempty"`
	Source      string `json:"source,omitempty"`
}

// MCPServer describes a configured MCP server.
type MCPServer struct {
	ID            string `json:"id"`
	Source        string `json:"source,omitempty"`
	Enabled       bool   `json:"enabled"`
	ToolCount     int    `json:"toolCount"`
	ResourceCount int    `json:"resourceCount"`
	Status        string `json:"status,omitempty"`
}

// BridgeCheckResult describes the result of a bridge connectivity probe.
type BridgeCheckResult struct {
	OK      bool           `json:"ok"`
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// Service is the backend contract consumed by the HTTP layer.
type Service interface {
	Status(ctx context.Context) (Status, error)
	Workspace(ctx context.Context) (WorkspaceDescriptor, error)
	Threads(ctx context.Context) ([]ThreadDescriptor, error)
	CreateThread(ctx context.Context, request CreateThreadRequest) (ThreadDescriptor, error)
	Thread(ctx context.Context, id string) (ThreadDescriptor, error)
	ActivateThread(ctx context.Context, id string) (ThreadDescriptor, error)
	Tasks(ctx context.Context, threadID string) ([]TaskDescriptor, error)
	CreateTask(ctx context.Context, threadID string, request CreateTaskRequest) (TaskDescriptor, error)
	UpdateTaskStatus(ctx context.Context, threadID string, taskID string, request UpdateTaskStatusRequest) (TaskDescriptor, error)
	Events(ctx context.Context, threadID string) ([]EventDescriptor, error)
	StreamEvents(ctx context.Context, threadID string) (<-chan EventDescriptor, error)
	Skills(ctx context.Context) ([]Skill, error)
	Tools(ctx context.Context) ([]Tool, error)
	MCPServers(ctx context.Context) ([]MCPServer, error)
	CheckBridge(ctx context.Context, request map[string]any) (BridgeCheckResult, error)
}

// NewNoopService returns a placeholder runtime service until the core runtime is wired in.
func NewNoopService() Service {
	return noopService{}
}

type noopService struct{}

func (noopService) Status(context.Context) (Status, error) {
	return Status{
		State:   "initializing",
		Ready:   false,
		Message: "runtime service not configured",
	}, nil
}

func (noopService) Skills(context.Context) ([]Skill, error) {
	return []Skill{}, nil
}

func (noopService) Workspace(context.Context) (WorkspaceDescriptor, error) {
	return WorkspaceDescriptor{}, nil
}

func (noopService) Threads(context.Context) ([]ThreadDescriptor, error) {
	return []ThreadDescriptor{}, nil
}

func (noopService) CreateThread(context.Context, CreateThreadRequest) (ThreadDescriptor, error) {
	return ThreadDescriptor{}, nil
}

func (noopService) Thread(context.Context, string) (ThreadDescriptor, error) {
	return ThreadDescriptor{}, nil
}

func (noopService) ActivateThread(context.Context, string) (ThreadDescriptor, error) {
	return ThreadDescriptor{}, nil
}

func (noopService) Tasks(context.Context, string) ([]TaskDescriptor, error) {
	return []TaskDescriptor{}, nil
}

func (noopService) CreateTask(context.Context, string, CreateTaskRequest) (TaskDescriptor, error) {
	return TaskDescriptor{}, nil
}

func (noopService) UpdateTaskStatus(context.Context, string, string, UpdateTaskStatusRequest) (TaskDescriptor, error) {
	return TaskDescriptor{}, nil
}

func (noopService) Events(context.Context, string) ([]EventDescriptor, error) {
	return []EventDescriptor{}, nil
}

func (noopService) StreamEvents(context.Context, string) (<-chan EventDescriptor, error) {
	ch := make(chan EventDescriptor)
	close(ch)
	return ch, nil
}

func (noopService) Tools(context.Context) ([]Tool, error) {
	return []Tool{}, nil
}

func (noopService) MCPServers(context.Context) ([]MCPServer, error) {
	return []MCPServer{}, nil
}

func (noopService) CheckBridge(context.Context, map[string]any) (BridgeCheckResult, error) {
	return BridgeCheckResult{
		OK:      false,
		Message: "runtime service not configured",
	}, nil
}
