package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/session"
)

const (
	KindMessageAppend   = "thread.message.append"
	KindToolCallAppend  = "thread.toolcall.append"
	KindArtifactAppend  = "thread.artifact.append"
	KindRuntimeFlagSet  = "thread.runtimeflag.set"
	KindWorkspaceRead   = "workspace.read_file"
	KindWorkspaceList   = "workspace.list_files"
	KindWorkspaceSearch = "workspace.search_text"
	restartFailureLabel = "interrupted by runtime restart"
)

var (
	ErrUnsupportedTaskKind = errors.New("unsupported task kind")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrPathOutsideWorkspace = errors.New("path outside workspace")
)

type Registry interface {
	Task(threadID string, taskID string) (session.Task, error)
	Tasks(threadID string) ([]session.Task, bool)
	Thread(threadID string) (session.Thread, bool)
	Workspace() session.Workspace
	UpdateTaskStatus(threadID string, taskID string, input session.UpdateTaskStatusInput) (session.Task, error)
	AppendMessage(threadID string, input session.AppendMessageInput) (session.MessageRecord, error)
	AppendToolCall(threadID string, input session.AppendToolCallInput) (session.ToolCallRecord, error)
	AppendArtifact(threadID string, input session.AppendArtifactInput) (session.ArtifactRecord, error)
	SetRuntimeFlag(threadID string, input session.SetRuntimeFlagInput) (session.RuntimeFlagRecord, error)
}

type Runner struct {
	registry Registry
}

func New(registry Registry) *Runner {
	return &Runner{registry: registry}
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
			if task.Status != "running" {
				continue
			}
			if _, err := r.registry.UpdateTaskStatus(threadID, task.ID, session.UpdateTaskStatusInput{
				Status:        "failed",
				ResultSummary: restartFailureLabel,
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
		}
	}
	return nil
}

func (r *Runner) RunTask(_ context.Context, threadID string, taskID string) (session.Task, error) {
	task, err := r.registry.Task(threadID, taskID)
	if err != nil {
		return session.Task{}, err
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

	summary, execErr := r.execute(threadID, task)
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

func (r *Runner) execute(threadID string, task session.Task) (string, error) {
	thread, ok := r.registry.Thread(threadID)
	if !ok {
		return "", session.ErrThreadNotFound
	}

	switch task.Kind {
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
		return fmt.Sprintf("read %s: %s", filepath.Base(resolvedPath), compactSummary(string(bytes), 240)), nil
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
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedTaskKind, task.Kind)
	}
}

func ensureReadAllowed(mode policy.Mode) error {
	switch mode {
	case policy.ReadOnly, policy.WorkspaceWrite, policy.FullAccess:
		return nil
	case "", policy.AskUser:
		return fmt.Errorf("%w: approval required", ErrPermissionDenied)
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
