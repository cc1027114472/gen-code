package runner

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"llmtrace/internal/core/mcp"
	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/provider"
	"llmtrace/internal/core/session"
)

const (
	KindMessageAppend               = "thread.message.append"
	KindToolCallAppend              = "thread.toolcall.append"
	KindArtifactAppend              = "thread.artifact.append"
	KindRuntimeFlagSet              = "thread.runtimeflag.set"
	KindWorkspaceRead               = "workspace.read_file"
	KindWorkspaceList               = "workspace.list_files"
	KindWorkspaceSearch             = "workspace.search_text"
	KindWorkspaceStat               = "workspace.stat_file"
	KindWorkspaceReadBatch          = "workspace.read_files_batch"
	KindWorkspaceListFiltered       = "workspace.list_files_filtered"
	KindWorkspaceSearchDetailed     = "workspace.search_text_detailed"
	KindWorkspaceApplyPatch         = "workspace.apply_patch"
	KindWorkspaceApplyPatchRollback = "workspace.apply_patch.rollback"
	KindMCPToolInvoke               = "mcp.tool.invoke"
	KindModelResponse               = "model.response.create"
	KindAgentRun                    = "agent.run"
	restartFailureLabel             = "interrupted by runtime restart"
	recoveryGracePeriod             = 5 * time.Second
)

var recoveryNow = time.Now

var (
	ErrUnsupportedTaskKind  = errors.New("unsupported task kind")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrPathOutsideWorkspace = errors.New("path outside workspace")
	ErrApprovalRequired     = errors.New("approval required")
)

type Registry interface {
	Task(threadID string, taskID string) (session.Task, error)
	Tasks(threadID string) ([]session.Task, bool)
	Thread(threadID string) (session.Thread, bool)
	Workspace() session.Workspace
	CreateTask(threadID string, input session.CreateTaskInput) (session.Task, bool)
	UpdateTaskStatus(threadID string, taskID string, input session.UpdateTaskStatusInput) (session.Task, error)
	Approvals(threadID string) ([]session.ApprovalRecord, bool)
	ApprovalByTask(threadID string, taskID string) (session.ApprovalRecord, error)
	CreateApproval(threadID string, input session.CreateApprovalInput) (session.ApprovalRecord, error)
	UpdateApproval(threadID string, taskID string, input session.UpdateApprovalInput) (session.ApprovalRecord, error)
	CreateWriteExecution(threadID string, input session.CreateWriteExecutionInput) (session.WriteExecutionRecord, error)
	WriteExecutions(threadID string) ([]session.WriteExecutionRecord, bool)
	Messages(threadID string) ([]session.MessageRecord, bool)
	AppendMessage(threadID string, input session.AppendMessageInput) (session.MessageRecord, error)
	AppendToolCall(threadID string, input session.AppendToolCallInput) (session.ToolCallRecord, error)
	AppendArtifact(threadID string, input session.AppendArtifactInput) (session.ArtifactRecord, error)
	SetRuntimeFlag(threadID string, input session.SetRuntimeFlagInput) (session.RuntimeFlagRecord, error)
}

type Runner struct {
	registry Registry
	models   ModelExecutor
	mcp      *mcp.Manager
}

type ModelExecutor interface {
	CreateResponse(ctx context.Context, request provider.ResponseRequest) (provider.ResponseResult, error)
}

func New(registry Registry, models ModelExecutor) *Runner {
	return &Runner{registry: registry, models: models}
}

func (r *Runner) WithMCP(manager *mcp.Manager) *Runner {
	r.mcp = manager
	return r
}

func (r *Runner) RecoverInterruptedTasks() error {
	if r == nil || r.registry == nil {
		return nil
	}
	type sortedIDs interface {
		SortedIDs() []string
	}
	idSource, ok := r.registry.(sortedIDs)
	if !ok {
		return nil
	}
	for _, threadID := range idSource.SortedIDs() {
		tasks, ok := r.registry.Tasks(threadID)
		if !ok {
			continue
		}
		for _, task := range tasks {
			if task.Kind == KindAgentRun {
				handled, err := r.recoverInterruptedAgentTask(threadID, task)
				if err != nil {
					return err
				}
				if handled {
					continue
				}
			}
			if task.Status != "running" {
				continue
			}
			if !shouldRecoverRunningTask(task) {
				continue
			}
			if err := r.markRecoveredAsFailed(threadID, task, restartFailureLabel); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *Runner) recoverInterruptedAgentTask(threadID string, task session.Task) (bool, error) {
	if task.Kind != KindAgentRun {
		return false, nil
	}
	state, err := parseAgentRunState(task.AgentState)
	if err != nil {
		if task.Status == "running" {
			return true, r.markRecoveredAsFailed(threadID, task, restartFailureLabel)
		}
		return false, nil
	}
	childID := strings.TrimSpace(state.WaitingChildTaskID)
	switch task.Status {
	case "waiting_for_approval":
		if childID == "" {
			return true, r.markRecoveredAsFailed(threadID, task, "agent recovery failed: waiting_for_approval without child task")
		}
		child, childErr := r.registry.Task(threadID, childID)
		if childErr != nil {
			return true, r.markRecoveredAsFailed(threadID, task, "agent recovery failed: approval child task not found")
		}
		if child.Status == "needs_approval" {
			return true, nil
		}
		if child.Status == "completed" {
			_, resumeErr := r.resumeAgentRun(context.Background(), threadID, task)
			return true, resumeErr
		}
		return true, r.markRecoveredAsFailed(threadID, task, fmt.Sprintf("agent recovery failed: approval child task is %s", child.Status))
	case "waiting_for_task":
		if childID == "" {
			return true, r.markRecoveredAsFailed(threadID, task, "agent recovery failed: waiting_for_task without child task")
		}
		child, childErr := r.registry.Task(threadID, childID)
		if childErr != nil {
			return true, r.markRecoveredAsFailed(threadID, task, "agent recovery failed: child task not found")
		}
		if child.Status == "completed" {
			_, resumeErr := r.resumeAgentRun(context.Background(), threadID, task)
			return true, resumeErr
		}
		return true, r.markRecoveredAsFailed(threadID, task, fmt.Sprintf("agent recovery failed: child task is %s", child.Status))
	case "running":
		if !shouldRecoverRunningTask(task) {
			return true, nil
		}
		return true, r.markRecoveredAsFailed(threadID, task, restartFailureLabel)
	default:
		return false, nil
	}
}

func shouldRecoverRunningTask(task session.Task) bool {
	if task.UpdatedAt.IsZero() {
		return true
	}
	return recoveryNow().Sub(task.UpdatedAt) >= recoveryGracePeriod
}

func (r *Runner) markRecoveredAsFailed(threadID string, task session.Task, summary string) error {
	if _, err := r.registry.UpdateTaskStatus(threadID, task.ID, session.UpdateTaskStatusInput{
		Status:        "failed",
		ResultSummary: summary,
		WaitingStatus: strPtr(""),
	}); err != nil {
		return err
	}
	if recorder, ok := r.registry.(interface {
		AppendRuntimeEvent(threadID string, eventType string, message string) error
	}); ok {
		if err := recorder.AppendRuntimeEvent(threadID, "task.recovered_as_failed", fmt.Sprintf("Task %s was recovered as failed after runtime restart", task.ID)); err != nil {
			return err
		}
	}
	if _, err := r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
		ToolID:  "task.recovery",
		Status:  "failed",
		Summary: fmt.Sprintf("Task %s was marked failed after runtime restart", task.ID),
	}); err != nil {
		return err
	}
	return nil
}

