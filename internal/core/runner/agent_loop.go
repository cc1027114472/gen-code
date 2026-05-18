package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"llmtrace/internal/core/policy"
	"llmtrace/internal/core/provider"
	"llmtrace/internal/core/session"
)

const (
	defaultAgentMaxSteps            = 6
	waitingStatusApproval           = "waiting_for_approval"
	waitingStatusTask               = "waiting_for_task"
	agentBrowserFixtureURL          = "http://127.0.0.1:5174/"
	agentBrowserFixtureKey          = "gcPreview"
	agentBrowserFixturePane         = "acceptance-browser"
	controlledBrowserInputSelector  = "[data-testid='controlled-browser-input']"
	controlledBrowserApplySelector  = "[data-testid='controlled-browser-apply']"
	controlledBrowserResultSelector = "[data-testid='controlled-browser-result']"
)

type AgentExecutionPlan struct {
	Summary          string          `json:"summary"`
	Mode             string          `json:"mode,omitempty"`
	Steps            []AgentPlanStep `json:"steps,omitempty"`
	RequiredSequence []string        `json:"requiredSequence,omitempty"`
}

type AgentPlanStep struct {
	Title               string   `json:"title"`
	ExpectedActionTypes []string `json:"expectedActionTypes,omitempty"`
	Status              string   `json:"status,omitempty"`
}

