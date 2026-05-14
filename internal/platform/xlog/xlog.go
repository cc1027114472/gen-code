package xlog

import (
	"context"

	"llmtrace/internal/logger"
	"llmtrace/internal/platform/xtrace"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger = logger.Logger

type contextKey string

const (
	FieldApp       = "app"
	FieldMethod    = "method"
	FieldPath      = "path"
	FieldRequestID = "request_id"
	FieldTaskID    = "task_id"
)

const loggerKey contextKey = "xlog.logger"

// zapLogger 是基于 Zap 的日志实现。
type zapLogger struct {
	base *zap.SugaredLogger
}

// New 用于创建指定级别的日志对象。
func New(level string) (Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.Encoding = "console"
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000")
	cfg.EncoderConfig.LevelKey = "level"
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	cfg.EncoderConfig.CallerKey = "caller"
	cfg.DisableStacktrace = true

	if err := cfg.Level.UnmarshalText([]byte(level)); err != nil {
		return nil, err
	}

	base, err := cfg.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}

	return zapLogger{base: base.Sugar()}, nil
}

// ContextWithLogger 用于将日志对象写入上下文。
func ContextWithLogger(ctx context.Context, log Logger) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, loggerKey, log)
}

// WithLogger 用于将日志对象写入上下文。
func WithLogger(ctx context.Context, log Logger) context.Context {
	return ContextWithLogger(ctx, log)
}

// FromContext 用于从上下文中获取日志对象并附加链路字段。
func FromContext(ctx context.Context, fallback Logger) Logger {
	if ctx != nil {
		if log, ok := ctx.Value(loggerKey).(Logger); ok && log != nil {
			return WithContext(log, ctx)
		}
	}
	if fallback == nil {
		return nil
	}
	return WithContext(fallback, ctx)
}

// FromStdContext 用于从标准上下文中获取日志对象。
func FromStdContext(ctx context.Context, fallback Logger) Logger {
	return FromContext(ctx, fallback)
}

// WithContext 用于为日志对象附加上下文字段。
func WithContext(log Logger, ctx context.Context) Logger {
	if log == nil {
		return nil
	}
	if ctx == nil {
		return log
	}

	keyvals := make([]any, 0, 4)
	if requestID := xtrace.RequestID(ctx); requestID != "" {
		keyvals = append(keyvals, FieldRequestID, requestID)
	}
	if taskID := xtrace.TaskID(ctx); taskID != "" {
		keyvals = append(keyvals, FieldTaskID, taskID)
	}
	if len(keyvals) == 0 {
		return log
	}

	return log.With(keyvals...)
}

// WithRequestID 用于向上下文写入请求 ID。
func WithRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return xtrace.WithRequestID(ctx, requestID)
}

// RequestID 用于从上下文中读取请求 ID。
func RequestID(ctx context.Context) string {
	return xtrace.RequestID(ctx)
}

// WithTaskID 用于向上下文写入任务 ID。
func WithTaskID(ctx context.Context, taskID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return xtrace.WithTaskID(ctx, taskID)
}

// TaskID 用于从上下文中读取任务 ID。
func TaskID(ctx context.Context) string {
	return xtrace.TaskID(ctx)
}

// Debug 用于输出调试级别日志。
func (l zapLogger) Debug(msg string, keyvals ...any) {
	l.base.Debugw(msg, keyvals...)
}

// Info 用于输出信息级别日志。
func (l zapLogger) Info(msg string, keyvals ...any) {
	l.base.Infow(msg, keyvals...)
}

// Warn 用于输出警告级别日志。
func (l zapLogger) Warn(msg string, keyvals ...any) {
	l.base.Warnw(msg, keyvals...)
}

// Error 用于输出错误级别日志。
func (l zapLogger) Error(msg string, keyvals ...any) {
	l.base.Errorw(msg, keyvals...)
}

// With 用于派生附带额外字段的日志对象。
func (l zapLogger) With(keyvals ...any) Logger {
	return zapLogger{base: l.base.With(keyvals...)}
}
