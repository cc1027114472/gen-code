package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"llmtrace/internal/core/session"
)

const (
	KindMessageAppend   = "thread.message.append"
	KindToolCallAppend  = "thread.toolcall.append"
	KindArtifactAppend  = "thread.artifact.append"
	KindRuntimeFlagSet  = "thread.runtimeflag.set"
	restartFailureLabel = "interrupted by runtime restart"
)

var (
	ErrUnsupportedTaskKind = errors.New("unsupported task kind")
)

type Registry interface {
	Task(threadID string, taskID string) (session.Task, error)
	Tasks(threadID string) ([]session.Task, bool)
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

	if _, err := r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
		Status:        "running",
		ResultSummary: "",
	}); err != nil {
		return session.Task{}, err
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

	summary, execErr := r.execute(threadID, task)
	if execErr != nil {
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

	return r.registry.UpdateTaskStatus(threadID, taskID, session.UpdateTaskStatusInput{
		Status:        "completed",
		ResultSummary: summary,
	})
}

func (r *Runner) execute(threadID string, task session.Task) (string, error) {
	switch task.Kind {
	case KindMessageAppend:
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
