package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/core/mcp"
	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/provider"
	"llmtrace/internal/core/runner"
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
	RuntimeSourceDetail    string      `json:"runtime_source_detail"`
	RuntimeTrust           string      `json:"runtime_trust"`
	CanonicalRuntimeURL    string      `json:"canonical_runtime_url"`
	StateStore             string      `json:"state_store"`
	StatePath              string      `json:"state_path"`
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
	version        string
	skillGroup     skill.Group
	permission     policy.Mode
	desktopStatus  string
	appServer      string
	goBridge       string
	canonicalURL   string
	projectRoot    string
	tools          *tool.Registry
	skills         *skill.Manager
	mcp            *mcp.Manager
	providers      *provider.Registry
	providerClient *provider.Client
	prober         *provider.Prober
	session        *session.Registry
	runner         *runner.Runner
}

// NewService constructs a Service with static runtime metadata.
func NewService(version string, group skill.Group, permission policy.Mode, projectRoot string, tools *tool.Registry, skills *skill.Manager, mcpManager *mcp.Manager, providers *provider.Registry, sessions *session.Registry) *Service {
	return newService(version, group, permission, projectRoot, tools, skills, mcpManager, providers, sessions, true)
}

// NewServiceWithoutRecovery constructs a Service without startup task recovery.
func NewServiceWithoutRecovery(version string, group skill.Group, permission policy.Mode, projectRoot string, tools *tool.Registry, skills *skill.Manager, mcpManager *mcp.Manager, providers *provider.Registry, sessions *session.Registry) *Service {
	return newService(version, group, permission, projectRoot, tools, skills, mcpManager, providers, sessions, false)
}

