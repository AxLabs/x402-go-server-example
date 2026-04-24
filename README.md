# x402-go-server-example

An example Go HTTP server for x402 that monetizes resources with the [x402](https://github.com/x402-foundation/x402) protocol. This repo is a **resource server** (seller) example: it exposes paid endpoints and delegates protocol handling to the official [x402 Go SDK](https://github.com/x402-foundation/x402/go).

> **Protocol source of truth.** All request parsing, 402 challenge generation, header names (`PAYMENT-SIGNATURE`, `PAYMENT-REQUIRED`, `PAYMENT-RESPONSE`), facilitator communication, and EIP-712 signature verification live in the SDK.
>
> **Example scope.** This repository is intentionally small and educational. It demonstrates SDK integration patterns, not a complete production template.

---

## What this repo provides

- A chi router wired to the SDK `net/http` middleware.
- Two example paid endpoints (`GET /paid/hello`, `POST /paid/echo`) with protocol-agnostic business handlers.
- Two unpaid endpoints (`GET /healthz`, `GET /info`).
- Structured `log/slog` logging with request IDs.
- Environment-driven configuration with network profile selection.
- A mock-facilitator-backed integration test suite.

---

## Architecture in one picture

```text
        ┌───────── HTTP ─────────┐
        │                        │
    chi router                   │
        │                        │
        ├── /healthz, /info ─────┼──► business handler (no payment)
        │                        │
        └── /paid/* subrouter    │
                │                │
          SDK middleware         │
          (nethttp.X402Payment)  │
                │                │
          ┌─────┴─────────────┐  │
          │ EVM exact scheme  │  │
          │  + Facilitator    │  │
          │    client         │  │
          └───────────────────┘  │
                │                │
            business handler ────┘
```

- [internal/x402/middleware.go](internal/x402/middleware.go) constructs SDK objects.
- [internal/httpapi/router.go](internal/httpapi/router.go) applies x402 middleware on the `/paid` subrouter.
- [internal/httpapi/handlers/](internal/httpapi/handlers) contains handlers that do not depend on x402 internals.

For more detail see [docs/architecture.md](docs/architecture.md) and [docs/flow.md](docs/flow.md).

---

## How to use this example

Run this repo as the seller server:

```bash
cp .env.example .env
# edit .env: set PAY_TO_ADDRESS at minimum
make run
```

By default this example uses **Base Sepolia** (`NETWORK_NAME=base-sepolia`, `CHAIN_ID=84532`) as the active network profile. This is a temporary default due to dependency compatibility (see [Network configuration](#network-configuration) below).

### Step 1: Call free and paid routes

Unauthenticated requests to a paid route return an SDK-generated 402 with `PAYMENT-REQUIRED`:

```bash
curl -i http://localhost:8080/paid/hello
# HTTP/1.1 402 Payment Required
# PAYMENT-REQUIRED: {"x402Version":2,"accepts":[...]}
```

The health and info endpoints are always free:

```bash
curl -s http://localhost:8080/healthz | jq
curl -s http://localhost:8080/info | jq
```

### Step 2: Use the client example to pay

Use the companion client repository, `x402-go-client-example`, as the payer.

- Request a paid route and read `PAYMENT-REQUIRED`.
- Build and sign the payment payload.
- Retry with `PAYMENT-SIGNATURE`.

On success this server returns `200` and includes `PAYMENT-RESPONSE` with the settlement transaction hash.

### Step 3: Switch to Neo X Testnet (prepared profile)

Neo X Testnet is the intended primary network for this example and is fully supported by configuration. It is temporarily inactive because the current dependency stack requires go-ethereum v1.16, which Neo X does not yet support. To switch, set these values in `.env`:

```bash
NETWORK_NAME=neo-x-testnet
CHAIN_ID=12227332
RPC_URL=https://neoxt4seed1.ngd.network
EXPLORER_URL=https://xt4scan.ngd.network
PAYMENT_ASSET=USDC
# PAYMENT_NETWORK=eip155:12227332
```

No code changes are required — only configuration.

---

## Goal of this example

This repository focuses on one thing: showing a clean way to add x402 to a Go HTTP server while keeping business handlers x402-agnostic and keeping network selection configuration-driven.

---

## Configuration

See [.env.example](.env.example) for the full list.

| Variable | Required | Default | Notes |
|---|---|---|---|
| `NETWORK_NAME` | no | `base-sepolia` | Network profile label used for docs/logging/info output. |
| `CHAIN_ID` | no | `84532` | Primary network selector; used to derive `eip155:<chain_id>`. |
| `RPC_URL` | no | `https://sepolia.base.org` | RPC endpoint for the selected profile. |
| `EXPLORER_URL` | no | `https://sepolia.basescan.org` | Explorer URL for the selected profile. |
| `PAYMENT_ASSET` | no | `USDC` | Asset label for operational context. |
| `PAYMENT_NETWORK` | no | derived from `CHAIN_ID` | Optional compatibility override for CAIP-2 network. |
| `PAY_TO_ADDRESS` | yes | — | Seller EOA receiving payments. |
| `FACILITATOR_BASE_URL` | no | `https://x402.org/facilitator` | Must support the configured network. |
| `FACILITATOR_TIMEOUT` | no | `30s` | Timeout for facilitator HTTP requests. |
| `PAID_HELLO_PRICE` | no | `$0.01` | USD string resolved by SDK against facilitator support. |
| `PAID_ECHO_PRICE` | no | `$0.005` | USD string resolved by SDK against facilitator support. |
| `PAYMENT_MAX_TIMEOUT` | no | `300s` | Advertised on 402 payment requirements. |
| `SERVER_ADDR` | no | `:8080` | HTTP listen address. |
| `LOG_LEVEL` | no | `info` | `debug\|info\|warn\|error`. |

---

## Network configuration

All network-specific values are centralized in the configuration layer and selected via environment variables. The server is fully chain-agnostic — no network-specific logic exists in handlers or protocol code.

**Current default: Base Sepolia.** This is a temporary default. The project's dependency stack requires go-ethereum v1.16, which Neo X does not yet support. Base Sepolia is used in the meantime for compatibility.

**Prepared profile: Neo X Testnet.** Neo X Testnet is the intended primary network. Its configuration is preserved in `.env.example` and in the source code comments. Once Neo X supports go-ethereum v1.16, switching back requires only changing environment variables — no code changes.

---

## Development

```bash
make build    # build ./bin/x402-server
make test     # go test -v -race ./...
make run      # go run ./cmd/server
```

The test suite includes:

- Unit tests for config loading ([internal/config/config_test.go](internal/config/config_test.go)).
- Handler-level tests for `/paid/hello` and `/paid/echo`.
- Integration tests with a mock facilitator that verify SDK middleware behavior on unauthenticated paid requests ([test/integration_test.go](test/integration_test.go)).

---

## Repository layout

```text
cmd/server/                 entrypoint (loads config, builds middleware, serves)
internal/config/            env-driven configuration + tests
internal/logging/           slog wrapper
internal/version/           build-info vars
internal/x402/              thin factory over the x402 Go SDK (middleware.go)
internal/httpapi/           chi router + request-logging + request-id middleware
internal/httpapi/handlers/  business handlers (health, info, paid_hello, paid_echo)
test/                       integration tests
docs/                       architecture + request flow notes
```
