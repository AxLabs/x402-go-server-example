# Architecture

`x402-go-server-example` is a deliberately thin wrapper around the [x402 Go SDK](https://github.com/x402-foundation/x402/go). The goal is that everything protocol-related is delegated to the SDK, and this repo only owns the operational concerns every HTTP service has: configuration, logging, routing, health, and info.

## Layer diagram

```text
┌───────────────────────────────────────────────────────────────────┐
│ cmd/server/main.go                                                │
│   • loads config.Config (env)                                     │
│   • builds internal/x402.Middleware (SDK-backed)                  │
│   • constructs httpapi.NewRouter                                  │
│   • runs net/http.Server with graceful shutdown                   │
└───────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│ internal/httpapi/router.go                                        │
│   chi.Mux with:                                                   │
│     • request-id + slog request-logging middleware                │
│     • GET /healthz, GET /info  (unpaid)                           │
│     • /paid subrouter:                                            │
│         Use(cfg.X402Middleware)  ← from internal/x402             │
│         GET  /paid/hello                                          │
│         POST /paid/echo                                           │
└───────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│ internal/x402/middleware.go                                       │
│   Single entrypoint that calls into the SDK:                      │
│     • x402http.NewHTTPFacilitatorClient({URL, Timeout})           │
│     • evmserver.NewExactEvmScheme()                               │
│     • nethttp.X402Payment(nethttp.Config{                         │
│         Routes:      RoutesConfig{…}                              │
│         Facilitator: …                                            │
│         Schemes:     {NewExactEvmScheme()}                        │
│       })                                                          │
│   Returns a plain func(http.Handler) http.Handler.                │
└───────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│ x402 Go SDK (github.com/x402-foundation/x402/go)                  │
│   • parses PAYMENT-SIGNATURE                                      │
│   • builds 402 / PAYMENT-REQUIRED                                 │
│   • verifies signatures via EVM exact scheme                      │
│   • calls facilitator /verify and /settle                         │
│   • writes PAYMENT-RESPONSE on success                            │
└───────────────────────────────────────────────────────────────────┘
```

## Component responsibilities

### `cmd/server`

- Wires config -> middleware -> router -> HTTP server.
- Owns graceful shutdown via `context.Context` and `SIGINT/SIGTERM`.

### `internal/config`

- Reads environment variables (with defaults).
- Validates required fields (`PAY_TO_ADDRESS`, prices, network, facilitator URL).
- Uses USD price strings and CAIP-2 networks so that nothing in this repo has to know about stablecoin contract addresses; the SDK resolves those through the facilitator's `/supported` response.

### `internal/logging`

- A tiny `log/slog` wrapper that respects `LOG_LEVEL`.
- No protocol knowledge.

### `internal/x402`

- The **only** package in the repo that imports the SDK.
- Exposes `Config`, `RouteSpec`, and `Middleware(Config) (func(http.Handler) http.Handler, error)`.
- Validates inputs and translates our `RouteSpec` list into the SDK's `x402http.RoutesConfig`.

### `internal/httpapi`

- `router.go` builds the chi router and plugs the SDK middleware onto the `/paid` subrouter via `r.Use(cfg.X402Middleware)`.
- `middleware/` contains request-id and request-logging middleware.
- `handlers/` contains **protocol-agnostic** handlers. They do not import the SDK and do not touch any payment-related context keys.

### `test/`

- Integration tests that build the full router against a mock facilitator (serving `/supported`, `/verify`, `/settle`) and assert:

  - unpaid routes return 200,
  - paid routes return the SDK-issued 402 with `PAYMENT-REQUIRED`,
  - unknown routes return a JSON 404.

## Design decisions

- **No custom x402 code.** Hand-rolling the protocol duplicates work the SDK already does and invites bugs on every spec change. The `internal/x402` package is a ~50-line factory and nothing more.
- **Handlers are ignorant of x402.** If the SDK middleware invokes a handler, payment already succeeded; the SDK writes `PAYMENT-RESPONSE` on the response after the handler returns. Handlers just write their JSON body.
- **USD pricing.** Prices are USD strings such as `$0.01`. The SDK maps these to on-chain amounts using the facilitator's supported kinds. This keeps `.env.example` readable and portable across networks.
- **CAIP-2 networks.** `eip155:84532` (Base Sepolia) is the default because the SDK ships a default stablecoin parser for it; production deployments typically override to `eip155:8453` (Base mainnet) or similar.

## What is deliberately absent

- No custom `X-Payment` / `X-Payment-Required` headers; the SDK uses `PAYMENT-SIGNATURE` / `PAYMENT-REQUIRED` / `PAYMENT-RESPONSE`.
- No hand-written facilitator client; `x402http.HTTPFacilitatorClient` is used directly.
- No verify/settle orchestration; `nethttp.X402Payment` does this.
- No payment models in `internal/`; the SDK's types are used where needed (and nowhere else, since handlers don't need them).
