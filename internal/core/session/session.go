package session

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"llmtrace/internal/core/policy"
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
	ID                  string      `json:"id"`
	WorkspaceID         string      `json:"workspace_id"`
	Name                string      `json:"name"`
	Status              string      `json:"status"`
	ActiveModel         string      `json:"active_model"`
	PermissionMode      policy.Mode `json:"permission_mode"`
	MessageHistory      []string    `json:"message_history"`
	ToolHistory         []string    `json:"tool_history"`
	TaskState           []string    `json:"task_state"`
	ArtifactPaths       []string    `json:"artifact_paths"`
	RuntimeFlags        []string    `json:"runtime_flags"`
	CreatedAt           time.Time   `json:"created_at"`
	MessageHistoryCount int         `json:"message_history_count"`
	ToolCallCount       int         `json:"tool_call_count"`
	ArtifactCount       int         `json:"artifact_count"`
	IsActive            bool        `json:"is_active"`
}

// Task describes a minimal task tracked under a thread.
type Task struct {
	ID        string    `json:"id"`
	ThreadID  string    `json:"thread_id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
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
}

// UpdateTaskStatusInput collects the minimum fields required to update a task status.
type UpdateTaskStatusInput struct {
	Status string
}

var (
	// ErrThreadNotFound is returned when a thread lookup fails.
	ErrThreadNotFound = errors.New("thread not found")
	// ErrTaskNotFound is returned when a task lookup fails.
	ErrTaskNotFound = errors.New("task not found")
	// ErrInvalidTaskStatus is returned when a task status is unsupported.
	ErrInvalidTaskStatus = errors.New("invalid task status")
)

// Registry holds the in-memory workspace and thread set for the current runtime process.
type Registry struct {
	mu               sync.RWMutex
	workspace        Workspace
	threads          map[string]Thread
	tasks            map[string][]Task
	events           map[string][]Event
	order            []string
	activeThreadID   string
	nextThreadNumber int
	nextTaskNumber   int
	nextEventNumber  int
}

// NewRegistry creates the default single-workspace registry for the given project root.
func NewRegistry(projectRoot string) *Registry {
	workspaceID := filepath.Base(projectRoot)
	if workspaceID == "" || workspaceID == "." || workspaceID == string(filepath.Separator) {
		workspaceID = "workspace"
	}

	sharedDocsRoot := projectRoot
	if docsCandidate := filepath.Join(projectRoot, "docs"); docsCandidate != "" {
		sharedDocsRoot = docsCandidate
	}

	return &Registry{
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
	}
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
		MessageHistory: []string{},
		ToolHistory:    []string{},
		TaskState:      []string{},
		ArtifactPaths:  []string{},
		RuntimeFlags:   []string{},
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
		ID:        fmt.Sprintf("task-%d", taskNumber),
		ThreadID:  threadID,
		Title:     title,
		Status:    "queued",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	r.tasks[threadID] = append(r.tasks[threadID], task)
	thread.TaskState = append(thread.TaskState, task.ID)
	r.threads[threadID] = thread
	r.appendEventLocked(threadID, "task.created", fmt.Sprintf("%s queued on %s", title, thread.Name))
	return task, true
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
	for index, task := range tasks {
		if task.ID != taskID {
			continue
		}

		task.Status = input.Status
		task.UpdatedAt = time.Now().UTC()
		tasks[index] = task
		r.tasks[threadID] = tasks
		r.threads[threadID] = thread
		r.appendEventLocked(threadID, "task.updated", fmt.Sprintf("%s moved to %s on %s", task.Title, task.Status, thread.Name))
		return task, nil
	}

	return Task{}, ErrTaskNotFound
}

func snapshotThread(thread Thread, activeThreadID string) Thread {
	thread.MessageHistoryCount = len(thread.MessageHistory)
	thread.ToolCallCount = len(thread.ToolHistory)
	thread.ArtifactCount = len(thread.ArtifactPaths)
	thread.IsActive = thread.ID == activeThreadID
	thread.MessageHistory = append([]string(nil), thread.MessageHistory...)
	thread.ToolHistory = append([]string(nil), thread.ToolHistory...)
	thread.TaskState = append([]string(nil), thread.TaskState...)
	thread.ArtifactPaths = append([]string(nil), thread.ArtifactPaths...)
	thread.RuntimeFlags = append([]string(nil), thread.RuntimeFlags...)
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
}

func isSupportedTaskStatus(status string) bool {
	switch status {
	case "queued", "running", "completed", "failed":
		return true
	default:
		return false
	}
}

// SortedIDs returns the thread ids in stable creation order.
func (r *Registry) SortedIDs() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := append([]string(nil), r.order...)
	sort.Strings(ids)
	return ids
}