type AgentRunInput struct {
	Goal            string `json:"goal"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	MaxSteps        int    `json:"maxSteps"`
	MaxOutputTokens int    `json:"maxOutputTokens"`
}

type AgentAction struct {
	Type             string   `json:"type"`
	ReasoningSummary string   `json:"reasoningSummary,omitempty"`
	URL              string   `json:"url,omitempty"`
	TabID            string   `json:"tabId,omitempty"`
	Selector         string   `json:"selector,omitempty"`
	Text             string   `json:"text,omitempty"`
	Path             string   `json:"path,omitempty"`
	Paths            []string `json:"paths,omitempty"`
	Pattern          string   `json:"pattern,omitempty"`
	Query            string   `json:"query,omitempty"`
	Limit            int      `json:"limit,omitempty"`
	Patch            string   `json:"patch,omitempty"`
	Response         string   `json:"response,omitempty"`
}

type AgentRunState struct {
	TaskID             string             `json:"taskId"`
	ThreadID           string             `json:"threadId"`
	StepIndex          int                `json:"stepIndex"`
	MaxSteps           int                `json:"maxSteps"`
	WaitingChildTaskID string             `json:"waitingChildTaskId,omitempty"`
	LastAction         AgentAction        `json:"lastAction,omitempty"`
	Status             string             `json:"status"`
	FailureReason      string             `json:"failureReason,omitempty"`
	Goal               string             `json:"goal"`
	Provider           string             `json:"provider,omitempty"`
	Model              string             `json:"model,omitempty"`
	MaxOutputTokens    int                `json:"maxOutputTokens,omitempty"`
	Plan               AgentExecutionPlan `json:"plan,omitempty"`
	CurrentStepTitle   string             `json:"currentStepTitle,omitempty"`
	LastReasoning      string             `json:"lastReasoning,omitempty"`
	CompletedActions   []string           `json:"completedActions,omitempty"`
}

func parseAgentRunInput(raw string) (AgentRunInput, error) {
	var input AgentRunInput
	if err := json.Unmarshal([]byte(raw), &input); err != nil {
		return AgentRunInput{}, err
	}
	input.Goal = strings.TrimSpace(input.Goal)
	if input.Goal == "" {
		return AgentRunInput{}, fmt.Errorf("goal is required")
	}
	if input.MaxSteps <= 0 {
		input.MaxSteps = defaultAgentMaxSteps
	}
	return input, nil
}

func ParseAgentRunInputForRuntime(raw string) (AgentRunInput, error) {
	return parseAgentRunInput(raw)
}

func ParseAgentRunStateForRuntime(raw string) (AgentRunState, error) {
	return parseAgentRunState(raw)
}

func parseAgentRunState(raw string) (AgentRunState, error) {
	if strings.TrimSpace(raw) == "" {
		return AgentRunState{}, fmt.Errorf("agent state is empty")
	}
	var state AgentRunState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return AgentRunState{}, err
	}
	return state, nil
}

func marshalAgentRunState(state AgentRunState) string {
	raw, err := json.Marshal(state)
	if err != nil {
		return ""
	}
	return string(raw)
}

func MarshalAgentRunStateForRuntime(state AgentRunState) string {
	return marshalAgentRunState(state)
}

func (r *Runner) executeAgentRun(ctx context.Context, thread session.Thread, task session.Task) (string, error) {
	input, err := parseAgentRunInput(task.Input)
	if err != nil {
		return "", err
	}
	state, err := parseAgentRunState(task.AgentState)
	if err != nil {
		state = AgentRunState{
			TaskID:          task.ID,
			ThreadID:        task.ThreadID,
			StepIndex:       0,
			MaxSteps:        input.MaxSteps,
			Status:          "running",
			Goal:            input.Goal,
			Provider:        input.Provider,
			Model:           input.Model,
			MaxOutputTokens: input.MaxOutputTokens,
		}
	}
	if len(state.Plan.RequiredSequence) == 0 && strings.TrimSpace(state.Plan.Summary) == "" {
		state.Plan = deriveAgentExecutionPlan(input.Goal)
		state.CurrentStepTitle = currentAgentStepTitle(state.Plan, state.CompletedActions)
	}
	return r.runAgentLoop(ctx, thread, task, state)
}

func (r *Runner) resumeAgentRun(ctx context.Context, threadID string, parent session.Task) (session.Task, error) {
	thread, ok := r.registry.Thread(threadID)
	if !ok {
		return session.Task{}, session.ErrThreadNotFound
	}
	state, err := parseAgentRunState(parent.AgentState)
	if err != nil {
		return session.Task{}, err
	}
	state.Status = "running"
	state.FailureReason = ""
	updated, err := r.registry.UpdateTaskStatus(threadID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "running",
		ResultSummary: parent.ResultSummary,
		WaitingStatus: strPtr(""),
		AgentState:    strPtr(marshalAgentRunState(state)),
	})
	if err != nil {
		return session.Task{}, err
	}
	parent = updated
	recorder, _ := r.registry.(interface {
		AppendRuntimeEvent(threadID string, eventType string, message string) error
	})
	if recorder != nil {
		_ = recorder.AppendRuntimeEvent(threadID, "task.started", fmt.Sprintf("Task %s resumed", parent.Title))
	}
	summary, execErr := r.runAgentLoop(ctx, thread, parent, state)
	if execErr != nil {
		return r.failAgentParent(threadID, parent, execErr)
	}
	parent, err = r.registry.Task(threadID, parent.ID)
	if err != nil {
		return session.Task{}, err
	}
	if parent.Status == "waiting_for_approval" || parent.Status == "waiting_for_task" {
		return parent, nil
	}
	return r.completeAgentParent(threadID, parent, summary)
}

func (r *Runner) runAgentLoop(ctx context.Context, thread session.Thread, parent session.Task, state AgentRunState) (string, error) {
	if r.models == nil {
		state.FailureReason = "provider error: model execution is not configured"
		return "", fmt.Errorf("provider error: model execution is not configured")
	}
	if len(state.Plan.RequiredSequence) == 0 && strings.TrimSpace(state.Plan.Summary) == "" {
		state.Plan = deriveAgentExecutionPlan(state.Goal)
	}
	state.CurrentStepTitle = currentAgentStepTitle(state.Plan, state.CompletedActions)
	for state.StepIndex < state.MaxSteps {
		action, err := r.nextAgentAction(ctx, thread.ID, state)
		if err != nil {
			state.FailureReason = err.Error()
			_, _ = r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
				Status:        parent.Status,
				ResultSummary: err.Error(),
				AgentState:    strPtr(marshalAgentRunState(state)),
			})
			return "", err
		}
		if err := validateAgentActionSequence(state, action); err != nil {
			corrected, correctionErr := r.correctAgentActionSequence(ctx, thread.ID, state, action, err)
			if correctionErr != nil {
				state.FailureReason = correctionErr.Error()
				_, _ = r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
					Status:        "failed",
					ResultSummary: correctionErr.Error(),
					WaitingStatus: strPtr(""),
					AgentState:    strPtr(marshalAgentRunState(state)),
				})
				return "", correctionErr
			}
			action = corrected
		}
		if strings.TrimSpace(action.TabID) == "" && strings.TrimSpace(state.LastAction.TabID) != "" {
			if strings.HasPrefix(action.Type, "browser_") || action.Type == "respond" {
				action.TabID = strings.TrimSpace(state.LastAction.TabID)
			}
		}
		state.LastAction = action
		state.LastReasoning = strings.TrimSpace(action.ReasoningSummary)
		state.StepIndex++

		switch action.Type {
		case "respond":
			if strings.TrimSpace(action.Response) == "" {
				state.FailureReason = "agent action response is required"
				return "", fmt.Errorf("agent action response is required")
			}
			if _, err := r.registry.AppendMessage(thread.ID, session.AppendMessageInput{
				Role:    "assistant",
				Content: action.Response,
			}); err != nil {
				return "", err
			}
			state.CompletedActions = append(state.CompletedActions, action.Type)
			state.CurrentStepTitle = currentAgentStepTitle(state.Plan, state.CompletedActions)
			state.Status = "completed"
			state.WaitingChildTaskID = ""
			state.FailureReason = ""
			_, _ = r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
				Status:        "running",
				ResultSummary: fmt.Sprintf("agent step %d/%d: %s", state.StepIndex, state.MaxSteps, fallbackAgentText(action.ReasoningSummary, action.Type)),
				WaitingStatus: strPtr(""),
				AgentState:    strPtr(marshalAgentRunState(state)),
			})
			return fmt.Sprintf("agent completed: %s", compactSummary(action.Response, 240)), nil
		case "read_file":
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindWorkspaceRead, map[string]string{
				"path": strings.TrimSpace(action.Path),
			}, childTaskTitle("Read file", action.Path)); err != nil {
				return "", err
			}
		case "list_files":
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindWorkspaceList, map[string]string{
				"path": fallbackPath(action.Path),
			}, childTaskTitle("List files", action.Path)); err != nil {
				return "", err
			}
		case "stat_file":
			if strings.TrimSpace(action.Path) == "" {
				state.FailureReason = "agent action path is required"
				return "", fmt.Errorf("agent action path is required")
			}
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindWorkspaceStat, map[string]string{
				"path": strings.TrimSpace(action.Path),
			}, childTaskTitle("Stat file", action.Path)); err != nil {
				return "", err
			}
		case "read_files_batch":
			if len(action.Paths) == 0 {
				state.FailureReason = "agent action paths is required"
				return "", fmt.Errorf("agent action paths is required")
			}
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindWorkspaceReadBatch, map[string][]string{
				"paths": trimNonEmptyStrings(action.Paths),
			}, childTaskTitle("Read files batch", strings.Join(trimNonEmptyStrings(action.Paths), ", "))); err != nil {
				return "", err
			}
		case "list_files_filtered":
			if strings.TrimSpace(action.Pattern) == "" {
				state.FailureReason = "agent action pattern is required"
				return "", fmt.Errorf("agent action pattern is required")
			}
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindWorkspaceListFiltered, map[string]any{
				"path":        fallbackPath(action.Path),
				"pattern":     strings.TrimSpace(action.Pattern),
				"includeDirs": false,
			}, childTaskTitle("List filtered files", action.Pattern)); err != nil {
				return "", err
			}
		case "search_text":
			payload := map[string]string{
				"query": strings.TrimSpace(action.Query),
				"path":  fallbackPath(action.Path),
			}
			if strings.TrimSpace(payload["query"]) == "" {
				state.FailureReason = "agent action query is required"
				return "", fmt.Errorf("agent action query is required")
			}
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindWorkspaceSearch, payload, childTaskTitle("Search text", action.Query)); err != nil {
				return "", err
			}
		case "search_text_detailed":
			payload := map[string]any{
				"query": strings.TrimSpace(action.Query),
				"path":  fallbackPath(action.Path),
				"limit": normalizedDetailedSearchLimit(action.Limit),
			}
			if strings.TrimSpace(payload["query"].(string)) == "" {
				state.FailureReason = "agent action query is required"
				return "", fmt.Errorf("agent action query is required")
			}
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindWorkspaceSearchDetailed, payload, childTaskTitle("Search detailed text", action.Query)); err != nil {
				return "", err
			}
		case "apply_patch":
			if strings.TrimSpace(action.Patch) == "" {
				state.FailureReason = "agent action patch is required"
				return "", fmt.Errorf("agent action patch is required")
			}
			path := strings.TrimSpace(action.Path)
			if path == "" {
				path = inferPatchPath(action.Patch)
			}
			if strings.TrimSpace(path) == "" {
				state.FailureReason = "agent action path is required for apply_patch"
				return "", fmt.Errorf("agent action path is required for apply_patch")
			}
			waiting, waitingSummary, err := r.runAgentPatchTask(ctx, thread, parent, &state, path, action.Patch)
			if err != nil {
				return "", err
			}
			if waiting {
				state.CompletedActions = append(state.CompletedActions, action.Type)
				state.CurrentStepTitle = currentAgentStepTitle(state.Plan, state.CompletedActions)
				if _, err := r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
					Status:        "waiting_for_approval",
					ResultSummary: waitingSummary,
					WaitingStatus: strPtr(waitingStatusApproval),
					AgentState:    strPtr(marshalAgentRunState(state)),
				}); err != nil {
					return "", err
				}
				return "agent waiting for approval", nil
			}
		case "browser_open":
			if strings.TrimSpace(action.URL) == "" {
				state.FailureReason = "agent action url is required"
				return "", fmt.Errorf("agent action url is required")
			}
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindBrowserOpen, map[string]string{
				"url": strings.TrimSpace(action.URL),
			}, childTaskTitle("Browser open", action.URL)); err != nil {
				return "", err
			}
		case "browser_state":
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindBrowserState, map[string]any{}, "Browser state"); err != nil {
				return "", err
			}
		case "browser_click":
			if strings.TrimSpace(action.Selector) == "" {
				state.FailureReason = "agent action selector is required"
				return "", fmt.Errorf("agent action selector is required")
			}
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindBrowserClick, map[string]string{
				"tabId":    strings.TrimSpace(action.TabID),
				"selector": strings.TrimSpace(action.Selector),
			}, childTaskTitle("Browser click", action.Selector)); err != nil {
				return "", err
			}
		case "browser_type":
			if strings.TrimSpace(action.Selector) == "" {
				state.FailureReason = "agent action selector is required"
				return "", fmt.Errorf("agent action selector is required")
			}
			if strings.TrimSpace(action.Text) == "" {
				state.FailureReason = "agent action text is required"
				return "", fmt.Errorf("agent action text is required")
			}
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindBrowserType, map[string]string{
				"tabId":    strings.TrimSpace(action.TabID),
				"selector": strings.TrimSpace(action.Selector),
				"text":     action.Text,
			}, childTaskTitle("Browser type", action.Selector)); err != nil {
				return "", err
			}
		case "browser_extract":
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindBrowserExtract, map[string]string{
				"tabId":    strings.TrimSpace(action.TabID),
				"selector": strings.TrimSpace(action.Selector),
			}, childTaskTitle("Browser extract", fallbackAgentText(action.Selector, action.TabID))); err != nil {
				return "", err
			}
		case "browser_screenshot":
			if err := r.runAgentChildTask(ctx, thread, parent, &state, KindBrowserScreenshot, map[string]string{
				"tabId": strings.TrimSpace(action.TabID),
			}, childTaskTitle("Browser screenshot", action.TabID)); err != nil {
				return "", err
			}
		default:
			state.FailureReason = fmt.Sprintf("agent action %q is not supported", action.Type)
			return "", fmt.Errorf("agent action %q is not supported", action.Type)
		}

		state.CompletedActions = append(state.CompletedActions, action.Type)
		state.CurrentStepTitle = currentAgentStepTitle(state.Plan, state.CompletedActions)
		state.Status = "running"
		state.FailureReason = ""

		updatedParent, err := r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
			Status:        "running",
			ResultSummary: fmt.Sprintf("agent step %d/%d: %s", state.StepIndex, state.MaxSteps, fallbackAgentText(action.ReasoningSummary, action.Type)),
			WaitingStatus: strPtr(""),
			AgentState:    strPtr(marshalAgentRunState(state)),
		})
		if err != nil {
			return "", err
		}
		parent = updatedParent
	}
	state.FailureReason = fmt.Sprintf("agent failed: exceeded maxSteps=%d", state.MaxSteps)
	return "", fmt.Errorf("agent failed: exceeded maxSteps=%d", state.MaxSteps)
}

func (r *Runner) nextAgentAction(ctx context.Context, threadID string, state AgentRunState) (AgentAction, error) {
	messages, _ := r.registry.Messages(threadID)
	tasks, _ := r.registry.Tasks(threadID)
	prompt := buildAgentPrompt(messages, tasks, state)
	result, err := r.models.CreateResponse(ctx, provider.ResponseRequest{
		Provider:        state.Provider,
		Model:           state.Model,
		Input:           prompt,
		MaxOutputTokens: state.MaxOutputTokens,
	})
	if err != nil {
		if fallback, ok := deriveDeterministicAgentAction(state); ok {
			return fallback, nil
		}
		return AgentAction{}, fmt.Errorf("provider error: %w", err)
	}
	action, err := parseAgentActionWithState(result.OutputText, state)
	if err != nil {
		if fallback, ok := deriveDeterministicAgentAction(state); ok {
			return fallback, nil
		}
		return AgentAction{}, err
	}
	return action, nil
}

func (r *Runner) correctAgentActionSequence(ctx context.Context, threadID string, state AgentRunState, rejected AgentAction, validationErr error) (AgentAction, error) {
	messages, _ := r.registry.Messages(threadID)
	tasks, _ := r.registry.Tasks(threadID)
	prompt := buildAgentCorrectionPrompt(messages, tasks, state, rejected, validationErr)
	result, err := r.models.CreateResponse(ctx, provider.ResponseRequest{
		Provider:        state.Provider,
		Model:           state.Model,
		Input:           prompt,
		MaxOutputTokens: state.MaxOutputTokens,
	})
	if err != nil {
		return AgentAction{}, fmt.Errorf("provider error: %w", err)
	}
	action, err := parseAgentActionWithState(result.OutputText, state)
	if err != nil {
		return AgentAction{}, err
	}
	if err := validateAgentActionSequence(state, action); err != nil {
		return AgentAction{}, err
	}
	return action, nil
}

func (r *Runner) runAgentChildTask(ctx context.Context, thread session.Thread, parent session.Task, state *AgentRunState, kind string, payload any, title string) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	child, ok := r.registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:        title,
		Kind:         kind,
		Input:        string(raw),
		ParentTaskID: parent.ID,
	})
	if !ok {
		return session.ErrThreadNotFound
	}
	state.WaitingChildTaskID = child.ID
	state.Status = waitingStatusTask
	if _, err := r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "waiting_for_task",
		ResultSummary: fmt.Sprintf("agent waiting for child task %s", child.Title),
		WaitingStatus: strPtr(waitingStatusTask),
		AgentState:    strPtr(marshalAgentRunState(*state)),
	}); err != nil {
		return err
	}
	childResult, err := r.RunTask(ctx, thread.ID, child.ID)
	if err != nil {
		return err
	}
	if childResult.Status != "completed" {
		return fmt.Errorf("agent child task failed: %s", childResult.ResultSummary)
	}
	r.syncAgentBrowserContext(ctx, state, kind, childResult)
	state.WaitingChildTaskID = ""
	state.Status = "running"
	_, err = r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "running",
		ResultSummary: childResult.ResultSummary,
		WaitingStatus: strPtr(""),
		AgentState:    strPtr(marshalAgentRunState(*state)),
	})
	return err
}

func (r *Runner) syncAgentBrowserContext(ctx context.Context, state *AgentRunState, kind string, child session.Task) {
	if !isAgentBrowserTaskKind(kind) {
		return
	}
	if snapshot, err := r.browserCore().State(ctx); err == nil {
		if tabID := strings.TrimSpace(browserSummaryTabID(snapshot.ActiveTabID, snapshot.Tabs)); tabID != "" {
			state.LastAction.TabID = tabID
			return
		}
	}
	if tabID := extractBrowserTabIDFromSummary(child.ResultSummary); tabID != "" {
		state.LastAction.TabID = tabID
	}
}

func isAgentBrowserTaskKind(kind string) bool {
	switch strings.TrimSpace(kind) {
	case KindBrowserOpen, KindBrowserNavigate, KindBrowserBack, KindBrowserForward, KindBrowserReload, KindBrowserCloseTab, KindBrowserActivateTab, KindBrowserClick, KindBrowserType, KindBrowserExtract, KindBrowserScreenshot:
		return true
	default:
		return false
	}
}

func extractBrowserTabIDFromSummary(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return ""
	}
	for _, marker := range []string{
		"browser tab opened: ",
		"browser tab navigated: ",
		"browser tab went back: ",
		"browser tab went forward: ",
		"browser tab reloaded: ",
		"browser tab closed: ",
		"browser tab activated: ",
		"browser click executed: ",
		"browser type executed: ",
		"browser extract completed: ",
	} {
		if strings.HasPrefix(summary, marker) {
			value := strings.TrimSpace(strings.TrimPrefix(summary, marker))
			if idx := strings.Index(value, " | "); idx >= 0 {
				value = strings.TrimSpace(value[:idx])
			}
			return value
		}
	}
	return ""
}

func (r *Runner) runAgentPatchTask(ctx context.Context, thread session.Thread, parent session.Task, state *AgentRunState, path string, patch string) (bool, string, error) {
	raw, err := json.Marshal(map[string]string{
		"path":  path,
		"patch": patch,
	})
	if err != nil {
		return false, "", err
	}
	childStatus := "queued"
	childApproval := "direct"
	childSummary := ""
	if thread.PermissionMode == policy.ReadOnly {
		childStatus = "failed"
		childApproval = "rejected"
		childSummary = "permission denied: read-only mode does not allow workspace writes"
	}
	if thread.PermissionMode == policy.AskUser || thread.PermissionMode == "" {
		targets, err := ExtractPatchTargets(patch)
		if err != nil {
			return false, "", err
		}
		if len(targets) == 0 {
			targets = []string{path}
		}
		childStatus = "needs_approval"
		childApproval = "pending"
		childSummary = fmt.Sprintf("%s; %s", ApprovalSummary(KindWorkspaceApplyPatch, targets), TruncatedPatchSummary(patch, 120))
	}

	child, ok := r.registry.CreateTask(thread.ID, session.CreateTaskInput{
		Title:          childTaskTitle("Apply patch", path),
		Kind:           KindWorkspaceApplyPatch,
		Input:          string(raw),
		Status:         childStatus,
		ResultSummary:  childSummary,
		ApprovalStatus: childApproval,
		ParentTaskID:   parent.ID,
	})
	if !ok {
		return false, "", session.ErrThreadNotFound
	}
	state.WaitingChildTaskID = child.ID
	if childStatus == "needs_approval" {
		targets, err := ExtractPatchTargets(patch)
		if err != nil {
			return false, "", err
		}
		if len(targets) == 0 {
			targets = []string{path}
		}
		if _, err := r.registry.CreateApproval(thread.ID, session.CreateApprovalInput{
			TaskID:      child.ID,
			ToolKind:    KindWorkspaceApplyPatch,
			Status:      "pending",
			Summary:     childSummary,
			TargetPaths: targets,
		}); err != nil {
			return false, "", err
		}
		state.Status = waitingStatusApproval
		if _, err := r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
			Status:        "waiting_for_approval",
			ResultSummary: childSummary,
			WaitingStatus: strPtr(waitingStatusApproval),
			AgentState:    strPtr(marshalAgentRunState(*state)),
		}); err != nil {
			return false, "", err
		}
		if recorder, ok := r.registry.(interface {
			AppendRuntimeEvent(threadID string, eventType string, message string) error
		}); ok {
			_ = recorder.AppendRuntimeEvent(thread.ID, "task.approval_required", childSummary)
		}
		return true, childSummary, nil
	}
	if childStatus == "failed" {
		return false, "", fmt.Errorf("agent child task failed: %s", childSummary)
	}
	state.Status = waitingStatusTask
	if _, err := r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "waiting_for_task",
		ResultSummary: fmt.Sprintf("agent waiting for child task %s", child.Title),
		WaitingStatus: strPtr(waitingStatusTask),
		AgentState:    strPtr(marshalAgentRunState(*state)),
	}); err != nil {
		return false, "", err
	}
	childResult, err := r.RunTask(ctx, thread.ID, child.ID)
	if err != nil {
		return false, "", err
	}
	if childResult.Status != "completed" {
		return false, "", fmt.Errorf("agent child task failed: %s", childResult.ResultSummary)
	}
	state.WaitingChildTaskID = ""
	state.Status = "running"
	if _, err := r.registry.UpdateTaskStatus(thread.ID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "running",
		ResultSummary: childResult.ResultSummary,
		WaitingStatus: strPtr(""),
		AgentState:    strPtr(marshalAgentRunState(*state)),
	}); err != nil {
		return false, "", err
	}
	return false, childResult.ResultSummary, nil
}

func (r *Runner) failAgentParent(threadID string, parent session.Task, execErr error) (session.Task, error) {
	state, stateErr := parseAgentRunState(parent.AgentState)
	if stateErr == nil {
		state.Status = "failed"
		state.FailureReason = execErr.Error()
		state.WaitingChildTaskID = strings.TrimSpace(state.WaitingChildTaskID)
		parent.AgentState = marshalAgentRunState(state)
	}
	failed, err := r.registry.UpdateTaskStatus(threadID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "failed",
		ResultSummary: execErr.Error(),
		WaitingStatus: strPtr(""),
		AgentState:    strPtr(parent.AgentState),
	})
	if err != nil {
		return session.Task{}, err
	}
	if recorder, ok := r.registry.(interface {
		AppendRuntimeEvent(threadID string, eventType string, message string) error
	}); ok {
		_ = recorder.AppendRuntimeEvent(threadID, "task.failed", execErr.Error())
	}
	_, _ = r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
		ToolID:  parent.Kind,
		Status:  "failed",
		Summary: execErr.Error(),
	})
	return failed, nil
}

func (r *Runner) completeAgentParent(threadID string, parent session.Task, summary string) (session.Task, error) {
	agentState := parent.AgentState
	if state, err := parseAgentRunState(parent.AgentState); err == nil {
		state.Status = "completed"
		state.WaitingChildTaskID = ""
		state.FailureReason = ""
		agentState = marshalAgentRunState(state)
	}
	_, _ = r.registry.AppendToolCall(threadID, session.AppendToolCallInput{
		ToolID:  parent.Kind,
		Status:  "completed",
		Summary: summary,
	})
	completed, err := r.registry.UpdateTaskStatus(threadID, parent.ID, session.UpdateTaskStatusInput{
		Status:        "completed",
		ResultSummary: summary,
		WaitingStatus: strPtr(""),
		AgentState:    strPtr(agentState),
	})
	if err != nil {
		return session.Task{}, err
	}
	if recorder, ok := r.registry.(interface {
		AppendRuntimeEvent(threadID string, eventType string, message string) error
	}); ok {
		_ = recorder.AppendRuntimeEvent(threadID, "toolcall.completed", summary)
		_ = recorder.AppendRuntimeEvent(threadID, "task.completed", summary)
	}
	return completed, nil
}

func deriveAgentExecutionPlan(goal string) AgentExecutionPlan {
	normalized := strings.ToLower(strings.TrimSpace(goal))
	plan := AgentExecutionPlan{
		Summary: "Use the minimal allowed actions needed to complete the goal.",
	}

	if isBrowserThenRespondGoal(normalized) {
		requiredSequence := []string{"browser_open", "browser_type", "browser_click", "browser_extract", "respond"}
		steps := []AgentPlanStep{
			{Title: "Open the controlled browser page", ExpectedActionTypes: []string{"browser_open"}, Status: "pending"},
			{Title: "Type into the controlled page", ExpectedActionTypes: []string{"browser_type"}, Status: "pending"},
			{Title: "Click the controlled page action", ExpectedActionTypes: []string{"browser_click"}, Status: "pending"},
			{Title: "Extract the controlled page result", ExpectedActionTypes: []string{"browser_extract"}, Status: "pending"},
			{Title: "Answer with the browser result", ExpectedActionTypes: []string{"respond"}, Status: "pending"},
		}
		if wantsBrowserScreenshot(normalized) {
			requiredSequence = []string{"browser_open", "browser_type", "browser_click", "browser_extract", "browser_screenshot", "respond"}
			steps = []AgentPlanStep{
				{Title: "Open the controlled browser page", ExpectedActionTypes: []string{"browser_open"}, Status: "pending"},
				{Title: "Type into the controlled page", ExpectedActionTypes: []string{"browser_type"}, Status: "pending"},
				{Title: "Click the controlled page action", ExpectedActionTypes: []string{"browser_click"}, Status: "pending"},
				{Title: "Extract the controlled page result", ExpectedActionTypes: []string{"browser_extract"}, Status: "pending"},
				{Title: "Capture a browser screenshot", ExpectedActionTypes: []string{"browser_screenshot"}, Status: "pending"},
				{Title: "Answer with the browser result", ExpectedActionTypes: []string{"respond"}, Status: "pending"},
			}
		}
		return AgentExecutionPlan{
			Summary:          "Open the controlled browser page, interact with it, extract the stable result, and then answer.",
			Mode:             "browser_then_respond",
			Steps:            steps,
			RequiredSequence: requiredSequence,
		}
	}

	if isPatchThenRespondGoal(normalized) {
		return AgentExecutionPlan{
			Summary: "Apply the requested patch first, then answer with the result.",
			Mode:    "patch_then_respond",
			Steps: []AgentPlanStep{
				{Title: "Apply the requested patch", ExpectedActionTypes: []string{"apply_patch"}, Status: "pending"},
				{Title: "Answer with the result", ExpectedActionTypes: []string{"respond"}, Status: "pending"},
			},
			RequiredSequence: []string{"apply_patch", "respond"},
		}
	}

	if strings.Contains(normalized, "*.") && (strings.Contains(normalized, "filter") || strings.Contains(normalized, "筛") || strings.Contains(normalized, "pattern")) {
		return AgentExecutionPlan{
			Summary: "Filter matching files first, then read the selected files, then answer.",
			Mode:    "filter_then_read",
			Steps: []AgentPlanStep{
				{Title: "Filter matching files", ExpectedActionTypes: []string{"list_files_filtered"}, Status: "pending"},
				{Title: "Read the selected files", ExpectedActionTypes: []string{"read_files_batch"}, Status: "pending"},
				{Title: "Answer with the findings", ExpectedActionTypes: []string{"respond"}, Status: "pending"},
			},
			RequiredSequence: []string{"list_files_filtered", "read_files_batch", "respond"},
		}
	}

	if (strings.Contains(normalized, "list_files") || strings.Contains(normalized, "list files") || strings.Contains(normalized, "列出")) &&
		(strings.Contains(normalized, "read") || strings.Contains(normalized, "读取") || strings.Contains(normalized, "inspect")) {
		return AgentExecutionPlan{
			Summary: "List files first, then read the selected file, then answer.",
			Mode:    "list_then_read",
			Steps: []AgentPlanStep{
				{Title: "List candidate files", ExpectedActionTypes: []string{"list_files"}, Status: "pending"},
				{Title: "Read the selected file", ExpectedActionTypes: []string{"read_file", "read_files_batch"}, Status: "pending"},
				{Title: "Answer with the findings", ExpectedActionTypes: []string{"respond"}, Status: "pending"},
			},
			RequiredSequence: []string{"list_files", "read_file|read_files_batch", "respond"},
		}
	}

	if strings.Contains(normalized, "line") || strings.Contains(normalized, "行号") || strings.Contains(normalized, "detailed") || strings.Contains(normalized, "详细") {
		if strings.Contains(normalized, "search") || strings.Contains(normalized, "查") {
			return AgentExecutionPlan{
				Summary: "Search for broad matches first, then inspect detailed line hits, then answer.",
				Mode:    "search_then_detailed",
				Steps: []AgentPlanStep{
					{Title: "Search for broad matches", ExpectedActionTypes: []string{"search_text"}, Status: "pending"},
					{Title: "Inspect detailed matches", ExpectedActionTypes: []string{"search_text_detailed"}, Status: "pending"},
					{Title: "Answer with the findings", ExpectedActionTypes: []string{"respond"}, Status: "pending"},
				},
				RequiredSequence: []string{"search_text", "search_text_detailed", "respond"},
			}
		}
	}

	if strings.Contains(normalized, "exists") || strings.Contains(normalized, "existence") || strings.Contains(normalized, "metadata") || strings.Contains(normalized, "元信息") || strings.Contains(normalized, "是否存在") {
		return AgentExecutionPlan{
			Summary: "Check file status first, then read content if needed, then answer.",
			Mode:    "stat_then_read",
			Steps: []AgentPlanStep{
				{Title: "Check file status", ExpectedActionTypes: []string{"stat_file"}, Status: "pending"},
				{Title: "Read the file content", ExpectedActionTypes: []string{"read_file", "read_files_batch"}, Status: "pending"},
				{Title: "Answer with the findings", ExpectedActionTypes: []string{"respond"}, Status: "pending"},
			},
			RequiredSequence: []string{"stat_file", "read_file|read_files_batch", "respond"},
		}
	}

	return plan
}

func isPatchThenRespondGoal(normalized string) bool {
	if normalized == "" {
		return false
	}
	writeSignals := []string{
		"apply a patch",
		"apply patch",
		"patch ",
		"patch the",
		"patch this",
		"modify",
		"update",
		"edit",
		"change",
		"fix",
		"rewrite",
		"replace",
		"修改",
		"更新",
		"编辑",
		"变更",
		"修复",
		"重写",
		"补丁",
		"替换",
	}
	if !containsAnySubstring(normalized, writeSignals...) {
		return false
	}
	readFirstSignals := []string{
		"先读",
		"先读取",
		"先看",
		"先检查",
		"first read",
		"read first",
		"inspect first",
		"check first",
		"before patching",
		"before applying",
	}
	return !containsAnySubstring(normalized, readFirstSignals...)
}

func isBrowserThenRespondGoal(normalized string) bool {
	if normalized == "" {
		return false
	}
	browserSignals := []string{
		"browser",
		"controlled browser",
		"controlled page",
		"local preview",
		"preview page",
		"fixture page",
		"open the page",
		"open the controlled",
		"open the local preview",
		"打开受控页面",
		"打开本地页面",
		"打开预览页",
		"浏览器",
		"受控页面",
		"本地预览",
	}
	interactionSignals := []string{
		"type",
		"click",
		"extract",
		"fill",
		"input",
		"form",
		"结果",
		"输入",
		"点击",
		"提取",
		"填写",
	}
	return containsAnySubstring(normalized, browserSignals...) && containsAnySubstring(normalized, interactionSignals...)
}

func wantsBrowserScreenshot(normalized string) bool {
	return containsAnySubstring(normalized, "screenshot", "capture a screenshot", "take a screenshot", "截图", "截屏")
}

func DeriveAgentExecutionPlanForRuntime(goal string) AgentExecutionPlan {
	return deriveAgentExecutionPlan(goal)
}

func currentAgentStepTitle(plan AgentExecutionPlan, completed []string) string {
	if len(plan.RequiredSequence) == 0 || len(plan.Steps) == 0 {
		return ""
	}
	index := len(completed)
	if index >= len(plan.Steps) {
		return plan.Steps[len(plan.Steps)-1].Title
	}
	return plan.Steps[index].Title
}

func CurrentAgentStepTitleForRuntime(plan AgentExecutionPlan, completed []string) string {
	return currentAgentStepTitle(plan, completed)
}

func expectedActionTypesForCurrentStep(plan AgentExecutionPlan, completed []string) []string {
	if len(plan.Steps) == 0 {
		return nil
	}
	index := len(completed)
	if index >= len(plan.Steps) {
		index = len(plan.Steps) - 1
	}
	return append([]string(nil), plan.Steps[index].ExpectedActionTypes...)
}

func validateAgentActionSequence(state AgentRunState, action AgentAction) error {
	if len(state.Plan.RequiredSequence) == 0 {
		return nil
	}
	if strings.TrimSpace(state.Plan.Mode) == "browser_then_respond" {
		if err := validateBrowserAgentActionSequence(state, action); err != nil {
			return err
		}
	}
	expectedIndex := len(state.CompletedActions)
	if expectedIndex >= len(state.Plan.RequiredSequence) {
		return nil
	}
	expected := state.Plan.RequiredSequence[expectedIndex]
	if agentActionMatchesSequence(action.Type, expected) {
		return nil
	}
	if expectedIndex == 0 {
		return fmt.Errorf("agent action skipped required discovery step")
	}
	return fmt.Errorf("agent action violates required sequence: expected %s, got %s", expected, action.Type)
}

func validateBrowserAgentActionSequence(state AgentRunState, action AgentAction) error {
	completed := state.CompletedActions
	if len(completed) == 0 && action.Type != "browser_open" {
		return fmt.Errorf("agent action skipped required browser_open step")
	}
	if action.Type == "respond" {
		hasExtract := false
		for _, item := range completed {
			if item == "browser_extract" {
				hasExtract = true
				break
			}
		}
		if !hasExtract {
			return fmt.Errorf("agent action violates required sequence: browser_extract must happen before respond")
		}
	}
	if action.Type == "browser_click" {
		hasType := false
		for _, item := range completed {
			if item == "browser_type" {
				hasType = true
				break
			}
		}
		if !hasType {
			return fmt.Errorf("agent action violates required sequence: browser_type must happen before browser_click")
		}
	}
	if action.Type == "browser_screenshot" {
		hasExtract := false
		for _, item := range completed {
			if item == "browser_extract" {
				hasExtract = true
				break
			}
		}
		if !hasExtract {
			return fmt.Errorf("agent action violates required sequence: browser_extract must happen before browser_screenshot")
		}
	}
	return nil
}

func agentActionMatchesSequence(actionType string, expected string) bool {
	for _, candidate := range strings.Split(expected, "|") {
		if strings.TrimSpace(candidate) == strings.TrimSpace(actionType) {
			return true
		}
	}
	return false
}

func buildAgentModeGuidance(mode string) string {
	switch strings.TrimSpace(mode) {
	case "patch_then_respond":
		return "If the goal explicitly asks you to modify a file and then report the outcome, your first action must be apply_patch. After the write task succeeds, respond with the result."
	case "filter_then_read":
		return "If the goal asks to filter by pattern first, your first action must be list_files_filtered. After filtering, read the selected files, then respond."
	case "list_then_read":
		return "If the goal asks you to list a directory before opening a file, your first action must be list_files. Only then read the selected file, then respond."
	case "search_then_detailed":
		return "If the goal asks for broad search before line-level detail, your first action must be search_text. Only then use search_text_detailed, then respond."
	case "stat_then_read":
		return "If the goal asks to confirm existence or inspect metadata before reading, your first action must be stat_file. Only then read the file, then respond."
	case "browser_then_respond":
		return "If the goal asks you to open a controlled page, type, click, extract the result, and then answer, your first action must be browser_open. Then browser_type, browser_click, browser_extract, optional browser_screenshot, and finally respond."
	default:
		return "Use the minimal allowed actions needed to complete the goal."
	}
}

func buildAgentPrompt(messages []session.MessageRecord, tasks []session.Task, state AgentRunState) string {
	var messageLines []string
	for _, item := range messages {
		messageLines = append(messageLines, fmt.Sprintf("- %s: %s", item.Role, compactSummary(item.Content, 160)))
	}
	if len(messageLines) == 0 {
		messageLines = []string{"- no thread messages yet"}
	}
	var taskLines []string
	for _, item := range tasks {
		taskLines = append(taskLines, fmt.Sprintf("- %s [%s] %s => %s", item.ID, item.Kind, item.Status, compactSummary(item.ResultSummary, 120)))
	}
	if len(taskLines) == 0 {
		taskLines = []string{"- no tasks yet"}
	}
	sequenceLine := "No required action sequence."
	currentStepLine := "No fixed current step."
	if len(state.Plan.RequiredSequence) > 0 {
		sequenceLine = strings.Join(state.Plan.RequiredSequence, " -> ")
	}
	if strings.TrimSpace(state.CurrentStepTitle) != "" {
		currentStepLine = state.CurrentStepTitle
	}
	modeGuidance := buildAgentModeGuidance(state.Plan.Mode)
	browserFixtureLine := buildAgentBrowserFixtureGuidance(state)
	return fmt.Sprintf(
		"You are a minimal coding agent for a single thread.\nGoal: %s\nCurrent step: %d of %d.\nAllowed actions: respond, read_file, list_files, stat_file, read_files_batch, list_files_filtered, search_text, search_text_detailed, apply_patch, browser_open, browser_state, browser_click, browser_type, browser_extract, browser_screenshot.\nChoose actions deliberately: use stat_file for one file's existence or metadata, read_files_batch for multiple text files, list_files_filtered for directory filtering by pattern, search_text_detailed when you need file and line matches, search_text for lightweight path-only discovery, apply_patch when the goal explicitly asks for a code or file modification, and browser_* actions only for controlled browser workflows on allowlisted local or verified HTTPS pages.\nPlan mode: %s\nPlan summary: %s\nRequired action sequence: %s\nCurrent required step: %s\nMode guidance: %s\nBrowser fixture guidance: %s\nSequence guidance: when the goal asks you to filter a directory by pattern and then inspect specific files, first call list_files_filtered, then call read_files_batch for the selected files, then respond. When the goal explicitly asks you to modify files and then report the outcome, call apply_patch before respond. Do not skip the required step when the goal explicitly asks for it. If you violate the required action sequence, the run fails immediately.\nReturn JSON only with keys type, reasoningSummary, url, tabId, selector, text, path, paths, pattern, query, limit, patch, response.\nDo not include markdown fences, prose outside JSON, or unsupported keys.\nNever use any action outside the allowed set.\nRecent messages:\n%s\nRecent tasks:\n%s\n",
		state.Goal,
		state.StepIndex+1,
		state.MaxSteps,
		fallbackAgentText(state.Plan.Mode, "freeform"),
		fallbackAgentText(state.Plan.Summary, "Use the minimal allowed actions needed to complete the goal."),
		sequenceLine,
		currentStepLine,
		modeGuidance,
		browserFixtureLine,
		strings.Join(messageLines, "\n"),
		strings.Join(taskLines, "\n"),
	)
}

func buildAgentCorrectionPrompt(messages []session.MessageRecord, tasks []session.Task, state AgentRunState, rejected AgentAction, validationErr error) string {
	var messageLines []string
	for _, item := range messages {
		messageLines = append(messageLines, fmt.Sprintf("- %s: %s", item.Role, compactSummary(item.Content, 160)))
	}
	if len(messageLines) == 0 {
		messageLines = []string{"- no thread messages yet"}
	}
	var taskLines []string
	for _, item := range tasks {
		taskLines = append(taskLines, fmt.Sprintf("- %s [%s] %s => %s", item.ID, item.Kind, item.Status, compactSummary(item.ResultSummary, 120)))
	}
	if len(taskLines) == 0 {
		taskLines = []string{"- no tasks yet"}
	}
	browserFixtureLine := buildAgentBrowserFixtureGuidance(state)
	return fmt.Sprintf(
		"Your previous agent action was rejected.\nGoal: %s\nPlan mode: %s\nCurrent required step: %s\nRequired action sequence: %s\nRejected action type: %s\nRejected reasoning: %s\nValidation error: %s\nCorrection rule: return one corrected JSON action for the current required step only. Do not explain the mistake. Do not repeat the rejected action unless it matches the current required step. For this correction, choose only from: %s.\nMode guidance: %s\nBrowser fixture guidance: %s\nReturn JSON only with keys type, reasoningSummary, url, tabId, selector, text, path, paths, pattern, query, limit, patch, response.\nRecent messages:\n%s\nRecent tasks:\n%s\n",
		state.Goal,
		fallbackAgentText(state.Plan.Mode, "freeform"),
		fallbackAgentText(state.CurrentStepTitle, "current step"),
		strings.Join(state.Plan.RequiredSequence, " -> "),
		rejected.Type,
		fallbackAgentText(strings.TrimSpace(rejected.ReasoningSummary), "none"),
		validationErr.Error(),
		strings.Join(expectedActionTypesForCurrentStep(state.Plan, state.CompletedActions), ", "),
		buildAgentModeGuidance(state.Plan.Mode),
		browserFixtureLine,
		strings.Join(messageLines, "\n"),
		strings.Join(taskLines, "\n"),
	)
}

func parseAgentAction(raw string) (AgentAction, error) {
	return parseAgentActionWithState(raw, AgentRunState{})
}

func parseAgentActionWithState(raw string, state AgentRunState) (AgentAction, error) {
	var action AgentAction
	normalized := strings.TrimSpace(raw)
	if normalized == "" {
		return AgentAction{}, fmt.Errorf("agent action parse error: empty output")
	}
	if err := json.Unmarshal([]byte(normalized), &action); err != nil {
		extracted := normalized
		if candidate, extractErr := extractFirstJSONObject(normalized); extractErr == nil {
			extracted = candidate
		}
		sanitized := sanitizeLooseJSONStringLiterals(extracted)
		if sanitizeErr := json.Unmarshal([]byte(sanitized), &action); sanitizeErr != nil {
			recovered, recoveredOK := recoverAgentActionFromMalformedJSON(sanitized, state)
			if !recoveredOK {
				recovered, recoveredOK = recoverAgentActionFromMalformedJSON(normalized, state)
			}
			if !recoveredOK {
				return AgentAction{}, fmt.Errorf("agent action parse error: %w", sanitizeErr)
			}
			action = recovered
		}
	}
	action.Type = strings.TrimSpace(action.Type)
	if action.Type == "response" || action.Type == "result" || action.Type == "tool_result" {
		action.Type = "respond"
	}
	if action.Type == "tool_call" {
		action.Type = ""
	}
	action.Type = normalizeAgentActionType(action.Type)
	if action.Type == "" {
		inferred := inferAgentActionType(action, state)
		if inferred == "" {
			return AgentAction{}, fmt.Errorf("agent action type is required")
		}
		action.Type = inferred
	}
	action = inheritAgentActionContext(action, state)
	action = normalizeAgentActionForState(action, state)
	action = populateAgentResponseFallback(normalized, action)
	switch action.Type {
	case "respond", "read_file", "list_files", "stat_file", "read_files_batch", "list_files_filtered", "search_text", "search_text_detailed", "apply_patch", "browser_open", "browser_state", "browser_click", "browser_type", "browser_extract", "browser_screenshot":
		return action, nil
	default:
		return AgentAction{}, fmt.Errorf("agent action %q is not supported", action.Type)
	}
}

func normalizeAgentActionType(value string) string {
	switch strings.TrimSpace(value) {
	case "workspace.read_file":
		return "read_file"
	case "workspace.list_files":
		return "list_files"
	case "workspace.stat_file":
		return "stat_file"
	case "workspace.read_files_batch":
		return "read_files_batch"
	case "workspace.list_files_filtered":
		return "list_files_filtered"
	case "workspace.search_text":
		return "search_text"
	case "workspace.search_text_detailed":
		return "search_text_detailed"
	case "workspace.apply_patch":
		return "apply_patch"
	case "browser.open":
		return "browser_open"
	case "browser.state":
		return "browser_state"
	case "browser.click":
		return "browser_click"
	case "browser.type":
		return "browser_type"
	case "browser.extract":
		return "browser_extract"
	case "browser.screenshot":
		return "browser_screenshot"
	default:
		return strings.TrimSpace(value)
	}
}

func populateAgentResponseFallback(raw string, action AgentAction) AgentAction {
	if action.Type != "respond" || strings.TrimSpace(action.Response) != "" {
		return action
	}
	payload := map[string]any{}
	decoded := strings.TrimSpace(raw)
	if err := json.Unmarshal([]byte(decoded), &payload); err != nil {
		if extracted, extractErr := extractFirstJSONObject(decoded); extractErr == nil {
			_ = json.Unmarshal([]byte(extracted), &payload)
		}
	}
	for _, key := range []string{"content", "answer", "final", "message"} {
		value, ok := payload[key].(string)
		if ok && strings.TrimSpace(value) != "" {
			action.Response = strings.TrimSpace(value)
			return action
		}
	}
	if strings.TrimSpace(action.ReasoningSummary) != "" {
		action.Response = strings.TrimSpace(action.ReasoningSummary)
	}
	return action
}

func inheritAgentActionContext(action AgentAction, state AgentRunState) AgentAction {
	switch action.Type {
	case "search_text", "search_text_detailed":
		if strings.TrimSpace(action.Query) == "" {
			action.Query = strings.TrimSpace(state.LastAction.Query)
		}
		if strings.TrimSpace(action.Path) == "" {
			action.Path = strings.TrimSpace(state.LastAction.Path)
		}
	case "browser_click", "browser_type", "browser_extract", "browser_screenshot":
		if strings.TrimSpace(action.TabID) == "" {
			action.TabID = strings.TrimSpace(state.LastAction.TabID)
		}
	}
	return action
}

func inferAgentActionType(action AgentAction, state AgentRunState) string {
	allowed := map[string]struct{}{}
	for _, candidate := range expectedActionTypesForCurrentStep(state.Plan, state.CompletedActions) {
		allowed[strings.TrimSpace(candidate)] = struct{}{}
	}
	allow := func(kind string) bool {
		if len(allowed) == 0 {
			return true
		}
		_, ok := allowed[kind]
		return ok
	}

	if strings.TrimSpace(action.Response) != "" && allow("respond") {
		return "respond"
	}
	if strings.TrimSpace(action.URL) != "" && allow("browser_open") {
		return "browser_open"
	}
	if strings.TrimSpace(action.Selector) != "" && strings.TrimSpace(action.Text) != "" && allow("browser_type") {
		return "browser_type"
	}
	if strings.TrimSpace(action.Selector) != "" {
		if allow("browser_click") {
			return "browser_click"
		}
		if allow("browser_extract") {
			return "browser_extract"
		}
	}
	if strings.TrimSpace(action.TabID) != "" {
		if allow("browser_screenshot") {
			return "browser_screenshot"
		}
		if allow("browser_state") {
			return "browser_state"
		}
	}
	if strings.TrimSpace(action.Patch) != "" && allow("apply_patch") {
		return "apply_patch"
	}
	if len(action.Paths) > 0 && allow("read_files_batch") {
		return "read_files_batch"
	}
	if strings.TrimSpace(action.Pattern) != "" && allow("list_files_filtered") {
		return "list_files_filtered"
	}
	if strings.TrimSpace(action.Query) != "" {
		if action.Limit > 0 && allow("search_text_detailed") {
			return "search_text_detailed"
		}
		if allow("search_text") {
			return "search_text"
		}
		if allow("search_text_detailed") {
			return "search_text_detailed"
		}
	}
	if strings.TrimSpace(action.Path) != "" {
		switch {
		case allow("stat_file") && strings.Contains(strings.ToLower(state.CurrentStepTitle), "status"):
			return "stat_file"
		case allow("read_file"):
			return "read_file"
		case allow("list_files"):
			return "list_files"
		case allow("stat_file"):
			return "stat_file"
		}
	}
	if len(allowed) == 1 {
		for kind := range allowed {
			return kind
		}
	}
	return ""
}

func buildAgentBrowserFixtureGuidance(state AgentRunState) string {
	if strings.TrimSpace(state.Plan.Mode) != "browser_then_respond" {
		return "No fixed browser fixture URL."
	}
	return fmt.Sprintf(
		"Use the controlled local fixture for this thread. The canonical browser_open URL is %s. Use these stable selectors: browser_type -> %s, browser_click -> %s, browser_extract -> %s. Do not substitute another localhost port such as 3000 or generic selectors such as input, button, or #result.",
		buildControlledBrowserFixtureURL(state.ThreadID),
		controlledBrowserInputSelector,
		controlledBrowserApplySelector,
		controlledBrowserResultSelector,
	)
}

func normalizeAgentActionForState(action AgentAction, state AgentRunState) AgentAction {
	switch strings.TrimSpace(state.Plan.Mode) {
	case "browser_then_respond":
		if action.Type == "browser_open" {
			canonicalURL := buildControlledBrowserFixtureURL(state.ThreadID)
			if !isCanonicalControlledBrowserURL(strings.TrimSpace(action.URL), state.ThreadID) {
				action.URL = canonicalURL
				if strings.TrimSpace(action.ReasoningSummary) == "" {
					action.ReasoningSummary = "Open the controlled browser fixture for the current thread"
				}
			}
		}
		switch action.Type {
		case "browser_type":
			if shouldNormalizeControlledBrowserSelector(action.Selector, action.Type) {
				action.Selector = controlledBrowserInputSelector
			}
		case "browser_click":
			if shouldNormalizeControlledBrowserSelector(action.Selector, action.Type) {
				action.Selector = controlledBrowserApplySelector
			}
		case "browser_extract":
			if shouldNormalizeControlledBrowserSelector(action.Selector, action.Type) {
				action.Selector = controlledBrowserResultSelector
			}
		}
	case "search_then_detailed":
		if action.Type == "search_text" || action.Type == "search_text_detailed" {
			if strings.TrimSpace(action.Query) == "" {
				action.Query = inferSearchGoalQuery(state.Goal)
			}
			if strings.TrimSpace(action.Path) == "" {
				action.Path = inferSearchGoalPath(state.Goal)
			}
		}
	case "filter_then_read":
		switch action.Type {
		case "list_files_filtered":
			if strings.TrimSpace(action.Path) == "" {
				action.Path = inferGoalDirectoryPath(state.Goal)
			}
			if strings.TrimSpace(action.Pattern) == "" {
				action.Pattern = inferGoalPattern(state.Goal)
			}
		case "read_files_batch":
			if len(action.Paths) == 0 {
				action.Paths = inferGoalFilePaths(state.Goal)
			}
		}
	case "list_then_read":
		switch action.Type {
		case "list_files":
			if strings.TrimSpace(action.Path) == "" {
				action.Path = inferGoalDirectoryPath(state.Goal)
			}
		case "read_file":
			if strings.TrimSpace(action.Path) == "" {
				action.Path = inferGoalSingleFilePath(state.Goal)
			}
		case "read_files_batch":
			if len(action.Paths) == 0 {
				if path := inferGoalSingleFilePath(state.Goal); path != "" {
					action.Paths = []string{path}
				}
			}
		}
	case "stat_then_read":
		switch action.Type {
		case "stat_file", "read_file":
			if strings.TrimSpace(action.Path) == "" {
				action.Path = inferGoalSingleFilePath(state.Goal)
			}
		case "read_files_batch":
			if len(action.Paths) == 0 {
				if path := inferGoalSingleFilePath(state.Goal); path != "" {
					action.Paths = []string{path}
				}
			}
		}
	}
	return action
}

var (
	agentGoalPathPattern  = regexp.MustCompile(`[A-Za-z0-9._-]+(?:[\\/][A-Za-z0-9._-]+)+`)
	agentGoalFilePattern  = regexp.MustCompile(`[A-Za-z0-9._-]+\.[A-Za-z0-9._-]+`)
	agentGoalQueryPattern = regexp.MustCompile(`(?i)\bfor\s+["'` + "`" + `]?([A-Za-z0-9_.:-]+)`)
	agentGoalPatternToken = regexp.MustCompile(`\*\.[A-Za-z0-9._-]+`)
	agentLooseStringField = regexp.MustCompile(`(?is)"([A-Za-zA-Z0-9_]+)"\s*:\s*"((?:\\.|[^"\\])*)"`)
	agentLooseLimitField  = regexp.MustCompile(`(?is)"limit"\s*:\s*(\d+)`)
	agentLoosePathsField  = regexp.MustCompile(`(?is)"paths"\s*:\s*\[(.*?)\]`)
	agentLooseQuotedValue = regexp.MustCompile(`(?is)"((?:\\.|[^"\\])*)"`)
)

