package handlers

import (
	"net/http"
)

// HealthHandler handles health check requests.
type HealthHandler struct{}

// NewHealthHandler creates a new health handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthResponse is the response for the health endpoint.
type HealthResponse struct {
	OK bool `json:"ok"`
}

// ServeHTTP handles GET /healthz requests.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, HealthResponse{OK: true})
}
