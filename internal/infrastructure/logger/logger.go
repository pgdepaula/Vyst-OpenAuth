package logger

import (
	"context"
	"log/slog"
	"os"
	"sync"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
)

// Config holds the configuration for the logger.
type Config struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "json", "text"
}

// Wrapper wraps slog.Logger to implement our Logger interface.
type Wrapper struct {
	*slog.Logger
}

var (
	globalLogger *Wrapper
	once         sync.Once
)

// New creates a new Logger instance with the given configuration.
func New(cfg Config) *Wrapper {
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
		// Add source location to logs for better debugging
		AddSource: true,
	}

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)
	return &Wrapper{Logger: logger}
}

// SetGlobal sets the global logger instance.
func SetGlobal(l *Wrapper) {
	once.Do(func() {
		globalLogger = l
		slog.SetDefault(l.Logger)
	})
}

// Global returns the global logger instance.
func Global() *Wrapper {
	if globalLogger == nil {
		// Default to a basic info/text logger if not initialized
		return New(Config{Level: "info", Format: "text"})
	}
	return globalLogger
}

// Debug logs at Debug level.
func (l *Wrapper) Debug(msg string, args ...any) {
	l.Logger.Debug(msg, args...)
}

// Info logs at Info level.
func (l *Wrapper) Info(msg string, args ...any) {
	l.Logger.Info(msg, args...)
}

// Warn logs at Warn level.
func (l *Wrapper) Warn(msg string, args ...any) {
	l.Logger.Warn(msg, args...)
}

// Error logs at Error level.
func (l *Wrapper) Error(msg string, args ...any) {
	l.Logger.Error(msg, args...)
}

// With returns a new Logger with the given attributes.
func (l *Wrapper) With(args ...any) ports.Logger {
	return &Wrapper{Logger: l.Logger.With(args...)}
}

// WithContext returns the logger from the context if it exists,
// otherwise returns the current logger.
// This allows services to do `s.logger.WithContext(ctx).Info(...)` and get the request-scoped logger.
func (l *Wrapper) WithContext(ctx context.Context) ports.Logger {
	if v := ctx.Value(ports.LoggerContextKey{}); v != nil {
		if logger, ok := v.(ports.Logger); ok {
			return logger
		}
	}
	return l
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
