package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/middleware"
	"llmtrace/internal/platform/xresp"
)

const (
	errCodeInvalidBridgeCheckPayload = 1001
	errCodeInvalidCreateThreadBody   = 1002
	errCodeInvalidCreateWorkspaceBody = 1004
	errCodeInvalidCreateTaskBody     = 1005
	errCodeInvalidTaskStatusBody     = 1001
	errCodeInvalidRunTaskBody        = 1011
	errCodeInvalidApproveTaskBody    = 1012
	errCodeInvalidRejectTaskBody     = 1013
	errCodeInvalidThreadPreferencesBody = 1014
	errCodeInvalidMessageBody        = 1006
	errCodeInvalidToolCallBody       = 1008
	errCodeInvalidArtifactBody       = 1009
	errCodeInvalidRuntimeFlagBody    = 1010
)

// RuntimeHandler exposes codex-style runtime discovery and bridge endpoints.
type RuntimeHandler struct {
	runtime runtimecontract.Service
}

// NewRuntimeHandler creates the runtime API handler.
func NewRuntimeHandler(runtime runtimecontract.Service) *RuntimeHandler {
	if runtime == nil {
		runtime = runtimecontract.NewNoopService()
	}

	return &RuntimeHandler{
		runtime: runtime,
	}
}

// Status returns the current runtime state.
func (h *RuntimeHandler) Status(c *gin.Context) {
	data, err := h.runtime.Status(c.Request.Context())
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Workspace returns the current workspace descriptor.
func (h *RuntimeHandler) Workspace(c *gin.Context) {
	data, err := h.runtime.Workspace(c.Request.Context())
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Workspaces returns the registered workspace descriptors.
func (h *RuntimeHandler) Workspaces(c *gin.Context) {
	data, err := h.runtime.Workspaces(c.Request.Context())
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// CreateWorkspace registers a new workspace root.
func (h *RuntimeHandler) CreateWorkspace(c *gin.Context) {
	var payload runtimecontract.CreateWorkspaceRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidCreateWorkspaceBody, "invalid create workspace payload")
		return
	}

	data, err := h.runtime.CreateWorkspace(c.Request.Context(), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// ActivateWorkspace marks a workspace as active.
func (h *RuntimeHandler) ActivateWorkspace(c *gin.Context) {
	data, err := h.runtime.ActivateWorkspace(c.Request.Context(), c.Param("id"), runtimecontract.ActivateWorkspaceRequest{})
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Threads returns the registered thread descriptors.
func (h *RuntimeHandler) Threads(c *gin.Context) {
	data, err := h.runtime.Threads(c.Request.Context())
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// CreateThread registers a new thread.
func (h *RuntimeHandler) CreateThread(c *gin.Context) {
	var payload runtimecontract.CreateThreadRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidCreateThreadBody, "invalid create thread payload")
		return
	}

	data, err := h.runtime.CreateThread(c.Request.Context(), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Thread returns a single thread descriptor by id.
func (h *RuntimeHandler) Thread(c *gin.Context) {
	data, err := h.runtime.Thread(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// ActivateThread marks a thread as active.
func (h *RuntimeHandler) ActivateThread(c *gin.Context) {
	data, err := h.runtime.ActivateThread(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// UpdateThreadPreferences updates thread-level model and reasoning preferences.
func (h *RuntimeHandler) UpdateThreadPreferences(c *gin.Context) {
	var payload runtimecontract.UpdateThreadPreferencesRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidThreadPreferencesBody, "invalid thread preferences payload")
		return
	}

	data, err := h.runtime.UpdateThreadPreferences(c.Request.Context(), c.Param("id"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Tasks returns the tasks under the given thread.
func (h *RuntimeHandler) Tasks(c *gin.Context) {
	data, err := h.runtime.Tasks(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// CreateTask registers a task under the given thread.
func (h *RuntimeHandler) CreateTask(c *gin.Context) {
	var payload runtimecontract.CreateTaskRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidCreateTaskBody, "invalid create task payload")
		return
	}

	data, err := h.runtime.CreateTask(c.Request.Context(), c.Param("id"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// RunTask executes an existing thread-local task.
func (h *RuntimeHandler) RunTask(c *gin.Context) {
	var payload runtimecontract.RunTaskRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidRunTaskBody, "invalid run task payload")
		return
	}

	data, err := h.runtime.RunTask(c.Request.Context(), c.Param("id"), c.Param("taskId"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Approvals returns the approvals under the given thread.
func (h *RuntimeHandler) Approvals(c *gin.Context) {
	data, err := h.runtime.Approvals(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// WriteExecutions returns the write execution audit records under the given thread.
func (h *RuntimeHandler) WriteExecutions(c *gin.Context) {
	data, err := h.runtime.WriteExecutions(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// ApproveTask approves an existing thread-local task.
func (h *RuntimeHandler) ApproveTask(c *gin.Context) {
	var payload runtimecontract.ApproveTaskRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidApproveTaskBody, "invalid approve task payload")
		return
	}

	data, err := h.runtime.ApproveTask(c.Request.Context(), c.Param("id"), c.Param("taskId"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// RejectTask rejects an existing thread-local task.
func (h *RuntimeHandler) RejectTask(c *gin.Context) {
	var payload runtimecontract.RejectTaskRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidRejectTaskBody, "invalid reject task payload")
		return
	}

	data, err := h.runtime.RejectTask(c.Request.Context(), c.Param("id"), c.Param("taskId"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Messages returns the messages under the given thread.
func (h *RuntimeHandler) Messages(c *gin.Context) {
	data, err := h.runtime.Messages(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// AppendMessage appends a message under the given thread.
func (h *RuntimeHandler) AppendMessage(c *gin.Context) {
	var payload runtimecontract.CreateMessageRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidMessageBody, "invalid create message payload")
		return
	}

	data, err := h.runtime.AppendMessage(c.Request.Context(), c.Param("id"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// ToolCalls returns the tool calls under the given thread.
func (h *RuntimeHandler) ToolCalls(c *gin.Context) {
	data, err := h.runtime.ToolCalls(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// AppendToolCall appends a tool call under the given thread.
func (h *RuntimeHandler) AppendToolCall(c *gin.Context) {
	var payload runtimecontract.CreateToolCallRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidToolCallBody, "invalid create tool call payload")
		return
	}

	data, err := h.runtime.AppendToolCall(c.Request.Context(), c.Param("id"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Artifacts returns the artifacts under the given thread.
func (h *RuntimeHandler) Artifacts(c *gin.Context) {
	data, err := h.runtime.Artifacts(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// AppendArtifact appends an artifact under the given thread.
func (h *RuntimeHandler) AppendArtifact(c *gin.Context) {
	var payload runtimecontract.CreateArtifactRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidArtifactBody, "invalid create artifact payload")
		return
	}

	data, err := h.runtime.AppendArtifact(c.Request.Context(), c.Param("id"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// RuntimeFlags returns the runtime flags under the given thread.
func (h *RuntimeHandler) RuntimeFlags(c *gin.Context) {
	data, err := h.runtime.RuntimeFlags(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// SetRuntimeFlag upserts a runtime flag under the given thread.
func (h *RuntimeHandler) SetRuntimeFlag(c *gin.Context) {
	var payload runtimecontract.SetRuntimeFlagRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidRuntimeFlagBody, "invalid runtime flag payload")
		return
	}

	data, err := h.runtime.SetRuntimeFlag(c.Request.Context(), c.Param("id"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// UpdateTaskStatus updates the status for a thread-local task.
func (h *RuntimeHandler) UpdateTaskStatus(c *gin.Context) {
	var payload runtimecontract.UpdateTaskStatusRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidTaskStatusBody, "invalid task status payload")
		return
	}

	data, err := h.runtime.UpdateTaskStatus(c.Request.Context(), c.Param("id"), c.Param("taskId"), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// Events returns the events under the given thread.
func (h *RuntimeHandler) Events(c *gin.Context) {
	data, err := h.runtime.Events(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// StreamEvents returns a minimal SSE stream for the given thread.
func (h *RuntimeHandler) StreamEvents(c *gin.Context) {
	stream, err := h.runtime.StreamEvents(c.Request.Context(), c.Param("id"), runtimecontract.StreamEventsRequest{
		Limit:     parsePositiveInt(c.Query("limit"), 200),
		SinceID:   c.Query("sinceId"),
		SinceTime: c.Query("sinceTime"),
	})
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	c.Status(http.StatusOK)
	c.Writer.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			_, _ = fmt.Fprint(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		case event, ok := <-stream:
			if !ok {
				return
			}
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				return
			}
			_, _ = fmt.Fprintf(c.Writer, "event: %s\n", event.Type)
			_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
			c.Writer.Flush()
		}
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	var value int
	if _, err := fmt.Sscanf(raw, "%d", &value); err != nil || value <= 0 {
		return fallback
	}
	return value
}

// Skills returns the available skills exposed by the runtime.
func (h *RuntimeHandler) Skills(c *gin.Context) {
	data, err := h.runtime.Skills(c.Request.Context())
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// Tools returns the available tools exposed by the runtime.
func (h *RuntimeHandler) Tools(c *gin.Context) {
	data, err := h.runtime.Tools(c.Request.Context())
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// Providers returns the configured model providers exposed by the runtime.
func (h *RuntimeHandler) Providers(c *gin.Context) {
	data, err := h.runtime.Providers(c.Request.Context())
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// ProbeProvider runs a lightweight connectivity probe for the requested provider.
func (h *RuntimeHandler) ProbeProvider(c *gin.Context) {
	data, err := h.runtime.ProbeProvider(c.Request.Context(), c.Param("kind"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

// MCPServers returns the configured MCP servers exposed by the runtime.
func (h *RuntimeHandler) MCPServers(c *gin.Context) {
	data, err := h.runtime.MCPServers(c.Request.Context())
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
}

// CheckBridge validates runtime bridge connectivity using a passthrough JSON payload.
func (h *RuntimeHandler) CheckBridge(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		xresp.BadRequest(c, errCodeInvalidBridgeCheckPayload, "invalid bridge check payload")
		return
	}

	data, err := h.runtime.CheckBridge(c.Request.Context(), payload)
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, data)
}

func writeRuntimeError(c *gin.Context, err error) {
	if requestLogger := middleware.GetLogger(c); requestLogger != nil {
		requestLogger.Error("runtime handler failed", "status", http.StatusInternalServerError, "error", err)
	}

	xresp.WriteError(c, err)
}
