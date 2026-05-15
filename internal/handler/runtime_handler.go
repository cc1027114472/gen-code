package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/middleware"
	"llmtrace/internal/platform/xresp"
)

const (
	errCodeInvalidBridgeCheckPayload = 1001
	errCodeInvalidCreateThreadBody   = 1002
	errCodeInvalidCreateTaskBody     = 1005
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

// Events returns the events under the given thread.
func (h *RuntimeHandler) Events(c *gin.Context) {
	data, err := h.runtime.Events(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeRuntimeError(c, err)
		return
	}

	xresp.OK(c, gin.H{"items": data})
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
