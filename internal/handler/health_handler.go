package handler

import (
	"github.com/gin-gonic/gin"

	"llmtrace/internal/platform/xresp"
)

// HealthHandler 提供服务健康检查接口。
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器。
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Healthz 返回服务当前的健康状态。
func (h *HealthHandler) Healthz(c *gin.Context) {
	xresp.OK(c, gin.H{"status": "ok"})
}
