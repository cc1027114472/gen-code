package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapLogger 基于 zap 实现项目内统一日志接口。
type zapLogger struct {
	base *zap.SugaredLogger
}

// New 根据日志级别创建 zap 日志实例。
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

	return zapLogger{
		base: base.Sugar(),
	}, nil
}

// Debug 输出调试级别日志。
func (l zapLogger) Debug(msg string, keyvals ...any) {
	l.base.Debugw(msg, keyvals...)
}

// Info 输出信息级别日志。
func (l zapLogger) Info(msg string, keyvals ...any) {
	l.base.Infow(msg, keyvals...)
}

// Warn 输出警告级别日志。
func (l zapLogger) Warn(msg string, keyvals ...any) {
	l.base.Warnw(msg, keyvals...)
}

// Error 输出错误级别日志。
func (l zapLogger) Error(msg string, keyvals ...any) {
	l.base.Errorw(msg, keyvals...)
}

// With 为日志实例附加上下文字段。
func (l zapLogger) With(keyvals ...any) Logger {
	return zapLogger{
		base: l.base.With(keyvals...),
	}
}
