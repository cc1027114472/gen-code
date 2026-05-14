package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"llmtrace/internal/logger"
	"llmtrace/internal/platform/xlog"
	"llmtrace/internal/response"
)

// Recovery 用于拦截 panic 并返回统一错误响应。
func Recovery(base logger.Logger) gin.HandlerFunc {
	if base == nil {
		panic("base logger is required")
	}

	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		requestLogger := xlog.FromContext(c.Request.Context(), GetLogger(c))
		if requestLogger == nil {
			requestLogger = xlog.WithContext(base, c.Request.Context())
		}

		requestLogger.Error("panic recovered", "panic", recovered)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.ErrorBody{
			Code:    http.StatusInternalServerError,
			Message: "internal server error",
		})
	})
}