func newService(version string, group skill.Group, permission policy.Mode, projectRoot string, tools *tool.Registry, skills *skill.Manager, mcpManager *mcp.Manager, providers *provider.Registry, sessions *session.Registry, recoverInterrupted bool) *Service {
	if sessions == nil {
		sessions = session.NewRegistry(projectRoot)
	}
	if providers == nil {
		providers = provider.NewRegistry("")
	}
	providerClient := provider.NewClient(providers)
	taskRunner := runner.New(sessions, providerClient)
	if recoverInterrupted {
		_ = taskRunner.RecoverInterruptedTasks()
	}
	return &Service{
		version:        version,
		skillGroup:     group,
		permission:     permission,
		desktopStatus:  "ready",
		appServer:      "running",
		goBridge:       "ready for verification",
		canonicalURL:   canonicalRuntimeURL(""),
		projectRoot:    projectRoot,
		tools:          tools,
		skills:         skills,
		mcp:            mcpManager,
		providers:      providers,
		providerClient: providerClient,
		prober:         provider.NewProber(),
		session:        sessions,
		runner:         taskRunner,
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
		RuntimeSource:          "remote-app-server",
		RuntimeSourceDetail:    "canonical shared runtime served by the app-server entry",
		RuntimeTrust:           "canonical",
		CanonicalRuntimeURL:    s.canonicalURL,
		StateStore:             s.session.StateStoreName(),
		StatePath:              s.session.StatePath(),
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

// SetCanonicalRuntimeURL pins the runtime status to the app-server's canonical base URL.
func (s *Service) SetCanonicalRuntimeURL(raw string) {
	s.canonicalURL = canonicalRuntimeURL(raw)
}

func canonicalRuntimeURL(raw string) string {
	if value := strings.TrimSpace(raw); value != "" {
		return strings.TrimRight(value, "/")
	}
	if value := strings.TrimSpace(os.Getenv("GENCODE_RUNTIME_BASE_URL")); value != "" {
		return strings.TrimRight(value, "/")
	}
	return "http://127.0.0.1:10008"
}

// Status returns the app-server contract view of the current runtime summary.
func (s *Service) Status(context.Context) (runtimecontract.Status, error) {
	snapshot := s.Snapshot()
	return runtimecontract.Status{
		State:               snapshot.AppServerStatus,
		Ready:               snapshot.AppServerStatus == "running",
		Message:             snapshot.GoBridgeStatus,
		RuntimeSource:       snapshot.RuntimeSource,
		RuntimeSourceDetail: snapshot.RuntimeSourceDetail,
		RuntimeTrust:        snapshot.RuntimeTrust,
		CanonicalRuntimeURL: snapshot.CanonicalRuntimeURL,
		StateStore:          snapshot.StateStore,
		StatePath:           snapshot.StatePath,
		WorkspaceID:         snapshot.WorkspaceID,
		ProjectRoot:         snapshot.ProjectRoot,
		ThreadCount:         snapshot.ThreadCount,
		ActiveThreadID:      snapshot.ActiveThreadID,
		TaskCount:           snapshot.ActiveThreadTaskCount,
		EventCount:          snapshot.ActiveThreadEventCount,
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
		result = append(result, toTaskDescriptor(item))
	}
	return result, nil
}

// CreateTask registers a task under the given thread.
func (s *Service) CreateTask(_ context.Context, threadID string, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	request = normalizeCreateTaskRequest(request)
	thread, ok := s.session.Thread(threadID)
	if !ok {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
	}

	if request.Kind == runner.KindWorkspaceApplyPatch {
		return s.createPatchTask(thread, request)
	}
	if request.Kind == runner.KindWorkspaceApplyPatchRollback {
		return s.createRollbackTask(thread, request)
	}
	if request.Kind == runner.KindAgentRun {
		return s.createAgentTask(thread, request)
	}

	item, ok := s.session.CreateTask(threadID, session.CreateTaskInput{
		Title: request.Title,
		Kind:  request.Kind,
		Input: request.Input,
	})
	if !ok {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
	}
	return toTaskDescriptor(item), nil
}

func normalizeCreateTaskRequest(request runtimecontract.CreateTaskRequest) runtimecontract.CreateTaskRequest {
	request.Title = strings.TrimSpace(request.Title)
	request.Kind = strings.TrimSpace(request.Kind)
	request.Input = strings.TrimSpace(request.Input)
	if request.Kind != runner.KindModelResponse {
		if request.Kind != runner.KindAgentRun {
			return request
		}
		if request.Input == "" {
			request.Input = `{"goal":""}`
			return request
		}
		if strings.HasPrefix(request.Input, "{") && strings.HasSuffix(request.Input, "}") {
			var payload struct {
				Goal            string `json:"goal"`
				Provider        string `json:"provider"`
				Model           string `json:"model"`
				MaxSteps        int    `json:"maxSteps"`
				MaxOutputTokens int    `json:"maxOutputTokens"`
			}
			if err := json.Unmarshal([]byte(request.Input), &payload); err == nil {
				if strings.TrimSpace(payload.Goal) == "" {
					payload.Goal = request.Input
				}
				normalized, marshalErr := json.Marshal(payload)
				if marshalErr == nil {
					request.Input = string(normalized)
					return request
				}
			}
		}
		normalized, err := json.Marshal(map[string]string{
			"goal": request.Input,
		})
		if err == nil {
			request.Input = string(normalized)
		}
		return request
	}
	if request.Input == "" {
		request.Input = `{"input":""}`
		return request
	}
	if strings.HasPrefix(request.Input, "{") && strings.HasSuffix(request.Input, "}") {
		var payload struct {
			Provider        string `json:"provider"`
			Model           string `json:"model"`
			Input           string `json:"input"`
			MaxOutputTokens int    `json:"maxOutputTokens"`
		}
		if err := json.Unmarshal([]byte(request.Input), &payload); err == nil {
			if strings.TrimSpace(payload.Input) == "" {
				payload.Input = request.Input
			}
			normalized, marshalErr := json.Marshal(payload)
			if marshalErr == nil {
				request.Input = string(normalized)
				return request
			}
		}
	}
	normalized, err := json.Marshal(map[string]string{
		"input": request.Input,
	})
	if err == nil {
		request.Input = string(normalized)
	}
	return request
}

// RunTask executes a queued task under the given thread.
func (s *Service) RunTask(ctx context.Context, threadID string, taskID string, _ runtimecontract.RunTaskRequest) (runtimecontract.TaskDescriptor, error) {
	item, err := s.runner.RunTask(ctx, threadID, taskID)
	if err != nil {
		switch {
		case errors.Is(err, session.ErrThreadNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		case errors.Is(err, session.ErrTaskNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1007, "task not found")
		case errors.Is(err, session.ErrTaskAlreadyRunning):
			return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1011, "thread already has a running task")
		case errors.Is(err, runner.ErrApprovalRequired):
			return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1014, "task approval required")
		default:
			return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1012, err.Error())
		}
	}
	return toTaskDescriptor(item), nil
}

