package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/state"
)

// Workspace describes the shared runtime container for the currently opened project.
type Workspace struct {
	ID                string    `json:"id"`
	ProjectRoot       string    `json:"project_root"`
	SharedDocsRoot    string    `json:"shared_docs_root"`
	CreatedAt         time.Time `json:"created_at"`
	ActiveThreadID    string    `json:"active_thread_id"`
	ActiveThreadCount int       `json:"active_thread_count"`
	IsActive          bool      `json:"is_active"`
}

// Thread describes an isolated work session under a workspace.
type Thread struct {
	ID                  string              `json:"id"`
	WorkspaceID         string              `json:"workspace_id"`
	Name                string              `json:"name"`
	Status              string              `json:"status"`
	ActiveModel         string              `json:"active_model"`
	ReasoningEffort     string              `json:"reasoning_effort"`
	PermissionMode      policy.Mode         `json:"permission_mode"`
	MessageHistory      []MessageRecord     `json:"message_history"`
	ToolHistory         []ToolCallRecord    `json:"tool_history"`
	TaskState           []string            `json:"task_state"`
	ArtifactPaths       []ArtifactRecord    `json:"artifact_paths"`
	RuntimeFlags        []RuntimeFlagRecord `json:"runtime_flags"`
	CreatedAt           time.Time           `json:"created_at"`
	MessageHistoryCount int                 `json:"message_history_count"`
	ToolCallCount       int                 `json:"tool_call_count"`
	ArtifactCount       int                 `json:"artifact_count"`
	IsActive            bool                `json:"is_active"`
}

// Task describes a minimal task tracked under a thread.
type Task struct {
	ID             string    `json:"id"`
	ThreadID       string    `json:"thread_id"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
	Kind           string    `json:"kind"`
	Input          string    `json:"input"`
	ResultSummary  string    `json:"result_summary"`
	ApprovalStatus string    `json:"approval_status"`
	ParentTaskID   string    `json:"parent_task_id"`
	WaitingStatus  string    `json:"waiting_status"`
	AgentState     string    `json:"agent_state"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// MessageRecord describes a persisted thread-local message.
type MessageRecord struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ToolCallRecord describes a persisted thread-local tool call summary.
type ToolCallRecord struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	ToolID    string    `json:"tool_id"`
	Status    string    `json:"status"`
	Summary   string    `json:"summary"`
	CreatedAt time.Time `json:"created_at"`
}

// ArtifactRecord describes a persisted thread-local artifact reference.
type ArtifactRecord struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	Path      string    `json:"path"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

// RuntimeFlagRecord describes a persisted thread-local runtime flag.
type RuntimeFlagRecord struct {
	ThreadID  string    `json:"thread_id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Event describes a timestamped thread event for logs and activity panels.
type Event struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	Type      string    `json:"type"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

// ApprovalRecord describes a persisted thread-local approval state.
type ApprovalRecord struct {
	ID          string    `json:"id"`
	ThreadID    string    `json:"thread_id"`
	TaskID      string    `json:"task_id"`
	ToolKind    string    `json:"tool_kind"`
	Status      string    `json:"status"`
	Summary     string    `json:"summary"`
	TargetPaths []string  `json:"target_paths"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// WriteExecutionRecord describes a persisted audit trail for a thread-local write execution.
type WriteExecutionRecord struct {
	ID                    string                       `json:"id"`
	ThreadID              string                       `json:"thread_id"`
	TaskID                string                       `json:"task_id"`
	ApprovalID            string                       `json:"approval_id"`
	ToolKind              string                       `json:"tool_kind"`
	Operation             string                       `json:"operation"`
	RelatedExecutionID    string                       `json:"related_execution_id"`
	Status                string                       `json:"status"`
	TargetPaths           []string                     `json:"target_paths"`
	PatchHash             string                       `json:"patch_hash"`
	PatchSummary          string                       `json:"patch_summary"`
	BeforeSnapshotSummary string                       `json:"before_snapshot_summary"`
	AfterSnapshotSummary  string                       `json:"after_snapshot_summary"`
	RollbackPayload       []WriteExecutionFileSnapshot `json:"rollback_payload"`
	ResultSummary         string                       `json:"result_summary"`
	CreatedAt             time.Time                    `json:"created_at"`
	UpdatedAt             time.Time                    `json:"updated_at"`
}

// WriteExecutionFileSnapshot stores the minimum file state required for a controlled rollback.
type WriteExecutionFileSnapshot struct {
	Path          string `json:"path"`
	BeforeExists  bool   `json:"beforeExists"`
	BeforeContent string `json:"beforeContent"`
	BeforeHash    string `json:"beforeHash"`
	AfterExists   bool   `json:"afterExists"`
	AfterHash     string `json:"afterHash"`
}

// CreateThreadInput collects optional fields for creating a thread.
type CreateThreadInput struct {
	Name           string
	ActiveModel    string
	ReasoningEffort string
	PermissionMode policy.Mode
}

// CreateWorkspaceInput collects optional fields for registering a workspace.
type CreateWorkspaceInput struct {
	ProjectRoot    string
	SharedDocsRoot string
}

// UpdateThreadPreferencesInput collects the minimum fields required to update thread preferences.
type UpdateThreadPreferencesInput struct {
	ActiveModel     *string
	ReasoningEffort *string
}

// CreateTaskInput collects the minimum task fields for a thread-local task.
type CreateTaskInput struct {
	Title          string
	Kind           string
	Input          string
	Status         string
	ResultSummary  string
	ApprovalStatus string
	ParentTaskID   string
	WaitingStatus  string
	AgentState     string
}

// AppendMessageInput collects the minimum message fields for a thread-local message.
type AppendMessageInput struct {
	Role    string
	Content string
}

// AppendToolCallInput collects the minimum fields for a thread-local tool call.
type AppendToolCallInput struct {
	ToolID  string
	Status  string
	Summary string
}

// AppendArtifactInput collects the minimum fields for a thread-local artifact.
type AppendArtifactInput struct {
	Path string
	Kind string
}

// SetRuntimeFlagInput collects the minimum fields for a thread-local runtime flag.
type SetRuntimeFlagInput struct {
	Key   string
	Value string
}

// UpdateTaskStatusInput collects the minimum fields required to update a task status.
type UpdateTaskStatusInput struct {
	Status         string
	ResultSummary  string
	ApprovalStatus *string
	WaitingStatus  *string
	AgentState     *string
}

