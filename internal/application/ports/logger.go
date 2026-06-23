package ports

import "context"

// Logger defines the interface for logging operations in the application layer.
// It abstracts the underlying logging implementation (e.g., slog, zap)...
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
	WithContext(ctx context.Context) Logger
}

// LoggerContextKey is the key used to store/retrieve the logger from context.
type LoggerContextKey struct{}
