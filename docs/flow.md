# Request flow

This document traces an HTTP request from the client to a paid handler,
showing where each concern is handled. The key point: almost everything is
in the x402 Go SDK; this repo just arranges chi and hands the SDK its
config.

## Unpaid request — e.g. `GET /healthz`

```
Client ──► chi.Router ──► request-id mw ──► request-logging mw ──► healthHandler ──► 200
```

No SDK code runs. The `/paid` subrouter is never entered.

## Paid request — unauthenticated

```
Client ──► chi.Router ──► request-id mw ──► logging mw ──► /paid subrouter
                                                               │
                                                               ▼
                                                 SDK middleware (nethttp.X402Payment)
                                                               │
                                                        no PAYMENT-SIGNATURE
                                                               │
                                                               ▼
                                               402 Payment Required
                                               PAYMENT-REQUIRED: {x402Version, accepts:[…]}
                                               body: null
```

What the SDK does here:
1. Inspects `PAYMENT-SIGNATURE` — finds it missing.
2. Looks up the route in its `RoutesConfig`.
3. Builds a `PaymentRequirements` list using the `PaymentOption` we supplied
   (scheme=`exact`, network, payTo, price, timeout).
4. Serializes that into the `PAYMENT-REQUIRED` response header and returns
   HTTP 402. The business handler is never invoked.

## Paid request — authenticated

```
Client ──► chi.Router ──► request-id mw ──► logging mw ──► /paid subrouter
                                                               │
                                                               ▼
                                                 SDK middleware
                                                               │
                                                   PAYMENT-SIGNATURE present
                                                               │
                                        ┌──────────────────────┴───────────────────────┐
                                        │  SDK decodes base64 JSON, picks the          │
                                        │  PaymentPayload whose (scheme, network)      │
                                        │  matches our route, hands it to the EVM      │
                                        │  exact scheme server.                        │
                                        └──────────────────────┬───────────────────────┘
                                                               │
                                                               ▼
                                           EVM exact scheme structural checks
                                                               │
                                                               ▼
                                     HTTPFacilitatorClient.Verify(payload, requirements)
                                                               │
                                                               ▼
                                                     verify OK? ─── no ──► 402
                                                               │ yes
                                                               ▼
                                                      business handler runs
                                                               │
                                                               ▼
                                     HTTPFacilitatorClient.Settle(payload, requirements)
                                                               │
                                                               ▼
                                               PAYMENT-RESPONSE: {…, transaction}
                                                               │
                                                               ▼
                                                       200 OK + body
```

Things worth noting:

- **Order**: verify → run handler → settle. This is the SDK's default for
  `X402Payment`; it ensures we don't pay before we know we can produce the
  resource, and don't produce the resource before we know the payer is valid.
- **No context leakage**: the handler is not given any payment fields from
  the SDK. If a handler ever needs payer/transaction data, the SDK exposes
  it; we chose not to expose it because our example handlers don't need it.
- **`PAYMENT-RESPONSE` header** is written by the SDK after the handler
  returns but before the response is flushed to the client. The client uses
  this to learn the settlement tx hash.

## Error paths

- **Invalid JSON body on `POST /paid/echo`** — `paid_echo` handler returns
  400 **before** any body processing occurs. The SDK middleware has already
  verified payment at this point, so a settlement attempt still follows
  only on success. (Our example keeps this simple; a real server would
  tie settlement to a successful handler outcome more carefully.)
- **Unknown path** — chi's `NotFoundHandler` returns a JSON 404; the SDK
  middleware is never reached because the path is not matched by the
  `/paid` subrouter.
- **Facilitator unreachable at startup** — `nethttp.X402Payment` with
  `SyncFacilitatorOnStart: true` surfaces this as an error from
  `x402.Middleware(...)`, which `cmd/server/main.go` treats as fatal.

## Where to look in the code

| Step                             | File                                                               |
|----------------------------------|--------------------------------------------------------------------|
| Env → Config                     | [internal/config/config.go](../internal/config/config.go)          |
| SDK wiring                       | [internal/x402/middleware.go](../internal/x402/middleware.go)      |
| chi routes + middleware plug-in  | [internal/httpapi/router.go](../internal/httpapi/router.go)        |
| Business handlers (no x402)      | [internal/httpapi/handlers/](../internal/httpapi/handlers)         |
| Startup, graceful shutdown       | [cmd/server/main.go](../cmd/server/main.go)                        |
| Mock-facilitator integration test| [test/integration_test.go](../test/integration_test.go)            |
