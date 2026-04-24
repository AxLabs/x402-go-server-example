package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPaidHelloHandler_Success(t *testing.T) {
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
	if !resp.OK {
		t.Error("expected OK=true")
	}
	if resp.Message != "payment accepted" {
		t.Errorf("expected message=payment accepted, got %q", resp.Message)
	}
	if resp.Resource != "hello" {
		t.Errorf("expected resource=hello, got %q", resp.Resource)
	}
	if resp.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
}
