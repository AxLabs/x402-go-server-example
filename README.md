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
        └── configured paid route│
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
- [internal/httpapi/router.go](internal/httpapi/router.go) — dynamically registers configured paid routes and applies SDK middleware per route.
- [internal/httpapi/handlers/](internal/httpapi/handlers) — handlers that know nothing about x402.

For more detail see [docs/architecture.md](docs/architecture.md) and [docs/flow.md](docs/flow.md).

---

## How to use this example

Run this repo as the seller server:

```bash
cp .env.example .env
# edit .env: set FACILITATOR_BASE_URL and PAYMENT_CONFIG_FILE
# edit payment-config.example.yaml with your real addresses/amounts
make run
```

This server reads paid-route definitions from `PAYMENT_CONFIG_FILE`. Each route declares a concrete business `handler` (`paid_hello` or `paid_echo`) plus explicit `accepts` options (`scheme`, `network`, `asset`, `amount`, `payTo`, optional `maxTimeoutSeconds`, optional `extra`) that map directly to x402 `PaymentRequirements`.

You can offer multiple accepts per route (e.g. Base Sepolia USDC via `eip3009` and ZCHF via `permit2`). See `payment-config.example.yaml`.

### Step 1: Call free and paid routes

Unauthenticated requests to a paid route return an SDK-generated 402 with the `PAYMENT-REQUIRED` header carrying a base64-encoded JSON challenge with the accepts list:

```bash
curl -i http://localhost:8080/paid/hello
# HTTP/1.1 402 Payment Required
# PAYMENT-REQUIRED: eyJ4NDAyVmVyc2lvbiI6MiwiYWNjZXB0cyI6W119
```

Decode helper:

```bash
curl -si http://localhost:8080/paid/hello \
  | awk -F': ' '/^PAYMENT-REQUIRED:/ {print $2}' \
  | tr -d '\r\n' \
  | base64 --decode | jq .
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
| `FACILITATOR_BASE_URL` | yes      | —                                | Must speak x402.                       |
| `FACILITATOR_TIMEOUT`  | no       | `30s`                            |                                        |
| `PAYMENT_CONFIG_FILE`  | yes      | —                                | Path to YAML accepts config.           |
| `SERVER_ADDR`          | no       | `:8080`                          |                                        |
| `LOG_LEVEL`            | no       | `info`                           | `debug\|info\|warn\|error`             |

### Payment YAML structure

Use [payment-config.example.yaml](payment-config.example.yaml) as the template. The canonical shape is:

```yaml
payment:
  routes:
    - method: GET
      path: /paid/hello
      handler: paid_hello
      description: Paid hello resource
      accepts:
        - scheme: exact
          network: eip155:12227332
          asset: 0x2222222222222222222222222222222222222222
          amount: "1000000000000000000"
          payTo: 0x1111111111111111111111111111111111111111
          maxTimeoutSeconds: 300
          extra:
            name: xGAS
            version: "1"
            assetTransferMethod: eip3009
```

`handler` is required per route and must be one of `paid_hello` or `paid_echo`. `scheme` is currently constrained to `exact`.

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
