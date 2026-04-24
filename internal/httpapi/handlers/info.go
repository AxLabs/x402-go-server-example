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
	Network        string      `json:"network"`
	NetworkName    string      `json:"networkName"`
	ChainID        string      `json:"chainId"`
	RPCURL         string      `json:"rpcUrl"`
	ExplorerURL    string      `json:"explorerUrl"`
	PaymentAsset   string      `json:"paymentAsset"`
	Scheme         string      `json:"scheme"`
	PayTo          string      `json:"payToAddress"`
	Pricing        PricingInfo `json:"pricing"`
}

// PricingInfo holds endpoint pricing information (USD strings resolved by SDK).
type PricingInfo struct {
	PaidHello EndpointPricing `json:"paidHello"`
	PaidEcho  EndpointPricing `json:"paidEcho"`
}

// EndpointPricing holds price information for an endpoint.
type EndpointPricing struct {
	Path  string `json:"path"`
	Price string `json:"price"`
}

// ServeHTTP handles GET /info requests.
func (h *InfoHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vInfo := version.Info()

	JSON(w, http.StatusOK, InfoResponse{
		Service:        "x402-go-server-example",
		Version:        vInfo.Version,
		Commit:         vInfo.Commit,
		BuildTime:      vInfo.BuildTime,
		FacilitatorURL: h.cfg.Facilitator.BaseURL,
		Network:        h.cfg.Payment.Network,
		NetworkName:    h.cfg.Network.Name,
		ChainID:        h.cfg.Network.ChainID,
		RPCURL:         h.cfg.Network.RPCURL,
		ExplorerURL:    h.cfg.Network.ExplorerURL,
		PaymentAsset:   h.cfg.Network.PaymentAsset,
		Scheme:         "exact",
		PayTo:          h.cfg.Payment.PayToAddress,
		Pricing: PricingInfo{
			PaidHello: EndpointPricing{Path: "/paid/hello", Price: h.cfg.Payment.PaidHelloPrice},
			PaidEcho:  EndpointPricing{Path: "/paid/echo", Price: h.cfg.Payment.PaidEchoPrice},
		},
	})
}
