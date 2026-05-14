package xlog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFromContextFallsBack 用于验证上下文缺少日志时会回退到默认日志。
func TestFromContextFallsBack(t *testing.T) {
	log, err := New("info")
	require.NoError(t, err)

	require.NotNil(t, FromContext(context.Background(), log))
}

// TestContextWithLogger 用于验证日志对象可以通过上下文传递。
func TestContextWithLogger(t *testing.T) {
	log, err := New("info")
	require.NoError(t, err)

	ctx := WithRequestID(context.Background(), "req-1")
	ctx = ContextWithLogger(ctx, log)

	require.NotNil(t, FromContext(ctx, nil))
}
