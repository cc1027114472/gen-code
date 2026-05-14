package middleware

import (
	"github.com/gin-gonic/gin"

	"llmtrace/internal/logger"
	"llmtrace/internal/platform/xlog"
)

const contextLoggerKey = "logger"

// ContextLogger 用于为请求上下文注入日志对象。
func ContextLogger(base logger.Logger, appName string) gin.HandlerFunc {
	if base == nil {
		panic("base logger is required")
	}

	return func(c *gin.Context) {
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		requestLogger := base.With(
			xlog.FieldApp, appName,
			xlog.FieldMethod, c.Request.Method,
			xlog.FieldPath, path,
		)

		c.Set(contextLoggerKey, requestLogger)
		c.Request = c.Request.WithContext(xlog.WithLogger(c.Request.Context(), requestLogger))
		c.Next()
	}
}

// GetLogger 用于从 Gin 上下文中读取请求日志对象。
func GetLogger(c *gin.Context) logger.Logger {
	if value, ok := c.Get(contextLoggerKey); ok {
		if requestLogger, ok := value.(logger.Logger); ok {
			return requestLogger
		}
	}

	return nil
}
