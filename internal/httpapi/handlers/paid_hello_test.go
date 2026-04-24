package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPaidHelloHandler_TimestampIsRFC3339(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/paid/hello", nil)
	rec := httptest.NewRecorder()

	NewPaidHelloHandler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PaidHelloResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if _, err := time.Parse(time.RFC3339, resp.Timestamp); err != nil {
		t.Fatalf("expected RFC3339 timestamp, got %q (%v)", resp.Timestamp, err)
	}
}
