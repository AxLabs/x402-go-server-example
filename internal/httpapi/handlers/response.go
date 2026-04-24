// Package handlers provides HTTP request handlers.
package handlers

import (
	"encoding/json"
	"net/http"
)

// Response helpers

// JSON writes a JSON response with the given status code.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Best effort only: headers/status may already be written.
		return
	}
}

// ErrorJSON writes a JSON error response.
func ErrorJSON(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}
