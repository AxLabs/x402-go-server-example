// Package httpapi provides the HTTP API router and server setup.
package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// Server wraps an HTTP server with graceful shutdown support.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// NewServer creates a new HTTP server.
func NewServer(addr string, handler http.Handler, readTimeout, writeTimeout time.Duration, logger *slog.Logger) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
		},
		logger: logger.With("component", "http_server"),
	}
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", "addr", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}