// CreateApprovalInput collects the minimum fields for a thread-local approval.
type CreateApprovalInput struct {
	TaskID      string
	ToolKind    string
	Status      string
	Summary     string
	TargetPaths []string
}

// UpdateApprovalInput collects the minimum fields required to update an approval.
type UpdateApprovalInput struct {
	Status      string
	Summary     string
	TargetPaths []string
}

// CreateWriteExecutionInput collects the minimum fields required to append a write execution audit record.
type CreateWriteExecutionInput struct {
	TaskID                string
	ApprovalID            string
	ToolKind              string
	Operation             string
	RelatedExecutionID    string
	Status                string
	TargetPaths           []string
	PatchHash             string
	PatchSummary          string
	BeforeSnapshotSummary string
	AfterSnapshotSummary  string
	RollbackPayload       []WriteExecutionFileSnapshot
	ResultSummary         string
}

var (
	// ErrWorkspaceNotFound is returned when a workspace lookup fails.
	ErrWorkspaceNotFound = errors.New("workspace not found")
	// ErrThreadNotFound is returned when a thread lookup fails.
	ErrThreadNotFound = errors.New("thread not found")
	// ErrTaskNotFound is returned when a task lookup fails.
	ErrTaskNotFound = errors.New("task not found")
	// ErrInvalidTaskStatus is returned when a task status is unsupported.
	ErrInvalidTaskStatus = errors.New("invalid task status")
	// ErrTaskAlreadyRunning is returned when a thread already has a running task.
	ErrTaskAlreadyRunning = errors.New("thread already has a running task")
	// ErrApprovalNotFound is returned when a task approval lookup fails.
	ErrApprovalNotFound = errors.New("approval not found")
)

func defaultWorkspace(projectRoot string) Workspace {
	workspaceID := filepath.Base(projectRoot)
	if workspaceID == "" || workspaceID == "." || workspaceID == string(filepath.Separator) {
		workspaceID = "workspace"
	}

	sharedDocsRoot := projectRoot
	if docsCandidate := filepath.Join(projectRoot, "docs"); docsCandidate != "" {
		sharedDocsRoot = docsCandidate
	}

	return Workspace{
		ID:             workspaceID,
		ProjectRoot:    projectRoot,
		SharedDocsRoot: sharedDocsRoot,
		CreatedAt:      time.Now().UTC(),
		IsActive:       true,
	}
}

// Registry holds the in-memory workspace and thread set for the current runtime process.
type Registry struct {
	mu               sync.RWMutex
	store            *state.Store
	bus              *EventBus
	workspaces       map[string]Workspace
	workspaceOrder   []string
	activeWorkspaceID string
	threads          map[string]Thread
	tasks            map[string][]Task
	events           map[string][]Event
	approvals        map[string][]ApprovalRecord
	writeExecutions  map[string][]WriteExecutionRecord
	order            []string
	nextThreadNumber int
	nextTaskNumber   int
	nextEventNumber  int
	nextMessageNum   int
	nextToolCallNum  int
	nextArtifactNum  int
	nextApprovalNum  int
	nextWriteExecNum int
}

// NewRegistry creates the default single-workspace registry for the given project root.
func NewRegistry(projectRoot string) *Registry {
	registry, err := NewRegistryWithStore(projectRoot, nil)
	if err != nil {
		panic(err)
	}
	return registry
}

// NewRegistryWithStore creates a registry and optionally hydrates it from a state store.
func NewRegistryWithStore(projectRoot string, store *state.Store) (*Registry, error) {
	workspace := defaultWorkspace(projectRoot)

	registry := &Registry{
		store: store,
		bus:   NewEventBus(64),
		workspaces: map[string]Workspace{
			workspace.ID: workspace,
		},
		workspaceOrder:    []string{workspace.ID},
		activeWorkspaceID: workspace.ID,
		threads:          map[string]Thread{},
		tasks:            map[string][]Task{},
		events:           map[string][]Event{},
		approvals:        map[string][]ApprovalRecord{},
		writeExecutions:  map[string][]WriteExecutionRecord{},
		order:            []string{},
		nextThreadNumber: 1,
		nextTaskNumber:   1,
		nextEventNumber:  1,
		nextMessageNum:   1,
		nextToolCallNum:  1,
		nextArtifactNum:  1,
		nextApprovalNum:  1,
		nextWriteExecNum: 1,
	}

	if store != nil {
		if err := registry.restoreFromStore(); err != nil {
			return nil, err
		}
		if err := registry.persistWorkspacesLocked(); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

// Workspace returns the current workspace descriptor.
func (r *Registry) Workspace() Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.snapshotWorkspaceLocked(r.activeWorkspaceID)
}

// ActiveThreadID returns the currently active thread identifier, if any.
func (r *Registry) ActiveThreadID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	workspace, ok := r.workspaces[r.activeWorkspaceID]
	if !ok {
		return ""
	}
	return workspace.ActiveThreadID
}

// Threads returns all known threads sorted by creation order.
func (r *Registry) Threads() []Thread {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Thread, 0, len(r.order))
	activeThreadID := r.activeThreadIDLocked()
	activeWorkspaceID := r.activeWorkspaceID
	for _, id := range r.order {
		thread, ok := r.threads[id]
		if !ok {
			continue
		}
		if thread.WorkspaceID != activeWorkspaceID {
			continue
		}
		items = append(items, snapshotThread(thread, activeThreadID))
	}
	return items
}

// Thread returns the thread with the given id.
func (r *Registry) Thread(id string) (Thread, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	thread, ok := r.threads[id]
	if !ok {
		return Thread{}, false
	}
	return snapshotThread(thread, r.activeThreadIDLocked()), true
}

// Workspaces returns all registered workspaces sorted by creation order.
func (r *Registry) Workspaces() []Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Workspace, 0, len(r.workspaceOrder))
	for _, id := range r.workspaceOrder {
		items = append(items, r.snapshotWorkspaceLocked(id))
	}
	return items
}

// ActiveWorkspaceID returns the current active workspace identifier, if any.
func (r *Registry) ActiveWorkspaceID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.activeWorkspaceID
}

