package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	handler := NewHealthHandler()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	expected := `{"ok":true}`
	body := rec.Body.String()
	// Trim newline added by json.Encoder
	if body != expected+"\n" {
		t.Errorf("expected body %q, got %q", expected, body)
	}
}
