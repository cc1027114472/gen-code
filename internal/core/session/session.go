package session

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
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
	ActiveThreadCount int       `json:"active_thread_count"`
}

// Thread describes an isolated work session under a workspace.
type Thread struct {
	ID                  string              `json:"id"`
	WorkspaceID         string              `json:"workspace_id"`
	Name                string              `json:"name"`
	Status              string              `json:"status"`
	ActiveModel         string              `json:"active_model"`
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
	ID            string    `json:"id"`
	ThreadID      string    `json:"thread_id"`
	Title         string    `json:"title"`
	Status        string    `json:"status"`
	Kind          string    `json:"kind"`
	Input         string    `json:"input"`
	ResultSummary string    `json:"result_summary"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
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

// CreateThreadInput collects optional fields for creating a thread.
type CreateThreadInput struct {
	Name           string
	ActiveModel    string
	PermissionMode policy.Mode
}

// CreateTaskInput collects the minimum task fields for a thread-local task.
type CreateTaskInput struct {
	Title string
	Kind  string
	Input string
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
	Status        string
	ResultSummary string
}

var (
	// ErrThreadNotFound is returned when a thread lookup fails.
	ErrThreadNotFound = errors.New("thread not found")
	// ErrTaskNotFound is returned when a task lookup fails.
	ErrTaskNotFound = errors.New("task not found")
	// ErrInvalidTaskStatus is returned when a task status is unsupported.
	ErrInvalidTaskStatus = errors.New("invalid task status")
	// ErrTaskAlreadyRunning is returned when a thread already has a running task.
	ErrTaskAlreadyRunning = errors.New("thread already has a running task")
)

// Registry holds the in-memory workspace and thread set for the current runtime process.
type Registry struct {
	mu               sync.RWMutex
	store            *state.Store
	bus              *EventBus
	workspace        Workspace
	threads          map[string]Thread
	tasks            map[string][]Task
	events           map[string][]Event
	order            []string
	activeThreadID   string
	nextThreadNumber int
	nextTaskNumber   int
	nextEventNumber  int
	nextMessageNum   int
	nextToolCallNum  int
	nextArtifactNum  int
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
	workspaceID := filepath.Base(projectRoot)
	if workspaceID == "" || workspaceID == "." || workspaceID == string(filepath.Separator) {
		workspaceID = "workspace"
	}

	sharedDocsRoot := projectRoot
	if docsCandidate := filepath.Join(projectRoot, "docs"); docsCandidate != "" {
		sharedDocsRoot = docsCandidate
	}

	registry := &Registry{
		store: store,
		bus:   NewEventBus(64),
		workspace: Workspace{
			ID:             workspaceID,
			ProjectRoot:    projectRoot,
			SharedDocsRoot: sharedDocsRoot,
			CreatedAt:      time.Now().UTC(),
		},
		threads:          map[string]Thread{},
		tasks:            map[string][]Task{},
		events:           map[string][]Event{},
		order:            []string{},
		nextThreadNumber: 1,
		nextTaskNumber:   1,
		nextEventNumber:  1,
		nextMessageNum:   1,
		nextToolCallNum:  1,
		nextArtifactNum:  1,
	}

	if store != nil {
		if err := registry.restoreFromStore(); err != nil {
			return nil, err
		}
		if err := registry.persistWorkspaceLocked(); err != nil {
			return nil, err
		}
	}

	return registry, nil
}

// Workspace returns the current workspace descriptor.
func (r *Registry) Workspace() Workspace {
	r.mu.RLock()
	defer r.mu.RUnlock()

	workspace := r.workspace
	workspace.ActiveThreadCount = len(r.threads)
	return workspace
}

// ActiveThreadID returns the currently active thread identifier, if any.
func (r *Registry) ActiveThreadID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.activeThreadID
}

// Threads returns all known threads sorted by creation order.
func (r *Registry) Threads() []Thread {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Thread, 0, len(r.order))
	for _, id := range r.order {
		thread, ok := r.threads[id]
		if !ok {
			continue
		}
		items = append(items, snapshotThread(thread, r.activeThreadID))
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
	return snapshotThread(thread, r.activeThreadID), true
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
		WorkspaceID:    r.workspace.ID,
		Name:           name,
		Status:         "idle",
		ActiveModel:    input.ActiveModel,
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
	r.order = append(r.order, threadID)
	if r.activeThreadID == "" {
		r.activeThreadID = threadID
	}
	r.appendEventLocked(threadID, "thread.created", fmt.Sprintf("%s created", name))
	_ = r.persistThreadLocked(thread)
	_ = r.persistWorkspaceLocked()

	return snapshotThread(thread, r.activeThreadID)
}

// ActivateThread marks the given thread as the active thread for the runtime.
func (r *Registry) ActivateThread(id string) (Thread, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	thread, ok := r.threads[id]
	if !ok {
		return Thread{}, false
	}
	r.activeThreadID = id
	r.appendEventLocked(id, "thread.activated", fmt.Sprintf("%s activated", thread.Name))
	_ = r.persistWorkspaceLocked()
	return snapshotThread(thread, r.activeThreadID), true
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

	task := Task{
		ID:            fmt.Sprintf("task-%d", taskNumber),
		ThreadID:      threadID,
		Title:         title,
		Status:        "queued",
		Kind:          input.Kind,
		Input:         input.Input,
		ResultSummary: "",
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}

	r.tasks[threadID] = append(r.tasks[threadID], task)
	thread.TaskState = append(thread.TaskState, task.ID)
	r.threads[threadID] = thread
	r.appendEventLocked(threadID, "task.created", fmt.Sprintf("%s queued on %s", title, thread.Name))
	_ = r.persistTaskLocked(task)
	_ = r.persistThreadLocked(thread)
	return task, true
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
	case "queued", "running", "completed", "failed":
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

	if snapshot.Workspace.ID != "" {
		r.workspace.ID = snapshot.Workspace.ID
		r.workspace.ProjectRoot = snapshot.Workspace.ProjectRoot
		r.workspace.SharedDocsRoot = snapshot.Workspace.SharedDocsRoot
		r.workspace.CreatedAt = snapshot.Workspace.CreatedAt
		r.activeThreadID = snapshot.Workspace.ActiveThreadID
	}

	for _, item := range snapshot.Threads {
		thread := Thread{
			ID:             item.ID,
			WorkspaceID:    item.WorkspaceID,
			Name:           item.Name,
			Status:         item.Status,
			ActiveModel:    item.ActiveModel,
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
			ID:            item.ID,
			ThreadID:      item.ThreadID,
			Title:         item.Title,
			Status:        item.Status,
			Kind:          item.Kind,
			Input:         item.Input,
			ResultSummary: item.ResultSummary,
			CreatedAt:     item.CreatedAt,
			UpdatedAt:     item.UpdatedAt,
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

	return nil
}

func (r *Registry) persistWorkspaceLocked() error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveWorkspace(state.WorkspaceRecord{
		ID:             r.workspace.ID,
		ProjectRoot:    r.workspace.ProjectRoot,
		SharedDocsRoot: r.workspace.SharedDocsRoot,
		CreatedAt:      r.workspace.CreatedAt,
		ActiveThreadID: r.activeThreadID,
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
		PermissionMode: string(thread.PermissionMode),
		CreatedAt:      thread.CreatedAt,
	})
}

func (r *Registry) persistTaskLocked(task Task) error {
	if r.store == nil {
		return nil
	}
	return r.store.SaveTask(state.TaskRecord{
		ID:            task.ID,
		ThreadID:      task.ThreadID,
		Title:         task.Title,
		Status:        task.Status,
		Kind:          task.Kind,
		Input:         task.Input,
		ResultSummary: task.ResultSummary,
		CreatedAt:     task.CreatedAt,
		UpdatedAt:     task.UpdatedAt,
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