func inferSearchGoalPath(goal string) string {
	return strings.TrimSpace(agentGoalPathPattern.FindString(goal))
}

func inferSearchGoalQuery(goal string) string {
	matches := agentGoalQueryPattern.FindStringSubmatch(goal)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

func inferGoalPattern(goal string) string {
	return strings.TrimSpace(agentGoalPatternToken.FindString(goal))
}

func inferGoalPathTokens(goal string) []string {
	seen := map[string]struct{}{}
	tokens := make([]string, 0)
	for _, token := range agentGoalPathPattern.FindAllString(goal, -1) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		tokens = append(tokens, token)
	}
	return tokens
}

func inferGoalDirectoryPath(goal string) string {
	for _, token := range inferGoalPathTokens(goal) {
		if !strings.Contains(filepathBase(token), ".") {
			return token
		}
	}
	return ""
}

func inferGoalFilePaths(goal string) []string {
	paths := make([]string, 0)
	for _, token := range inferGoalPathTokens(goal) {
		if strings.Contains(filepathBase(token), ".") {
			paths = append(paths, token)
		}
	}
	return paths
}

func inferGoalSingleFilePath(goal string) string {
	paths := inferGoalFilePaths(goal)
	if len(paths) > 0 {
		return paths[0]
	}
	for _, token := range agentGoalFilePattern.FindAllString(goal, -1) {
		token = strings.TrimSpace(token)
		if token != "" && !strings.HasPrefix(token, "*.") {
			return token
		}
	}
	return ""
}

