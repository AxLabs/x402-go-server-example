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

Unauthenticated requests to a paid route return an SDK-generated 402. The `PAYMENT-REQUIRED` header carries the base64-encoded payment requirements JSON.

HTTP header names are case-insensitive; Go renders this header as `Payment-Required` in `curl -i` output:

```bash
curl -i http://localhost:8080/paid/hello
# HTTP/1.1 402 Payment Required
# Payment-Required: <base64-encoded-payment-requirements-json>
```

Decode helper:

```bash
curl -si http://localhost:8080/paid/hello \
  | awk -F': ' 'tolower($1) == "payment-required" {print $2}' \
  | tr -d '\r\n' \
  | base64 --decode | jq .
```

The health and info endpoints are always free:

```bash
curl -s http://localhost:8080/healthz | jq
curl -s http://localhost:8080/info | jq
```

`/healthz` is a liveness endpoint for health checks. `/info` returns service metadata and the configured paid routes with their payment options.

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
| `FACILITATOR_TIMEOUT`  | no       | `120s`                           |                                        |
| `PAYMENT_CONFIG_FILE`  | yes      | —                                | Path to YAML accepts config.           |
| `SERVER_ADDR`          | no       | `:8080`                          |                                        |
| `READ_TIMEOUT`         | no       | `120s`                           | HTTP server read timeout.              |
| `WRITE_TIMEOUT`        | no       | `120s`                           | HTTP server write timeout.             |
| `REQUEST_TIMEOUT`      | no       | `120s`                           | x402 middleware request timeout.       |
| `SHUTDOWN_TIMEOUT`     | no       | `30s`                            | Graceful shutdown timeout.             |
| `LOG_LEVEL`            | no       | `info`                           | `debug\|info\|warn\|error`             |
| `CORS_ALLOWED_ORIGINS` | no       | —                                | Comma-separated browser origins (React client URL). |

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

## Deploying on Coolify

This repo ships a `Dockerfile`, `docker-compose.yml`, and `docker-compose.coolify.yml` for production-style deployment (same pattern as the [ax402](https://github.com/AxLabs/ax402) facilitator wrapper).

### Prerequisites

- A running **x402 facilitator** reachable from the server container (e.g. your Coolify facilitator URL).
- `payment-config.example.yaml` updated with your real `payTo` addresses (or add `payment-config.yaml` and point `PAYMENT_CONFIG_FILE` at it).

### Coolify setup

1. **New Application** → connect GitHub → select `x402-go-server-example`.
2. **Build Pack**: **Docker Compose** (not Nixpacks — this project needs Go 1.24).
3. **Base directory**: `/` · **Compose file**: `docker-compose.coolify.yml` (when facilitator is on the same Coolify host).
4. Set environment variables (minimum):

   | Variable | Example |
   |----------|---------|
   | `FACILITATOR_BASE_URL` | `https://facilitator.yourdomain.com` |
   | `FACILITATOR_TIMEOUT` | `120s` |
   | `PAYMENT_CONFIG_FILE` | `/app/payment-config.yaml` (default in image) |
   | `SERVER_HOST_PORT` | `8080` or another free host port |

5. **Domain** → container port **8080** · **Health check**: `GET /healthz`.
6. Deploy.

7. If the React client runs on another domain, set `CORS_ALLOWED_ORIGINS` to that origin (comma-separated), e.g. `https://t8pyvsim60q2xbll4i7r9wz0.app.mf.axlabs.net`.

The server syncs with the facilitator on startup. If `FACILITATOR_BASE_URL` is wrong or the facilitator is down, the container will exit during boot.

### Reaching the facilitator from the server container (Coolify)

Coolify runs each app on its own Docker network. **`http://172.17.0.1:8088` often times out** — that is not a reliable way to reach another stack.

**Recommended:** join the facilitator’s Docker network via compose (Coolify env), then use the **internal** service URL (container port **8080**, not host **8088**).

**Stable vs ephemeral names**

| Stable (use these) | Changes every deploy (do not use in URLs) |
|--------------------|---------------------------------------------|
| Docker network `hg20xignraizz6za0lx0ydja` (Coolify app UUID) | Container `facilitator-hg20xignraizz6za0lx0ydja-105356082802` |
| Compose hostname `facilitator` (port **8080**) | Host port `8088` (only for curl on the VPS host) |

1. On the Coolify host, find the facilitator network name:

   ```bash
   docker ps --format '{{.Names}}' | grep facilitator
   # example: facilitator-hg20xignraizz6za0lx0ydja-075039758871

   docker inspect facilitator-hg20xignraizz6za0lx0ydja-075039758871 \
     --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{"\n"}}{{end}}'
   # example output: hg20xignraizz6za0lx0ydja
   ```

2. On the **server** Coolify app, set:

   ```env
   FACILITATOR_DOCKER_NETWORK=hg20xignraizz6za0lx0ydja
   FACILITATOR_BASE_URL=http://facilitator:8080
   ```

   Use your actual network name from step 1 (often the facilitator app’s UUID).

3. In Coolify **Docker Compose Location**, set `./docker-compose.coolify.yml` (not the base file alone).

4. Redeploy the server.

5. If `lookup facilitator` still fails, resolve the live hostname from the shared network:

   ```bash
   docker network inspect hg20xignraizz6za0lx0ydja \
     --format '{{range .Containers}}{{.Name}} {{.IPv4Address}}{{"\n"}}{{end}}'
   ```

   Then try `FACILITATOR_BASE_URL=http://<facilitator-container-name>:8080` or `http://<ip>:8080`.

**Note:** Coolify’s **Connect to Predefined Network** checkbox alone often does **not** register the `facilitator` DNS name (you may see `lookup facilitator … server misbehaving`). Use `docker-compose.coolify.yml` + `FACILITATOR_DOCKER_NETWORK` instead (you can uncheck Predefined Network).

**Verify from a running server container:**

```bash
docker exec -it <server-container> wget -qO- http://facilitator:8080/health
```

**Alternatives:**

| `FACILITATOR_BASE_URL` | When it works |
|------------------------|----------------|
| `http://facilitator:8080` | Same Docker network (best on Coolify) |
| `https://ax402.app.mf.axlabs.net` | Public HTTPS + domain configured on facilitator |
| `http://host.docker.internal:8088` | Host port publish + `extra_hosts` (may still fail on some hosts) |

Do **not** use host port `8088` in the URL when talking to the `facilitator` service name — use **8080** (inside the container).

### Local Docker Compose

```bash
cp .env.example .env
# edit .env: FACILITATOR_BASE_URL, payment YAML payTo addresses
docker compose up -d --build
docker compose logs -f server
```

Verify:

```bash
curl -s http://localhost:8080/healthz | jq
curl -si http://localhost:8080/paid/hello | head -20
```

If port `8080` is taken on the host, set `SERVER_HOST_PORT` in `.env` (e.g. `8081`).

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
internal/httpapi/           chi router + request logging, request IDs, CORS, recovery
internal/httpapi/handlers/  business handlers (health, info, paid_hello, paid_echo)
test/                       integration tests
docs/                       architecture + request flow notes
Dockerfile                  production image (Go 1.24, Alpine runtime)
docker-compose.yml          local / Coolify compose service
```
