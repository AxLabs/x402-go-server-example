package handlers

import (
	"net/http"

	"github.com/AxLabs/x402-go-server-example/internal/config"
	"github.com/AxLabs/x402-go-server-example/internal/version"
)

// InfoHandler handles info requests.
type InfoHandler struct {
	cfg *config.Config
}

// NewInfoHandler creates a new info handler.
func NewInfoHandler(cfg *config.Config) *InfoHandler {
	return &InfoHandler{cfg: cfg}
}

// InfoResponse is the response for the info endpoint.
type InfoResponse struct {
	Service        string      `json:"service"`
	Version        string      `json:"version"`
	Commit         string      `json:"commit"`
	BuildTime      string      `json:"buildTime"`
	FacilitatorURL string      `json:"facilitatorUrl"`
	Pricing        PricingInfo `json:"pricing"`
}

// PricingInfo holds endpoint pricing information.
type PricingInfo struct {
	Routes []EndpointPricing `json:"routes"`
}

// EndpointPricing holds price information for an endpoint.
type EndpointPricing struct {
	Method      string          `json:"method"`
	Path        string          `json:"path"`
	Handler     string          `json:"handler"`
	Description string          `json:"description,omitempty"`
	Accepts     []PaymentAccept `json:"accepts"`
}

// PaymentAccept describes one payment option as shown in the /info response.
type PaymentAccept struct {
	Scheme            string                 `json:"scheme"`
	Network           string                 `json:"network"`
	Asset             string                 `json:"asset"`
	Amount            string                 `json:"amount"`
	PayTo             string                 `json:"payTo"`
	MaxTimeoutSeconds int                    `json:"maxTimeoutSeconds,omitempty"`
	Extra             map[string]interface{} `json:"extra,omitempty"`
}

// ServeHTTP handles GET /info requests.
func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vInfo := version.Info()
	routes := make([]EndpointPricing, 0, len(h.cfg.Payment.Routes))
	for _, route := range h.cfg.Payment.Routes {
		accepts := make([]PaymentAccept, 0, len(route.Accepts))
		for _, accept := range route.Accepts {
			accepts = append(accepts, PaymentAccept{
				Scheme:            accept.Scheme,
				Network:           accept.Network,
				Asset:             accept.Asset,
				Amount:            accept.Amount,
				PayTo:             accept.PayTo,
				MaxTimeoutSeconds: accept.MaxTimeoutSeconds,
				Extra:             accept.Extra,
			})
		}
		routes = append(routes, EndpointPricing{
			Method:      route.Method,
			Path:        route.Path,
			Handler:     route.Handler,
			Description: route.Description,
			Accepts:     accepts,
		})
	}

	JSON(w, http.StatusOK, InfoResponse{
		Service:        "x402-go-server-example",
		Version:        vInfo.Version,
		Commit:         vInfo.Commit,
		BuildTime:      vInfo.BuildTime,
		FacilitatorURL: h.cfg.Facilitator.BaseURL,
		Pricing: PricingInfo{
			Routes: routes,
		},
	})
}
