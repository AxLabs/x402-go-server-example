package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"
)

// PaidEchoHandler handles paid echo requests.
//
// See PaidHelloHandler for the payment-flow context: all x402 protocol
// handling happens in the SDK middleware wrapped around this handler.
type PaidEchoHandler struct{}

// NewPaidEchoHandler creates a new paid echo handler.
func NewPaidEchoHandler() *PaidEchoHandler {
	return &PaidEchoHandler{}
}

// EchoRequest is the request body for the echo endpoint.
type EchoRequest struct {
	Message string `json:"message"`
}

// PaidEchoResponse is the JSON body for /paid/echo.
type PaidEchoResponse struct {
	OK        bool          `json:"ok"`
	Message   string        `json:"message"`
	Resource  string        `json:"resource"`
	Timestamp string        `json:"timestamp"`
	Echo      EchoedMessage `json:"echo"`
}

// EchoedMessage contains the echoed content.
type EchoedMessage struct {
	OriginalMessage string `json:"originalMessage"`
}

// ServeHTTP handles POST /paid/echo requests.
func (h *PaidEchoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid_request", "failed to read request body")
		return
	}

	var req EchoRequest
	if err := json.Unmarshal(body, &req); err != nil {
		ErrorJSON(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Message == "" {
		ErrorJSON(w, http.StatusBadRequest, "missing_message", "message field is required")
		return
	}

	JSON(w, http.StatusOK, PaidEchoResponse{
		OK:        true,
		Message:   "payment accepted",
		Resource:  "echo",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Echo:      EchoedMessage{OriginalMessage: req.Message},
	})
}