// CreateWorkspace registers a new workspace root and makes it active when it is the first workspace.
func (r *Registry) CreateWorkspace(input CreateWorkspaceInput) (Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	projectRoot := strings.TrimSpace(input.ProjectRoot)
	if projectRoot == "" {
		return Workspace{}, fmt.Errorf("project root is required")
	}
	workspace := defaultWorkspace(projectRoot)
	if strings.TrimSpace(input.SharedDocsRoot) != "" {
		workspace.SharedDocsRoot = strings.TrimSpace(input.SharedDocsRoot)
	}
	if existing, ok := r.workspaces[workspace.ID]; ok {
		existing.ProjectRoot = workspace.ProjectRoot
		existing.SharedDocsRoot = workspace.SharedDocsRoot
		r.workspaces[workspace.ID] = existing
		_ = r.persistWorkspaceLocked(workspace.ID)
		return r.snapshotWorkspaceLocked(workspace.ID), nil
	}
	if len(r.workspaces) > 0 {
		workspace.IsActive = false
	}
	r.workspaces[workspace.ID] = workspace
	r.workspaceOrder = append(r.workspaceOrder, workspace.ID)
	if strings.TrimSpace(r.activeWorkspaceID) == "" {
		r.activeWorkspaceID = workspace.ID
		r.setActiveWorkspaceLocked(workspace.ID)
	}
	_ = r.persistWorkspaceLocked(workspace.ID)
	return r.snapshotWorkspaceLocked(workspace.ID), nil
}

// ActivateWorkspace marks the workspace with the given id as active.
func (r *Registry) ActivateWorkspace(id string) (Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.workspaces[id]; !ok {
		return Workspace{}, ErrWorkspaceNotFound
	}
	r.activeWorkspaceID = id
	r.setActiveWorkspaceLocked(id)
	_ = r.persistWorkspacesLocked()
	return r.snapshotWorkspaceLocked(id), nil
}

// UpdateThreadPreferences updates the persisted model and reasoning selection for a thread.
func (r *Registry) UpdateThreadPreferences(id string, input UpdateThreadPreferencesInput) (Thread, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[id]
	if !ok {
		return Thread{}, ErrThreadNotFound
	}
	if input.ActiveModel != nil {
		thread.ActiveModel = strings.TrimSpace(*input.ActiveModel)
	}
	if input.ReasoningEffort != nil {
		thread.ReasoningEffort = strings.TrimSpace(*input.ReasoningEffort)
	}
	r.threads[id] = thread
	_ = r.persistThreadLocked(thread)
	return snapshotThread(thread, r.activeThreadIDLocked()), nil
}

// CreateThread creates a new isolated thread under the current workspace.
func (r *Registry) CreateThread(input CreateThreadInput) Thread {
	r.mu.Lock()
	defer r.mu.Unlock()

	threadNumber := r.nextThreadNumber
	r.nextThreadNumber++

	threadID := fmt.Sprintf("thread-%d", threadNumber)
	name := input.Name
	if name == "" {
		name = fmt.Sprintf("Thread %d", threadNumber)
	}

	mode := input.PermissionMode
	if mode == "" {
		mode = policy.DefaultMode()
	}

	thread := Thread{
		ID:             threadID,
		WorkspaceID:    r.activeWorkspaceID,
		Name:           name,
		Status:         "idle",
		ActiveModel:    input.ActiveModel,
		ReasoningEffort: input.ReasoningEffort,
		PermissionMode: mode,
		MessageHistory: []MessageRecord{},
		ToolHistory:    []ToolCallRecord{},
		TaskState:      []string{},
		ArtifactPaths:  []ArtifactRecord{},
		RuntimeFlags:   []RuntimeFlagRecord{},
		CreatedAt:      time.Now().UTC(),
	}

	r.threads[threadID] = thread
	r.tasks[threadID] = []Task{}
	r.events[threadID] = []Event{}
	r.approvals[threadID] = []ApprovalRecord{}
	r.writeExecutions[threadID] = []WriteExecutionRecord{}
	r.order = append(r.order, threadID)
	workspace := r.workspaces[r.activeWorkspaceID]
	if strings.TrimSpace(workspace.ActiveThreadID) == "" {
		workspace.ActiveThreadID = threadID
		r.workspaces[r.activeWorkspaceID] = workspace
	}
	r.appendEventLocked(threadID, "thread.created", fmt.Sprintf("%s created", name))
	_ = r.persistThreadLocked(thread)
	_ = r.persistWorkspaceLocked(r.activeWorkspaceID)

	return snapshotThread(thread, r.activeThreadIDLocked())
}

// ActivateThread marks the given thread as the active thread for the runtime.
func (r *Registry) ActivateThread(id string) (Thread, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[id]
	if !ok {
		return Thread{}, false
	}
	r.activeWorkspaceID = thread.WorkspaceID
	r.setActiveWorkspaceLocked(thread.WorkspaceID)
	workspace := r.workspaces[thread.WorkspaceID]
	workspace.ActiveThreadID = id
	r.workspaces[thread.WorkspaceID] = workspace
	r.appendEventLocked(id, "thread.activated", fmt.Sprintf("%s activated", thread.Name))
	_ = r.persistWorkspaceLocked(thread.WorkspaceID)
	return snapshotThread(thread, r.activeThreadIDLocked()), true
}

// Tasks returns the tasks registered under the given thread.
func (r *Registry) Tasks(threadID string) ([]Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.threads[threadID]; !ok {
		return nil, false
	}
	items := append([]Task(nil), r.tasks[threadID]...)
	return items, true
}

// Events returns the recorded events under the given thread.
func (r *Registry) Events(threadID string) ([]Event, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.threads[threadID]; !ok {
		return nil, false
	}
	items := append([]Event(nil), r.events[threadID]...)
	return items, true
}

// Approvals returns the approval records under the given thread.
func (r *Registry) Approvals(threadID string) ([]ApprovalRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.threads[threadID]; !ok {
		return nil, false
	}
	items := append([]ApprovalRecord(nil), r.approvals[threadID]...)
	for index := range items {
		items[index].TargetPaths = append([]string(nil), items[index].TargetPaths...)
	}
	return items, true
}

// WriteExecutions returns the persisted write execution audit records under the given thread.
func (r *Registry) WriteExecutions(threadID string) ([]WriteExecutionRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.threads[threadID]; !ok {
		return nil, false
	}
	items := append([]WriteExecutionRecord(nil), r.writeExecutions[threadID]...)
	for index := range items {
		items[index].TargetPaths = append([]string(nil), items[index].TargetPaths...)
		items[index].RollbackPayload = append([]WriteExecutionFileSnapshot(nil), items[index].RollbackPayload...)
	}
	return items, true
}