func deriveDeterministicAgentAction(state AgentRunState) (AgentAction, bool) {
	expected := expectedActionTypesForCurrentStep(state.Plan, state.CompletedActions)
	if len(expected) != 1 {
		return AgentAction{}, false
	}
	switch expected[0] {
	case "list_files_filtered":
		path := inferGoalDirectoryPath(state.Goal)
		pattern := inferGoalPattern(state.Goal)
		if path == "" || pattern == "" {
			return AgentAction{}, false
		}
		return AgentAction{
			Type:             "list_files_filtered",
			Path:             path,
			Pattern:          pattern,
			ReasoningSummary: "Use the required filtered directory listing step from the goal",
		}, true
	case "read_files_batch":
		paths := trimNonEmptyStrings(inferGoalFilePaths(state.Goal))
		if len(paths) == 0 {
			if path := inferGoalSingleFilePath(state.Goal); path != "" {
				paths = []string{path}
			}
		}
		if len(paths) == 0 {
			return AgentAction{}, false
		}
		return AgentAction{
			Type:             "read_files_batch",
			Paths:            paths,
			ReasoningSummary: "Read the explicit file targets referenced in the goal",
		}, true
	case "list_files":
		path := inferGoalDirectoryPath(state.Goal)
		if path == "" {
			return AgentAction{}, false
		}
		return AgentAction{
			Type:             "list_files",
			Path:             path,
			ReasoningSummary: "List the target directory named in the goal",
		}, true
	case "read_file":
		path := inferGoalSingleFilePath(state.Goal)
		if path == "" {
			return AgentAction{}, false
		}
		return AgentAction{
			Type:             "read_file",
			Path:             path,
			ReasoningSummary: "Read the target file named in the goal",
		}, true
	case "stat_file":
		path := inferGoalSingleFilePath(state.Goal)
		if path == "" {
			return AgentAction{}, false
		}
		return AgentAction{
			Type:             "stat_file",
			Path:             path,
			ReasoningSummary: "Check the target file metadata named in the goal",
		}, true
	case "search_text":
		query := inferSearchGoalQuery(state.Goal)
		path := inferSearchGoalPath(state.Goal)
		if query == "" || path == "" {
			return AgentAction{}, false
		}
		return AgentAction{
			Type:             "search_text",
			Query:            query,
			Path:             path,
			ReasoningSummary: "Run the broad search required by the goal",
		}, true
	case "search_text_detailed":
		query := inferSearchGoalQuery(state.Goal)
		path := inferSearchGoalPath(state.Goal)
		if query == "" || path == "" {
			return AgentAction{}, false
		}
		return AgentAction{
			Type:             "search_text_detailed",
			Query:            query,
			Path:             path,
			Limit:            normalizedDetailedSearchLimit(20),
			ReasoningSummary: "Inspect detailed line hits for the required search query",
		}, true
	case "browser_open":
		return AgentAction{
			Type:             "browser_open",
			URL:              buildControlledBrowserFixtureURL(state.ThreadID),
			ReasoningSummary: "Open the controlled browser fixture for this thread",
		}, true
	case "browser_type":
		return AgentAction{
			Type:             "browser_type",
			Selector:         controlledBrowserInputSelector,
			Text:             "browser demo text",
			ReasoningSummary: "Type the stable controlled-browser input text",
		}, true
	case "browser_click":
		return AgentAction{
			Type:             "browser_click",
			Selector:         controlledBrowserApplySelector,
			ReasoningSummary: "Click the stable controlled-browser apply control",
		}, true
	case "browser_extract":
		return AgentAction{
			Type:             "browser_extract",
			Selector:         controlledBrowserResultSelector,
			ReasoningSummary: "Extract the stable controlled-browser result text",
		}, true
	case "browser_screenshot":
		return AgentAction{
			Type:             "browser_screenshot",
			ReasoningSummary: "Capture a screenshot of the current browser tab",
		}, true
	case "apply_patch":
		path := inferGoalSingleFilePath(state.Goal)
		if path == "" {
			return AgentAction{}, false
		}
		line := inferGoalPatchLine(state.Goal)
		if line == "" {
			return AgentAction{}, false
		}
		patch := buildDeterministicLineAppendPatch(path, line)
		return AgentAction{
			Type:             "apply_patch",
			Path:             path,
			Patch:            patch,
			ReasoningSummary: "Apply the exact single-line patch requested by the goal",
		}, true
	case "respond":
		response := deriveDeterministicAgentResponse(state)
		if response == "" {
			return AgentAction{}, false
		}
		return AgentAction{
			Type:             "respond",
			Response:         response,
			ReasoningSummary: "Answer concisely using the completed task results",
		}, true
	default:
		return AgentAction{}, false
	}
}

