// Package httpapi provides the HTTP API router and server setup.
package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

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
	// internal/x402 package. It is applied to each configured paid route so
	// those routes go through the SDK's payment gating. It is injected
	// here (rather than built internally) so that tests can swap in a
	// middleware backed by a mock facilitator.
	X402Middleware func(http.Handler) http.Handler
}

// PaidRoutes returns the route specs used by both the router and the SDK
// middleware. Keeping a single source of these specs ensures the chi
// routes and the SDK's RoutesConfig stay in sync.
func PaidRoutes(cfg *config.Config) []x402.RouteSpec {
	routes := make([]x402.RouteSpec, 0, len(cfg.Payment.Routes))
	for _, r := range cfg.Payment.Routes {
		route := x402.RouteSpec{
			Method:      r.Method,
			Path:        r.Path,
			Description: r.Description,
			Accepts:     make([]x402.AcceptOption, 0, len(r.Accepts)),
		}
		for _, accept := range r.Accepts {
			route.Accepts = append(route.Accepts, x402.AcceptOption{
				Scheme:            accept.Scheme,
				Network:           accept.Network,
				PayTo:             accept.PayTo,
				Asset:             accept.Asset,
				Amount:            accept.Amount,
				MaxTimeoutSeconds: accept.MaxTimeoutSeconds,
				Extra:             accept.Extra,
			})
		}
		routes = append(routes, route)
	}

	return routes
}

// NewRouter creates and configures the HTTP router.
func NewRouter(cfg RouterConfig) (http.Handler, error) {
	r := chi.NewRouter()

	// Global middleware (not x402-related).
	r.Use(middleware.Recoverer(cfg.Logger))
	r.Use(middleware.RequestLogger(cfg.Logger))
	r.Use(middleware.ContentTypeJSON)

	// Public endpoints.
	r.Get("/healthz", handlers.NewHealthHandler().ServeHTTP)
	r.Get("/info", handlers.NewInfoHandler(cfg.Config).ServeHTTP)

	// Paid endpoints: all configured paid routes are gated by the SDK middleware.
	// The SDK middleware runs ProcessHTTPRequest (which returns 402 or the
	// verified payment) and then, after the handler returns, runs
	// ProcessSettlement and appends the PAYMENT-RESPONSE header.
	for i := range cfg.Config.Payment.Routes {
		paidRoute := cfg.Config.Payment.Routes[i]
		if isReservedPublicRoute(paidRoute.Method, paidRoute.Path) {
			return nil, fmt.Errorf("payment.routes[%d] cannot use reserved public route: %s %s", i, paidRoute.Method, paidRoute.Path)
		}
		handler, err := paidRouteHandler(paidRoute.Handler)
		if err != nil {
			return nil, fmt.Errorf("payment.routes[%d]: %w", i, err)
		}

		r.Group(func(gr chi.Router) {
			if cfg.X402Middleware != nil {
				gr.Use(cfg.X402Middleware)
			}
			gr.Method(paidRoute.Method, paidRoute.Path, handler)
		})
	}

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		handlers.ErrorJSON(w, http.StatusNotFound, "not_found", "endpoint not found")
	})

	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		handlers.ErrorJSON(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	})

	return r, nil
}

func paidRouteHandler(name string) (http.Handler, error) {
	switch name {
	case config.PaidHandlerHello:
		return handlers.NewPaidHelloHandler(), nil
	case config.PaidHandlerEcho:
		return handlers.NewPaidEchoHandler(), nil
	default:
		return nil, fmt.Errorf("unsupported handler %q", name)
	}
}

func isReservedPublicRoute(method, path string) bool {
	return (method == http.MethodGet && path == "/healthz") || (method == http.MethodGet && path == "/info")
}
