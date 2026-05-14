package xtrace

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRequestIDRoundTrip 用于验证请求 ID 可以完整写入并读取。
func TestRequestIDRoundTrip(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")

	require.Equal(t, "req-123", RequestID(ctx))
	require.Equal(t, "req-123", RequestIDFromContext(ctx))
}

// TestEnsureRequestIDReusesExistingValue 用于验证已有请求 ID 会被直接复用。
func TestEnsureRequestIDReusesExistingValue(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")

	nextCtx, requestID := EnsureRequestID(ctx)

	require.Equal(t, "req-123", requestID)
	require.Equal(t, "req-123", RequestID(nextCtx))
}

// TestEnsureTaskIDGeneratesValue 用于验证任务 ID 缺失时会自动生成。
func TestEnsureTaskIDGeneratesValue(t *testing.T) {
	ctx, taskID := EnsureTaskID(context.Background())

	require.NotEmpty(t, taskID)
	require.Equal(t, taskID, TaskID(ctx))
}

// TestContextWithRequestIDAcceptsNilContext 用于验证空上下文也能写入请求 ID。
func TestContextWithRequestIDAcceptsNilContext(t *testing.T) {
	ctx := ContextWithRequestID(nil, "req-234")

	require.Equal(t, "req-234", RequestIDFromContext(ctx))
}
