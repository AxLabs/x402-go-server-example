// Package httpapi provides the HTTP API router and server setup.
package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	x402core "github.com/x402-foundation/x402/go"

	"github.com/AxLabs/x402-go-server-example/internal/config"
	"github.com/AxLabs/x402-go-server-example/internal/httpapi/handlers"
	"github.com/AxLabs/x402-go-server-example/internal/httpapi/middleware"
	"github.com/AxLabs/x402-go-server-example/internal/x402"
)

// RouterConfig holds dependencies for creating the router.
type RouterConfig struct {
	Config *config.Config
	Logger *slog.Logger
	// X402Middleware is the net/http middleware returned by the SDK-backed
	// internal/x402 package. It is applied to the /paid subrouter so that
	// all paid routes go through the SDK's payment gating. It is injected
	// here (rather than built internally) so that tests can swap in a
	// middleware backed by a mock facilitator.
	X402Middleware func(http.Handler) http.Handler
}

// PaidRoutes returns the route specs used by both the router and the SDK
// middleware. Keeping a single source of these specs ensures the chi
// routes and the SDK's RoutesConfig stay in sync.
func PaidRoutes(cfg *config.Config) []x402.RouteSpec {
	return []x402.RouteSpec{
		{
			Method:      http.MethodGet,
			Path:        "/paid/hello",
			Price:       cfg.Payment.PaidHelloPrice,
			Description: "Paid hello resource",
		},
		{
			Method:      http.MethodPost,
			Path:        "/paid/echo",
			Price:       cfg.Payment.PaidEchoPrice,
			Description: "Paid echo resource",
		},
	}
}

// PaidNetwork returns the CAIP-2 network used for paid routes.
func PaidNetwork(cfg *config.Config) x402core.Network {
	return x402core.Network(cfg.Payment.Network)
}

// NewRouter creates and configures the HTTP router.
func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	// Global middleware (not x402-related).
	r.Use(middleware.Recoverer(cfg.Logger))
	r.Use(middleware.RequestLogger(cfg.Logger))
	r.Use(middleware.ContentTypeJSON)

	// Public endpoints.
	r.Get("/healthz", handlers.NewHealthHandler().ServeHTTP)
	r.Get("/info", handlers.NewInfoHandler(cfg.Config).ServeHTTP)

	// Paid endpoints: all /paid/* routes are gated by the SDK middleware.
	// The SDK middleware runs ProcessHTTPRequest (which returns 402 or the
	// verified payment) and then, after the handler returns, runs
	// ProcessSettlement and appends the PAYMENT-RESPONSE header.
	r.Route("/paid", func(r chi.Router) {
		if cfg.X402Middleware != nil {
			r.Use(cfg.X402Middleware)
		}
		r.Get("/hello", handlers.NewPaidHelloHandler().ServeHTTP)
		r.Post("/echo", handlers.NewPaidEchoHandler().ServeHTTP)
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		handlers.ErrorJSON(w, http.StatusNotFound, "not_found", "endpoint not found")
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		handlers.ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	return r
}