func (r *Runner) RunTask(ctx context.Context, threadID string, taskID string) (session.Task, error) {
	task, err := r.registry.Task(threadID, taskID)
	if err != nil {
		return session.Task{}, err
	}
	if task.Kind == KindAgentRun {
		return r.runAgentTask(ctx, threadID, task)
	}
	if task.Status == "needs_approval" {
		return session.Task{}, ErrApprovalRequired
	}
	recorder, _ := r.registry.(interface {
		AppendRuntimeEvent(threadID string, eventType string, message string) error
	})

	if _, err := r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
		Status:        "running",
		ResultSummary: "",
	}); err != nil {
		return session.Task{}, err
	}
	if recorder != nil {
		_ = recorder.AppendRuntimeEvent(threadID, "task.started", fmt.Sprintf("Task %s started", task.Title))
	}

	toolCall, err := r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
		ToolID:  task.Kind,
		Status:  "running",
		Summary: "task execution started",
	})
	if err != nil {
		_, _ = r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
			Status:        "failed",
			ResultSummary: err.Error(),
		})
		return session.Task{}, err
	}
	if recorder != nil {
		_ = recorder.AppendRuntimeEvent(threadID, "toolcall.started", fmt.Sprintf("Tool call %s started", task.Kind))
	}

	summary, execErr := r.execute(ctx, threadID, task)
	if execErr != nil {
		if recorder != nil {
			_ = recorder.AppendRuntimeEvent(threadID, "toolcall.failed", execErr.Error())
		}
		_, _ = r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
			ToolID:  task.Kind,
			Status:  "failed",
			Summary: execErr.Error(),
		})
		failed, updateErr := r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
			Status:        "failed",
			ResultSummary: execErr.Error(),
		})
		if updateErr != nil {
			return session.Task{}, updateErr
		}
		if recorder != nil {
			_ = recorder.AppendRuntimeEvent(threadID, "task.failed", execErr.Error())
		}
		_ = toolCall
		return failed, nil
	}

	_, err = r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
		ToolID:  task.Kind,
		Status:  "completed",
		Summary: summary,
	})
	if err != nil {
		return session.Task{}, err
	}
	if recorder != nil {
		_ = recorder.AppendRuntimeEvent(threadID, "toolcall.completed", summary)
	}

	result, err := r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
		Status:        "completed",
		ResultSummary: summary,
	})
	if err != nil {
		return session.Task{}, err
	}
	if recorder != nil {
		_ = recorder.AppendRuntimeEvent(threadID, "task.completed", summary)
	}
	return result, nil
}

func (r *Runner) runAgentTask(ctx context.Context, threadID string, task session.Task) (session.Task, error) {
	recorder, _ := r.registry.(interface {
		AppendRuntimeEvent(threadID string, eventType string, message string) error
	})
	if task.Status == "needs_approval" {
		return session.Task{}, ErrApprovalRequired
	}
	if _, err := r.registry.UpdateTaskStatus(threadID, task.ID, session.UpdateTaskStatusInput{
		Status:        "running",
		ResultSummary: task.ResultSummary,
		WaitingStatus: strPtr(""),
	}); err != nil {
		return session.Task{}, err
	}
	if recorder != nil {
		_ = recorder.AppendRuntimeEvent(threadID, "task.started", fmt.Sprintf("Task %s started", task.Title))
	}
	_, _ = r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
		ToolID:  task.Kind,
		Status:  "running",
		Summary: "agent loop started",
	})

	thread, ok := r.registry.Thread(threadID)
	if !ok {
		return session.Task{}, session.ErrThreadNotFound
	}
	parent, err := r.registry.Task(threadID, task.ID)
	if err != nil {
		return session.Task{}, err
	}
	summary, execErr := r.executeAgentRun(ctx, thread, parent)
	if execErr != nil {
		return r.failAgentParent(threadID, parent, execErr)
	}
	parent, err = r.registry.Task(threadID, task.ID)
	if err != nil {
		return session.Task{}, err
	}
	if parent.Status == "waiting_for_approval" || parent.Status == "waiting_for_task" {
		if recorder != nil {
			_ = recorder.AppendRuntimeEvent(threadID, "toolcall.completed", summary)
		}
		_, _ = r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
			ToolID:  task.Kind,
			Status:  "completed",
			Summary: summary,
		})
		return parent, nil
	}
	return r.completeAgentParent(threadID, parent, summary)
}