// ApprovalByTask returns the approval associated with a task.
func (r *Registry) ApprovalByTask(threadID string, taskID string) (ApprovalRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.threads[threadID]; !ok {
		return ApprovalRecord{}, ErrThreadNotFound
	}
	for _, item := range r.approvals[threadID] {
		if item.TaskID == taskID {
			item.TargetPaths = append([]string(nil), item.TargetPaths...)
			return item, nil
		}
	}
	return ApprovalRecord{}, ErrApprovalNotFound
}

// SubscribeEvents subscribes to real-time events for the given thread.
func (r *Registry) SubscribeEvents(threadID string) (<-chan Event, func(), error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.threads[threadID]; !ok {
		return nil, nil, ErrThreadNotFound
	}
	ch, cancel := r.bus.Subscribe(threadID)
	return ch, cancel, nil
}

// AppendRuntimeEvent appends a runtime-scoped event and broadcasts it in real time.
func (r *Registry) AppendRuntimeEvent(threadID string, eventType string, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.threads[threadID]; !ok {
		return ErrThreadNotFound
	}
	r.appendEventLocked(threadID, eventType, message)
	return nil
}

// Messages returns the recorded thread-local messages.
func (r *Registry) Messages(threadID string) ([]MessageRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return nil, false
	}
	items := append([]MessageRecord(nil), thread.MessageHistory...)
	return items, true
}

// ToolCalls returns the recorded thread-local tool calls.
func (r *Registry) ToolCalls(threadID string) ([]ToolCallRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return nil, false
	}
	items := append([]ToolCallRecord(nil), thread.ToolHistory...)
	return items, true
}

// Artifacts returns the recorded thread-local artifacts.
func (r *Registry) Artifacts(threadID string) ([]ArtifactRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return nil, false
	}
	items := append([]ArtifactRecord(nil), thread.ArtifactPaths...)
	return items, true
}

// RuntimeFlags returns the recorded thread-local runtime flags.
func (r *Registry) RuntimeFlags(threadID string) ([]RuntimeFlagRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return nil, false
	}
	items := append([]RuntimeFlagRecord(nil), thread.RuntimeFlags...)
	return items, true
}

// CreateTask registers a new queued task under the given thread.
func (r *Registry) CreateTask(threadID string, input CreateTaskInput) (Task, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return Task{}, false
	}

	taskNumber := r.nextTaskNumber
	r.nextTaskNumber++

	title := input.Title
	if title == "" {
		title = fmt.Sprintf("Task %d", taskNumber)
	}

	status := input.Status
	if status == "" {
		status = "queued"
	}
	task := Task{
		ID:             fmt.Sprintf("task-%d", taskNumber),
		ThreadID:       threadID,
		Title:          title,
		Status:         status,
		Kind:           input.Kind,
		Input:          input.Input,
		ResultSummary:  input.ResultSummary,
		ApprovalStatus: input.ApprovalStatus,
		ParentTaskID:   input.ParentTaskID,
		WaitingStatus:  input.WaitingStatus,
		AgentState:     input.AgentState,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	r.tasks[threadID] = append(r.tasks[threadID], task)
	thread.TaskState = append(thread.TaskState, task.ID)
	r.threads[threadID] = thread
	r.appendEventLocked(threadID, "task.created", fmt.Sprintf("%s queued on %s", title, thread.Name))
	_ = r.persistTaskLocked(task)
	_ = r.persistThreadLocked(thread)
	return task, true
}

// CreateApproval creates or replaces a thread-local approval for a task.
func (r *Registry) CreateApproval(threadID string, input CreateApprovalInput) (ApprovalRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return ApprovalRecord{}, ErrThreadNotFound
	}
	if _, err := r.taskLocked(threadID, input.TaskID); err != nil {
		return ApprovalRecord{}, err
	}

	now := time.Now().UTC()
	record := ApprovalRecord{
		ID:          fmt.Sprintf("approval-%d", r.nextApprovalNum),
		ThreadID:    threadID,
		TaskID:      input.TaskID,
		ToolKind:    input.ToolKind,
		Status:      input.Status,
		Summary:     input.Summary,
		TargetPaths: append([]string(nil), input.TargetPaths...),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if record.Status == "" {
		record.Status = "pending"
	}

	items := r.approvals[threadID]
	for index, item := range items {
		if item.TaskID != input.TaskID {
			continue
		}
		record.ID = item.ID
		record.CreatedAt = item.CreatedAt
		record.UpdatedAt = now
		items[index] = record
		r.approvals[threadID] = items
		_ = r.persistApprovalLocked(record)
		_ = r.persistThreadLocked(thread)
		return record, nil
	}

	r.nextApprovalNum++
	r.approvals[threadID] = append(r.approvals[threadID], record)
	_ = r.persistApprovalLocked(record)
	_ = r.persistThreadLocked(thread)
	return record, nil
}

// UpdateApproval updates an approval associated with a task.
func (r *Registry) UpdateApproval(threadID string, taskID string, input UpdateApprovalInput) (ApprovalRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return ApprovalRecord{}, ErrThreadNotFound
	}

	items := r.approvals[threadID]
	for index, item := range items {
		if item.TaskID != taskID {
			continue
		}
		if input.Status != "" {
			item.Status = input.Status
		}
		if input.Summary != "" {
			item.Summary = input.Summary
		}
		if input.TargetPaths != nil {
			item.TargetPaths = append([]string(nil), input.TargetPaths...)
		}
		item.UpdatedAt = time.Now().UTC()
		items[index] = item
		r.approvals[threadID] = items
		_ = r.persistApprovalLocked(item)
		_ = r.persistThreadLocked(thread)
		item.TargetPaths = append([]string(nil), item.TargetPaths...)
		return item, nil
	}

	return ApprovalRecord{}, ErrApprovalNotFound
}

