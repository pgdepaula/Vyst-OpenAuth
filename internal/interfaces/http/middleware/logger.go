package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"

	"github.com/pgdepaula/vyst-openauth/internal/application/ports"
	"github.com/pgdepaula/vyst-openauth/internal/infrastructure/logger"
)

// RequestLogger is a middleware that logs the start and end of each request.
// It also injects a request-scoped logger into the context.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := middleware.GetReqID(r.Context())

		// Create a logger with request fields
		// We use the global logger as the base
		logFields := []any{
			"req_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_ip", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		}

		// Add OpenTelemetry TraceID/SpanID if available
		if span := trace.SpanFromContext(r.Context()); span.SpanContext().IsValid() {
			logFields = append(logFields,
				"trace_id", span.SpanContext().TraceID().String(),
				"span_id", span.SpanContext().SpanID().String(),
			)
		}

		log := logger.Global().With(logFields...)

		// Inject logger into context using the port's key
		ctx := context.WithValue(r.Context(), ports.LoggerContextKey{}, log)

		// Wrap response writer to capture status code
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Log request start
		log.Info("Request started")

		defer func() {
			// Log request completion
			duration := time.Since(start)
			log.Info("Request completed",
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration", duration.String(),
				"duration_ms", duration.Milliseconds(),
			)
		}()

		next.ServeHTTP(ww, r.WithContext(ctx))
	})
}

// GetLogger retrieves the logger from the context.
// If no logger is found, it returns the global logger.
func GetLogger(ctx context.Context) ports.Logger {
	if l, ok := ctx.Value(ports.LoggerContextKey{}).(ports.Logger); ok {
		return l
	}
	return logger.Global()
}
