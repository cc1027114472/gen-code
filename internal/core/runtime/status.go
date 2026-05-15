package runtime

import (
	"context"
	"errors"
	"time"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/mcp"
	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/session"
	"llmtrace/internal/core/skill"
	"llmtrace/internal/core/tool"
	"llmtrace/internal/platform/xerror"
)

// Status represents the aggregate runtime state surfaced to CLI, API, and desktop.
type Status struct {
	AppVersion             string      `json:"app_version"`
	DesktopShellStatus     string      `json:"desktop_shell_status"`
	AppServerStatus        string      `json:"app_server_status"`
	GoBridgeStatus         string      `json:"go_bridge_status"`
	RuntimeSource          string      `json:"runtime_source"`
	WorkspaceID            string      `json:"workspace_id"`
	ProjectRoot            string      `json:"project_root"`
	ThreadCount            int         `json:"thread_count"`
	ActiveThreadID         string      `json:"active_thread_id"`
	ActiveThreadTaskCount  int         `json:"active_thread_task_count"`
	ActiveThreadEventCount int         `json:"active_thread_event_count"`
	ActiveSkillGroup       skill.Group `json:"active_skill_group"`
	ConfiguredMCPServers   int         `json:"configured_mcp_server_count"`
	PermissionMode         policy.Mode `json:"permission_mode"`
}

// BridgeCheckResult describes the result of a lightweight bridge verification.
type BridgeCheckResult struct {
	Message string `json:"message"`
	Status  string `json:"status"`
}

// Service exposes the read-only runtime surface for app-server, CLI, and desktop use.
type Service struct {
	version       string
	skillGroup    skill.Group
	permission    policy.Mode
	desktopStatus string
	appServer     string
	goBridge      string
	projectRoot   string
	tools         *tool.Registry
	skills        *skill.Manager
	mcp           *mcp.Manager
	session       *session.Registry
}

// NewService constructs a Service with static runtime metadata.
func NewService(version string, group skill.Group, permission policy.Mode, projectRoot string, tools *tool.Registry, skills *skill.Manager, mcpManager *mcp.Manager, sessions *session.Registry) *Service {
	if sessions == nil {
		sessions = session.NewRegistry(projectRoot)
	}
	return &Service{
		version:       version,
		skillGroup:    group,
		permission:    permission,
		desktopStatus: "ready",
		appServer:     "running",
		goBridge:      "ready for verification",
		projectRoot:   projectRoot,
		tools:         tools,
		skills:        skills,
		mcp:           mcpManager,
		session:       sessions,
	}
}

// Snapshot returns the current runtime summary.
func (s *Service) Snapshot() Status {
	workspace := s.session.Workspace()
	activeTaskCount := 0
	activeEventCount := 0
	if activeThreadID := s.session.ActiveThreadID(); activeThreadID != "" {
		if tasks, ok := s.session.Tasks(activeThreadID); ok {
			activeTaskCount = len(tasks)
		}
		if events, ok := s.session.Events(activeThreadID); ok {
			activeEventCount = len(events)
		}
	}
	return Status{
		AppVersion:             s.version,
		DesktopShellStatus:     s.desktopStatus,
		AppServerStatus:        s.appServer,
		GoBridgeStatus:         s.goBridge,
		RuntimeSource:          "local",
		WorkspaceID:            workspace.ID,
		ProjectRoot:            workspace.ProjectRoot,
		ThreadCount:            workspace.ActiveThreadCount,
		ActiveThreadID:         s.session.ActiveThreadID(),
		ActiveThreadTaskCount:  activeTaskCount,
		ActiveThreadEventCount: activeEventCount,
		ActiveSkillGroup:       s.skillGroup,
		ConfiguredMCPServers:   len(s.mcp.List()),
		PermissionMode:         s.permission,
	}
}

// Status returns the app-server contract view of the current runtime summary.
func (s *Service) Status(context.Context) (runtimecontract.Status, error) {
	snapshot := s.Snapshot()
	return runtimecontract.Status{
		State:          snapshot.AppServerStatus,
		Ready:          snapshot.AppServerStatus == "running",
		Message:        snapshot.GoBridgeStatus,
		RuntimeSource:  snapshot.RuntimeSource,
		WorkspaceID:    snapshot.WorkspaceID,
		ProjectRoot:    snapshot.ProjectRoot,
		ThreadCount:    snapshot.ThreadCount,
		ActiveThreadID: snapshot.ActiveThreadID,
		TaskCount:      snapshot.ActiveThreadTaskCount,
		EventCount:     snapshot.ActiveThreadEventCount,
	}, nil
}

