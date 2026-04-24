// Package logging provides structured logging utilities using slog.
package logging

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

// contextKey is the type for context keys in this package.
type contextKey string

const (
	// RequestIDKey is the context key for request IDs.
	RequestIDKey contextKey = "request_id"
)

// NewLogger creates a new slog.Logger with the specified level.
func NewLogger(levelStr string) *slog.Logger {
	level := parseLevel(levelStr)

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}

// parseLevel parses a log level string into slog.Level.
func parseLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID returns a new context with the request ID.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// RequestIDFromContext extracts the request ID from context.
func RequestIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// LoggerWithRequestID returns a logger with the request ID attribute.
func LoggerWithRequestID(logger *slog.Logger, ctx context.Context) *slog.Logger {
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		return logger.With("request_id", requestID)
	}
	return logger
}