// CreateWriteExecution appends a write execution audit record under the given thread.
func (r *Registry) CreateWriteExecution(threadID string, input CreateWriteExecutionInput) (WriteExecutionRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return WriteExecutionRecord{}, ErrThreadNotFound
	}
	if _, err := r.taskLocked(threadID, input.TaskID); err != nil {
		return WriteExecutionRecord{}, err
	}

	now := time.Now().UTC()
	record := WriteExecutionRecord{
		ID:                    fmt.Sprintf("writeexec-%d", r.nextWriteExecNum),
		ThreadID:              threadID,
		TaskID:                input.TaskID,
		ApprovalID:            input.ApprovalID,
		ToolKind:              input.ToolKind,
		Operation:             input.Operation,
		RelatedExecutionID:    input.RelatedExecutionID,
		Status:                input.Status,
		TargetPaths:           append([]string(nil), input.TargetPaths...),
		PatchHash:             input.PatchHash,
		PatchSummary:          input.PatchSummary,
		BeforeSnapshotSummary: input.BeforeSnapshotSummary,
		AfterSnapshotSummary:  input.AfterSnapshotSummary,
		RollbackPayload:       append([]WriteExecutionFileSnapshot(nil), input.RollbackPayload...),
		ResultSummary:         input.ResultSummary,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if record.Operation == "" {
		record.Operation = "apply"
	}
	if record.Status == "" {
		record.Status = "completed"
	}

	r.nextWriteExecNum++
	r.writeExecutions[threadID] = append(r.writeExecutions[threadID], record)
	_ = r.persistWriteExecutionLocked(record)
	_ = r.persistThreadLocked(thread)
	return record, nil
}

// AppendMessage appends a thread-local message and persists it.
func (r *Registry) AppendMessage(threadID string, input AppendMessageInput) (MessageRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return MessageRecord{}, ErrThreadNotFound
	}

	record := MessageRecord{
		ID:        fmt.Sprintf("message-%d", r.nextMessageNum),
		ThreadID:  threadID,
		Role:      input.Role,
		Content:   input.Content,
		CreatedAt: time.Now().UTC(),
	}
	r.nextMessageNum++
	thread.MessageHistory = append(thread.MessageHistory, record)
	r.threads[threadID] = thread
	r.appendEventLocked(threadID, "message.appended", fmt.Sprintf("%s message appended on %s", record.Role, thread.Name))
	_ = r.persistMessageLocked(record)
	_ = r.persistThreadLocked(thread)
	return record, nil
}

// AppendToolCall appends a thread-local tool call and persists it.
func (r *Registry) AppendToolCall(threadID string, input AppendToolCallInput) (ToolCallRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return ToolCallRecord{}, ErrThreadNotFound
	}

	record := ToolCallRecord{
		ID:        fmt.Sprintf("toolcall-%d", r.nextToolCallNum),
		ThreadID:  threadID,
		ToolID:    input.ToolID,
		Status:    input.Status,
		Summary:   input.Summary,
		CreatedAt: time.Now().UTC(),
	}
	r.nextToolCallNum++
	thread.ToolHistory = append(thread.ToolHistory, record)
	r.threads[threadID] = thread
	r.appendEventLocked(threadID, "toolcall.appended", fmt.Sprintf("%s tool call recorded on %s", record.ToolID, thread.Name))
	_ = r.persistToolCallLocked(record)
	_ = r.persistThreadLocked(thread)
	return record, nil
}

// AppendArtifact appends a thread-local artifact and persists it.
func (r *Registry) AppendArtifact(threadID string, input AppendArtifactInput) (ArtifactRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return ArtifactRecord{}, ErrThreadNotFound
	}

	record := ArtifactRecord{
		ID:        fmt.Sprintf("artifact-%d", r.nextArtifactNum),
		ThreadID:  threadID,
		Path:      input.Path,
		Kind:      input.Kind,
		CreatedAt: time.Now().UTC(),
	}
	r.nextArtifactNum++
	thread.ArtifactPaths = append(thread.ArtifactPaths, record)
	r.threads[threadID] = thread
	r.appendEventLocked(threadID, "artifact.appended", fmt.Sprintf("%s artifact recorded on %s", record.Kind, thread.Name))
	_ = r.persistArtifactLocked(record)
	_ = r.persistThreadLocked(thread)
	return record, nil
}

// SetRuntimeFlag upserts a thread-local runtime flag and persists it.
func (r *Registry) SetRuntimeFlag(threadID string, input SetRuntimeFlagInput) (RuntimeFlagRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return RuntimeFlagRecord{}, ErrThreadNotFound
	}

	record := RuntimeFlagRecord{
		ThreadID:  threadID,
		Key:       input.Key,
		Value:     input.Value,
		UpdatedAt: time.Now().UTC(),
	}
	replaced := false
	for index, item := range thread.RuntimeFlags {
		if item.Key != input.Key {
			continue
		}
		thread.RuntimeFlags[index] = record
		replaced = true
		break
	}
	if !replaced {
		thread.RuntimeFlags = append(thread.RuntimeFlags, record)
	}
	r.threads[threadID] = thread
	r.appendEventLocked(threadID, "runtimeflag.updated", fmt.Sprintf("%s flag updated on %s", record.Key, thread.Name))
	_ = r.persistRuntimeFlagLocked(record)
	_ = r.persistThreadLocked(thread)
	return record, nil
}

// UpdateTaskStatus updates the lifecycle state for a task under the given thread.
func (r *Registry) UpdateTaskStatus(threadID string, taskID string, input UpdateTaskStatusInput) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[threadID]
	if !ok {
		return Task{}, ErrThreadNotFound
	}

	if !isSupportedTaskStatus(input.Status) {
		return Task{}, ErrInvalidTaskStatus
	}

	tasks := r.tasks[threadID]
	if input.Status == "running" {
		for _, item := range tasks {
			if item.ID != taskID && item.Status == "running" {
				return Task{}, ErrTaskAlreadyRunning
			}
		}
	}
	for index, task := range tasks {
		if task.ID != taskID {
			continue
		}

		task.Status = input.Status
		task.ResultSummary = input.ResultSummary
		if input.ApprovalStatus != nil {
			task.ApprovalStatus = *input.ApprovalStatus
		}
		if input.WaitingStatus != nil {
			task.WaitingStatus = *input.WaitingStatus
		}
		if input.AgentState != nil {
			task.AgentState = *input.AgentState
		}
		task.UpdatedAt = time.Now().UTC()
		tasks[index] = task
		r.tasks[threadID] = tasks
		r.threads[threadID] = thread
		r.appendEventLocked(threadID, "task.updated", fmt.Sprintf("%s moved to %s on %s", task.Title, task.Status, thread.Name))
		_ = r.persistTaskLocked(task)
		return task, nil
	}

	return Task{}, ErrTaskNotFound
}

