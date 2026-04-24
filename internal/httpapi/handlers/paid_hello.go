package handlers

import (
	"net/http"
	"time"
)

// PaidHelloHandler handles paid hello requests.
//
// Payment verification and settlement are handled entirely by the x402 SDK
// middleware wrapping this handler; this handler only produces the
// business response once payment has been verified. The settlement
// transaction hash is returned to the client in the PAYMENT-RESPONSE
// header added by the SDK after this handler completes.
type PaidHelloHandler struct{}

// NewPaidHelloHandler creates a new paid hello handler.
func NewPaidHelloHandler() *PaidHelloHandler {
	return &PaidHelloHandler{}
}

// PaidHelloResponse is the JSON body for /paid/hello.
type PaidHelloResponse struct {
	OK        bool   `json:"ok"`
	Message   string `json:"message"`
	Resource  string `json:"resource"`
	Timestamp string `json:"timestamp"`
}

// ServeHTTP handles GET /paid/hello requests.
func (h *PaidHelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	JSON(w, http.StatusOK, PaidHelloResponse{
		OK:        true,
		Message:   "payment accepted",
		Resource:  "hello",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