func inferGoalPatchLine(goal string) string {
	lower := strings.ToLower(goal)
	markers := []string{
		"adds one line saying ",
		"adding one line saying ",
	}
	index := -1
	markerLength := 0
	for _, marker := range markers {
		index = strings.Index(lower, marker)
		if index >= 0 {
			markerLength = len(marker)
			break
		}
	}
	if index < 0 {
		return ""
	}
	start := index + markerLength
	rest := goal[start:]
	for _, separator := range []string{", then", " then", ". Do not", ".do not"} {
		if split := strings.Index(strings.ToLower(rest), separator); split >= 0 {
			rest = rest[:split]
			break
		}
	}
	return strings.TrimSpace(strings.Trim(rest, `"'`+"`"))
}

func buildDeterministicLineAppendPatch(path string, line string) string {
	return fmt.Sprintf(
		"*** Begin Patch\n*** Update File: %s\n@@\n+%s\n*** End Patch",
		path,
		line,
	)
}

func deriveDeterministicAgentResponse(state AgentRunState) string {
	switch strings.TrimSpace(state.Plan.Mode) {
	case "browser_then_respond":
		return "Controlled browser result: browser demo text"
	case "patch_then_respond":
		if text := inferGoalPatchLine(state.Goal); text != "" {
			return fmt.Sprintf("I added the line %q.", text)
		}
	case "filter_then_read":
		if paths := inferGoalFilePaths(state.Goal); len(paths) > 0 {
			return fmt.Sprintf("I inspected %s.", strings.Join(paths, " and "))
		}
	case "list_then_read":
		if path := inferGoalSingleFilePath(state.Goal); path != "" {
			return fmt.Sprintf("I listed the target directory and inspected %s.", path)
		}
	case "stat_then_read":
		if path := inferGoalSingleFilePath(state.Goal); path != "" {
			return fmt.Sprintf("I checked %s and read its contents.", path)
		}
	case "search_then_detailed":
		if query := inferSearchGoalQuery(state.Goal); query != "" {
			if path := inferSearchGoalPath(state.Goal); path != "" {
				return fmt.Sprintf("I searched %s in %s and inspected the detailed matches.", query, path)
			}
			return fmt.Sprintf("I searched %s and inspected the detailed matches.", query)
		}
	}
	return ""
}