func snapshotThread(thread Thread, activeThreadID string) Thread {
	thread.MessageHistoryCount = len(thread.MessageHistory)
	thread.ToolCallCount = len(thread.ToolHistory)
	thread.ArtifactCount = len(thread.ArtifactPaths)
	thread.IsActive = thread.ID == activeThreadID
	thread.MessageHistory = append([]MessageRecord(nil), thread.MessageHistory...)
	thread.ToolHistory = append([]ToolCallRecord(nil), thread.ToolHistory...)
	thread.TaskState = append([]string(nil), thread.TaskState...)
	thread.ArtifactPaths = append([]ArtifactRecord(nil), thread.ArtifactPaths...)
	thread.RuntimeFlags = append([]RuntimeFlagRecord(nil), thread.RuntimeFlags...)
	return thread
}

func (r *Registry) snapshotWorkspaceLocked(id string) Workspace {
	workspace, ok := r.workspaces[id]
	if !ok {
		return Workspace{}
	}
	count := 0
	for _, thread := range r.threads {
		if thread.WorkspaceID == id {
			count++
		}
	}
	workspace.ActiveThreadCount = count
	workspace.IsActive = id == r.activeWorkspaceID
	return workspace
}

func (r *Registry) activeThreadIDLocked() string {
	workspace, ok := r.workspaces[r.activeWorkspaceID]
	if !ok {
		return ""
	}
	return workspace.ActiveThreadID
}

func (r *Registry) setActiveWorkspaceLocked(id string) {
	for workspaceID, workspace := range r.workspaces {
		workspace.IsActive = workspaceID == id
		r.workspaces[workspaceID] = workspace
	}
}

func (r *Registry) appendEventLocked(threadID string, eventType string, message string) {
	event := Event{
		ID:        fmt.Sprintf("event-%d", r.nextEventNumber),
		ThreadID:  threadID,
		Type:      eventType,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}
	r.nextEventNumber++
	r.events[threadID] = append(r.events[threadID], event)
	_ = r.persistEventLocked(event)
	if dropped := r.bus.Publish(threadID, event); dropped && eventType != "event.dropped" {
		dropEvent := Event{
			ID:        fmt.Sprintf("event-%d", r.nextEventNumber),
			ThreadID:  threadID,
			Type:      "event.dropped",
			Message:   "event bus dropped the oldest buffered event",
			CreatedAt: time.Now().UTC(),
		}
		r.nextEventNumber++
		r.events[threadID] = append(r.events[threadID], dropEvent)
		_ = r.persistEventLocked(dropEvent)
		_ = r.bus.Publish(threadID, dropEvent)
	}
}

func isSupportedTaskStatus(status string) bool {
	switch status {
	case "queued", "running", "completed", "failed", "needs_approval", "waiting_for_approval", "waiting_for_task":
		return true
	default:
		return false
	}
}

// Task returns the task with the given id under the given thread.
func (r *Registry) Task(threadID string, taskID string) (Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if _, ok := r.threads[threadID]; !ok {
		return Task{}, ErrThreadNotFound
	}
	for _, item := range r.tasks[threadID] {
		if item.ID == taskID {
			return item, nil
		}
	}
	return Task{}, ErrTaskNotFound
}

func (r *Registry) taskLocked(threadID string, taskID string) (Task, error) {
	if _, ok := r.threads[threadID]; !ok {
		return Task{}, ErrThreadNotFound
	}
	for _, item := range r.tasks[threadID] {
		if item.ID == taskID {
			return item, nil
		}
	}
	return Task{}, ErrTaskNotFound
}

// SortedIDs returns the thread ids in stable creation order.
func (r *Registry) SortedIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := append([]string(nil), r.order...)
	sort.Strings(ids)
	return ids
}

// StateStoreName returns the backing persistence engine name.
func (r *Registry) StateStoreName() string {
	if r.store == nil {
		return ""
	}
	return state.StoreName
}

// StatePath returns the resolved persistence file path.
func (r *Registry) StatePath() string {
	if r.store == nil {
		return ""
	}
	return r.store.Path()
}

