package logger

// Logger is the minimal application logging surface used by middleware.
type Logger interface {
	Debug(msg string, keyvals ...any)
	Info(msg string, keyvals ...any)
	Warn(msg string, keyvals ...any)
	Error(msg string, keyvals ...any)
	With(keyvals ...any) Logger
}