func recoverAgentActionFromMalformedJSON(raw string, state AgentRunState) (AgentAction, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return AgentAction{}, false
	}
	stringFields := map[string]string{}
	for _, match := range agentLooseStringField.FindAllStringSubmatch(raw, -1) {
		if len(match) < 3 {
			continue
		}
		value, err := strconv.Unquote(`"` + match[2] + `"`)
		if err != nil {
			value = match[2]
		}
		stringFields[match[1]] = strings.TrimSpace(value)
	}
	action := AgentAction{
		Type:             firstNonEmptyString(stringFields["type"]),
		ReasoningSummary: stringFields["reasoningSummary"],
		URL:              stringFields["url"],
		TabID:            stringFields["tabId"],
		Selector:         stringFields["selector"],
		Text:             stringFields["text"],
		Path:             stringFields["path"],
		Pattern:          stringFields["pattern"],
		Query:            stringFields["query"],
		Patch:            stringFields["patch"],
		Response: firstNonEmptyString(
			stringFields["response"],
			stringFields["content"],
			stringFields["answer"],
			stringFields["final"],
			stringFields["message"],
		),
	}
	if match := agentLooseLimitField.FindStringSubmatch(raw); len(match) > 1 {
		if limit, err := strconv.Atoi(match[1]); err == nil {
			action.Limit = limit
		}
	}
	if match := agentLoosePathsField.FindStringSubmatch(raw); len(match) > 1 {
		for _, item := range agentLooseQuotedValue.FindAllStringSubmatch(match[1], -1) {
			if len(item) < 2 {
				continue
			}
			value, err := strconv.Unquote(`"` + item[1] + `"`)
			if err != nil {
				value = item[1]
			}
			value = strings.TrimSpace(value)
			if value != "" {
				action.Paths = append(action.Paths, value)
			}
		}
	}

	action.Type = strings.TrimSpace(action.Type)
	if action.Type == "response" || action.Type == "result" || action.Type == "tool_result" {
		action.Type = "respond"
	}
	if action.Type == "tool_call" {
		action.Type = ""
	}
	action.Type = normalizeAgentActionType(action.Type)
	if action.Type == "" {
		action.Type = inferAgentActionType(action, state)
	}
	action = inheritAgentActionContext(action, state)
	action = normalizeAgentActionForState(action, state)
	action = populateAgentResponseFallback(raw, action)
	if strings.TrimSpace(action.Type) == "" {
		return AgentAction{}, false
	}
	return action, true
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func filepathBase(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.ReplaceAll(path, "\\", "/")
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func buildControlledBrowserFixtureURL(threadID string) string {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return fmt.Sprintf("%s?%s=1&pane=%s", agentBrowserFixtureURL, agentBrowserFixtureKey, agentBrowserFixturePane)
	}
	return fmt.Sprintf(
		"%s?%s=1&pane=%s&threadId=%s",
		agentBrowserFixtureURL,
		agentBrowserFixtureKey,
		agentBrowserFixturePane,
		threadID,
	)
}

