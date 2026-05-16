package appserver

import (
	"github.com/gin-gonic/gin"

	"llmtrace/internal/appserver/runtimecontract"
	"llmtrace/internal/config"
	"llmtrace/internal/core/runtime"
	"llmtrace/internal/handler"
	"llmtrace/internal/logger"
	"llmtrace/internal/router"
)

type RuntimeService = runtimecontract.Service

// HTTPConfig carries router-facing settings for the app server.
type HTTPConfig struct {
	AppName        string
	TrustedProxies []string
	HTTPAccess     bool
}

// NewEngine assembles the codex-style app server HTTP routes.
func NewEngine(base logger.Logger, cfg HTTPConfig, runtime RuntimeService) (*gin.Engine, error) {
	healthHandler := handler.NewHealthHandler()
	runtimeHandler := handler.NewRuntimeHandler(runtime)

	return router.New(base, cfg.AppName, cfg.TrustedProxies, cfg.HTTPAccess, router.Handlers{
		Health:  healthHandler,
		Runtime: runtimeHandler,
	})
}

// NewRuntimeService returns the default runtime-backed service used by the app server.
func NewRuntimeService() RuntimeService {
	cfg, err := config.Load()
	if err != nil {
		return runtime.NewDefaultService()
	}
	return runtime.NewDefaultServiceWithProviders(cfg.Providers)
}
