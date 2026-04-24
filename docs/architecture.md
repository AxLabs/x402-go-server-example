# Architecture

`x402-go-server-example` is a deliberately thin wrapper around the [x402 Go SDK](https://github.com/x402-foundation/x402/go). Everything protocol-related is delegated to the SDK, and this repo owns only operational concerns common to HTTP services: configuration, logging, routing, health, and info.

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
│ x402 Go SDK (github.com/x402-foundation/x402/go)                 │
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
- Validates required fields (`PAY_TO_ADDRESS`, prices, network profile, facilitator URL).
- Keeps network differences in config (`NETWORK_NAME`, `CHAIN_ID`, `RPC_URL`, `EXPLORER_URL`, `PAYMENT_ASSET`) and derives SDK CAIP-2 network values from `CHAIN_ID`.

### `internal/logging`

- A tiny `log/slog` wrapper that respects `LOG_LEVEL`.
- No protocol knowledge.

### `internal/x402`

- The **only** package in the repo that imports the SDK.
- Exposes `Config`, `RouteSpec`, and `Middleware(Config) (func(http.Handler) http.Handler, error)`.
- Validates inputs and translates our `RouteSpec` list into the SDK `x402http.RoutesConfig`.

### `internal/httpapi`

- `router.go` builds the chi router and plugs SDK middleware onto `/paid` via `r.Use(cfg.X402Middleware)`.
- `middleware/` contains request-id and request-logging middleware.
- `handlers/` contains protocol-agnostic handlers that do not import the SDK and do not touch payment context keys.

### `test/`

- Integration tests build the full router against a mock facilitator (`/supported`, `/verify`, `/settle`) and assert:

  - unpaid routes return 200,
  - paid routes return SDK-issued 402 with `PAYMENT-REQUIRED`,
  - unknown routes return JSON 404.

## Design decisions

- **No custom x402 code.** Hand-rolling protocol behavior duplicates SDK logic and creates spec-drift risk.
- **Handlers are x402-agnostic.** If SDK middleware invokes a handler, payment checks have already passed; SDK writes `PAYMENT-RESPONSE` after handler execution.
- **Config-driven network selection.** Base Sepolia is the current default profile (`CHAIN_ID=84532` -> `eip155:84532`) due to go-ethereum v1.16 compatibility. Neo X Testnet (`CHAIN_ID=12227332` -> `eip155:12227332`) is the intended primary profile and remains fully prepared in configuration — switching back requires only changing environment variables.
- **USD pricing.** Prices stay as readable USD strings (for example `$0.01`) and are resolved by the SDK using facilitator-supported kinds.

## What is deliberately absent

- No custom `X-Payment` / `X-Payment-Required` headers; SDK uses `PAYMENT-SIGNATURE` / `PAYMENT-REQUIRED` / `PAYMENT-RESPONSE`.
- No hand-written facilitator client; `x402http.HTTPFacilitatorClient` is used directly.
- No verify/settle orchestration; `nethttp.X402Payment` handles it.
- No duplicated payment models in `internal/`; SDK types are used where needed.