func isCanonicalControlledBrowserURL(raw string, threadID string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" {
		return false
	}
	if parsed.Host != "127.0.0.1:5174" && parsed.Host != "localhost:5174" {
		return false
	}
	query := parsed.Query()
	if query.Get(agentBrowserFixtureKey) != "1" {
		return false
	}
	if query.Get("pane") != agentBrowserFixturePane {
		return false
	}
	if strings.TrimSpace(threadID) == "" {
		return true
	}
	return query.Get("threadId") == threadID
}

func shouldNormalizeControlledBrowserSelector(selector string, actionType string) bool {
	normalized := strings.TrimSpace(strings.ToLower(selector))
	switch actionType {
	case "browser_type":
		return normalized == "" || normalized == "input" || normalized == "#input" || normalized == "textarea" || strings.Contains(normalized, "stable-input")
	case "browser_click":
		return normalized == "" || normalized == "button" || normalized == "#apply" || normalized == ".apply" || strings.Contains(normalized, "stable-apply")
	case "browser_extract":
		return normalized == "" || normalized == "#result" || normalized == ".result" || normalized == "result" || strings.Contains(normalized, "stable-result")
	default:
		return false
	}
}

func extractFirstJSONObject(raw string) (string, error) {
	start := strings.IndexByte(raw, '{')
	if start < 0 {
		return "", fmt.Errorf("no json object found")
	}

	depth := 0
	inString := false
	escaped := false
	for index := start; index < len(raw); index++ {
		ch := raw[index]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return raw[start : index+1], nil
			}
		}
	}

	return "", fmt.Errorf("json object is not closed")
}