func (r *Registry) restoreFromStore() error {
	snapshot, err := r.store.Load()
	if err != nil {
		return err
	}

	if len(snapshot.Workspaces) > 0 {
		r.workspaces = map[string]Workspace{}
		r.workspaceOrder = r.workspaceOrder[:0]
		for _, item := range snapshot.Workspaces {
			r.workspaces[item.ID] = Workspace{
				ID:             item.ID,
				ProjectRoot:    item.ProjectRoot,
				SharedDocsRoot: item.SharedDocsRoot,
				CreatedAt:      item.CreatedAt,
				ActiveThreadID: item.ActiveThreadID,
				IsActive:       item.IsActive,
			}
			r.workspaceOrder = append(r.workspaceOrder, item.ID)
			if item.IsActive {
				r.activeWorkspaceID = item.ID
			}
		}
		if strings.TrimSpace(r.activeWorkspaceID) == "" {
			r.activeWorkspaceID = snapshot.ActiveWorkspaceID
		}
		if strings.TrimSpace(r.activeWorkspaceID) == "" && len(r.workspaceOrder) > 0 {
			r.activeWorkspaceID = r.workspaceOrder[0]
			r.setActiveWorkspaceLocked(r.activeWorkspaceID)
		}
	} else if snapshot.Workspace.ID != "" {
		r.workspaces = map[string]Workspace{
			snapshot.Workspace.ID: {
				ID:             snapshot.Workspace.ID,
				ProjectRoot:    snapshot.Workspace.ProjectRoot,
				SharedDocsRoot: snapshot.Workspace.SharedDocsRoot,
				CreatedAt:      snapshot.Workspace.CreatedAt,
				ActiveThreadID: snapshot.Workspace.ActiveThreadID,
				IsActive:       true,
			},
		}
		r.workspaceOrder = []string{snapshot.Workspace.ID}
		r.activeWorkspaceID = snapshot.Workspace.ID
	}

	for _, item := range snapshot.Threads {
		thread := Thread{
			ID:             item.ID,
			WorkspaceID:    item.WorkspaceID,
			Name:           item.Name,
			Status:         item.Status,
			ActiveModel:    item.ActiveModel,
			ReasoningEffort: item.ReasoningEffort,
			PermissionMode: policy.Mode(item.PermissionMode),
			MessageHistory: []MessageRecord{},
			ToolHistory:    []ToolCallRecord{},
			TaskState:      []string{},
			ArtifactPaths:  []ArtifactRecord{},
			RuntimeFlags:   []RuntimeFlagRecord{},
			CreatedAt:      item.CreatedAt,
		}
		r.threads[item.ID] = thread
		r.order = append(r.order, item.ID)
	}

	for _, item := range snapshot.Tasks {
		task := Task{
			ID:             item.ID,
			ThreadID:       item.ThreadID,
			Title:          item.Title,
			Status:         item.Status,
			Kind:           item.Kind,
			Input:          item.Input,
			ResultSummary:  item.ResultSummary,
			ApprovalStatus: item.ApprovalStatus,
			ParentTaskID:   item.ParentTaskID,
			WaitingStatus:  item.WaitingStatus,
			AgentState:     item.AgentState,
			CreatedAt:      item.CreatedAt,
			UpdatedAt:      item.UpdatedAt,
		}
		r.tasks[item.ThreadID] = append(r.tasks[item.ThreadID], task)
		thread := r.threads[item.ThreadID]
		thread.TaskState = append(thread.TaskState, item.ID)
		r.threads[item.ThreadID] = thread
	}

	for _, item := range snapshot.Messages {
		thread := r.threads[item.ThreadID]
		thread.MessageHistory = append(thread.MessageHistory, MessageRecord{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Role:      item.Role,
			Content:   item.Content,
			CreatedAt: item.CreatedAt,
		})
		r.threads[item.ThreadID] = thread
	}

	for _, item := range snapshot.ToolCalls {
		thread := r.threads[item.ThreadID]
		thread.ToolHistory = append(thread.ToolHistory, ToolCallRecord{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			ToolID:    item.ToolID,
			Status:    item.Status,
			Summary:   item.Summary,
			CreatedAt: item.CreatedAt,
		})
		r.threads[item.ThreadID] = thread
	}

	for _, item := range snapshot.Artifacts {
		thread := r.threads[item.ThreadID]
		thread.ArtifactPaths = append(thread.ArtifactPaths, ArtifactRecord{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Path:      item.Path,
			Kind:      item.Kind,
			CreatedAt: item.CreatedAt,
		})
		r.threads[item.ThreadID] = thread
	}

	for _, item := range snapshot.Flags {
		thread := r.threads[item.ThreadID]
		thread.RuntimeFlags = append(thread.RuntimeFlags, RuntimeFlagRecord{
			ThreadID:  item.ThreadID,
			Key:       item.Key,
			Value:     item.Value,
			UpdatedAt: item.UpdatedAt,
		})
		r.threads[item.ThreadID] = thread
	}

	for _, item := range snapshot.Events {
		r.events[item.ThreadID] = append(r.events[item.ThreadID], Event{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Type:      item.Type,
			Message:   item.Message,
			CreatedAt: item.CreatedAt,
		})
	}

	for _, item := range snapshot.Approvals {
		r.approvals[item.ThreadID] = append(r.approvals[item.ThreadID], ApprovalRecord{
			ID:          item.ID,
			ThreadID:    item.ThreadID,
			TaskID:      item.TaskID,
			ToolKind:    item.ToolKind,
			Status:      item.Status,
			Summary:     item.Summary,
			TargetPaths: decodeTargetPaths(item.TargetPaths),
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	for _, item := range snapshot.WriteExecutions {
		r.writeExecutions[item.ThreadID] = append(r.writeExecutions[item.ThreadID], WriteExecutionRecord{
			ID:                    item.ID,
			ThreadID:              item.ThreadID,
			TaskID:                item.TaskID,
			ApprovalID:            item.ApprovalID,
			ToolKind:              item.ToolKind,
			Operation:             item.Operation,
			RelatedExecutionID:    item.RelatedExecutionID,
			Status:                item.Status,
			TargetPaths:           decodeTargetPaths(item.TargetPaths),
			PatchHash:             item.PatchHash,
			PatchSummary:          item.PatchSummary,
			BeforeSnapshotSummary: item.BeforeSnapshotSummary,
			AfterSnapshotSummary:  item.AfterSnapshotSummary,
			RollbackPayload:       decodeRollbackPayload(item.RollbackPayload),
			ResultSummary:         item.ResultSummary,
			CreatedAt:             item.CreatedAt,
			UpdatedAt:             item.UpdatedAt,
		})
	}

	r.nextThreadNumber = state.MaxSuffix(r.order, "thread-") + 1
	if r.nextThreadNumber == 1 && len(r.order) == 0 {
		r.nextThreadNumber = 1
	}

	taskIDs := make([]string, 0)
	for _, items := range r.tasks {
		for _, item := range items {
			taskIDs = append(taskIDs, item.ID)
		}
	}
	r.nextTaskNumber = state.MaxSuffix(taskIDs, "task-") + 1
	if r.nextTaskNumber == 1 && len(taskIDs) == 0 {
		r.nextTaskNumber = 1
	}

	eventIDs := make([]string, 0)
	for _, items := range r.events {
		for _, item := range items {
			eventIDs = append(eventIDs, item.ID)
		}
	}
	r.nextEventNumber = state.MaxSuffix(eventIDs, "event-") + 1
	if r.nextEventNumber == 1 && len(eventIDs) == 0 {
		r.nextEventNumber = 1
	}

	messageIDs := make([]string, 0)
	toolCallIDs := make([]string, 0)
	artifactIDs := make([]string, 0)
	for _, thread := range r.threads {
		for _, item := range thread.MessageHistory {
			messageIDs = append(messageIDs, item.ID)
		}
		for _, item := range thread.ToolHistory {
			toolCallIDs = append(toolCallIDs, item.ID)
		}
		for _, item := range thread.ArtifactPaths {
			artifactIDs = append(artifactIDs, item.ID)
		}
	}
	r.nextMessageNum = state.MaxSuffix(messageIDs, "message-") + 1
	if r.nextMessageNum == 1 && len(messageIDs) == 0 {
		r.nextMessageNum = 1
	}
	r.nextToolCallNum = state.MaxSuffix(toolCallIDs, "toolcall-") + 1
	if r.nextToolCallNum == 1 && len(toolCallIDs) == 0 {
		r.nextToolCallNum = 1
	}
	r.nextArtifactNum = state.MaxSuffix(artifactIDs, "artifact-") + 1
	if r.nextArtifactNum == 1 && len(artifactIDs) == 0 {
		r.nextArtifactNum = 1
	}

	approvalIDs := make([]string, 0)
	for _, items := range r.approvals {
		for _, item := range items {
			approvalIDs = append(approvalIDs, item.ID)
		}
	}
	r.nextApprovalNum = state.MaxSuffix(approvalIDs, "approval-") + 1
	if r.nextApprovalNum == 1 && len(approvalIDs) == 0 {
		r.nextApprovalNum = 1
	}

	writeExecIDs := make([]string, 0)
	for _, items := range r.writeExecutions {
		for _, item := range items {
			writeExecIDs = append(writeExecIDs, item.ID)
		}
	}
	r.nextWriteExecNum = state.MaxSuffix(writeExecIDs, "writeexec-") + 1
	if r.nextWriteExecNum == 1 && len(writeExecIDs) == 0 {
		r.nextWriteExecNum = 1
	}

	return nil
}

func (r *Registry) persistWorkspacesLocked() error {
	if r.store == nil {
		return nil
	}
	for _, id := range r.workspaceOrder {
		if err := r.persistWorkspaceLocked(id); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) persistWorkspaceLocked(id string) error {
	if r.store == nil {
		return nil
	}
	workspace, ok := r.workspaces[id]
	if !ok {
		return ErrWorkspaceNotFound
	}
	return r.store.SaveWorkspace(state.WorkspaceRecord{
		ID:             workspace.ID,
		ProjectRoot:    workspace.ProjectRoot,
		SharedDocsRoot: workspace.SharedDocsRoot,
		CreatedAt:      workspace.CreatedAt,
		ActiveThreadID: workspace.ActiveThreadID,
		IsActive:       workspace.ID == r.activeWorkspaceID,
	})
}

func (r *Registry) persistThreadLocked(thread Thread) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveThread(state.ThreadRecord{
		ID:             thread.ID,
		WorkspaceID:    thread.WorkspaceID,
		Name:           thread.Name,
		Status:         thread.Status,
		ActiveModel:    thread.ActiveModel,
		ReasoningEffort: thread.ReasoningEffort,
		PermissionMode: string(thread.PermissionMode),
		CreatedAt:      thread.CreatedAt,
	})
}

func (r *Registry) persistTaskLocked(task Task) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveTask(state.TaskRecord{
		ID:             task.ID,
		ThreadID:       task.ThreadID,
		Title:          task.Title,
		Status:         task.Status,
		Kind:           task.Kind,
		Input:          task.Input,
		ResultSummary:  task.ResultSummary,
		ApprovalStatus: task.ApprovalStatus,
		ParentTaskID:   task.ParentTaskID,
		WaitingStatus:  task.WaitingStatus,
		AgentState:     task.AgentState,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
	})
}

func (r *Registry) persistEventLocked(event Event) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveEvent(state.EventRecord{
		ID:        event.ID,
		ThreadID:  event.ThreadID,
		Type:      event.Type,
		Message:   event.Message,
		CreatedAt: event.CreatedAt,
	})
}

func (r *Registry) persistMessageLocked(item MessageRecord) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveMessage(state.MessageRecord{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		Role:      item.Role,
		Content:   item.Content,
		CreatedAt: item.CreatedAt,
	})
}

