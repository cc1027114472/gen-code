package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"llmtrace/internal/platform/xlog"
)

const (
	RequestIDHeader = "X-Request-ID"
	requestIDKey    = "request_id"
)

// RequestID 用于为请求注入或透传请求 ID。
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.NewString()
		}

		c.Set(requestIDKey, requestID)
		c.Request = c.Request.WithContext(xlog.WithRequestID(c.Request.Context(), requestID))
		c.Writer.Header().Set(RequestIDHeader, requestID)
		c.Next()
	}
}

// GetRequestID 用于从 Gin 上下文中读取请求 ID。
func GetRequestID(c *gin.Context) string {
	if value, ok := c.Get(requestIDKey); ok {
		if requestID, ok := value.(string); ok {
			return requestID
		}
	}

	return ""
}

// WithTaskID 用于向上下文写入任务 ID。
func WithTaskID(ctx context.Context, taskID string) context.Context {
	return xlog.WithTaskID(ctx, taskID)
}

// GetTaskID 用于从 Gin 上下文中读取任务 ID。
func GetTaskID(c *gin.Context) string {
	return xlog.TaskID(c.Request.Context())
}