func (r *Runner) ApproveTask(ctx context.Context, threadID string, taskID string) (session.Task, error) {
	approval, err := r.registry.ApprovalByTask(threadID, taskID)
	if err != nil {
		return session.Task{}, err
	}
	if approval.Status == "rejected" {
		return session.Task{}, fmt.Errorf("approval already rejected")
	}

	approved := "approved"
	if _, err := r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
		Status:         "queued",
		ResultSummary:  approval.Summary,
		ApprovalStatus: &approved,
	}); err != nil {
		return session.Task{}, err
	}
	if _, err := r.registry.UpdateApproval(threadID, taskID, session.UpdateApprovalInput{
		Status:  "approved",
		Summary: approval.Summary,
	}); err != nil {
		return session.Task{}, err
	}
	if recorder, ok := r.registry.(interface {
		AppendRuntimeEvent(threadID string, eventType string, message string) error
	}); ok {
		_ = recorder.AppendRuntimeEvent(threadID, "task.approved", approval.Summary)
		_ = recorder.AppendRuntimeEvent(threadID, "toolcall.approved", approval.Summary)
	}

	result, err := r.RunTask(ctx, threadID, taskID)
	if err != nil {
		return session.Task{}, err
	}
	if result.Status == "completed" {
		_, updateErr := r.registry.UpdateApproval(threadID, taskID, session.UpdateApprovalInput{
			Status:  "executed",
			Summary: result.ResultSummary,
		})
		if updateErr == nil {
			executed := "executed"
			result, _ = r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
				Status:         result.Status,
				ResultSummary:  result.ResultSummary,
				ApprovalStatus: &executed,
			})
		}
	}
	parent, parentErr := r.parentTask(threadID, taskID)
	if parentErr == nil && parent.Kind == KindAgentRun {
		return r.resumeAgentRun(ctx, threadID, parent)
	}
	return result, nil
}

func (r *Runner) RejectTask(threadID string, taskID string) (session.Task, error) {
	approval, err := r.registry.ApprovalByTask(threadID, taskID)
	if err != nil {
		return session.Task{}, err
	}
	rejectedSummary := approval.Summary
	if rejectedSummary == "" {
		rejectedSummary = "approval rejected"
	}
	rejectedSummary = "approval rejected: " + rejectedSummary
	rejected := "rejected"
	task, err := r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
		Status:         "failed",
		ResultSummary:  rejectedSummary,
		ApprovalStatus: &rejected,
	})
	if err != nil {
		return session.Task{}, err
	}
	if _, err := r.registry.UpdateApproval(threadID, taskID, session.UpdateApprovalInput{
		Status:  "rejected",
		Summary: rejectedSummary,
	}); err != nil {
		return session.Task{}, err
	}
	if recorder, ok := r.registry.(interface {
		AppendRuntimeEvent(threadID string, eventType string, message string) error
	}); ok {
		_ = recorder.AppendRuntimeEvent(threadID, "task.rejected", rejectedSummary)
		_ = recorder.AppendRuntimeEvent(threadID, "toolcall.rejected", rejectedSummary)
	}
	parent, parentErr := r.parentTask(threadID, taskID)
	if parentErr == nil && parent.Kind == KindAgentRun {
		parentFailedSummary := fmt.Sprintf("agent failed: child approval rejected: %s", rejectedSummary)
		failedParent, updateErr := r.registry.UpdateTaskStatus(threadID, parent.ID, session.UpdateTaskStatusInput{
			Status:        "failed",
			ResultSummary: parentFailedSummary,
			WaitingStatus: strPtr(""),
		})
		if updateErr != nil {
			return session.Task{}, updateErr
		}
		_, _ = r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
			ToolID:  parent.Kind,
			Status:  "failed",
			Summary: parentFailedSummary,
		})
		if recorder, ok := r.registry.(interface {
			AppendRuntimeEvent(threadID string, eventType string, message string) error
		}); ok {
			_ = recorder.AppendRuntimeEvent(threadID, "task.failed", parentFailedSummary)
			_ = recorder.AppendRuntimeEvent(threadID, "toolcall.failed", parentFailedSummary)
		}
		return failedParent, nil
	}
	return task, nil
}