// Approvals returns the approval descriptors under the given thread.
func (s *Service) Approvals(_ context.Context, threadID string) ([]runtimecontract.ApprovalDescriptor, error) {
	items, ok := s.session.Approvals(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}
	result := make([]runtimecontract.ApprovalDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.ApprovalDescriptor{
			ID:          item.ID,
			ThreadID:    item.ThreadID,
			TaskID:      item.TaskID,
			ToolKind:    item.ToolKind,
			Status:      item.Status,
			Summary:     item.Summary,
			TargetPaths: append([]string(nil), item.TargetPaths...),
			CreatedAt:   item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// WriteExecutions returns the persisted write execution audit records under the given thread.
func (s *Service) WriteExecutions(_ context.Context, threadID string) ([]runtimecontract.WriteExecutionDescriptor, error) {
	items, ok := s.session.WriteExecutions(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}
	result := make([]runtimecontract.WriteExecutionDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.WriteExecutionDescriptor{
			ID:                 item.ID,
			ThreadID:           item.ThreadID,
			TaskID:             item.TaskID,
			ApprovalID:         item.ApprovalID,
			ToolKind:           item.ToolKind,
			Operation:          item.Operation,
			RelatedExecutionID: item.RelatedExecutionID,
			Status:             item.Status,
			TargetPaths:        append([]string(nil), item.TargetPaths...),
			PatchSummary:       item.PatchSummary,
			BeforeSummary:      item.BeforeSnapshotSummary,
			AfterSummary:       item.AfterSnapshotSummary,
			ResultSummary:      item.ResultSummary,
			CreatedAt:          item.CreatedAt.Format(time.RFC3339),
			UpdatedAt:          item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// ApproveTask approves and executes a pending task.
func (s *Service) ApproveTask(ctx context.Context, threadID string, taskID string, _ runtimecontract.ApproveTaskRequest) (runtimecontract.TaskDescriptor, error) {
	item, err := s.runner.ApproveTask(ctx, threadID, taskID)
	if err != nil {
		switch {
		case errors.Is(err, session.ErrThreadNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		case errors.Is(err, session.ErrTaskNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1007, "task not found")
		case errors.Is(err, session.ErrApprovalNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1015, "approval not found")
		default:
			return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1016, err.Error())
		}
	}
	return toTaskDescriptor(item), nil
}

// RejectTask rejects a pending task without executing it.
func (s *Service) RejectTask(_ context.Context, threadID string, taskID string, _ runtimecontract.RejectTaskRequest) (runtimecontract.TaskDescriptor, error) {
	item, err := s.runner.RejectTask(threadID, taskID)
	if err != nil {
		switch {
		case errors.Is(err, session.ErrThreadNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		case errors.Is(err, session.ErrTaskNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1007, "task not found")
		case errors.Is(err, session.ErrApprovalNotFound):
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1015, "approval not found")
		default:
			return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1017, err.Error())
		}
	}
	return toTaskDescriptor(item), nil
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
		ID:             item.ID,
		ThreadID:       item.ThreadID,
		Title:          item.Title,
		Status:         item.Status,
		Kind:           item.Kind,
		InputSummary:   item.Input,
		ResultSummary:  item.ResultSummary,
		ApprovalStatus: item.ApprovalStatus,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
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
func (s *Service) StreamEvents(ctx context.Context, threadID string, request runtimecontract.StreamEventsRequest) (<-chan runtimecontract.EventDescriptor, error) {
	items, ok := s.session.Events(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}
	subscribeCh, cancel, err := s.session.SubscribeEvents(threadID)
	if err != nil {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	out := make(chan runtimecontract.EventDescriptor, 256)
	go func() {
		defer close(out)
		defer cancel()

		replayItems := filterReplay(items, request)
		for _, item := range replayItems {
			select {
			case <-ctx.Done():
				return
			case out <- toEventDescriptor(item):
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-subscribeCh:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- toEventDescriptor(item):
				}
			}
		}
	}()

	return out, nil
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
			Kind:        item.Kind,
			ReadOnly:    item.ReadOnly,
			Executable:  item.Executable,
		})
	}
	return result, nil
}

// Providers returns the configured runtime model providers.
func (s *Service) Providers(context.Context) ([]runtimecontract.Provider, error) {
	items := s.providers.List()
	result := make([]runtimecontract.Provider, 0, len(items))
	for _, item := range items {
		recommendedStyle, reason := recommendedAPIStyle(item)
		result = append(result, runtimecontract.Provider{
			Kind:              string(item.Kind),
			Enabled:           item.Enabled,
			BaseURL:           item.BaseURL,
			DefaultModel:      item.DefaultModel,
			HasAuthToken:      item.HasAuthToken,
			SupportsChat:      item.SupportsChat,
			SupportsResponses: recommendedStyle == "openai-responses",
			PreferredAPIStyle: recommendedStyle,
			Recommended:       recommendedStyle != "",
			RecommendedReason: reason,
		})
	}
	return result, nil
}

// ProbeProvider performs a lightweight connectivity probe for a configured provider.
func (s *Service) ProbeProvider(ctx context.Context, kind string) (runtimecontract.ProviderProbeResult, error) {
	cfg, ok := s.providers.Resolve(provider.Kind(kind))
	if !ok {
		return runtimecontract.ProviderProbeResult{}, xerror.NotFound(1013, "provider not found")
	}
	result, err := s.prober.Probe(ctx, cfg)
	if err != nil {
		return runtimecontract.ProviderProbeResult{}, xerror.Internal(2002, err.Error())
	}
	return runtimecontract.ProviderProbeResult{
		Kind:              string(result.Kind),
		Reachable:         result.Reachable,
		PreferredAPIStyle: result.PreferredAPIStyle,
		Message:           result.Message,
		Details:           result.Details,
	}, nil
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
		result = append(result, runtimecontract.MCPServer{
			ID:            item.ID,
			Source:        item.Source,
			Enabled:       item.Enabled,
			ToolCount:     item.ToolCount,
			ResourceCount: item.ResourceCount,
			Status:        normalizeMCPServerStatus(item),
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

func toTaskDescriptor(item session.Task) runtimecontract.TaskDescriptor {
	descriptor := runtimecontract.TaskDescriptor{
		ID:             item.ID,
		ThreadID:       item.ThreadID,
		Title:          item.Title,
		Status:         item.Status,
		Kind:           item.Kind,
		InputSummary:   item.Input,
		ResultSummary:  item.ResultSummary,
		ApprovalStatus: item.ApprovalStatus,
		ParentTaskID:   item.ParentTaskID,
		WaitingStatus:  item.WaitingStatus,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339),
	}
	if item.Kind == runner.KindAgentRun {
		if state, err := runner.ParseAgentRunStateForRuntime(item.AgentState); err == nil {
			descriptor.AgentStep = state.StepIndex
			descriptor.AgentMaxSteps = state.MaxSteps
			descriptor.LatestChildTaskID = state.WaitingChildTaskID
			descriptor.AgentPlanSummary = state.Plan.Summary
			descriptor.AgentPlanMode = state.Plan.Mode
			descriptor.AgentCurrentStepTitle = state.CurrentStepTitle
			descriptor.AgentLastReasoning = state.LastReasoning
		}
		descriptor.ResultSummary = summarizeAgentTaskDescriptor(descriptor)
	}
	return descriptor
}

func summarizeAgentTaskDescriptor(task runtimecontract.TaskDescriptor) string {
	if strings.TrimSpace(task.ResultSummary) != "" {
		return strings.TrimSpace(task.ResultSummary)
	}

	progress := func() string {
		if task.AgentStep > 0 && task.AgentMaxSteps > 0 {
			return fmt.Sprintf("agent step %d/%d", task.AgentStep, task.AgentMaxSteps)
		}
		if task.AgentMaxSteps > 0 {
			return fmt.Sprintf("agent step 0/%d", task.AgentMaxSteps)
		}
		return "agent"
	}

	switch task.WaitingStatus {
	case "waiting_for_approval":
		if task.LatestChildTaskID != "" {
			return fmt.Sprintf("%s: waiting for approval for child task %s", progress(), task.LatestChildTaskID)
		}
		return fmt.Sprintf("%s: waiting for approval", progress())
	case "waiting_for_task":
		if task.LatestChildTaskID != "" {
			return fmt.Sprintf("%s: waiting for child task %s", progress(), task.LatestChildTaskID)
		}
		return fmt.Sprintf("%s: waiting for child task", progress())
	}

	switch task.Status {
	case "queued":
		return "agent queued"
	case "running":
		return fmt.Sprintf("%s: running", progress())
	case "completed":
		return "agent completed"
	case "failed":
		return "agent failed"
	default:
		return ""
	}
}

// Messages returns the message descriptors under the given thread.
func (s *Service) Messages(_ context.Context, threadID string) ([]runtimecontract.MessageDescriptor, error) {
	items, ok := s.session.Messages(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	result := make([]runtimecontract.MessageDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.MessageDescriptor{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Role:      item.Role,
			Content:   item.Content,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// AppendMessage appends a message under the given thread.
func (s *Service) AppendMessage(_ context.Context, threadID string, request runtimecontract.CreateMessageRequest) (runtimecontract.MessageDescriptor, error) {
	item, err := s.session.AppendMessage(threadID, session.AppendMessageInput{
		Role:    request.Role,
		Content: request.Content,
	})
	if err != nil {
		if errors.Is(err, session.ErrThreadNotFound) {
			return runtimecontract.MessageDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		return runtimecontract.MessageDescriptor{}, xerror.Internal(2001, "failed to append message")
	}

	return runtimecontract.MessageDescriptor{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		Role:      item.Role,
		Content:   item.Content,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ToolCalls returns the tool call descriptors under the given thread.
func (s *Service) ToolCalls(_ context.Context, threadID string) ([]runtimecontract.ToolCallDescriptor, error) {
	items, ok := s.session.ToolCalls(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	result := make([]runtimecontract.ToolCallDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.ToolCallDescriptor{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			ToolID:    item.ToolID,
			Status:    item.Status,
			Summary:   item.Summary,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// AppendToolCall appends a tool call under the given thread.
func (s *Service) AppendToolCall(_ context.Context, threadID string, request runtimecontract.CreateToolCallRequest) (runtimecontract.ToolCallDescriptor, error) {
	item, err := s.session.AppendToolCall(threadID, session.AppendToolCallInput{
		ToolID:  request.ToolID,
		Status:  request.Status,
		Summary: request.Summary,
	})
	if err != nil {
		if errors.Is(err, session.ErrThreadNotFound) {
			return runtimecontract.ToolCallDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		return runtimecontract.ToolCallDescriptor{}, xerror.Internal(2001, "failed to append tool call")
	}

	return runtimecontract.ToolCallDescriptor{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		ToolID:    item.ToolID,
		Status:    item.Status,
		Summary:   item.Summary,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}, nil
}

// Artifacts returns the artifact descriptors under the given thread.
func (s *Service) Artifacts(_ context.Context, threadID string) ([]runtimecontract.ArtifactDescriptor, error) {
	items, ok := s.session.Artifacts(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	result := make([]runtimecontract.ArtifactDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.ArtifactDescriptor{
			ID:        item.ID,
			ThreadID:  item.ThreadID,
			Path:      item.Path,
			Kind:      item.Kind,
			CreatedAt: item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// AppendArtifact appends an artifact under the given thread.
func (s *Service) AppendArtifact(_ context.Context, threadID string, request runtimecontract.CreateArtifactRequest) (runtimecontract.ArtifactDescriptor, error) {
	item, err := s.session.AppendArtifact(threadID, session.AppendArtifactInput{
		Path: request.Path,
		Kind: request.Kind,
	})
	if err != nil {
		if errors.Is(err, session.ErrThreadNotFound) {
			return runtimecontract.ArtifactDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		return runtimecontract.ArtifactDescriptor{}, xerror.Internal(2001, "failed to append artifact")
	}

	return runtimecontract.ArtifactDescriptor{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		Path:      item.Path,
		Kind:      item.Kind,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}, nil
}

// RuntimeFlags returns the runtime flag descriptors under the given thread.
func (s *Service) RuntimeFlags(_ context.Context, threadID string) ([]runtimecontract.RuntimeFlagDescriptor, error) {
	items, ok := s.session.RuntimeFlags(threadID)
	if !ok {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	result := make([]runtimecontract.RuntimeFlagDescriptor, 0, len(items))
	for _, item := range items {
		result = append(result, runtimecontract.RuntimeFlagDescriptor{
			ThreadID:  item.ThreadID,
			Key:       item.Key,
			Value:     item.Value,
			UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// SetRuntimeFlag upserts a runtime flag under the given thread.
func (s *Service) SetRuntimeFlag(_ context.Context, threadID string, request runtimecontract.SetRuntimeFlagRequest) (runtimecontract.RuntimeFlagDescriptor, error) {
	item, err := s.session.SetRuntimeFlag(threadID, session.SetRuntimeFlagInput{
		Key:   request.Key,
		Value: request.Value,
	})
	if err != nil {
		if errors.Is(err, session.ErrThreadNotFound) {
			return runtimecontract.RuntimeFlagDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		return runtimecontract.RuntimeFlagDescriptor{}, xerror.Internal(2001, "failed to set runtime flag")
	}

	return runtimecontract.RuntimeFlagDescriptor{
		ThreadID:  item.ThreadID,
		Key:       item.Key,
		Value:     item.Value,
		UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
	}, nil
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

func normalizeMCPServerStatus(item mcp.ServerDescriptor) string {
	status := strings.ToLower(strings.TrimSpace(item.Status))
	switch status {
	case "enabled", "disabled", "degraded", "unreachable":
		return status
	}
	if !item.Enabled {
		return "disabled"
	}
	if item.ToolCount == 0 && item.ResourceCount == 0 {
		return "degraded"
	}
	return "enabled"
}

func toEventDescriptor(item session.Event) runtimecontract.EventDescriptor {
	return runtimecontract.EventDescriptor{
		ID:        item.ID,
		ThreadID:  item.ThreadID,
		Type:      item.Type,
		Message:   item.Message,
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}
}

func limitReplay(items []session.Event, limit int) []session.Event {
	if limit <= 0 || len(items) <= limit {
		return append([]session.Event(nil), items...)
	}
	return append([]session.Event(nil), items[len(items)-limit:]...)
}

func filterReplay(items []session.Event, request runtimecontract.StreamEventsRequest) []session.Event {
	filtered := make([]session.Event, 0, len(items))
	sinceIDFound := request.SinceID == ""
	var sinceTime time.Time
	if request.SinceTime != "" {
		if parsed, err := time.Parse(time.RFC3339, request.SinceTime); err == nil {
			sinceTime = parsed
		}
	}

	for _, item := range items {
		if !sinceIDFound {
			if item.ID == request.SinceID {
				sinceIDFound = true
			}
			continue
		}
		if !sinceTime.IsZero() && item.CreatedAt.Before(sinceTime) {
			continue
		}
		filtered = append(filtered, item)
	}

	limit := request.Limit
	if limit <= 0 {
		limit = 200
	}
	if len(filtered) <= limit {
		return append([]session.Event(nil), filtered...)
	}
	return append([]session.Event(nil), filtered[len(filtered)-limit:]...)
}

func recommendedAPIStyle(item provider.Descriptor) (string, string) {
	baseURL := strings.ToLower(item.BaseURL)
	model := strings.ToLower(item.DefaultModel)

	if item.Kind == provider.Anthropic && strings.Contains(model, "gpt-") {
		return "openai-responses", "configured as anthropic, but the selected model family is better matched by OpenAI Responses"
	}
	if item.Kind == provider.OpenAI {
		return "openai-responses", "OpenAI-compatible providers should prefer the Responses API"
	}
	if item.Kind == provider.Anthropic {
		return "anthropic", "Anthropic-compatible providers can start from the messages API"
	}
	if item.Kind == provider.Gemini && baseURL != "" {
		return "gemini", "Gemini providers should use the Gemini-native API contract"
	}
	return "", ""
}

func (s *Service) createPatchTask(thread session.Thread, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	_, patch, err := runner.ParsePatchInput(request.Input)
	if err != nil {
		return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1005, err.Error())
	}
	targets, err := runner.ExtractPatchTargets(patch)
	if err != nil {
		return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1005, err.Error())
	}
	summary := fmt.Sprintf("%s; %s", runner.ApprovalSummary(request.Kind, targets), runner.TruncatedPatchSummary(patch, 120))

	switch thread.PermissionMode {
	case policy.ReadOnly:
		item, ok := s.session.CreateTask(thread.ID, session.CreateTaskInput{
			Title:          request.Title,
			Kind:           request.Kind,
			Input:          request.Input,
			Status:         "failed",
			ResultSummary:  "permission denied: read-only mode does not allow workspace writes",
			ApprovalStatus: "rejected",
		})
		if !ok {
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		_ = s.session.AppendRuntimeEvent(thread.ID, "task.failed", item.ResultSummary)
		return toTaskDescriptor(item), nil
	case policy.AskUser, "":
		item, ok := s.session.CreateTask(thread.ID, session.CreateTaskInput{
			Title:          request.Title,
			Kind:           request.Kind,
			Input:          request.Input,
			Status:         "needs_approval",
			ResultSummary:  summary,
			ApprovalStatus: "pending",
		})
		if !ok {
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		if _, err := s.session.CreateApproval(thread.ID, session.CreateApprovalInput{
			TaskID:      item.ID,
			ToolKind:    request.Kind,
			Status:      "pending",
			Summary:     summary,
			TargetPaths: append([]string(nil), targets...),
		}); err != nil {
			return runtimecontract.TaskDescriptor{}, xerror.Internal(2001, err.Error())
		}
		_ = s.session.AppendRuntimeEvent(thread.ID, "task.approval_required", summary)
		return toTaskDescriptor(item), nil
	default:
		item, ok := s.session.CreateTask(thread.ID, session.CreateTaskInput{
			Title:          request.Title,
			Kind:           request.Kind,
			Input:          request.Input,
			Status:         "queued",
			ResultSummary:  "",
			ApprovalStatus: "direct",
		})
		if !ok {
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		return toTaskDescriptor(item), nil
	}
}

func (s *Service) createRollbackTask(thread session.Thread, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	writeExecutionID, err := runner.ParseRollbackInput(request.Input)
	if err != nil {
		return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1005, err.Error())
	}
	executions, ok := s.session.WriteExecutions(thread.ID)
	if !ok {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
	}
	var source *session.WriteExecutionRecord
	for index := range executions {
		if executions[index].ID == writeExecutionID {
			source = &executions[index]
			break
		}
	}
	if source == nil {
		return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1005, "write execution not found")
	}
	summary := runner.RollbackApprovalSummary(source.TargetPaths)

	switch thread.PermissionMode {
	case policy.ReadOnly:
		item, ok := s.session.CreateTask(thread.ID, session.CreateTaskInput{
			Title:          request.Title,
			Kind:           request.Kind,
			Input:          request.Input,
			Status:         "failed",
			ResultSummary:  "rollback failed: permission denied: read-only mode does not allow workspace writes",
			ApprovalStatus: "rejected",
		})
		if !ok {
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		_ = s.session.AppendRuntimeEvent(thread.ID, "task.failed", item.ResultSummary)
		return toTaskDescriptor(item), nil
	case "", policy.AskUser:
		item, ok := s.session.CreateTask(thread.ID, session.CreateTaskInput{
			Title:          request.Title,
			Kind:           request.Kind,
			Input:          request.Input,
			Status:         "needs_approval",
			ResultSummary:  summary,
			ApprovalStatus: "pending",
		})
		if !ok {
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		if _, err := s.session.CreateApproval(thread.ID, session.CreateApprovalInput{
			TaskID:      item.ID,
			ToolKind:    request.Kind,
			Status:      "pending",
			Summary:     summary,
			TargetPaths: append([]string(nil), source.TargetPaths...),
		}); err != nil {
			return runtimecontract.TaskDescriptor{}, xerror.Internal(2001, err.Error())
		}
		_ = s.session.AppendRuntimeEvent(thread.ID, "task.rollback_required", summary)
		return toTaskDescriptor(item), nil
	default:
		item, ok := s.session.CreateTask(thread.ID, session.CreateTaskInput{
			Title:          request.Title,
			Kind:           request.Kind,
			Input:          request.Input,
			Status:         "queued",
			ResultSummary:  "",
			ApprovalStatus: "direct",
		})
		if !ok {
			return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
		}
		return toTaskDescriptor(item), nil
	}
}

func (s *Service) createAgentTask(thread session.Thread, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	if strings.TrimSpace(request.Title) == "" {
		request.Title = "Agent run"
	}
	input, err := runner.ParseAgentRunInputForRuntime(request.Input)
	if err != nil {
		return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1005, err.Error())
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return runtimecontract.TaskDescriptor{}, xerror.Internal(2001, err.Error())
	}
	item, ok := s.session.CreateTask(thread.ID, session.CreateTaskInput{
		Title: request.Title,
		Kind:  request.Kind,
		Input: string(raw),
		AgentState: runner.MarshalAgentRunStateForRuntime(runner.AgentRunState{
			ThreadID:         thread.ID,
			StepIndex:        0,
			MaxSteps:         input.MaxSteps,
			Status:           "queued",
			Goal:             input.Goal,
			Provider:         input.Provider,
			Model:            input.Model,
			MaxOutputTokens:  input.MaxOutputTokens,
			Plan:             runner.DeriveAgentExecutionPlanForRuntime(input.Goal),
			CurrentStepTitle: runner.CurrentAgentStepTitleForRuntime(runner.DeriveAgentExecutionPlanForRuntime(input.Goal), nil),
		}),
	})
	if !ok {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
	}
	return toTaskDescriptor(item), nil
}