func containsAnySubstring(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

func sanitizeLooseJSONStringLiterals(raw string) string {
	var builder strings.Builder
	builder.Grow(len(raw) + 16)
	inString := false
	escaped := false
	for index := 0; index < len(raw); index++ {
		ch := raw[index]
		if inString {
			if escaped {
				builder.WriteByte(ch)
				escaped = false
				continue
			}
			switch ch {
			case '\\':
				builder.WriteByte(ch)
				escaped = true
			case '"':
				builder.WriteByte(ch)
				inString = false
			case '\n':
				builder.WriteString(`\n`)
			case '\r':
				builder.WriteString(`\r`)
			case '\t':
				builder.WriteString(`\t`)
			default:
				builder.WriteByte(ch)
			}
			continue
		}
		if ch == '"' {
			inString = true
		}
		builder.WriteByte(ch)
	}
	return builder.String()
}

func childTaskTitle(prefix string, suffix string) string {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return prefix
	}
	return fmt.Sprintf("%s %s", prefix, suffix)
}

func fallbackPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "."
	}
	return path
}

func inferPatchPath(patch string) string {
	targets, err := ExtractPatchTargets(patch)
	if err != nil || len(targets) == 0 {
		return ""
	}
	return targets[0]
}

func trimNonEmptyStrings(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		result = append(result, item)
	}
	return result
}

func strPtr(value string) *string {
	return &value
}

func fallbackAgentText(value string, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
