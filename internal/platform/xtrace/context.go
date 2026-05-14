package xtrace

import (
	"context"

	"github.com/google/uuid"
)

const HeaderRequestID = "X-Request-ID"

type contextKey string

const (
	requestIDKey contextKey = "xtrace.request_id"
	taskIDKey    contextKey = "xtrace.task_id"
)

// NewID 用于生成新的链路标识。
func NewID() string {
	return uuid.NewString()
}

// WithRequestID 用于向上下文写入请求 ID。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if requestID == "" {
		return ctx
	}

	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestID 用于从上下文中读取请求 ID。
func RequestID(ctx context.Context) string {
	return value(ctx, requestIDKey)
}

// ContextWithRequestID 用于向上下文写入请求 ID。
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return WithRequestID(ctx, requestID)
}

// RequestIDFromContext 用于从上下文中读取请求 ID。
func RequestIDFromContext(ctx context.Context) string {
	return RequestID(ctx)
}

// EnsureRequestID 用于确保上下文中存在请求 ID。
func EnsureRequestID(ctx context.Context) (context.Context, string) {
	if requestID := RequestID(ctx); requestID != "" {
		return ctx, requestID
	}

	requestID := NewID()
	return WithRequestID(ctx, requestID), requestID
}

// WithTaskID 用于向上下文写入任务 ID。
func WithTaskID(ctx context.Context, taskID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if taskID == "" {
		return ctx
	}

	return context.WithValue(ctx, taskIDKey, taskID)
}

// TaskID 用于从上下文中读取任务 ID。
func TaskID(ctx context.Context) string {
	return value(ctx, taskIDKey)
}

// ContextWithTaskID 用于向上下文写入任务 ID。
func ContextWithTaskID(ctx context.Context, taskID string) context.Context {
	return WithTaskID(ctx, taskID)
}

// TaskIDFromContext 用于从上下文中读取任务 ID。
func TaskIDFromContext(ctx context.Context) string {
	return TaskID(ctx)
}

// EnsureTaskID 用于确保上下文中存在任务 ID。
func EnsureTaskID(ctx context.Context) (context.Context, string) {
	if taskID := TaskID(ctx); taskID != "" {
		return ctx, taskID
	}

	taskID := NewID()
	return WithTaskID(ctx, taskID), taskID
}

// value 用于从上下文中读取字符串值。
func value(ctx context.Context, key contextKey) string {
	if ctx == nil {
		return ""
	}

	value, _ := ctx.Value(key).(string)
	return value
}
