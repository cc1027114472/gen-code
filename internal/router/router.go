package router

import (
	"github.com/gin-gonic/gin"

	"llmtrace/internal/handler"
	"llmtrace/internal/logger"
	"llmtrace/internal/middleware"
)

// Handlers collects the HTTP handlers registered by the router.
type Handlers struct {
	Health  *handler.HealthHandler
	Runtime *handler.RuntimeHandler
}

// New creates the HTTP router for the minimal service.
func New(base logger.Logger, appName string, trustedProxies []string, httpAccess bool, handlers Handlers) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(trustedProxies); err != nil {
		return nil, err
	}

	r.Use(middleware.RequestID())
	r.Use(middleware.ContextLogger(base, appName))
	if httpAccess {
		r.Use(middleware.AccessLog(base))
	}
	r.Use(middleware.Recovery(base))

	if handlers.Health != nil {
		r.GET("/healthz", handlers.Health.Healthz)
	}

	if handlers.Runtime != nil {
		api := r.Group("/api")
		api.GET("/runtime/status", handlers.Runtime.Status)
		api.GET("/workspace", handlers.Runtime.Workspace)
		api.GET("/threads", handlers.Runtime.Threads)
		api.POST("/threads", handlers.Runtime.CreateThread)
		api.GET("/threads/:id", handlers.Runtime.Thread)
		api.POST("/threads/:id/activate", handlers.Runtime.ActivateThread)
		api.GET("/threads/:id/tasks", handlers.Runtime.Tasks)
		api.POST("/threads/:id/tasks", handlers.Runtime.CreateTask)
		api.GET("/threads/:id/messages", handlers.Runtime.Messages)
		api.POST("/threads/:id/messages", handlers.Runtime.AppendMessage)
		api.GET("/threads/:id/tool-calls", handlers.Runtime.ToolCalls)
		api.POST("/threads/:id/tool-calls", handlers.Runtime.AppendToolCall)
		api.GET("/threads/:id/artifacts", handlers.Runtime.Artifacts)
		api.POST("/threads/:id/artifacts", handlers.Runtime.AppendArtifact)
		api.GET("/threads/:id/runtime-flags", handlers.Runtime.RuntimeFlags)
		api.POST("/threads/:id/runtime-flags", handlers.Runtime.SetRuntimeFlag)
		api.POST("/threads/:id/tasks/:taskId/status", handlers.Runtime.UpdateTaskStatus)
		api.GET("/threads/:id/events", handlers.Runtime.Events)
		api.GET("/threads/:id/events/stream", handlers.Runtime.StreamEvents)
		api.GET("/skills", handlers.Runtime.Skills)
		api.GET("/tools", handlers.Runtime.Tools)
		api.GET("/mcp/servers", handlers.Runtime.MCPServers)
		api.POST("/bridge/check", handlers.Runtime.CheckBridge)
	}

	return r, nil
}