func (r *Runner) execute(ctx context.Context, threadID string, task session.Task) (string, error) {
	thread, ok := r.registry.Thread(threadID)
	if !ok {
		return "", session.ErrThreadNotFound
	}

	switch task.Kind {
	case KindAgentRun:
		return r.executeAgentRun(ctx, thread, task)
	case KindModelResponse:
		if r.models == nil {
			return "", fmt.Errorf("provider error: model execution is not configured")
		}
		var input struct {
			Provider        string `json:"provider"`
			Model           string `json:"model"`
			Input           string `json:"input"`
			MaxOutputTokens int    `json:"maxOutputTokens"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		result, err := r.models.CreateResponse(ctx, provider.ResponseRequest{
			Provider:        input.Provider,
			Model:           input.Model,
			Input:           input.Input,
			MaxOutputTokens: input.MaxOutputTokens,
		})
		if err != nil {
			return "", fmt.Errorf("provider error: %w", err)
		}
		if _, err := r.registry.AppendMessage(threadID, session.AppendMessageInput{
			Role:    "assistant",
			Content: result.OutputText,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("response from %s: %s", fallbackModel(result.Model, input.Model), compactSummary(result.OutputText, 240)), nil
	case KindMessageAppend:
		if err := ensureThreadMutationAllowed(thread.PermissionMode); err != nil {
			return "", err
		}
		var input struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if _, err := r.registry.AppendMessage(threadID, session.AppendMessageInput{
			Role:    input.Role,
			Content: input.Content,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("message appended for role %s", input.Role), nil
	case KindWorkspaceRead:
		if err := ensureReadAllowed(thread.PermissionMode); err != nil {
			return "", err
		}
		var input struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		resolvedPath, err := resolveWorkspacePath(r.registry.Workspace().ProjectRoot, input.Path)
		if err != nil {
			return "", err
		}
		bytes, err := os.ReadFile(resolvedPath)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("read %s: %s", workspaceRelative(r.registry.Workspace().ProjectRoot, resolvedPath), compactSummary(string(bytes), 240)), nil
	case KindWorkspaceList:
		if err := ensureReadAllowed(thread.PermissionMode); err != nil {
			return "", err
		}
		var input struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		resolvedPath, err := resolveWorkspacePath(r.registry.Workspace().ProjectRoot, input.Path)
		if err != nil {
			return "", err
		}
		entries, err := os.ReadDir(resolvedPath)
		if err != nil {
			return "", err
		}
		names := make([]string, 0, len(entries))
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() {
				name += "/"
			}
			names = append(names, name)
		}
		sort.Strings(names)
		return fmt.Sprintf("listed %d entries in %s: %s", len(names), workspaceRelative(r.registry.Workspace().ProjectRoot, resolvedPath), compactSummary(strings.Join(names, ", "), 240)), nil
	case KindWorkspaceSearch:
		if err := ensureReadAllowed(thread.PermissionMode); err != nil {
			return "", err
		}
		var input struct {
			Query string `json:"query"`
			Path  string `json:"path"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if strings.TrimSpace(input.Query) == "" {
			return "", fmt.Errorf("query is required")
		}
		searchRoot, err := resolveWorkspacePath(r.registry.Workspace().ProjectRoot, input.Path)
		if err != nil {
			return "", err
		}
		matches, err := searchWorkspaceText(searchRoot, input.Query, r.registry.Workspace().ProjectRoot)
		if err != nil {
			return "", err
		}
		if len(matches) == 0 {
			return fmt.Sprintf("no matches for %q under %s", input.Query, workspaceRelative(r.registry.Workspace().ProjectRoot, searchRoot)), nil
		}
		return fmt.Sprintf("found %d matches for %q: %s", len(matches), input.Query, compactSummary(strings.Join(matches, " | "), 240)), nil
	case KindWorkspaceStat:
		if err := ensureReadAllowed(thread.PermissionMode); err != nil {
			return "", err
		}
		var input struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		resolvedPath, err := resolveWorkspacePath(r.registry.Workspace().ProjectRoot, input.Path)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(resolvedPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Sprintf("stat %s: missing", workspaceRelative(r.registry.Workspace().ProjectRoot, resolvedPath)), nil
			}
			return "", err
		}
		entryType := "file"
		if info.IsDir() {
			entryType = "dir"
		}
		return fmt.Sprintf("stat %s: %s, %d B, %s", workspaceRelative(r.registry.Workspace().ProjectRoot, resolvedPath), entryType, info.Size(), info.ModTime().UTC().Format(time.RFC3339)), nil
	case KindWorkspaceReadBatch:
		if err := ensureReadAllowed(thread.PermissionMode); err != nil {
			return "", err
		}
		var input struct {
			Paths []string `json:"paths"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if len(input.Paths) == 0 {
			return "", fmt.Errorf("paths is required")
		}
		paths := make([]string, 0, len(input.Paths))
		for _, candidate := range input.Paths {
			resolvedPath, err := resolveWorkspacePath(r.registry.Workspace().ProjectRoot, candidate)
			if err != nil {
				return "", err
			}
			info, err := os.Stat(resolvedPath)
			if err != nil {
				return "", err
			}
			if info.IsDir() {
				return "", fmt.Errorf("path %s is a directory", workspaceRelative(r.registry.Workspace().ProjectRoot, resolvedPath))
			}
			content, err := os.ReadFile(resolvedPath)
			if err != nil {
				return "", err
			}
			if !isLikelyText(content) {
				return "", fmt.Errorf("path %s is not a text file", workspaceRelative(r.registry.Workspace().ProjectRoot, resolvedPath))
			}
			paths = append(paths, workspaceRelative(r.registry.Workspace().ProjectRoot, resolvedPath))
		}
		return fmt.Sprintf("read %d files: %s", len(paths), compactSummary(strings.Join(paths, ", "), 240)), nil
	case KindWorkspaceListFiltered:
		if err := ensureReadAllowed(thread.PermissionMode); err != nil {
			return "", err
		}
		var input struct {
			Path        string `json:"path"`
			Pattern     string `json:"pattern"`
			IncludeDirs bool   `json:"includeDirs"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if strings.TrimSpace(input.Pattern) == "" {
			return "", fmt.Errorf("pattern is required")
		}
		searchRoot, err := resolveWorkspacePath(r.registry.Workspace().ProjectRoot, input.Path)
		if err != nil {
			return "", err
		}
		matches, err := listWorkspaceEntriesFiltered(searchRoot, r.registry.Workspace().ProjectRoot, input.Pattern, input.IncludeDirs)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("listed %d filtered entries in %s: %s", len(matches), workspaceRelative(r.registry.Workspace().ProjectRoot, searchRoot), compactSummary(strings.Join(matches, ", "), 240)), nil
	case KindWorkspaceSearchDetailed:
		if err := ensureReadAllowed(thread.PermissionMode); err != nil {
			return "", err
		}
		var input struct {
			Query string `json:"query"`
			Path  string `json:"path"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if strings.TrimSpace(input.Query) == "" {
			return "", fmt.Errorf("query is required")
		}
		limit := normalizedDetailedSearchLimit(input.Limit)
		searchRoot, err := resolveWorkspacePath(r.registry.Workspace().ProjectRoot, input.Path)
		if err != nil {
			return "", err
		}
		matches, err := searchWorkspaceTextDetailed(searchRoot, input.Query, r.registry.Workspace().ProjectRoot, limit)
		if err != nil {
			return "", err
		}
		if len(matches) == 0 {
			return fmt.Sprintf("no matches for %q under %s", input.Query, workspaceRelative(r.registry.Workspace().ProjectRoot, searchRoot)), nil
		}
		return fmt.Sprintf("found %d detailed matches for %q: %s", len(matches), input.Query, compactSummary(strings.Join(matches, " | "), 240)), nil
	case KindWorkspaceApplyPatch:
		approvedWrite := task.ApprovalStatus == "approved" || task.ApprovalStatus == "executed" || task.ApprovalStatus == "direct"
		if err := ensureWriteAllowed(thread.PermissionMode, approvedWrite); err != nil {
			return "", err
		}
		return r.executeWorkspaceApplyPatch(threadID, task)
	case KindWorkspaceApplyPatchRollback:
		approvedWrite := task.ApprovalStatus == "approved" || task.ApprovalStatus == "executed" || task.ApprovalStatus == "direct"
		if err := ensureWriteAllowed(thread.PermissionMode, approvedWrite); err != nil {
			return "", fmt.Errorf("rollback failed: %w", err)
		}
		return r.executeWorkspaceApplyPatchRollback(threadID, task)
	case KindToolCallAppend:
		var input struct {
			ToolID  string `json:"toolId"`
			Status  string `json:"status"`
			Summary string `json:"summary"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if _, err := r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
			ToolID:  input.ToolID,
			Status:  input.Status,
			Summary: input.Summary,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("tool call %s appended", input.ToolID), nil
	case KindArtifactAppend:
		var input struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if _, err := r.registry.AppendArtifact(threadID, session.AppendArtifactInput{
			Path: input.Path,
			Kind: input.Kind,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("artifact %s appended", input.Kind), nil
	case KindRuntimeFlagSet:
		var input struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if _, err := r.registry.SetRuntimeFlag(threadID, session.SetRuntimeFlagInput{
			Key:   input.Key,
			Value: input.Value,
		}); err != nil {
			return "", err
		}
		return fmt.Sprintf("runtime flag %s updated", input.Key), nil
	case KindMCPToolInvoke:
		var input struct {
			ServerID  string         `json:"serverId"`
			ToolName  string         `json:"toolName"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(task.Input), &input); err != nil {
			return "", err
		}
		if r.mcp == nil {
			return "", errors.New("mcp execution is not configured")
		}
		result, err := r.mcp.Invoke(ctx, mcp.InvokeRequest{
			ServerID:  input.ServerID,
			ToolName:  input.ToolName,
			Arguments: input.Arguments,
		})
		if err != nil {
			return "", err
		}
		return result.ResultSummary, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedTaskKind, task.Kind)
	}
}

func (r *Runner) parentTask(threadID string, taskID string) (session.Task, error) {
	task, err := r.registry.Task(threadID, taskID)
	if err != nil {
		return session.Task{}, err
	}
	if strings.TrimSpace(task.ParentTaskID) == "" {
		return session.Task{}, fmt.Errorf("parent task not found")
	}
	return r.registry.Task(threadID, task.ParentTaskID)
}

func (r *Runner) executeWorkspaceApplyPatch(threadID string, task session.Task) (string, error) {
	path, patch, err := ParsePatchInput(task.Input)
	if err != nil {
		return "", err
	}
	targets, err := ExtractPatchTargets(patch)
	if err != nil {
		return "", err
	}
	if len(targets) == 0 {
		targets = []string{path}
	}

	workspaceRoot := r.registry.Workspace().ProjectRoot
	resolvedPath, err := resolveWorkspacePath(workspaceRoot, path)
	if err != nil {
		return "", err
	}

	approvalID := ""
	if approval, approvalErr := r.registry.ApprovalByTask(threadID, task.ID); approvalErr == nil {
		approvalID = approval.ID
	}

	rollbackSnapshot, snapshotErr := captureRollbackSnapshot(resolvedPath, workspaceRelative(workspaceRoot, resolvedPath))
	if snapshotErr != nil {
		return "", snapshotErr
	}
	beforeSummary := snapshotFileSummary(resolvedPath)
	changed, execErr := applyWorkspacePatch(resolvedPath, patch)
	rollbackSnapshot.AfterExists, rollbackSnapshot.AfterHash = readFilePresenceAndHash(resolvedPath)
	afterSummary := snapshotFileSummary(resolvedPath)
	patchSummary := TruncatedPatchSummary(patch, 120)
	if execErr != nil {
		_, _ = r.recordWriteExecution(threadID, task, approvalID, "apply", "", "failed", targets, patch, patchSummary, beforeSummary, afterSummary, []session.WriteExecutionFileSnapshot{rollbackSnapshot}, execErr.Error())
		return "", execErr
	}

	resultSummary := PatchExecutionSummary(targets, changed)
	if _, err := r.recordWriteExecution(threadID, task, approvalID, "apply", "", "completed", targets, patch, patchSummary, beforeSummary, afterSummary, []session.WriteExecutionFileSnapshot{rollbackSnapshot}, resultSummary); err != nil {
		return "", err
	}
	return resultSummary, nil
}

func (r *Runner) executeWorkspaceApplyPatchRollback(threadID string, task session.Task) (string, error) {
	writeExecutionID, err := ParseRollbackInput(task.Input)
	if err != nil {
		return "", fmt.Errorf("rollback failed: %w", err)
	}

	executions, ok := r.registry.WriteExecutions(threadID)
	if !ok {
		return "", fmt.Errorf("rollback failed: %w", session.ErrThreadNotFound)
	}

	var source *session.WriteExecutionRecord
	for index := range executions {
		if executions[index].ID == writeExecutionID {
			source = &executions[index]
			break
		}
	}
	if source == nil {
		return "", fmt.Errorf("rollback failed: write execution not found")
	}
	if source.Operation != "apply" {
		return "", fmt.Errorf("rollback failed: write execution %s is not an apply execution", source.ID)
	}
	if source.Status != "completed" {
		return "", fmt.Errorf("rollback failed: write execution %s is not completed", source.ID)
	}

	latestApply := latestCompletedApplyExecution(executions)
	if latestApply == nil || latestApply.ID != source.ID {
		return "", fmt.Errorf("rollback failed: only the latest completed apply execution can be rolled back")
	}
	if len(source.RollbackPayload) == 0 {
		return "", fmt.Errorf("rollback failed: write execution %s has no rollback payload", source.ID)
	}

	approvalID := ""
	if approval, approvalErr := r.registry.ApprovalByTask(threadID, task.ID); approvalErr == nil {
		approvalID = approval.ID
	}

	workspaceRoot := r.registry.Workspace().ProjectRoot
	targets := append([]string(nil), source.TargetPaths...)
	beforeSummary := snapshotFileSummary(resolveRollbackPrimaryPath(workspaceRoot, source.RollbackPayload))
	changeSummary, afterSummary, rollbackErr := applyRollbackPayload(workspaceRoot, source.RollbackPayload)
	if rollbackErr != nil {
		failure := fmt.Sprintf("rollback failed: %s", rollbackErr.Error())
		_, _ = r.recordWriteExecution(threadID, task, approvalID, "rollback", source.ID, "failed", targets, "", rollbackPatchSummary(source), beforeSummary, beforeSummary, nil, failure)
		return "", fmt.Errorf("%s", failure)
	}

	resultSummary := RollbackExecutionSummary(targets, changeSummary)
	if _, err := r.recordWriteExecution(threadID, task, approvalID, "rollback", source.ID, "completed", targets, "", rollbackPatchSummary(source), beforeSummary, afterSummary, nil, resultSummary); err != nil {
		return "", err
	}
	return resultSummary, nil
}

func (r *Runner) recordWriteExecution(threadID string, task session.Task, approvalID string, operation string, relatedExecutionID string, status string, targets []string, patch string, patchSummary string, beforeSummary string, afterSummary string, rollbackPayload []session.WriteExecutionFileSnapshot, resultSummary string) (session.WriteExecutionRecord, error) {
	return r.registry.CreateWriteExecution(threadID, session.CreateWriteExecutionInput{
		TaskID:                task.ID,
		ApprovalID:            approvalID,
		ToolKind:              task.Kind,
		Operation:             operation,
		RelatedExecutionID:    relatedExecutionID,
		Status:                status,
		TargetPaths:           append([]string(nil), targets...),
		PatchHash:             patchHash(patch),
		PatchSummary:          patchSummary,
		BeforeSnapshotSummary: beforeSummary,
		AfterSnapshotSummary:  afterSummary,
		RollbackPayload:       append([]session.WriteExecutionFileSnapshot(nil), rollbackPayload...),
		ResultSummary:         resultSummary,
	})
}

func fallbackModel(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "unknown-model"
}

func ensureReadAllowed(mode policy.Mode) error {
	switch mode {
	case policy.ReadOnly, policy.WorkspaceWrite, policy.FullAccess, "", policy.AskUser:
		return nil
	default:
		return fmt.Errorf("%w: unsupported permission mode %s", ErrPermissionDenied, mode)
	}
}

func ensureThreadMutationAllowed(mode policy.Mode) error {
	switch mode {
	case policy.WorkspaceWrite, policy.FullAccess:
		return nil
	case "", policy.AskUser:
		return fmt.Errorf("%w: approval required", ErrPermissionDenied)
	default:
		return fmt.Errorf("%w: %s not allowed for thread mutation", ErrPermissionDenied, mode)
	}
}

func ensureWriteAllowed(mode policy.Mode, alreadyApproved bool) error {
	switch mode {
	case policy.WorkspaceWrite, policy.FullAccess:
		return nil
	case "", policy.AskUser:
		if alreadyApproved {
			return nil
		}
		return fmt.Errorf("%w: approval required", ErrPermissionDenied)
	case policy.ReadOnly:
		return fmt.Errorf("%w: %s does not allow workspace writes", ErrPermissionDenied, mode)
	default:
		return fmt.Errorf("%w: unsupported permission mode %s", ErrPermissionDenied, mode)
	}
}

func resolveWorkspacePath(workspaceRoot string, provided string) (string, error) {
	root, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return "", err
	}
	target := strings.TrimSpace(provided)
	if target == "" {
		target = "."
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	resolved, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	relative, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", ErrPathOutsideWorkspace
	}
	return resolved, nil
}

func workspaceRelative(workspaceRoot string, target string) string {
	relative, err := filepath.Rel(workspaceRoot, target)
	if err != nil || relative == "." {
		return "."
	}
	return filepath.ToSlash(relative)
}

func compactSummary(value string, max int) string {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if normalized == "" {
		return "(empty)"
	}
	runes := []rune(normalized)
	if len(runes) <= max {
		return normalized
	}
	return string(runes[:max]) + "..."
}

func searchWorkspaceText(searchRoot string, query string, workspaceRoot string) ([]string, error) {
	results := make([]string, 0)
	err := filepath.WalkDir(searchRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			base := entry.Name()
			if strings.HasPrefix(base, ".git") || base == "node_modules" || base == "dist" || base == "build" {
				return filepath.SkipDir
			}
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if !strings.Contains(string(content), query) {
			return nil
		}
		results = append(results, workspaceRelative(workspaceRoot, path))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(results)
	return results, nil
}

func searchWorkspaceTextDetailed(searchRoot string, query string, workspaceRoot string, limit int) ([]string, error) {
	results := make([]string, 0, limit)
	err := filepath.WalkDir(searchRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if shouldSkipWorkspaceEntry(entry) {
			return filepath.SkipDir
		}
		if entry.IsDir() {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil || !isLikelyText(content) {
			return nil
		}
		lines := splitTextLines(string(content))
		for lineIndex, line := range lines {
			if !strings.Contains(line, query) {
				continue
			}
			results = append(results, fmt.Sprintf("%s:%d: %s", workspaceRelative(workspaceRoot, path), lineIndex+1, compactSummary(line, 160)))
			if len(results) >= limit {
				return errSearchLimitReached
			}
		}
		return nil
	})
	if err != nil && !errors.Is(err, errSearchLimitReached) {
		return nil, err
	}
	sort.Strings(results)
	return results, nil
}

func listWorkspaceEntriesFiltered(searchRoot string, workspaceRoot string, pattern string, includeDirs bool) ([]string, error) {
	results := make([]string, 0)
	err := filepath.WalkDir(searchRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == searchRoot {
			return nil
		}
		if shouldSkipWorkspaceEntry(entry) {
			return filepath.SkipDir
		}
		if entry.IsDir() && !includeDirs {
			return nil
		}
		matched, err := filepath.Match(pattern, entry.Name())
		if err != nil {
			return err
		}
		if !matched {
			return nil
		}
		label := workspaceRelative(workspaceRoot, path)
		if entry.IsDir() {
			label += "/"
		}
		results = append(results, label)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(results)
	return results, nil
}

var errSearchLimitReached = errors.New("search limit reached")

func normalizedDetailedSearchLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func shouldSkipWorkspaceEntry(entry os.DirEntry) bool {
	if entry == nil || !entry.IsDir() {
		return false
	}
	base := entry.Name()
	return strings.HasPrefix(base, ".git") || base == "node_modules" || base == "dist" || base == "build"
}

func isLikelyText(content []byte) bool {
	if len(content) == 0 {
		return true
	}
	for _, b := range content {
		if b == 0 {
			return false
		}
	}
	return true
}

func applyWorkspacePatch(targetPath string, patch string) (string, error) {
	trimmed := strings.TrimSpace(patch)
	if trimmed == "" {
		return "", fmt.Errorf("patch is required")
	}
	lines := splitPatchLines(trimmed)
	if len(lines) < 2 || lines[0] != "*** Begin Patch" {
		return "", fmt.Errorf("unsupported patch format")
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "*** Delete File: ") {
			return "", fmt.Errorf("delete patch is not allowed")
		}
	}

	var op string
	var patchPath string
	start := -1
	for index, line := range lines {
		switch {
		case strings.HasPrefix(line, "*** Update File: "):
			op = "update"
			patchPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: "))
			start = index + 1
		case strings.HasPrefix(line, "*** Add File: "):
			op = "add"
			patchPath = strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: "))
			start = index + 1
		}
		if start != -1 {
			break
		}
	}
	if op == "" || patchPath == "" {
		return "", fmt.Errorf("patch must contain a file operation")
	}
	if filepath.Clean(filepath.FromSlash(patchPath)) != filepath.Clean(targetPath) && filepath.Base(filepath.Clean(filepath.FromSlash(patchPath))) != filepath.Base(targetPath) {
		if !sameNormalizedPath(filepath.Clean(filepath.FromSlash(patchPath)), filepath.Clean(targetPath)) {
			return "", fmt.Errorf("patch path does not match target path")
		}
	}

	switch op {
	case "add":
		return applyAddFilePatch(targetPath, lines[start:])
	case "update":
		return applyUpdateFilePatch(targetPath, lines[start:])
	default:
		return "", fmt.Errorf("unsupported patch operation")
	}
}

func sameNormalizedPath(left string, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr == nil && rightErr == nil {
		return strings.EqualFold(filepath.Clean(leftAbs), filepath.Clean(rightAbs))
	}
	return strings.EqualFold(filepath.ToSlash(filepath.Clean(left)), filepath.ToSlash(filepath.Clean(right)))
}

func applyAddFilePatch(targetPath string, lines []string) (string, error) {
	if _, err := os.Stat(targetPath); err == nil {
		return "", fmt.Errorf("target file already exists")
	}
	content := make([]string, 0)
	for _, line := range lines {
		if line == "*** End Patch" {
			break
		}
		if !strings.HasPrefix(line, "+") {
			if strings.TrimSpace(line) == "" {
				content = append(content, "")
				continue
			}
			return "", fmt.Errorf("add patch must contain only added lines")
		}
		content = append(content, strings.TrimPrefix(line, "+"))
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(targetPath, []byte(strings.Join(content, "\n")), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("created %d line(s)", len(content)), nil
}

func applyUpdateFilePatch(targetPath string, lines []string) (string, error) {
	originalBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return "", err
	}
	original := splitTextLines(string(originalBytes))
	result := make([]string, 0, len(original))
	sourceIndex := 0
	appliedLines := 0

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if line == "*** End Patch" {
			break
		}
		if strings.HasPrefix(line, "*** Move to: ") {
			return "", fmt.Errorf("move patch is not allowed")
		}
		if strings.HasPrefix(line, "@@") {
			continue
		}
		if line == "*** End of File" {
			continue
		}
		if line == "" {
			return "", fmt.Errorf("unexpected blank patch line")
		}
		marker := line[:1]
		text := line[1:]
		switch marker {
		case " ":
			found := findNextLine(original, sourceIndex, text)
			if found < 0 {
				return "", fmt.Errorf("patch context not found: %s", text)
			}
			result = append(result, original[sourceIndex:found+1]...)
			sourceIndex = found + 1
		case "-":
			found := findNextLine(original, sourceIndex, text)
			if found < 0 {
				return "", fmt.Errorf("patch removal not found: %s", text)
			}
			result = append(result, original[sourceIndex:found]...)
			sourceIndex = found + 1
			appliedLines++
		case "+":
			result = append(result, text)
			appliedLines++
		default:
			return "", fmt.Errorf("unsupported patch line: %s", line)
		}
	}
	result = append(result, original[sourceIndex:]...)
	updated := strings.Join(result, "\n")
	if err := os.WriteFile(targetPath, []byte(updated), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("updated %d line(s)", appliedLines), nil
}

func splitPatchLines(value string) []string {
	scanner := bufio.NewScanner(strings.NewReader(value))
	lines := make([]string, 0)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}

func splitTextLines(value string) []string {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, "\n")
	if normalized == "" {
		return []string{}
	}
	return strings.Split(normalized, "\n")
}

func findNextLine(lines []string, start int, want string) int {
	for index := start; index < len(lines); index++ {
		if lines[index] == want {
			return index
		}
	}
	return -1
}

func ExtractPatchTargets(raw string) ([]string, error) {
	lines := splitPatchLines(strings.TrimSpace(raw))
	targets := make([]string, 0)
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "*** Update File: "):
			targets = append(targets, strings.TrimSpace(strings.TrimPrefix(line, "*** Update File: ")))
		case strings.HasPrefix(line, "*** Add File: "):
			targets = append(targets, strings.TrimSpace(strings.TrimPrefix(line, "*** Add File: ")))
		case strings.HasPrefix(line, "*** Delete File: "):
			return nil, fmt.Errorf("delete patch is not allowed")
		}
	}
	if len(targets) == 0 {
		return nil, fmt.Errorf("patch does not declare target files")
	}
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(targets))
	for _, target := range targets {
		key := strings.ToLower(filepath.ToSlash(filepath.Clean(target)))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, target)
	}
	return unique, nil
}

func ApprovalSummary(kind string, targets []string) string {
	label := kind
	if label == "" {
		label = "write"
	}
	if len(targets) == 0 {
		return fmt.Sprintf("approval required for %s", label)
	}
	return fmt.Sprintf("approval required for %s on %s", label, strings.Join(targets, ", "))
}

func PatchExecutionSummary(targets []string, changeSummary string) string {
	label := strings.Join(targets, ", ")
	if label == "" {
		label = "workspace"
	}
	if strings.TrimSpace(changeSummary) == "" {
		return fmt.Sprintf("applied patch to %s", label)
	}
	return fmt.Sprintf("applied patch to %s: %s", label, changeSummary)
}

func TruncatedPatchSummary(raw string, max int) string {
	lines := splitPatchLines(strings.TrimSpace(raw))
	delta := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			delta++
		}
	}
	summary := fmt.Sprintf("%d patch line(s)", delta)
	if max <= 0 {
		return summary
	}
	return compactSummary(summary, max)
}

func patchHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func captureRollbackSnapshot(targetPath string, pathLabel string) (session.WriteExecutionFileSnapshot, error) {
	content, exists, err := readOptionalFile(targetPath)
	if err != nil {
		return session.WriteExecutionFileSnapshot{}, err
	}
	snapshot := session.WriteExecutionFileSnapshot{
		Path:          pathLabel,
		BeforeExists:  exists,
		BeforeContent: content,
	}
	if exists {
		snapshot.BeforeHash = patchHash(content)
	}
	return snapshot, nil
}

func readOptionalFile(targetPath string) (string, bool, error) {
	content, err := os.ReadFile(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return string(content), true, nil
}

func readFilePresenceAndHash(targetPath string) (bool, string) {
	content, exists, err := readOptionalFile(targetPath)
	if err != nil || !exists {
		return exists, ""
	}
	return true, patchHash(content)
}

func snapshotFileSummary(targetPath string) string {
	info, err := os.Stat(targetPath)
	if errors.Is(err, os.ErrNotExist) {
		return "missing file"
	}
	if err != nil {
		return fmt.Sprintf("snapshot unavailable: %s", err.Error())
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Sprintf("exists (%d bytes), unreadable: %s", info.Size(), err.Error())
	}

	lineCount := 0
	if len(content) > 0 {
		lineCount = len(splitTextLines(string(content)))
	}
	return fmt.Sprintf("exists, %d line(s), %d byte(s), sha256:%s", lineCount, len(content), patchHash(string(content))[:12])
}

func ParseRollbackInput(raw string) (string, error) {
	var input struct {
		WriteExecutionID string `json:"writeExecutionId"`
	}
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return "", err
	}
	if strings.TrimSpace(input.WriteExecutionID) == "" {
		return "", fmt.Errorf("writeExecutionId is required")
	}
	return input.WriteExecutionID, nil
}

func RollbackApprovalSummary(targets []string) string {
	label := strings.Join(targets, ", ")
	if strings.TrimSpace(label) == "" {
		label = "workspace"
	}
	return fmt.Sprintf("approval required for rollback of %s", label)
}

func RollbackExecutionSummary(targets []string, changeSummary string) string {
	label := strings.Join(targets, ", ")
	if strings.TrimSpace(label) == "" {
		label = "workspace"
	}
	if strings.TrimSpace(changeSummary) == "" {
		return fmt.Sprintf("rolled back patch on %s", label)
	}
	return fmt.Sprintf("rolled back patch on %s: %s", label, changeSummary)
}

func rollbackPatchSummary(source *session.WriteExecutionRecord) string {
	if source == nil {
		return "rollback"
	}
	if strings.TrimSpace(source.PatchSummary) == "" {
		return fmt.Sprintf("rollback of %s", source.ID)
	}
	return fmt.Sprintf("rollback of %s", source.PatchSummary)
}

func latestCompletedApplyExecution(items []session.WriteExecutionRecord) *session.WriteExecutionRecord {
	for index := len(items) - 1; index >= 0; index-- {
		if items[index].Operation == "apply" && items[index].Status == "completed" {
			return &items[index]
		}
	}
	return nil
}

func resolveRollbackPrimaryPath(workspaceRoot string, payload []session.WriteExecutionFileSnapshot) string {
	if len(payload) == 0 {
		return workspaceRoot
	}
	resolved, err := resolveWorkspacePath(workspaceRoot, payload[0].Path)
	if err != nil {
		return workspaceRoot
	}
	return resolved
}

func applyRollbackPayload(workspaceRoot string, payload []session.WriteExecutionFileSnapshot) (string, string, error) {
	changes := make([]string, 0, len(payload))
	afterSummaries := make([]string, 0, len(payload))
	for _, item := range payload {
		resolvedPath, err := resolveWorkspacePath(workspaceRoot, item.Path)
		if err != nil {
			return "", "", err
		}
		currentContent, currentExists, err := readOptionalFile(resolvedPath)
		if err != nil {
			return "", "", err
		}
		currentHash := ""
		if currentExists {
			currentHash = patchHash(currentContent)
		}
		if currentExists != item.AfterExists {
			return "", "", fmt.Errorf("file drift detected for %s", item.Path)
		}
		if currentExists && currentHash != item.AfterHash {
			return "", "", fmt.Errorf("file drift detected for %s", item.Path)
		}
		if item.BeforeExists {
			if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
				return "", "", err
			}
			if err := os.WriteFile(resolvedPath, []byte(item.BeforeContent), 0o644); err != nil {
				return "", "", err
			}
			changes = append(changes, fmt.Sprintf("restored %s", item.Path))
		} else {
			if currentExists {
				if err := os.Remove(resolvedPath); err != nil {
					return "", "", err
				}
			}
			changes = append(changes, fmt.Sprintf("removed %s", item.Path))
		}
		afterSummaries = append(afterSummaries, fmt.Sprintf("%s => %s", item.Path, snapshotFileSummary(resolvedPath)))
	}
	return strings.Join(changes, "; "), strings.Join(afterSummaries, " | "), nil
}

func ParsePatchInput(raw string) (string, string, error) {
	var input struct {
		Path  string `json:"path"`
		Patch string `json:"patch"`
	}
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return "", "", err
	}
	if strings.TrimSpace(input.Path) == "" {
		return "", "", fmt.Errorf("path is required")
	}
	if strings.TrimSpace(input.Patch) == "" {
		return "", "", fmt.Errorf("patch is required")
	}
	return input.Path, input.Patch, nil
}
