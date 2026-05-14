package router

import (
	"github.com/gin-gonic/gin"

	"llmtrace/internal/handler"
	"llmtrace/internal/logger"
	"llmtrace/internal/middleware"
)

// New creates the HTTP router for the minimal service.
func New(base logger.Logger, appName string, trustedProxies []string, httpAccess bool, health *handler.HealthHandler) (*gin.Engine, error) {
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

	r.GET("/healthz", health.Healthz)

	return r, nil
}
