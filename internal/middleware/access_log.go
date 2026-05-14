package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"llmtrace/internal/logger"
	"llmtrace/internal/platform/xlog"
)

// AccessLog 用于记录 HTTP 请求访问日志。
func AccessLog(base logger.Logger) gin.HandlerFunc {
	if base == nil {
		panic("base logger is required")
	}

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		requestLogger := xlog.FromContext(c.Request.Context(), GetLogger(c))
		if requestLogger == nil {
			requestLogger = xlog.WithContext(base, c.Request.Context())
		}

		requestLogger.Info("request completed",
			xlog.FieldMethod, c.Request.Method,
			xlog.FieldPath, c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
			"size", c.Writer.Size(),
		)
	}
}
