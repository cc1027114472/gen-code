package runtimecontract

import "context"

// Status describes the current runtime availability exposed by the app server.
type Status struct {
	State          string `json:"state"`
	Ready          bool   `json:"ready"`
	Message        string `json:"message,omitempty"`
	RuntimeSource  string `json:"runtimeSource,omitempty"`
	StateStore     string `json:"stateStore,omitempty"`
	StatePath      string `json:"statePath,omitempty"`
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
	ID            string `json:"id"`
	ThreadID      string `json:"threadId"`
	Title         string `json:"title"`
	Status        string `json:"status"`
	Kind          string `json:"kind,omitempty"`
	InputSummary  string `json:"inputSummary,omitempty"`
	ResultSummary string `json:"resultSummary,omitempty"`
	CreatedAt     string `json:"createdAt"`
	UpdatedAt     string `json:"updatedAt,omitempty"`
}

// MessageDescriptor describes a single thread-local message.
type MessageDescriptor struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
}

// ToolCallDescriptor describes a single thread-local tool call summary.
type ToolCallDescriptor struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	ToolID    string `json:"toolId"`
	Status    string `json:"status"`
	Summary   string `json:"summary"`
	CreatedAt string `json:"createdAt"`
}

// ArtifactDescriptor describes a single thread-local artifact summary.
type ArtifactDescriptor struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Path      string `json:"path"`
	Kind      string `json:"kind"`
	CreatedAt string `json:"createdAt"`
}

// RuntimeFlagDescriptor describes a single thread-local runtime flag.
type RuntimeFlagDescriptor struct {
	ThreadID  string `json:"threadId"`
	Key       string `json:"key"`
	Value     string `json:"value"`
	UpdatedAt string `json:"updatedAt"`
}

// EventDescriptor describes a single thread event for logs/activity views.
type EventDescriptor struct {
	ID        string `json:"id"`
	ThreadID  string `json:"threadId"`
	Type      string `json:"type"`
	Message   string `json:"message"`
	CreatedAt string `json:"createdAt"`
}

// StreamEventsRequest defines replay options for the event stream endpoint.
type StreamEventsRequest struct {
	Limit     int
	SinceID   string
	SinceTime string
}

// CreateTaskRequest defines the minimum request body for creating a thread task.
type CreateTaskRequest struct {
	Title string `json:"title"`
	Kind  string `json:"kind"`
	Input string `json:"input"`
}

// RunTaskRequest defines the minimum request body for running a task.
type RunTaskRequest struct{}

// CreateMessageRequest defines the minimum request body for appending a thread message.
type CreateMessageRequest struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CreateToolCallRequest defines the minimum request body for appending a thread tool call.
type CreateToolCallRequest struct {
	ToolID  string `json:"toolId"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

// CreateArtifactRequest defines the minimum request body for appending a thread artifact.
type CreateArtifactRequest struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
}

// SetRuntimeFlagRequest defines the minimum request body for upserting a thread runtime flag.
type SetRuntimeFlagRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	Kind        string `json:"kind,omitempty"`
	ReadOnly    bool   `json:"readOnly"`
	Executable  bool   `json:"executable"`
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
	RunTask(ctx context.Context, threadID string, taskID string, request RunTaskRequest) (TaskDescriptor, error)
	Messages(ctx context.Context, threadID string) ([]MessageDescriptor, error)
	AppendMessage(ctx context.Context, threadID string, request CreateMessageRequest) (MessageDescriptor, error)
	ToolCalls(ctx context.Context, threadID string) ([]ToolCallDescriptor, error)
	AppendToolCall(ctx context.Context, threadID string, request CreateToolCallRequest) (ToolCallDescriptor, error)
	Artifacts(ctx context.Context, threadID string) ([]ArtifactDescriptor, error)
	AppendArtifact(ctx context.Context, threadID string, request CreateArtifactRequest) (ArtifactDescriptor, error)
	RuntimeFlags(ctx context.Context, threadID string) ([]RuntimeFlagDescriptor, error)
	SetRuntimeFlag(ctx context.Context, threadID string, request SetRuntimeFlagRequest) (RuntimeFlagDescriptor, error)
	UpdateTaskStatus(ctx context.Context, threadID string, taskID string, request UpdateTaskStatusRequest) (TaskDescriptor, error)
	Events(ctx context.Context, threadID string) ([]EventDescriptor, error)
	StreamEvents(ctx context.Context, threadID string, request StreamEventsRequest) (<-chan EventDescriptor, error)
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

func (noopService) RunTask(context.Context, string, string, RunTaskRequest) (TaskDescriptor, error) {
	return TaskDescriptor{}, nil
}

func (noopService) Messages(context.Context, string) ([]MessageDescriptor, error) {
	return []MessageDescriptor{}, nil
}

func (noopService) AppendMessage(context.Context, string, CreateMessageRequest) (MessageDescriptor, error) {
	return MessageDescriptor{}, nil
}

func (noopService) ToolCalls(context.Context, string) ([]ToolCallDescriptor, error) {
	return []ToolCallDescriptor{}, nil
}

func (noopService) AppendToolCall(context.Context, string, CreateToolCallRequest) (ToolCallDescriptor, error) {
	return ToolCallDescriptor{}, nil
}

func (noopService) Artifacts(context.Context, string) ([]ArtifactDescriptor, error) {
	return []ArtifactDescriptor{}, nil
}

func (noopService) AppendArtifact(context.Context, string, CreateArtifactRequest) (ArtifactDescriptor, error) {
	return ArtifactDescriptor{}, nil
}

func (noopService) RuntimeFlags(context.Context, string) ([]RuntimeFlagDescriptor, error) {
	return []RuntimeFlagDescriptor{}, nil
}

func (noopService) SetRuntimeFlag(context.Context, string, SetRuntimeFlagRequest) (RuntimeFlagDescriptor, error) {
	return RuntimeFlagDescriptor{}, nil
}

func (noopService) UpdateTaskStatus(context.Context, string, string, UpdateTaskStatusRequest) (TaskDescriptor, error) {
	return TaskDescriptor{}, nil
}

func (noopService) Events(context.Context, string) ([]EventDescriptor, error) {
	return []EventDescriptor{}, nil
}

func (noopService) StreamEvents(context.Context, string, StreamEventsRequest) (<-chan EventDescriptor, error) {
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
