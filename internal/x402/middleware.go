// Package x402 wires the official x402 Go SDK
// (github.com/x402-foundation/x402/go) into this resource server.
//
// The SDK is the single source of truth for the x402 protocol: request
// parsing, 402 challenge generation, header names, facilitator verification
// and settlement are all implemented in the SDK and are NOT re-implemented
// here. This package only:
//
//  1. Builds a RoutesConfig describing which HTTP routes are paid and what
//     accepts options they advertise (explicit scheme/network/asset/amount).
//  2. Constructs an SDK HTTPFacilitatorClient pointing at our facilitator.
//  3. Registers the SDK's built-in EVM "exact" scheme server.
//  4. Returns an SDK-backed net/http middleware that can be plugged into
//     the chi router (chi is compatible with plain http.Handler middleware).
//
// Business handlers do NOT read any payment data from the request context;
// the SDK middleware runs before the handler and, on success, appends the
// PAYMENT-RESPONSE header (which carries the settlement transaction hash)
// to the response after the handler returns. Handlers just need to return
// their business payload.
package x402

import (
	"fmt"
	"net/http"
	"time"

	x402 "github.com/x402-foundation/x402/go"
	x402http "github.com/x402-foundation/x402/go/http"
	"github.com/x402-foundation/x402/go/http/nethttp"
	evmserver "github.com/x402-foundation/x402/go/mechanisms/evm/exact/server"
)

// Scheme is the x402 payment scheme served by this resource server.
// The EVM "exact" scheme is provided by the SDK.
const Scheme = "exact"

// RouteSpec describes one paid endpoint.
type RouteSpec struct {
	// Method is the HTTP verb (e.g. "GET").
	Method string
	// Path is the URL path (e.g. "/paid/hello").
	Path string
	// Description is advertised in the RoutesConfig.
	Description string
	// Accepts is the route's explicit x402 payment options.
	Accepts []AcceptOption
}

// AcceptOption mirrors x402 PaymentRequirements fields.
type AcceptOption struct {
	Scheme            string
	Network           string
	PayTo             string
	Asset             string
	Amount            string
	MaxTimeoutSeconds int
	Extra             map[string]interface{}
}

// Pattern returns the SDK route key, e.g. "GET /paid/hello".
func (r RouteSpec) Pattern() string {
	return fmt.Sprintf("%s %s", r.Method, r.Path)
}

// Config bundles everything needed to build the SDK-backed middleware.
type Config struct {
	// FacilitatorURL is the x402 facilitator base URL.
	FacilitatorURL string
	// FacilitatorTimeout is the HTTP timeout for facilitator calls.
	FacilitatorTimeout time.Duration
	// Routes is the list of paid endpoints.
	Routes []RouteSpec
	// SyncFacilitatorOnStart fetches supported kinds from the facilitator
	// during middleware construction. Set false for tests using a mock
	// facilitator that is not yet reachable at wire-up time.
	SyncFacilitatorOnStart bool
	// Timeout bounds each verify/settle round trip; defaults to 30s.
	Timeout time.Duration
}

// Middleware builds an net/http middleware that protects Config.Routes
// using the official x402 SDK. The returned middleware is safe to apply
// with chi via r.Use or r.With, as chi accepts any func(http.Handler) http.Handler.
func Middleware(cfg Config) (func(http.Handler) http.Handler, error) {
	if cfg.FacilitatorURL == "" {
		return nil, fmt.Errorf("x402: FacilitatorURL is required")
	}
	if len(cfg.Routes) == 0 {
		return nil, fmt.Errorf("x402: at least one route is required")
	}

	routes := make(x402http.RoutesConfig, len(cfg.Routes))
	seen := map[x402.Network]bool{}
	schemes := []nethttp.SchemeConfig{}
	for _, r := range cfg.Routes {
		opts := x402http.PaymentOptions{}
		for _, accept := range r.Accepts {
			// The SDK's ParsePrice expects a map[string]interface{} for
			// explicit asset+amount (not a typed struct).
			price := map[string]interface{}{
				"asset":  accept.Asset,
				"amount": accept.Amount,
			}
			if len(accept.Extra) > 0 {
				price["extra"] = accept.Extra
			}
			opts = append(opts, x402http.PaymentOption{
				Scheme:            accept.Scheme,
				PayTo:             accept.PayTo,
				Price:             price,
				Network:           x402.Network(accept.Network),
				MaxTimeoutSeconds: accept.MaxTimeoutSeconds,
				Extra:             accept.Extra,
			})

			net := x402.Network(accept.Network)
			if !seen[net] {
				seen[net] = true
				schemes = append(schemes, nethttp.SchemeConfig{
					Network: net,
					Server:  evmserver.NewExactEvmScheme(),
				})
			}
		}
		routes[r.Pattern()] = x402http.RouteConfig{
			Accepts:     opts,
			Description: r.Description,
		}
	}

	facClient := x402http.NewHTTPFacilitatorClient(&x402http.FacilitatorConfig{
		URL:     cfg.FacilitatorURL,
		Timeout: cfg.FacilitatorTimeout,
	})

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	if len(schemes) == 0 {
		return nil, fmt.Errorf("x402: at least one accept option is required")
	}

	mw := nethttp.X402Payment(nethttp.Config{
		Routes:                 routes,
		Facilitator:            facClient,
		Schemes:                schemes,
		SyncFacilitatorOnStart: cfg.SyncFacilitatorOnStart,
		Timeout:                timeout,
	})

	return mw, nil
}