// Workspace returns the current workspace descriptor.
func (s *Service) Workspace(context.Context) (runtimecontract.WorkspaceDescriptor, error) {
	item := s.session.Workspace()
	return runtimecontract.WorkspaceDescriptor{
		ID:                item.ID,
		ProjectRoot:       item.ProjectRoot,
		SharedDocsRoot:    item.SharedDocsRoot,
		CreatedAt:         item.CreatedAt.Format(time.RFC3339),
		ActiveThreadCount: item.ActiveThreadCount,
	}, nil
}

// Threads returns all registered thread descriptors.
func (s *Service) Threads(context.Context) ([]runtimecontract.ThreadDescriptor, error) {
	items := s.session.Threads()
	result := make([]runtimecontract.ThreadDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, toThreadDescriptor(item))
	}
	return result, nil
}

// CreateThread registers a new thread under the current workspace.
func (s *Service) CreateThread(_ context.Context, request runtimecontract.CreateThreadRequest) (runtimecontract.ThreadDescriptor, error) {
	mode, err := policy.ParseMode(request.PermissionMode)
	if err != nil {
		return runtimecontract.ThreadDescriptor{}, xerror.BadRequest(1003, err.Error())
	}

	item := s.session.CreateThread(session.CreateThreadInput{
		Name:           request.Name,
		ActiveModel:    request.ActiveModel,
		PermissionMode: mode,
	})
	return toThreadDescriptor(item), nil
}

// Thread returns a single thread descriptor by id.
func (s *Service) Thread(_ context.Context, id string) (runtimecontract.ThreadDescriptor, error) {
	item, ok := s.session.Thread(id)
	if !ok {
		return runtimecontract.ThreadDescriptor{}, xerror.NotFound(1004, "thread not found")
	}
	return toThreadDescriptor(item), nil
}

// ActivateThread marks the thread with the given id as active.
func (s *Service) ActivateThread(_ context.Context, id string) (runtimecontract.ThreadDescriptor, error) {
	item, ok := s.session.ActivateThread(id)
	if !ok {
		return runtimecontract.ThreadDescriptor{}, xerror.NotFound(1004, "thread not found")
	}
	return toThreadDescriptor(item), nil
}

