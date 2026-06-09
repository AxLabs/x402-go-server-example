# Architecture

`x402-go-server-example` is a deliberately thin wrapper around the [x402 Go SDK](https://github.com/x402-foundation/x402/go). The goal is that everything protocol-related is delegated to the SDK, and this repo only owns the operational concerns every HTTP service has: configuration, logging, routing, health, and info.

## Layer diagram

```text
┌───────────────────────────────────────────────────────────────────┐
│ cmd/server/main.go                                                │
│   • loads config.Config (env + PAYMENT_CONFIG_FILE YAML)          │
│   • builds internal/x402.Middleware (SDK-backed)                  │
│   • constructs httpapi.NewRouter                                  │
│   • runs net/http.Server with graceful shutdown                   │
└───────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│ internal/httpapi/router.go                                        │
│   chi.Mux with:                                                   │
│     • slog request logging with X-Request-ID propagation          │
│     • GET /healthz, GET /info  (unpaid)                           │
│     • paid routes loaded from PAYMENT_CONFIG_FILE                 │
│       each route gets cfg.X402Middleware and a bound handler      │
└───────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│ internal/x402/middleware.go                                       │
│   Single entrypoint that calls into the SDK:                      │
│     • x402http.NewHTTPFacilitatorClient({URL, Timeout})           │
│     • x402http.Newx402HTTPResourceServer(RoutesConfig{…}, …)      │
│     • registers evmserver.NewExactEvmScheme() per network         │
│     • initializes facilitator support at startup                   │
│     • nethttp.PaymentMiddlewareFromHTTPServer(…)                  │
│   Returns a plain func(http.Handler) http.Handler.                │
└───────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌───────────────────────────────────────────────────────────────────┐
│ x402 Go SDK (github.com/x402-foundation/x402/go)                  │
│   • parses PAYMENT-SIGNATURE                                      │
│   • builds 402 / PAYMENT-REQUIRED (base64-encoded JSON challenge) │
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
- Reads `PAYMENT_CONFIG_FILE` and loads route-level accepts from YAML.
- Validates required fields (`FACILITATOR_BASE_URL`, `payment.routes[*].handler`, `payment.routes[*].accepts[*]`).
- Normalizes `method` to uppercase and enforces `scheme=exact` for now.
- Uses explicit `scheme/network/asset/amount/payTo` entries matching x402 `PaymentRequirements`.

### `internal/logging`

- A tiny `log/slog` wrapper that respects `LOG_LEVEL`.
- No protocol knowledge.

### `internal/x402`

- The **only** package in the repo that imports the SDK.
- Exposes `Config`, `RouteSpec`, `AcceptOption`, and `Middleware(Config) (func(http.Handler) http.Handler, error)`.
- Translates explicit route `Accepts` entries directly into the SDK's `x402http.RoutesConfig`.

### `internal/httpapi`

- `router.go` builds the chi router and dynamically registers paid routes from config.
- Each configured paid route is bound to a concrete handler (`paid_hello` or `paid_echo`) and wrapped with the SDK middleware.
- `middleware/` contains CORS, recovery, JSON content-type, and request logging; the request logger also propagates/generates `X-Request-ID`.
- `handlers/` contains **protocol-agnostic** handlers. They do not import the SDK and do not touch any payment-related context keys.

### `test/`

- Integration tests that build the full router against a mock facilitator (serving `/supported`, `/verify`, `/settle`) and assert:

  - unpaid routes return 200,
  - paid routes return the SDK-issued 402 with `PAYMENT-REQUIRED`,
  - unknown routes return a JSON 404.

## Design decisions

- **No custom x402 code.** Hand-rolling the protocol duplicates work the SDK already does and invites bugs on every spec change. The `internal/x402` package is a ~50-line factory and nothing more.
- **Handlers are ignorant of x402.** If the SDK middleware invokes a handler, payment already succeeded; the SDK writes `PAYMENT-RESPONSE` on the response after the handler returns. Handlers just write their JSON body.
- **Explicit accepts only.** Every payment option is configured as an explicit accept entry with concrete `asset` and `amount`.
- **No primary/default distinction.** The server uses one uniform `accepts` model that mirrors the x402 API shape.
- **Explicit handler binding.** Every paid route declares its business handler in config; startup fails for unsupported handlers.
- **Exact-only scheme policy.** Config validation currently allows only `scheme: exact` until additional scheme servers are wired.
- **CAIP-2 networks.** Each accept entry specifies its own network, so routes can advertise multi-chain options directly.

## What is deliberately absent

- No custom `X-Payment` / `X-Payment-Required` headers; the SDK uses `PAYMENT-SIGNATURE` / `PAYMENT-REQUIRED` / `PAYMENT-RESPONSE`.
- No hand-written facilitator client; `x402http.HTTPFacilitatorClient` is used directly.
- No verify/settle orchestration; the SDK resource server and net/http middleware do this.
- No payment models in `internal/`; the SDK's types are used where needed (and nowhere else, since handlers don't need them).
