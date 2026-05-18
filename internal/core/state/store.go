package state

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	StoreName = "sqlite"
	dbFile    = "state.db"
)

// WorkspaceRecord is the persisted workspace row.
type WorkspaceRecord struct {
	ID             string
	ProjectRoot    string
	SharedDocsRoot string
	CreatedAt      time.Time
	ActiveThreadID string
	IsActive       bool
}

// ThreadRecord is the persisted thread row.
type ThreadRecord struct {
	ID             string
	WorkspaceID    string
	Name           string
	Status         string
	ActiveModel    string
	ReasoningEffort string
	PermissionMode string
	CreatedAt      time.Time
}

// TaskRecord is the persisted task row.
type TaskRecord struct {
	ID             string
	ThreadID       string
	Title          string
	Status         string
	Kind           string
	Input          string
	ResultSummary  string
	ApprovalStatus string
	ParentTaskID   string
	WaitingStatus  string
	AgentState     string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MessageRecord is the persisted thread message row.
type MessageRecord struct {
	ID        string
	ThreadID  string
	Role      string
	Content   string
	CreatedAt time.Time
}

// ToolCallRecord is the persisted thread tool call row.
type ToolCallRecord struct {
	ID        string
	ThreadID  string
	ToolID    string
	Status    string
	Summary   string
	CreatedAt time.Time
}

// ArtifactRecord is the persisted thread artifact row.
type ArtifactRecord struct {
	ID        string
	ThreadID  string
	Path      string
	Kind      string
	CreatedAt time.Time
}

// RuntimeFlagRecord is the persisted thread runtime flag row.
type RuntimeFlagRecord struct {
	ThreadID  string
	Key       string
	Value     string
	UpdatedAt time.Time
}

// EventRecord is the persisted event row.
type EventRecord struct {
	ID        string
	ThreadID  string
	Type      string
	Message   string
	CreatedAt time.Time
}

// ApprovalRecord is the persisted approval row.
type ApprovalRecord struct {
	ID          string
	ThreadID    string
	TaskID      string
	ToolKind    string
	Status      string
	Summary     string
	TargetPaths string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WriteExecutionRecord is the persisted write execution audit row.
type WriteExecutionRecord struct {
	ID                    string
	ThreadID              string
	TaskID                string
	ApprovalID            string
	ToolKind              string
	Operation             string
	RelatedExecutionID    string
	Status                string
	TargetPaths           string
	PatchHash             string
	PatchSummary          string
	BeforeSnapshotSummary string
	AfterSnapshotSummary  string
	RollbackPayload       string
	ResultSummary         string
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// Snapshot is the full persisted runtime state payload.
type Snapshot struct {
	Workspace       WorkspaceRecord
	Workspaces      []WorkspaceRecord
	ActiveWorkspaceID string
	Threads         []ThreadRecord
	Tasks           []TaskRecord
	Messages        []MessageRecord
	ToolCalls       []ToolCallRecord
	Artifacts       []ArtifactRecord
	Flags           []RuntimeFlagRecord
	Events          []EventRecord
	Approvals       []ApprovalRecord
	WriteExecutions []WriteExecutionRecord
}

// Store persists runtime state to SQLite.
type Store struct {
	path string
	db   *sql.DB
}

// PathForProject returns the fixed state DB path for a project root.
func PathForProject(projectRoot string) string {
	return filepath.Join(projectRoot, ".gen-code", dbFile)
}

// Open creates the SQLite store and runs the minimum schema migration.
func Open(projectRoot string) (*Store, error) {
	path := PathForProject(projectRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite wal mode: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set sqlite busy timeout: %w", err)
	}

	store := &Store{path: path, db: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

// Path returns the SQLite file path.
func (s *Store) Path() string {
	return s.path
}

// Close releases the SQLite handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Load returns the full persisted snapshot.
func (s *Store) Load() (Snapshot, error) {
	snapshot := Snapshot{}

	workspaceRows, err := s.db.Query(`
		SELECT id, project_root, shared_docs_root, created_at, active_thread_id, is_active
		FROM workspace
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load workspaces: %w", err)
	}
	defer workspaceRows.Close()

	for workspaceRows.Next() {
		var item WorkspaceRecord
		var createdAt string
		if err := workspaceRows.Scan(&item.ID, &item.ProjectRoot, &item.SharedDocsRoot, &createdAt, &item.ActiveThreadID, &item.IsActive); err != nil {
			return Snapshot{}, fmt.Errorf("scan workspace: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse workspace created_at: %w", err)
		}
		snapshot.Workspaces = append(snapshot.Workspaces, item)
		if item.IsActive {
			snapshot.ActiveWorkspaceID = item.ID
		}
	}
	if len(snapshot.Workspaces) > 0 {
		snapshot.Workspace = snapshot.Workspaces[0]
		if strings.TrimSpace(snapshot.ActiveWorkspaceID) == "" {
			snapshot.ActiveWorkspaceID = snapshot.Workspace.ID
		}
		for _, item := range snapshot.Workspaces {
			if item.ID == snapshot.ActiveWorkspaceID {
				snapshot.Workspace = item
				break
			}
		}
	}

	threadRows, err := s.db.Query(`
		SELECT id, workspace_id, name, status, active_model, reasoning_effort, permission_mode, created_at
		FROM threads
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load threads: %w", err)
	}
	defer threadRows.Close()

	for threadRows.Next() {
		var item ThreadRecord
		var created string
		if err := threadRows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.Status, &item.ActiveModel, &item.ReasoningEffort, &item.PermissionMode, &created); err != nil {
			return Snapshot{}, fmt.Errorf("scan thread: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse thread created_at: %w", err)
		}
		snapshot.Threads = append(snapshot.Threads, item)
	}

	taskRows, err := s.db.Query(`
		SELECT id, thread_id, title, status, kind, input, result_summary, approval_status, parent_task_id, waiting_status, agent_state, created_at, updated_at
		FROM tasks
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load tasks: %w", err)
	}
	defer taskRows.Close()

	for taskRows.Next() {
		var item TaskRecord
		var created, updated string
		if err := taskRows.Scan(&item.ID, &item.ThreadID, &item.Title, &item.Status, &item.Kind, &item.Input, &item.ResultSummary, &item.ApprovalStatus, &item.ParentTaskID, &item.WaitingStatus, &item.AgentState, &created, &updated); err != nil {
			return Snapshot{}, fmt.Errorf("scan task: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse task created_at: %w", err)
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339, updated)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse task updated_at: %w", err)
		}
		snapshot.Tasks = append(snapshot.Tasks, item)
	}

	messageRows, err := s.db.Query(`
		SELECT id, thread_id, role, content, created_at
		FROM thread_messages
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load messages: %w", err)
	}
	defer messageRows.Close()

	for messageRows.Next() {
		var item MessageRecord
		var created string
		if err := messageRows.Scan(&item.ID, &item.ThreadID, &item.Role, &item.Content, &created); err != nil {
			return Snapshot{}, fmt.Errorf("scan message: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse message created_at: %w", err)
		}
		snapshot.Messages = append(snapshot.Messages, item)
	}

	toolCallRows, err := s.db.Query(`
		SELECT id, thread_id, tool_id, status, summary, created_at
		FROM thread_tool_calls
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load tool calls: %w", err)
	}
	defer toolCallRows.Close()

	for toolCallRows.Next() {
		var item ToolCallRecord
		var created string
		if err := toolCallRows.Scan(&item.ID, &item.ThreadID, &item.ToolID, &item.Status, &item.Summary, &created); err != nil {
			return Snapshot{}, fmt.Errorf("scan tool call: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse tool call created_at: %w", err)
		}
		snapshot.ToolCalls = append(snapshot.ToolCalls, item)
	}

	artifactRows, err := s.db.Query(`
		SELECT id, thread_id, path, kind, created_at
		FROM thread_artifacts
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load artifacts: %w", err)
	}
	defer artifactRows.Close()

	for artifactRows.Next() {
		var item ArtifactRecord
		var created string
		if err := artifactRows.Scan(&item.ID, &item.ThreadID, &item.Path, &item.Kind, &created); err != nil {
			return Snapshot{}, fmt.Errorf("scan artifact: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse artifact created_at: %w", err)
		}
		snapshot.Artifacts = append(snapshot.Artifacts, item)
	}

	flagRows, err := s.db.Query(`
		SELECT thread_id, key, value, updated_at
		FROM thread_runtime_flags
		ORDER BY thread_id ASC, key ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load runtime flags: %w", err)
	}
	defer flagRows.Close()

	for flagRows.Next() {
		var item RuntimeFlagRecord
		var updated string
		if err := flagRows.Scan(&item.ThreadID, &item.Key, &item.Value, &updated); err != nil {
			return Snapshot{}, fmt.Errorf("scan runtime flag: %w", err)
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339, updated)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse runtime flag updated_at: %w", err)
		}
		snapshot.Flags = append(snapshot.Flags, item)
	}

	eventRows, err := s.db.Query(`
		SELECT id, thread_id, type, message, created_at
		FROM events
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load events: %w", err)
	}
	defer eventRows.Close()

	for eventRows.Next() {
		var item EventRecord
		var created string
		if err := eventRows.Scan(&item.ID, &item.ThreadID, &item.Type, &item.Message, &created); err != nil {
			return Snapshot{}, fmt.Errorf("scan event: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse event created_at: %w", err)
		}
		snapshot.Events = append(snapshot.Events, item)
	}

	approvalRows, err := s.db.Query(`
		SELECT id, thread_id, task_id, tool_kind, status, summary, target_paths, created_at, updated_at
		FROM thread_approvals
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load approvals: %w", err)
	}
	defer approvalRows.Close()

	for approvalRows.Next() {
		var item ApprovalRecord
		var created, updated string
		if err := approvalRows.Scan(&item.ID, &item.ThreadID, &item.TaskID, &item.ToolKind, &item.Status, &item.Summary, &item.TargetPaths, &created, &updated); err != nil {
			return Snapshot{}, fmt.Errorf("scan approval: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse approval created_at: %w", err)
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339, updated)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse approval updated_at: %w", err)
		}
		snapshot.Approvals = append(snapshot.Approvals, item)
	}

	writeExecutionRows, err := s.db.Query(`
		SELECT id, thread_id, task_id, approval_id, tool_kind, operation, related_execution_id, status, target_paths, patch_hash, patch_summary,
		       before_snapshot_summary, after_snapshot_summary, rollback_payload, result_summary, created_at, updated_at
		FROM thread_write_executions
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("load write executions: %w", err)
	}
	defer writeExecutionRows.Close()

	for writeExecutionRows.Next() {
		var item WriteExecutionRecord
		var created, updated string
		if err := writeExecutionRows.Scan(
			&item.ID,
			&item.ThreadID,
			&item.TaskID,
			&item.ApprovalID,
			&item.ToolKind,
			&item.Operation,
			&item.RelatedExecutionID,
			&item.Status,
			&item.TargetPaths,
			&item.PatchHash,
			&item.PatchSummary,
			&item.BeforeSnapshotSummary,
			&item.AfterSnapshotSummary,
			&item.RollbackPayload,
			&item.ResultSummary,
			&created,
			&updated,
		); err != nil {
			return Snapshot{}, fmt.Errorf("scan write execution: %w", err)
		}
		item.CreatedAt, err = time.Parse(time.RFC3339, created)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse write execution created_at: %w", err)
		}
		item.UpdatedAt, err = time.Parse(time.RFC3339, updated)
		if err != nil {
			return Snapshot{}, fmt.Errorf("parse write execution updated_at: %w", err)
		}
		snapshot.WriteExecutions = append(snapshot.WriteExecutions, item)
	}

	return snapshot, nil
}

// SaveWorkspace upserts the single workspace record.
func (s *Store) SaveWorkspace(item WorkspaceRecord) error {
	if item.IsActive {
		if _, err := s.db.Exec(`UPDATE workspace SET is_active = 0 WHERE id <> ?`, item.ID); err != nil {
			return fmt.Errorf("clear active workspace: %w", err)
		}
	}
	_, err := s.db.Exec(`
		INSERT INTO workspace (id, project_root, shared_docs_root, created_at, active_thread_id, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			project_root=excluded.project_root,
			shared_docs_root=excluded.shared_docs_root,
			created_at=excluded.created_at,
			active_thread_id=excluded.active_thread_id,
			is_active=excluded.is_active
	`, item.ID, item.ProjectRoot, item.SharedDocsRoot, item.CreatedAt.Format(time.RFC3339), item.ActiveThreadID, item.IsActive)
	if err != nil {
		return fmt.Errorf("save workspace: %w", err)
	}
	return nil
}

// SaveThread upserts a thread row.
func (s *Store) SaveThread(item ThreadRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO threads (id, workspace_id, name, status, active_model, reasoning_effort, permission_mode, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			workspace_id=excluded.workspace_id,
			name=excluded.name,
			status=excluded.status,
			active_model=excluded.active_model,
			reasoning_effort=excluded.reasoning_effort,
			permission_mode=excluded.permission_mode,
			created_at=excluded.created_at
	`, item.ID, item.WorkspaceID, item.Name, item.Status, item.ActiveModel, item.ReasoningEffort, item.PermissionMode, item.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save thread: %w", err)
	}
	return nil
}

// SaveTask upserts a task row.
func (s *Store) SaveTask(item TaskRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO tasks (id, thread_id, title, status, kind, input, result_summary, approval_status, parent_task_id, waiting_status, agent_state, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			thread_id=excluded.thread_id,
			title=excluded.title,
			status=excluded.status,
			kind=excluded.kind,
			input=excluded.input,
			result_summary=excluded.result_summary,
			approval_status=excluded.approval_status,
			parent_task_id=excluded.parent_task_id,
			waiting_status=excluded.waiting_status,
			agent_state=excluded.agent_state,
			created_at=excluded.created_at,
			updated_at=excluded.updated_at
	`, item.ID, item.ThreadID, item.Title, item.Status, item.Kind, item.Input, item.ResultSummary, item.ApprovalStatus, item.ParentTaskID, item.WaitingStatus, item.AgentState, item.CreatedAt.Format(time.RFC3339), item.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save task: %w", err)
	}
	return nil
}

// SaveMessage inserts a thread message row.
func (s *Store) SaveMessage(item MessageRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO thread_messages (id, thread_id, role, content, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, item.ID, item.ThreadID, item.Role, item.Content, item.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

// SaveToolCall inserts a thread tool call row.
func (s *Store) SaveToolCall(item ToolCallRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO thread_tool_calls (id, thread_id, tool_id, status, summary, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			thread_id=excluded.thread_id,
			tool_id=excluded.tool_id,
			status=excluded.status,
			summary=excluded.summary,
			created_at=excluded.created_at
	`, item.ID, item.ThreadID, item.ToolID, item.Status, item.Summary, item.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save tool call: %w", err)
	}
	return nil
}

// SaveArtifact inserts a thread artifact row.
func (s *Store) SaveArtifact(item ArtifactRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO thread_artifacts (id, thread_id, path, kind, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, item.ID, item.ThreadID, item.Path, item.Kind, item.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save artifact: %w", err)
	}
	return nil
}

// SaveRuntimeFlag upserts a thread runtime flag row.
func (s *Store) SaveRuntimeFlag(item RuntimeFlagRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO thread_runtime_flags (thread_id, key, value, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(thread_id, key) DO UPDATE SET
			value=excluded.value,
			updated_at=excluded.updated_at
	`, item.ThreadID, item.Key, item.Value, item.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save runtime flag: %w", err)
	}
	return nil
}

// SaveEvent inserts an event row.
func (s *Store) SaveEvent(item EventRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO events (id, thread_id, type, message, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, item.ID, item.ThreadID, item.Type, item.Message, item.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save event: %w", err)
	}
	return nil
}

// SaveApproval upserts an approval row.
func (s *Store) SaveApproval(item ApprovalRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO thread_approvals (id, thread_id, task_id, tool_kind, status, summary, target_paths, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			thread_id=excluded.thread_id,
			task_id=excluded.task_id,
			tool_kind=excluded.tool_kind,
			status=excluded.status,
			summary=excluded.summary,
			target_paths=excluded.target_paths,
			created_at=excluded.created_at,
			updated_at=excluded.updated_at
	`, item.ID, item.ThreadID, item.TaskID, item.ToolKind, item.Status, item.Summary, item.TargetPaths, item.CreatedAt.Format(time.RFC3339), item.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save approval: %w", err)
	}
	return nil
}

// SaveWriteExecution inserts a write execution audit row.
func (s *Store) SaveWriteExecution(item WriteExecutionRecord) error {
	_, err := s.db.Exec(`
		INSERT INTO thread_write_executions (
			id, thread_id, task_id, approval_id, tool_kind, operation, related_execution_id, status, target_paths, patch_hash, patch_summary,
			before_snapshot_summary, after_snapshot_summary, rollback_payload, result_summary, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			thread_id=excluded.thread_id,
			task_id=excluded.task_id,
			approval_id=excluded.approval_id,
			tool_kind=excluded.tool_kind,
			operation=excluded.operation,
			related_execution_id=excluded.related_execution_id,
			status=excluded.status,
			target_paths=excluded.target_paths,
			patch_hash=excluded.patch_hash,
			patch_summary=excluded.patch_summary,
			before_snapshot_summary=excluded.before_snapshot_summary,
			after_snapshot_summary=excluded.after_snapshot_summary,
			rollback_payload=excluded.rollback_payload,
			result_summary=excluded.result_summary,
			created_at=excluded.created_at,
			updated_at=excluded.updated_at
	`, item.ID, item.ThreadID, item.TaskID, item.ApprovalID, item.ToolKind, item.Operation, item.RelatedExecutionID, item.Status, item.TargetPaths, item.PatchHash, item.PatchSummary, item.BeforeSnapshotSummary, item.AfterSnapshotSummary, item.RollbackPayload, item.ResultSummary, item.CreatedAt.Format(time.RFC3339), item.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save write execution: %w", err)
	}
	return nil
}

func (s *Store) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS schema_version (
			version INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS workspace (
			id TEXT PRIMARY KEY,
			project_root TEXT NOT NULL,
			shared_docs_root TEXT NOT NULL,
			created_at TEXT NOT NULL,
			active_thread_id TEXT NOT NULL DEFAULT '',
			is_active INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS threads (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			name TEXT NOT NULL,
			status TEXT NOT NULL,
			active_model TEXT NOT NULL DEFAULT '',
			reasoning_effort TEXT NOT NULL DEFAULT '',
			permission_mode TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id TEXT PRIMARY KEY,
			thread_id TEXT NOT NULL,
			title TEXT NOT NULL,
			status TEXT NOT NULL,
			kind TEXT NOT NULL DEFAULT '',
			input TEXT NOT NULL DEFAULT '',
			result_summary TEXT NOT NULL DEFAULT '',
			approval_status TEXT NOT NULL DEFAULT '',
			parent_task_id TEXT NOT NULL DEFAULT '',
			waiting_status TEXT NOT NULL DEFAULT '',
			agent_state TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
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
			rollback_payload TEXT NOT NULL DEFAULT '[]',
			result_summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}

	for _, query := range queries {
		if _, err := s.db.Exec(query); err != nil {
			return fmt.Errorf("migrate sqlite schema: %w", err)
		}
	}

	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM schema_version`).Scan(&count); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if count == 0 {
		if _, err := s.db.Exec(`INSERT INTO schema_version(version) VALUES (1)`); err != nil {
			return fmt.Errorf("write schema version: %w", err)
		}
	}
	if _, err := s.db.Exec(`ALTER TABLE tasks ADD COLUMN kind TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add tasks.kind column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE workspace ADD COLUMN is_active INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add workspace.is_active column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE threads ADD COLUMN reasoning_effort TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add threads.reasoning_effort column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE tasks ADD COLUMN input TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add tasks.input column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE tasks ADD COLUMN result_summary TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add tasks.result_summary column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE tasks ADD COLUMN approval_status TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add tasks.approval_status column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE tasks ADD COLUMN parent_task_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add tasks.parent_task_id column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE tasks ADD COLUMN waiting_status TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add tasks.waiting_status column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE tasks ADD COLUMN agent_state TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add tasks.agent_state column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE thread_write_executions ADD COLUMN operation TEXT NOT NULL DEFAULT 'apply'`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add thread_write_executions.operation column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE thread_write_executions ADD COLUMN related_execution_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add thread_write_executions.related_execution_id column: %w", err)
	}
	if _, err := s.db.Exec(`ALTER TABLE thread_write_executions ADD COLUMN rollback_payload TEXT NOT NULL DEFAULT '[]'`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return fmt.Errorf("add thread_write_executions.rollback_payload column: %w", err)
	}
	return nil
}

// MaxSuffix extracts the largest trailing integer from IDs like thread-12.
func MaxSuffix(ids []string, prefix string) int {
	maxValue := 0
	for _, id := range ids {
		if !strings.HasPrefix(id, prefix) {
			continue
		}
		value, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
		if err == nil && value > maxValue {
			maxValue = value
		}
	}
	return maxValue
}