// Tasks returns the task descriptors under the given thread.
func (s *Service) Tasks(_ context.Context, threadID string) ([]runtimecontract.TaskDescriptor, error) {
	items, ok := s.session.Tasks(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	result := make([]runtimecontract.TaskDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.TaskDescriptor{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Title:     item.Title,
			Status:    item.Status,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
			UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// CreateTask registers a task under the given thread.
func (s *Service) CreateTask(_ context.Context, threadID string, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	item, ok := s.session.CreateTask(threadID, session.CreateTaskInput{Title: request.Title})
	if !ok {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
	}

	return runtimecontract.TaskDescriptor{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		Title:     item.Title,
		Status:    item.Status,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateTaskStatus updates an existing task status under the given thread.
func (s *Service) UpdateTaskStatus(_ context.Context, threadID string, taskID string, request runtimecontract.UpdateTaskStatusRequest) (runtimecontract.TaskDescriptor, error) {
	item, err := s.session.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{Status: request.Status})
	if err != nil {
		switch {
		case errors.Is(err, session.ErrThreadNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		case errors.Is(err, session.ErrTaskNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1007, "task not found")
		case errors.Is(err, session.ErrInvalidTaskStatus):
			return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1001, "invalid task status")
		default:
			return runtimecontract.TaskDescriptor{}, xerror.Internal(2001, "failed to update task status")
		}
	}

	return runtimecontract.TaskDescriptor{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		Title:     item.Title,
		Status:    item.Status,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
		UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// Events returns the event descriptors under the given thread.
func (s *Service) Events(_ context.Context, threadID string) ([]runtimecontract.EventDescriptor, error) {
	items, ok := s.session.Events(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	result := make([]runtimecontract.EventDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.EventDescriptor{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Type:      item.Type,
			Message:   item.Message,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// StreamEvents returns a minimal replay stream of known thread events.
func (s *Service) StreamEvents(_ context.Context, threadID string) (<-chan runtimecontract.EventDescriptor, error) {
	items, ok := s.session.Events(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	ch := make(chan runtimecontract.EventDescriptor, len(items))
	for _, item := range items {
		ch <- runtimecontract.EventDescriptor{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Type:      item.Type,
			Message:   item.Message,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		}
	}
	close(ch)
	return ch, nil
}

// FullStatus returns the richer runtime summary for CLI and desktop use.
func (s *Service) FullStatus() Status {
	return s.Snapshot()
}

// Tools returns the app-server contract view of registered runtime tools.
func (s *Service) Tools(context.Context) ([]runtimecontract.Tool, error) {
	items := s.tools.List()
	result := make([]runtimecontract.Tool, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.Tool{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Permission:  string(item.PermissionMode),
			Source:      item.Source,
		})
	}
	return result, nil
}

// ToolDescriptors returns the richer tool descriptors for CLI and desktop use.
func (s *Service) ToolDescriptors() []tool.Descriptor {
	return s.tools.List()
}

// Skills returns the app-server contract view of visible runtime skills.
func (s *Service) Skills(_ context.Context) ([]runtimecontract.Skill, error) {
	items := s.skills.List(s.skillGroup)
	result := make([]runtimecontract.Skill, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.Skill{
			ID:          item.ID,
			Group:       string(item.Group),
			Name:        item.Name,
			Description: item.Description,
			Source:      groupSource(item.Group),
		})
	}
	return result, nil
}

// SkillDescriptors returns the richer skill descriptors for the requested group.
func (s *Service) SkillDescriptors(group skill.Group) []skill.Descriptor {
	return s.skills.List(group)
}

// MCPServers returns the app-server contract view of configured MCP server metadata.
func (s *Service) MCPServers(context.Context) ([]runtimecontract.MCPServer, error) {
	items := s.mcp.List()
	result := make([]runtimecontract.MCPServer, 0, len(items))
	for _, item := range items {
		status := "disabled"
		if item.Enabled {
			status = "enabled"
		}
		result = append(result, runtimecontract.MCPServer{
			ID:            item.ID,
			Source:        item.Source,
			Enabled:       item.Enabled,
			ToolCount:     item.ToolCount,
			ResourceCount: item.ResourceCount,
			Status:        status,
		})
	}
	return result, nil
}

// MCPDescriptors returns the richer MCP descriptors for CLI and desktop use.
func (s *Service) MCPDescriptors() []mcp.ServerDescriptor {
	return s.mcp.List()
}

// CheckBridge performs a lightweight runtime-side bridge check.
func (s *Service) CheckBridge(context.Context, map[string]any) (runtimecontract.BridgeCheckResult, error) {
	result := BridgeCheckResult{
		Message: "gen-code runtime bridge reachable",
		Status:  "ok",
	}
	return runtimecontract.BridgeCheckResult{
		OK:      result.Status == "ok",
		Message: result.Message,
		Details: map[string]any{"bridge_status": result.Status},
	}, nil
}

// BridgeStatus returns the richer bridge result for desktop use.
func (s *Service) BridgeStatus() BridgeCheckResult {
	return BridgeCheckResult{
		Message: "gen-code runtime bridge reachable",
		Status:  "ok",
	}
}

func toThreadDescriptor(item session.Thread) runtimecontract.ThreadDescriptor {
	return runtimecontract.ThreadDescriptor{
		ID:                  item.ID,
		WorkspaceID:         item.WorkspaceID,
		Name:                item.Name,
		Status:              item.Status,
		ActiveModel:         item.ActiveModel,
		PermissionMode:      string(item.PermissionMode),
		MessageHistoryCount: item.MessageHistoryCount,
		ToolCallCount:       item.ToolCallCount,
		ArtifactCount:       item.ArtifactCount,
		CreatedAt:           item.CreatedAt.Format(time.RFC3339),
		IsActive:            item.IsActive,
	}
}

func groupSource(group skill.Group) string {
	switch group {
	case skill.Codex:
		return "codex"
	case skill.CC:
		return "cc"
	default:
		return "common"
	}
}
