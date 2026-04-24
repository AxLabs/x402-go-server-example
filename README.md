# x402-go-server-example

An example Go HTTP server for x402 that monetizes resources with the [x402](https://github.com/x402-foundation/x402) protocol. This repo is a **resource server** (seller) example: it exposes a handful of paid endpoints and delegates every part of the x402 protocol to the official [x402 Go SDK](https://github.com/x402-foundation/x402/go).

> **Protocol source of truth.** All request parsing, 402 challenge generation, header names (`PAYMENT-SIGNATURE`, `PAYMENT-REQUIRED`, `PAYMENT-RESPONSE`), facilitator communication and EIP-712 signature verification live in the SDK. This repo does not re-implement any of that.
>
> **Example scope.** This repository is intentionally small and educational. It demonstrates SDK integration patterns, not a complete production template.

---

## What this repo provides

- A chi router wired up to the SDK's `net/http` middleware.
- Two example paid endpoints (`GET /paid/hello`, `POST /paid/echo`) whose business handlers are intentionally protocol-agnostic.
- Two unpaid endpoints for operations (`GET /healthz`, `GET /info`).
- Structured `log/slog` logging with request IDs.
- Environment-driven configuration with sensible defaults.
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

- [internal/x402/middleware.go](internal/x402/middleware.go) — the only place where we construct SDK objects.
- [internal/httpapi/router.go](internal/httpapi/router.go) — plugs that middleware onto the `/paid` subrouter.
- [internal/httpapi/handlers/](internal/httpapi/handlers) — handlers that know nothing about x402.

For more detail see [docs/architecture.md](docs/architecture.md) and [docs/flow.md](docs/flow.md).

---

## How to use this example

Run this repo as the seller server:

```bash
cp .env.example .env
# edit .env: set PAY_TO_ADDRESS at minimum
make run
```

By default the server listens on `:8080`, points at `https://x402.org/facilitator`, and prices `/paid/hello` at `$0.01` and `/paid/echo` at `$0.005` on `eip155:84532` (Base Sepolia). The SDK resolves the concrete asset (USDC) via the facilitator's `/supported` endpoint at startup.

### Step 1: Call free and paid routes

Unauthenticated requests to a paid route return an SDK-generated 402 with the `PAYMENT-REQUIRED` header carrying the accepts list:

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
Run this server in one terminal, then follow the client repo instructions to:

- request a paid route and read `PAYMENT-REQUIRED`,
- build/sign the payment payload,
- retry with `PAYMENT-SIGNATURE`.

On success, this server returns `200` and includes `PAYMENT-RESPONSE` with the settlement transaction hash.

---

## Goal of this example

This repository focuses on one thing: showing the cleanest way to add x402 to a Go HTTP server by keeping business handlers x402-agnostic and letting the SDK own protocol behavior.

---

## Configuration

See [.env.example](.env.example) for the full list. Key variables:

| Variable               | Required | Default                          | Notes                                  |
|------------------------|----------|----------------------------------|----------------------------------------|
| `PAY_TO_ADDRESS`       | yes      | —                                | Seller EOA.                            |
| `PAYMENT_NETWORK`      | no       | `eip155:84532`                   | CAIP-2 id.                             |
| `PAID_HELLO_PRICE`     | no       | `$0.01`                          | USD string.                            |
| `PAID_ECHO_PRICE`      | no       | `$0.005`                         | USD string.                            |
| `PAYMENT_MAX_TIMEOUT`  | no       | `300s`                           | Advertised on 402.                     |
| `FACILITATOR_BASE_URL` | no       | `https://x402.org/facilitator`   | Must speak x402.                       |
| `FACILITATOR_TIMEOUT`  | no       | `30s`                            |                                        |
| `SERVER_ADDR`          | no       | `:8080`                          |                                        |
| `LOG_LEVEL`            | no       | `info`                           | `debug\|info\|warn\|error`             |

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
- Integration tests that spin up a mock facilitator and verify the SDK middleware issues the correct 402 challenge on unauthenticated requests ([test/integration_test.go](test/integration_test.go)).

A full verify/settle happy-path integration test is intentionally out of scope here: it requires a real EVM signer and is better covered in the x402 SDK's own test suite.

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
