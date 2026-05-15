package router

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/handler"
	"llmtrace/internal/platform/xerror"
	"llmtrace/internal/platform/xlog"
)

func TestNewRegistersCodexStyleRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log, err := xlog.New("debug")
	require.NoError(t, err)

	engine, err := New(log, "gen-code", []string{"127.0.0.1"}, false, Handlers{
		Health:  handler.NewHealthHandler(),
		Runtime: handler.NewRuntimeHandler(stubRuntimeService{}),
	})
	require.NoError(t, err)

	testCases := []struct {
		name           string
		method         string
		path           string
		body           string
		wantStatusCode int
		wantBody       []string
	}{
		{
			name:           "healthz",
			method:         http.MethodGet,
			path:           "/healthz",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"status":"ok"`},
		},
		{
			name:           "runtime status",
			method:         http.MethodGet,
			path:           "/api/runtime/status",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"state":"ready"`, `"ready":true`, `"workspaceId":"gen-code"`, `"threadCount":2`, `"activeThreadId":"thread-1"`, `"stateStore":"sqlite"`, `"statePath":"D:\\GOWorks\\gen-code-heji\\gen-code\\.gen-code\\state.db"`},
		},
		{
			name:           "workspace",
			method:         http.MethodGet,
			path:           "/api/workspace",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"gen-code"`, `"projectRoot":"D:\\GOWorks\\gen-code-heji\\gen-code"`},
		},
		{
			name:           "threads",
			method:         http.MethodGet,
			path:           "/api/threads",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"thread-1","workspaceId":"gen-code","name":"Thread 1"`},
		},
		{
			name:           "create thread",
			method:         http.MethodPost,
			path:           "/api/threads",
			body:           `{"name":"Design Thread","activeModel":"gpt-5","permissionMode":"workspace-write"}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"thread-3"`, `"name":"Design Thread"`, `"permissionMode":"workspace-write"`},
		},
		{
			name:           "thread by id",
			method:         http.MethodGet,
			path:           "/api/threads/thread-1",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"thread-1"`, `"isActive":true`},
		},
		{
			name:           "activate thread",
			method:         http.MethodPost,
			path:           "/api/threads/thread-2/activate",
			body:           `{}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"thread-2"`, `"isActive":true`},
		},
		{
			name:           "tasks",
			method:         http.MethodGet,
			path:           "/api/threads/thread-1/tasks",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"task-1","threadId":"thread-1","title":"Task 1","status":"queued","kind":"thread.message.append"`},
		},
		{
			name:           "messages",
			method:         http.MethodGet,
			path:           "/api/threads/thread-1/messages",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"message-1","threadId":"thread-1","role":"user","content":"Draft spec"`},
		},
		{
			name:           "append message",
			method:         http.MethodPost,
			path:           "/api/threads/thread-1/messages",
			body:           `{"role":"assistant","content":"Sure"}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"message-2"`, `"role":"assistant"`, `"content":"Sure"`},
		},
		{
			name:           "tool calls",
			method:         http.MethodGet,
			path:           "/api/threads/thread-1/tool-calls",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"toolcall-1","threadId":"thread-1","toolId":"bridge.check","status":"completed","summary":"Bridge ok"`},
		},
		{
			name:           "append tool call",
			method:         http.MethodPost,
			path:           "/api/threads/thread-1/tool-calls",
			body:           `{"toolId":"skills.list","status":"queued","summary":"Pending"}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"toolcall-2"`, `"toolId":"skills.list"`, `"summary":"Pending"`},
		},
		{
			name:           "artifacts",
			method:         http.MethodGet,
			path:           "/api/threads/thread-1/artifacts",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"artifact-1","threadId":"thread-1","path":"D:\\artifacts\\spec.md","kind":"markdown"`},
		},
		{
			name:           "append artifact",
			method:         http.MethodPost,
			path:           "/api/threads/thread-1/artifacts",
			body:           `{"path":"D:\\artifacts\\report.json","kind":"json"}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"artifact-2"`, `"kind":"json"`},
		},
		{
			name:           "runtime flags",
			method:         http.MethodGet,
			path:           "/api/threads/thread-1/runtime-flags",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"threadId":"thread-1","key":"preview","value":"ready"`},
		},
		{
			name:           "set runtime flag",
			method:         http.MethodPost,
			path:           "/api/threads/thread-1/runtime-flags",
			body:           `{"key":"draft","value":"saved"}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"threadId":"thread-1"`, `"key":"draft"`, `"value":"saved"`},
		},
		{
			name:           "create task",
			method:         http.MethodPost,
			path:           "/api/threads/thread-1/tasks",
			body:           `{"title":"Draft spec","kind":"thread.message.append","input":"{\"role\":\"user\",\"content\":\"Draft spec\"}"}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"task-2"`, `"threadId":"thread-1"`, `"title":"Draft spec"`, `"kind":"thread.message.append"`, `"updatedAt":"2026-05-15T00:00:00Z"`},
		},
		{
			name:           "run task",
			method:         http.MethodPost,
			path:           "/api/threads/thread-1/tasks/task-1/run",
			body:           `{}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"task-1"`, `"status":"completed"`, `"resultSummary":"message appended"`},
		},
		{
			name:           "update task status",
			method:         http.MethodPost,
			path:           "/api/threads/thread-1/tasks/task-1/status",
			body:           `{"status":"running"}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"id":"task-1"`, `"status":"running"`, `"updatedAt":"2026-05-15T00:05:00Z"`},
		},
		{
			name:           "events",
			method:         http.MethodGet,
			path:           "/api/threads/thread-1/events",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"event-1","threadId":"thread-1","type":"thread.created","message":"Thread 1 created"`},
		},
		{
			name:           "events stream",
			method:         http.MethodGet,
			path:           "/api/threads/thread-1/events/stream",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`event: thread.created`, `data: {"id":"event-1","threadId":"thread-1","type":"thread.created","message":"Thread 1 created","createdAt":"2026-05-15T00:00:00Z"}`},
		},
		{
			name:           "skills",
			method:         http.MethodGet,
			path:           "/api/skills",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"skill-1","group":"codex","name":"Skill One","description":"Skill description","source":"codex"}`},
		},
		{
			name:           "tools",
			method:         http.MethodGet,
			path:           "/api/tools",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"tool-1","name":"Tool One","description":"Tool description","permissionMode":"ask-user","source":"runtime"}`},
		},
		{
			name:           "mcp servers",
			method:         http.MethodGet,
			path:           "/api/mcp/servers",
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"items":[{"id":"server-1","source":"node_modules","enabled":true,"toolCount":2,"resourceCount":1,"status":"enabled"}`},
		},
		{
			name:           "bridge check",
			method:         http.MethodPost,
			path:           "/api/bridge/check",
			body:           `{"bridge":"stdio","target":"server-1"}`,
			wantStatusCode: http.StatusOK,
			wantBody:       []string{`"ok":true`, `"echo":{"bridge":"stdio","target":"server-1"}`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			if tc.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}

			rec := httptest.NewRecorder()
			engine.ServeHTTP(rec, req)

			require.Equal(t, tc.wantStatusCode, rec.Code)
			if tc.name != "events stream" {
				require.Contains(t, rec.Body.String(), `"code":0`)
			}
			for _, item := range tc.wantBody {
				require.Contains(t, rec.Body.String(), item)
			}
		})
	}
}

func TestNewReturnsRouteErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log, err := xlog.New("debug")
	require.NoError(t, err)

	engine, err := New(log, "gen-code", []string{"127.0.0.1"}, false, Handlers{
		Runtime: handler.NewRuntimeHandler(errorRuntimeService{}),
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/api/runtime/status", nil)
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":2001`)
}

func TestNewRejectsInvalidBridgePayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log, err := xlog.New("debug")
	require.NoError(t, err)

	engine, err := New(log, "gen-code", []string{"127.0.0.1"}, false, Handlers{
		Runtime: handler.NewRuntimeHandler(stubRuntimeService{}),
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/bridge/check", bytes.NewBufferString(`oops`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":1001`)
}

func TestNewRejectsInvalidCreateThreadPermission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log, err := xlog.New("debug")
	require.NoError(t, err)

	engine, err := New(log, "gen-code", []string{"127.0.0.1"}, false, Handlers{
		Runtime: handler.NewRuntimeHandler(stubRuntimeService{}),
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/threads", bytes.NewBufferString(`{"permissionMode":"invalid-mode"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":1003`)
}

func TestNewRejectsInvalidCreateTaskThread(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log, err := xlog.New("debug")
	require.NoError(t, err)

	engine, err := New(log, "gen-code", []string{"127.0.0.1"}, false, Handlers{
		Runtime: handler.NewRuntimeHandler(stubRuntimeService{}),
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/threads/missing/tasks", bytes.NewBufferString(`{"title":"Draft spec"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":1004`)
}

func TestNewRejectsInvalidTaskStatusPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	log, err := xlog.New("debug")
	require.NoError(t, err)

	engine, err := New(log, "gen-code", []string{"127.0.0.1"}, false, Handlers{
		Runtime: handler.NewRuntimeHandler(stubRuntimeService{}),
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/api/threads/thread-1/tasks/task-1/status", bytes.NewBufferString(`{"status":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	engine.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), `"code":1001`)
}

type stubRuntimeService struct{}

func (stubRuntimeService) Status(context.Context) (runtimecontract.Status, error) {
	return runtimecontract.Status{
		State:          "ready",
		Ready:          true,
		StateStore:     "sqlite",
		StatePath:      `D:\GOWorks\gen-code-heji\gen-code\.gen-code\state.db`,
		WorkspaceID:    "gen-code",
		ProjectRoot:    `D:\GOWorks\gen-code-heji\gen-code`,
		ThreadCount:    2,
		ActiveThreadID: "thread-1",
	}, nil
}

func (stubRuntimeService) Workspace(context.Context) (runtimecontract.WorkspaceDescriptor, error) {
	return runtimecontract.WorkspaceDescriptor{
		ID:                "gen-code",
		ProjectRoot:       `D:\GOWorks\gen-code-heji\gen-code`,
		SharedDocsRoot:    `D:\GOWorks\gen-code-heji\gen-code\docs`,
		CreatedAt:         "2026-05-15T00:00:00Z",
		ActiveThreadCount: 2,
	}, nil
}

func (stubRuntimeService) Threads(context.Context) ([]runtimecontract.ThreadDescriptor, error) {
	return []runtimecontract.ThreadDescriptor{
		{
			ID:                  "thread-1",
			WorkspaceID:         "gen-code",
			Name:                "Thread 1",
			Status:              "idle",
			PermissionMode:      "ask-user",
			MessageHistoryCount: 0,
			ToolCallCount:       0,
			ArtifactCount:       0,
			CreatedAt:           "2026-05-15T00:00:00Z",
			IsActive:            true,
		},
		{
			ID:                  "thread-2",
			WorkspaceID:         "gen-code",
			Name:                "Thread 2",
			Status:              "idle",
			PermissionMode:      "ask-user",
			MessageHistoryCount: 0,
			ToolCallCount:       0,
			ArtifactCount:       0,
			CreatedAt:           "2026-05-15T00:00:00Z",
			IsActive:            false,
		},
	}, nil
}

func (stubRuntimeService) CreateThread(_ context.Context, request runtimecontract.CreateThreadRequest) (runtimecontract.ThreadDescriptor, error) {
	if request.PermissionMode == "invalid-mode" {
		return runtimecontract.ThreadDescriptor{}, xerror.BadRequest(1003, `invalid permission mode "invalid-mode"`)
	}
	return runtimecontract.ThreadDescriptor{
		ID:                  "thread-3",
		WorkspaceID:         "gen-code",
		Name:                request.Name,
		Status:              "idle",
		ActiveModel:         request.ActiveModel,
		PermissionMode:      request.PermissionMode,
		MessageHistoryCount: 0,
		ToolCallCount:       0,
		ArtifactCount:       0,
		CreatedAt:           "2026-05-15T00:00:00Z",
		IsActive:            false,
	}, nil
}

func (stubRuntimeService) Thread(_ context.Context, id string) (runtimecontract.ThreadDescriptor, error) {
	return runtimecontract.ThreadDescriptor{
		ID:                  id,
		WorkspaceID:         "gen-code",
		Name:                "Thread 1",
		Status:              "idle",
		PermissionMode:      "ask-user",
		MessageHistoryCount: 0,
		ToolCallCount:       0,
		ArtifactCount:       0,
		CreatedAt:           "2026-05-15T00:00:00Z",
		IsActive:            id == "thread-1",
	}, nil
}

func (stubRuntimeService) ActivateThread(_ context.Context, id string) (runtimecontract.ThreadDescriptor, error) {
	return runtimecontract.ThreadDescriptor{
		ID:                  id,
		WorkspaceID:         "gen-code",
		Name:                "Thread 2",
		Status:              "idle",
		PermissionMode:      "ask-user",
		MessageHistoryCount: 0,
		ToolCallCount:       0,
		ArtifactCount:       0,
		CreatedAt:           "2026-05-15T00:00:00Z",
		IsActive:            true,
	}, nil
}

func (stubRuntimeService) Tasks(context.Context, string) ([]runtimecontract.TaskDescriptor, error) {
	return []runtimecontract.TaskDescriptor{{
		ID:            "task-1",
		ThreadID:      "thread-1",
		Title:         "Task 1",
		Status:        "queued",
		Kind:          "thread.message.append",
		InputSummary:  `{"role":"user","content":"Draft spec"}`,
		ResultSummary: "",
		CreatedAt:     "2026-05-15T00:00:00Z",
		UpdatedAt:     "2026-05-15T00:00:00Z",
	}}, nil
}

func (stubRuntimeService) Messages(context.Context, string) ([]runtimecontract.MessageDescriptor, error) {
	return []runtimecontract.MessageDescriptor{{
		ID:        "message-1",
		ThreadID:  "thread-1",
		Role:      "user",
		Content:   "Draft spec",
		CreatedAt: "2026-05-15T00:00:00Z",
	}}, nil
}

func (stubRuntimeService) AppendMessage(_ context.Context, threadID string, request runtimecontract.CreateMessageRequest) (runtimecontract.MessageDescriptor, error) {
	return runtimecontract.MessageDescriptor{
		ID:        "message-2",
		ThreadID:  threadID,
		Role:      request.Role,
		Content:   request.Content,
		CreatedAt: "2026-05-15T00:01:00Z",
	}, nil
}

func (stubRuntimeService) ToolCalls(context.Context, string) ([]runtimecontract.ToolCallDescriptor, error) {
	return []runtimecontract.ToolCallDescriptor{{
		ID:        "toolcall-1",
		ThreadID:  "thread-1",
		ToolID:    "bridge.check",
		Status:    "completed",
		Summary:   "Bridge ok",
		CreatedAt: "2026-05-15T00:00:00Z",
	}}, nil
}

func (stubRuntimeService) AppendToolCall(_ context.Context, threadID string, request runtimecontract.CreateToolCallRequest) (runtimecontract.ToolCallDescriptor, error) {
	return runtimecontract.ToolCallDescriptor{
		ID:        "toolcall-2",
		ThreadID:  threadID,
		ToolID:    request.ToolID,
		Status:    request.Status,
		Summary:   request.Summary,
		CreatedAt: "2026-05-15T00:02:00Z",
	}, nil
}

func (stubRuntimeService) Artifacts(context.Context, string) ([]runtimecontract.ArtifactDescriptor, error) {
	return []runtimecontract.ArtifactDescriptor{{
		ID:        "artifact-1",
		ThreadID:  "thread-1",
		Path:      `D:\artifacts\spec.md`,
		Kind:      "markdown",
		CreatedAt: "2026-05-15T00:00:00Z",
	}}, nil
}

func (stubRuntimeService) AppendArtifact(_ context.Context, threadID string, request runtimecontract.CreateArtifactRequest) (runtimecontract.ArtifactDescriptor, error) {
	return runtimecontract.ArtifactDescriptor{
		ID:        "artifact-2",
		ThreadID:  threadID,
		Path:      request.Path,
		Kind:      request.Kind,
		CreatedAt: "2026-05-15T00:03:00Z",
	}, nil
}

func (stubRuntimeService) RuntimeFlags(context.Context, string) ([]runtimecontract.RuntimeFlagDescriptor, error) {
	return []runtimecontract.RuntimeFlagDescriptor{{
		ThreadID:  "thread-1",
		Key:       "preview",
		Value:     "ready",
		UpdatedAt: "2026-05-15T00:00:00Z",
	}}, nil
}

func (stubRuntimeService) SetRuntimeFlag(_ context.Context, threadID string, request runtimecontract.SetRuntimeFlagRequest) (runtimecontract.RuntimeFlagDescriptor, error) {
	return runtimecontract.RuntimeFlagDescriptor{
		ThreadID:  threadID,
		Key:       request.Key,
		Value:     request.Value,
		UpdatedAt: "2026-05-15T00:04:00Z",
	}, nil
}

func (stubRuntimeService) CreateTask(_ context.Context, threadID string, request runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	if threadID == "missing" {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
	}
	if request.Title == "" || request.Kind == "" {
		return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1005, "invalid create task payload")
	}
	return runtimecontract.TaskDescriptor{
		ID:            "task-2",
		ThreadID:      threadID,
		Title:         request.Title,
		Status:        "queued",
		Kind:          request.Kind,
		InputSummary:  request.Input,
		ResultSummary: "",
		CreatedAt:     "2026-05-15T00:00:00Z",
		UpdatedAt:     "2026-05-15T00:00:00Z",
	}, nil
}

func (stubRuntimeService) RunTask(_ context.Context, threadID string, taskID string, _ runtimecontract.RunTaskRequest) (runtimecontract.TaskDescriptor, error) {
	if threadID == "missing" {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
	}
	if taskID == "missing" {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1007, "task not found")
	}
	return runtimecontract.TaskDescriptor{
		ID:            taskID,
		ThreadID:      threadID,
		Title:         "Task 1",
		Status:        "completed",
		Kind:          "thread.message.append",
		InputSummary:  `{"role":"user","content":"Draft spec"}`,
		ResultSummary: "message appended",
		CreatedAt:     "2026-05-15T00:00:00Z",
		UpdatedAt:     "2026-05-15T00:05:00Z",
	}, nil
}

func (stubRuntimeService) UpdateTaskStatus(_ context.Context, threadID string, taskID string, request runtimecontract.UpdateTaskStatusRequest) (runtimecontract.TaskDescriptor, error) {
	if threadID == "missing" {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1004, "thread not found")
	}
	if taskID == "missing" {
		return runtimecontract.TaskDescriptor{}, xerror.NotFound(1007, "task not found")
	}
	if request.Status == "" || request.Status == "paused" {
		return runtimecontract.TaskDescriptor{}, xerror.BadRequest(1001, "invalid task status")
	}

	return runtimecontract.TaskDescriptor{
		ID:        taskID,
		ThreadID:  threadID,
		Title:     "Task 1",
		Status:    request.Status,
		CreatedAt: "2026-05-15T00:00:00Z",
		UpdatedAt: "2026-05-15T00:05:00Z",
	}, nil
}

func (stubRuntimeService) Events(context.Context, string) ([]runtimecontract.EventDescriptor, error) {
	return []runtimecontract.EventDescriptor{{
		ID:        "event-1",
		ThreadID:  "thread-1",
		Type:      "thread.created",
		Message:   "Thread 1 created",
		CreatedAt: "2026-05-15T00:00:00Z",
	}}, nil
}

func (stubRuntimeService) StreamEvents(_ context.Context, threadID string) (<-chan runtimecontract.EventDescriptor, error) {
	if threadID == "missing" {
		return nil, xerror.NotFound(1004, "thread not found")
	}

	ch := make(chan runtimecontract.EventDescriptor, 1)
	ch <- runtimecontract.EventDescriptor{
		ID:        "event-1",
		ThreadID:  "thread-1",
		Type:      "thread.created",
		Message:   "Thread 1 created",
		CreatedAt: "2026-05-15T00:00:00Z",
	}
	close(ch)
	return ch, nil
}

func (stubRuntimeService) Skills(context.Context) ([]runtimecontract.Skill, error) {
	return []runtimecontract.Skill{{
		ID:          "skill-1",
		Group:       "codex",
		Name:        "Skill One",
		Description: "Skill description",
		Source:      "codex",
	}}, nil
}

func (stubRuntimeService) Tools(context.Context) ([]runtimecontract.Tool, error) {
	return []runtimecontract.Tool{{
		ID:          "tool-1",
		Name:        "Tool One",
		Description: "Tool description",
		Permission:  "ask-user",
		Source:      "runtime",
	}}, nil
}

func (stubRuntimeService) MCPServers(context.Context) ([]runtimecontract.MCPServer, error) {
	return []runtimecontract.MCPServer{{
		ID:            "server-1",
		Source:        "node_modules",
		Enabled:       true,
		ToolCount:     2,
		ResourceCount: 1,
		Status:        "enabled",
	}}, nil
}

func (stubRuntimeService) CheckBridge(_ context.Context, request map[string]any) (runtimecontract.BridgeCheckResult, error) {
	return runtimecontract.BridgeCheckResult{
		OK: true,
		Details: map[string]any{
			"echo": request,
		},
	}, nil
}

type errorRuntimeService struct{}

func (errorRuntimeService) Status(context.Context) (runtimecontract.Status, error) {
	return runtimecontract.Status{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Skills(context.Context) ([]runtimecontract.Skill, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Workspace(context.Context) (runtimecontract.WorkspaceDescriptor, error) {
	return runtimecontract.WorkspaceDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Threads(context.Context) ([]runtimecontract.ThreadDescriptor, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) CreateThread(context.Context, runtimecontract.CreateThreadRequest) (runtimecontract.ThreadDescriptor, error) {
	return runtimecontract.ThreadDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Thread(context.Context, string) (runtimecontract.ThreadDescriptor, error) {
	return runtimecontract.ThreadDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) ActivateThread(context.Context, string) (runtimecontract.ThreadDescriptor, error) {
	return runtimecontract.ThreadDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Tasks(context.Context, string) ([]runtimecontract.TaskDescriptor, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) CreateTask(context.Context, string, runtimecontract.CreateTaskRequest) (runtimecontract.TaskDescriptor, error) {
	return runtimecontract.TaskDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) RunTask(context.Context, string, string, runtimecontract.RunTaskRequest) (runtimecontract.TaskDescriptor, error) {
	return runtimecontract.TaskDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Messages(context.Context, string) ([]runtimecontract.MessageDescriptor, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) AppendMessage(context.Context, string, runtimecontract.CreateMessageRequest) (runtimecontract.MessageDescriptor, error) {
	return runtimecontract.MessageDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) ToolCalls(context.Context, string) ([]runtimecontract.ToolCallDescriptor, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) AppendToolCall(context.Context, string, runtimecontract.CreateToolCallRequest) (runtimecontract.ToolCallDescriptor, error) {
	return runtimecontract.ToolCallDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Artifacts(context.Context, string) ([]runtimecontract.ArtifactDescriptor, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) AppendArtifact(context.Context, string, runtimecontract.CreateArtifactRequest) (runtimecontract.ArtifactDescriptor, error) {
	return runtimecontract.ArtifactDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) RuntimeFlags(context.Context, string) ([]runtimecontract.RuntimeFlagDescriptor, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) SetRuntimeFlag(context.Context, string, runtimecontract.SetRuntimeFlagRequest) (runtimecontract.RuntimeFlagDescriptor, error) {
	return runtimecontract.RuntimeFlagDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) UpdateTaskStatus(context.Context, string, string, runtimecontract.UpdateTaskStatusRequest) (runtimecontract.TaskDescriptor, error) {
	return runtimecontract.TaskDescriptor{}, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Events(context.Context, string) ([]runtimecontract.EventDescriptor, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) StreamEvents(context.Context, string) (<-chan runtimecontract.EventDescriptor, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) Tools(context.Context) ([]runtimecontract.Tool, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) MCPServers(context.Context) ([]runtimecontract.MCPServer, error) {
	return nil, xerror.Internal(2001, "runtime unavailable")
}

func (errorRuntimeService) CheckBridge(context.Context, map[string]any) (runtimecontract.BridgeCheckResult, error) {
	return runtimecontract.BridgeCheckResult{}, xerror.Internal(2001, "runtime unavailable")
}