func (r *Registry) persistToolCallLocked(item ToolCallRecord) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveToolCall(state.ToolCallRecord{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		ToolID:    item.ToolID,
		Status:    item.Status,
		Summary:   item.Summary,
		CreatedAt: item.CreatedAt,
	})
}

func (r *Registry) persistArtifactLocked(item ArtifactRecord) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveArtifact(state.ArtifactRecord{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		Path:      item.Path,
		Kind:      item.Kind,
		CreatedAt: item.CreatedAt,
	})
}

func (r *Registry) persistRuntimeFlagLocked(item RuntimeFlagRecord) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveRuntimeFlag(state.RuntimeFlagRecord{
		ThreadID:  item.ThreadID,
		Key:       item.Key,
		Value:     item.Value,
		UpdatedAt: item.UpdatedAt,
	})
}

func (r *Registry) persistApprovalLocked(item ApprovalRecord) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveApproval(state.ApprovalRecord{
		ID:          item.ID,
		ThreadID:    item.ThreadID,
		TaskID:      item.TaskID,
		ToolKind:    item.ToolKind,
		Status:      item.Status,
		Summary:     item.Summary,
		TargetPaths: encodeTargetPaths(item.TargetPaths),
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	})
}

func (r *Registry) persistWriteExecutionLocked(item WriteExecutionRecord) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveWriteExecution(state.WriteExecutionRecord{
		ID:                    item.ID,
		ThreadID:              item.ThreadID,
		TaskID:                item.TaskID,
		ApprovalID:            item.ApprovalID,
		ToolKind:              item.ToolKind,
		Operation:             item.Operation,
		RelatedExecutionID:    item.RelatedExecutionID,
		Status:                item.Status,
		TargetPaths:           encodeTargetPaths(item.TargetPaths),
		PatchHash:             item.PatchHash,
		PatchSummary:          item.PatchSummary,
		BeforeSnapshotSummary: item.BeforeSnapshotSummary,
		AfterSnapshotSummary:  item.AfterSnapshotSummary,
		RollbackPayload:       encodeRollbackPayload(item.RollbackPayload),
		ResultSummary:         item.ResultSummary,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	})
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
	if raw == "" {
		return nil
	}
	var paths []string
	if err := json.Unmarshal([]byte(raw), &paths); err != nil {
		return nil
	}
	return paths
}

func encodeRollbackPayload(items []WriteExecutionFileSnapshot) string {
	if len(items) == 0 {
		return "[]"
	}
	encoded, err := json.Marshal(items)
	if err != nil {
		return "[]"
	}
	return string(encoded)
}

func decodeRollbackPayload(raw string) []WriteExecutionFileSnapshot {
	if raw == "" {
		return nil
	}
	var items []WriteExecutionFileSnapshot
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}
