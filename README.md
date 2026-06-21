# ms_home_liverpool

Go microservice that serves the **Home page layout** for Liverpool's digital channels. It replaces the Home logic from the `digital_bff` Node/NestJS monorepo and is designed to run on **Google Cloud Run**.

- Static blocks are resolved inline and ready to render.
- Dynamic blocks (personalized, session-dependent) are returned as **placeholders** — the frontend calls their individual resolve endpoints independently.
- Every outbound call is wrapped in a **circuit breaker** (no retries, 5 % failure threshold).
- Failure of one block never blanks the page.

---

## Table of contents

1. [Requirements](#requirements)
2. [Quick start](#quick-start)
3. [Configuration](#configuration)
4. [API reference](#api-reference)
   - [GET /home](#get-home)
   - [GET /home/blocks/{blockType}](#get-homeblocksblocktype)
   - [GET /healthz](#get-healthz)
   - [GET /readyz](#get-readyz)
5. [Error contract](#error-contract)
6. [cURL examples](#curl-examples)
7. [Running tests](#running-tests)
8. [Project structure](#project-structure)
9. [Architecture overview](#architecture-overview)
10. [Adding a new block resolver](#adding-a-new-block-resolver)
11. [Circuit breaker behaviour](#circuit-breaker-behaviour)
12. [Logging](#logging)
13. [OpenTelemetry](#opentelemetry)
14. [Docker build](#docker-build)
15. [Deployment (Cloud Run)](#deployment-cloud-run)

---

## Requirements

| Tool | Version |
|------|---------|
| Go   | 1.26.4  |
| Docker | 24+ (optional, for container builds) |
| `gcloud` CLI | any recent (optional, for Cloud Run deploy) |

---

## Quick start

```bash
# 1. Clone and enter the repo
git clone https://github.com/YMARTINEZM08/ms_home_ref.git ms_home_liverpool
cd ms_home_liverpool

# 2. Copy the env template and fill in your values
cp configs/.env.example configs/.env.local

# 3. Source the env and run
export $(grep -v '^#' configs/.env.local | xargs)
go run ./cmd/home
```

The server starts on `PORT` (default **8080**).

```
{"time":"…","level":"INFO","msg":"server listening","addr":":8080"}
```

> **Minimum required env var:** `CONTENT_SERVICE_URL` — the service will exit on startup without it.

---

## Configuration

All configuration is injected through environment variables. No config files are read at runtime.

| Variable | Required | Default | Description |
|---|---|---|---|
| `PORT` | no | `8080` | HTTP listen port |
| `SERVICE_NAME` | no | `ms-home-liverpool` | Used in OTel resource and access logs |
| `ENVIRONMENT` | no | `dev` | `dev` \| `qa` \| `staging` \| `prod` |
| `LOG_LEVEL` | no | `INFO` | `OFF` \| `ERROR` \| `WARN` \| `INFO` \| `DEBUG` \| `TRACE` |
| `CONTENT_SERVICE_URL` | **yes** | — | Base URL of the internal content-service proxy (e.g. `http://content-service:8090`) |
| `CONTENT_SERVICE_TIMEOUT_MS` | no | `30000` | Per-request timeout in milliseconds for content-service calls |
| `DEFAULT_BRAND` | no | `LP` | Fallback brand when `x-brand-id` header is absent |
| `BREAKER_FAILURE_RATIO` | no | `0.05` | Fraction of failures that trips the circuit breaker (0–1) |
| `BREAKER_MIN_REQUESTS` | no | `20` | Minimum requests in window before the ratio is evaluated |
| `BREAKER_OPEN_TIMEOUT_S` | no | `30` | Seconds the breaker stays open before a half-open probe |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | no | `""` | `host:port` of OTLP collector; tracing is a no-op when empty |
| `OTEL_SAMPLE_RATIO` | no | `1.0` | Trace sampling ratio (0.0–1.0) |

### Changing log level at runtime

`LOG_LEVEL` is read once at startup into a `slog.LevelVar`. To change it without redeploying, update the env variable in Cloud Run and trigger a new revision, or send `SIGHUP` locally (the level var is not yet wired to a signal — set `LOG_LEVEL` before starting for now).

---

## API reference

### `GET /home`

Returns the ordered Home page layout.

#### Request

| Where | Name | Required | Description |
|---|---|---|---|
| Query | `locale` | no | BCP-47 locale, e.g. `es-mx`. Defaults to `es-mx`. |
| Query | `channel` | no | `pocket` \| `kiosk` \| `mpos`. Omit for web. |
| Header | `x-brand-id` | no | Brand code, e.g. `LP`. Defaults to `LP`. Append `-PREVIEW` for preview mode. |
| Header | `x-preview` | no | `true` to request preview content. |
| Header | `x-correlation-id` | no | Propagated through logs and traces. Must match `^[a-zA-Z0-9\-_]{1,64}$`. |

#### Response `200 OK`

```json
{
  "blocks": [
    {
      "kind": "static",
      "id":   "uid-abc123",
      "type": "banner",
      "content": {
        "title": "Verano Liverpool",
        "image_url": "https://cdn.example.com/hero.jpg"
      }
    },
    {
      "kind":             "dynamic",
      "id":               "uid-def456",
      "type":             "products_list",
      "resolve_endpoint": "/home/blocks/products_list",
      "fallback":         "popular_products",
      "feature_flag_id":  "flag-products-list",
      "enabled":          true
    },
    {
      "kind":             "dynamic",
      "id":               "uid-ghi789",
      "type":             "container_greeting",
      "resolve_endpoint": "/home/blocks/container_greeting",
      "fallback":         "",
      "feature_flag_id":  "flag-greeting",
      "enabled":          false
    }
  ]
}
```

**Static blocks** (`kind: "static"`) carry their full `content` object — safe to render immediately and cache.

**Dynamic blocks** (`kind: "dynamic"`) carry only the placeholder contract:
- `resolve_endpoint` — path to call to get the block's data.
- `fallback` — identifier the frontend uses when the block is disabled or its endpoint fails.
- `feature_flag_id` — the flag that controls this block.
- `enabled: false` — block is intentionally disabled; frontend should render the `fallback` without calling `resolve_endpoint`.

---

### `GET /home/blocks/{blockType}`

Resolves one dynamic block's data independently. Each block type has its own circuit breaker.

#### Allowed values for `{blockType}`

| Value | Description |
|---|---|
| `products_list` | Carousel of products (groupby / salesforce source) |
| `banner_products` | Banner with embedded product tiles |
| `container_greeting` | Personalised greeting for authenticated users |
| `container_guest` | Content shown only to unauthenticated users |
| `container_shortcuts` | Quick-action shortcuts bar |
| `recommendation_product_list` | ML-driven product recommendations |
| `products_cards` | Card grid of products |

Any other value returns `400 BAD_REQUEST` immediately — no downstream call is made.

#### Request

| Where | Name | Required | Description |
|---|---|---|---|
| Query | `locale` | no | Defaults to `es-mx` |
| Query | `channel` | no | Same values as `/home` |
| Header | `x-brand-id` | no | Defaults to `LP` |
| Header | `x-correlation-id` | no | Propagated through traces |

> Only `locale`, `brand`, and `channel` are forwarded to resolvers. Arbitrary headers (e.g. `Authorization`) are never passed downstream.

#### Response `200 OK`

While block resolvers are stubs, the response is:

```json
{
  "block_type": "products_list",
  "stub":       true,
  "message":    "resolver for \"products_list\" is not yet implemented",
  "params": {
    "brand":   "LP",
    "channel": "",
    "locale":  "es-mx"
  }
}
```

When real resolvers are wired, the response shape is defined by that resolver's outbound adapter.

---

### `GET /healthz`

Liveness probe. Returns `200` as long as the process is running.

```json
{"status": "ok"}
```

---

### `GET /readyz`

Readiness probe. Returns `200` when the service is ready to receive traffic (config is valid).

```json
{"status": "ready"}
```

---

## Error contract

All error responses share a consistent envelope. **Internal details are never exposed.**

```json
{
  "error_code": "SERVICE_UNAVAILABLE",
  "message":    "service not available at this moment",
  "retryable":  true
}
```

| `error_code` | HTTP | `retryable` | Meaning |
|---|---|---|---|
| `BAD_REQUEST` | 400 | false | Invalid query param or header value |
| `NOT_FOUND` | 404 | false | Content entry does not exist |
| `BLOCK_TEMPORARILY_DISABLED` | 423 | false | Feature flag is off — render `fallback` |
| `SERVICE_UNAVAILABLE` | 503 | true | Circuit breaker open — downstream is degraded |
| `TIMEOUT` | 504 | true | Downstream call exceeded timeout |
| `UNEXPECTED_ERROR` | 500 | false | Unclassified failure |
| `CONFIGURATION_ERROR` | 500 | false | Service misconfiguration (startup only) |

**Frontend guidance:**
- `retryable: true` → safe to retry with back-off.
- `retryable: false` → do not retry; render the block's `fallback` or hide the block.
- `BLOCK_TEMPORARILY_DISABLED` → intentional toggle, not an outage; do not alert.

---

## cURL examples

All examples assume the service is running locally on `http://localhost:8080`.

### Home layout — default locale and brand

```bash
curl -s http://localhost:8080/home \
  -H 'x-brand-id: LP' \
  -H 'x-correlation-id: local-test-001' \
  | jq
```

### Home layout — specific locale and channel

```bash
curl -s "http://localhost:8080/home?locale=es-mx&channel=pocket" \
  -H 'x-brand-id: LP' \
  | jq '.blocks[] | {kind, type, id}'
```

### Home layout — preview mode

```bash
curl -s "http://localhost:8080/home?locale=es-mx" \
  -H 'x-brand-id: LP-PREVIEW' \
  | jq
```

Or equivalently:

```bash
curl -s "http://localhost:8080/home?locale=es-mx" \
  -H 'x-brand-id: LP' \
  -H 'x-preview: true' \
  | jq
```

### Count static vs dynamic blocks

```bash
curl -s http://localhost:8080/home -H 'x-brand-id: LP' \
  | jq '[.blocks[] | .kind] | group_by(.) | map({(.[0]): length}) | add'
```

### Resolve a dynamic block

```bash
curl -s "http://localhost:8080/home/blocks/products_list?locale=es-mx" \
  -H 'x-brand-id: LP' \
  | jq
```

### Resolve all dynamic block types (loop)

```bash
for block in products_list banner_products container_greeting \
             container_guest container_shortcuts \
             recommendation_product_list products_cards; do
  echo "── $block ──"
  curl -s "http://localhost:8080/home/blocks/$block?locale=es-mx" \
    -H 'x-brand-id: LP' | jq .
done
```

### Trigger a 400 — unknown block type

```bash
curl -s http://localhost:8080/home/blocks/unknown_type \
  | jq
# → {"error_code":"BAD_REQUEST","message":"The request contains invalid parameters.","retryable":false}
```

### Trigger a 400 — invalid brand header

```bash
curl -s http://localhost:8080/home \
  -H 'x-brand-id: <script>alert(1)</script>' \
  | jq
# → {"error_code":"BAD_REQUEST","message":"The request contains invalid parameters.","retryable":false}
```

### Trigger a 400 — invalid channel

```bash
curl -s "http://localhost:8080/home?channel=desktop" \
  -H 'x-brand-id: LP' \
  | jq
# → {"error_code":"BAD_REQUEST","message":"The request contains invalid parameters.","retryable":false}
```

### Liveness and readiness probes

```bash
curl -s http://localhost:8080/healthz | jq
curl -s http://localhost:8080/readyz  | jq
```

### Inspect response headers

```bash
curl -si http://localhost:8080/home -H 'x-brand-id: LP' | head -20
# Look for:
#   X-Content-Type-Options: nosniff
#   X-Frame-Options: DENY
#   X-Request-Id: <uuid>
```

---

## Running tests

```bash
# All tests
go test ./...

# With verbose output
go test -v ./...

# Force re-run (bypass cache)
go test -count=1 ./...

# Specific package
go test ./internal/adapters/inbound/http/...
go test ./pkg/breaker/...
go test ./internal/domain/home/...

# Race detector
go test -race ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### What is tested

| Package | Tests |
|---|---|
| `pkg/breaker` | Trips at 5 % failure ratio; no trips below min-requests; no retries; `IsOpen` detection |
| `internal/domain/home` | `IsDynamic` for all block types; `IsAllowedResolveType` allowlist and injection-string rejection |
| `internal/application/home` | `classify` — static vs dynamic; resolve-endpoint construction; disabled block; source-of-data signals; `handle=="client-side"` |
| `internal/adapters/outbound/contentservice` | Key-wrapper unwrap; uid inference; `container_grid` flattening; ordering; invalid item skipping; 404→`NOT_FOUND`; breaker open→`SERVICE_UNAVAILABLE`; invalid locale→`BAD_REQUEST`; SSRF (host always from config); context cancellation |
| `internal/adapters/inbound/http` | Happy path layout; invalid channel/brand→400; error code propagation; internal detail never leaked; security headers; unknown block type→400; path traversal; disabled block→423; service unavailable→503 |

---

## Project structure

```
ms_home_liverpool/
├── cmd/
│   └── home/
│       └── main.go                   # Entrypoint — delegates to bootstrap
├── configs/
│   └── .env.example                  # All env vars documented with defaults
├── deployments/
│   ├── Dockerfile                    # Multi-stage, distroless, non-root
│   └── service.yaml                  # Cloud Run service descriptor
├── docs/
│   ├── architecture.md
│   ├── decisions.md                  # Architecture Decision Records
│   ├── integrations.md               # content-service contract + env mapping
│   ├── error-handling.md             # AppError model, logging discipline
│   ├── security.md                   # Threat model + controls
│   ├── deployment.md                 # Cloud Run deployment guide
│   └── changelog.md
├── internal/
│   ├── domain/home/                  # Entities, ports, error model (zero infra deps)
│   │   ├── home.go                   # Layout, Block, StaticBlock, DynamicBlock
│   │   ├── block_type.go             # BlockType constants + IsDynamic + IsAllowedResolveType
│   │   ├── errors.go                 # AppError + named constructors
│   │   └── ports.go                  # HomeUseCase, ContentPort, BlockResolverPort interfaces
│   ├── application/
│   │   ├── home/
│   │   │   ├── service.go            # HomeService: layout composition
│   │   │   └── classify.go           # Static vs dynamic classification
│   │   └── blocks/
│   │       └── resolve.go            # BlockResolver registry + StubResolver
│   ├── adapters/
│   │   ├── inbound/http/             # package handler (avoids shadowing net/http)
│   │   │   ├── router.go             # Chi router + middleware wiring
│   │   │   ├── home_handler.go       # GET /home
│   │   │   ├── block_handler.go      # GET /home/blocks/{blockType}
│   │   │   ├── middleware.go         # request-id, correlation-id, access log
│   │   │   ├── response.go           # Domain → JSON mapping + writeJSON
│   │   │   ├── errors.go             # AppError → HTTP status/body
│   │   │   └── health.go             # /healthz, /readyz
│   │   └── outbound/contentservice/
│   │       ├── client.go             # HTTP client + breaker + SSRF-safe URL
│   │       ├── mapper.go             # content-service payload → []RawBlock
│   │       └── normalize.go          # Flatten container_grid / unwrap key-wrappers
│   ├── config/
│   │   └── config.go                 # Env-driven config + validation
│   └── bootstrap/
│       └── app.go                    # DI wiring, server, OTel, graceful shutdown
└── pkg/
    ├── breaker/
    │   └── breaker.go                # Generic circuit breaker (no retries, 5 % trip)
    ├── httpx/
    │   ├── client.go                 # Keep-alive http.Client factory
    │   ├── context.go                # RequestID / CorrelationID context helpers
    │   ├── mask.go                   # MaskSensitiveHeaders (auth/cookies/tokens)
    │   └── curl.go                   # BuildCurlCommand for DEBUG logging
    ├── logger/
    │   └── logger.go                 # slog JSON logger with LevelVar
    └── observability/
        └── otel.go                   # OTel tracer init (noop when endpoint empty)
```

---

## Architecture overview

Dependencies point **inward**. The domain layer imports nothing from infrastructure.

```
┌─────────────────────────────────────────────────────────────┐
│  Inbound adapters        Application         Domain          │
│  ┌──────────────────┐   ┌────────────────┐  ┌────────────┐  │
│  │  GET /home       │──▶│  HomeService   │─▶│  Layout    │  │
│  │  GET /home/blocks│──▶│  BlockRegistry │  │  Block     │  │
│  │  /healthz /readyz│   └────────────────┘  │  AppError  │  │
│  └──────────────────┘          │             │  Ports     │  │
│                                ▼             └────────────┘  │
│  Outbound adapters                                           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │  contentservice.Client  ──[breaker]──▶ content-service  │ │
│  │  StubResolver (×7)      ──[breaker]──▶ (TBD per block)  │ │
│  └─────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

**Block classification** (`internal/application/home/classify.go`):

A block is **dynamic** when any of these is true:
- Its `_content_type_uid` is one of the 7 dynamic types.
- Its `source_of_data` is `groupby`, `salesforce`, `recently_viewed`, `jewel`, or `lob`.
- Its `handle` is `"client-side"`.

Everything else is **static**.

---

## Adding a new block resolver

1. **Register the block type** in `internal/domain/home/block_type.go`:
   - Add a constant to the `BlockType` constants block.
   - Add it to `dynamicBlockTypes` and `allowedBlockTypes`.

2. **Implement the resolver** — create `internal/adapters/outbound/<name>/resolver.go` and implement `blocks.Resolver`:
   ```go
   type MyResolver struct { /* breaker-wrapped http.Client */ }

   func (r *MyResolver) Resolve(ctx context.Context, params map[string]string) (map[string]any, *domain.AppError) {
       // call downstream, wrap every HTTP call with pkg/breaker
       // log errors exactly once here; return *domain.AppError
   }
   ```

3. **Wire it** in `internal/bootstrap/app.go`:
   ```go
   blockRegistry.Register(domain.BlockTypeMyNew, &myresolver.MyResolver{ /* deps */ })
   ```

4. **Write tests** — `httptest` for the outbound call, handler test for the new block type path.

That's it. The HTTP handler, routing, and error mapping are already wired — you only need to provide the resolver.

---

## Circuit breaker behaviour

Every outbound call goes through `pkg/breaker.Breaker[T]` (backed by `sony/gobreaker/v2`).

| State | Behaviour |
|---|---|
| **Closed** (healthy) | Requests pass through normally |
| **Open** (tripped) | Requests fail immediately with `SERVICE_UNAVAILABLE` — no downstream call made |
| **Half-open** (probing) | One probe request allowed; success → Closed; failure → Open again |

**Trip condition:** `failures / requests >= BREAKER_FAILURE_RATIO` when `requests >= BREAKER_MIN_REQUESTS`.

**No retries.** A failed call is counted once and fails fast. Retrying under load would amplify pressure on a degrading downstream.

**One breaker per dependency.** The content-service breaker is independent of each block resolver's breaker. A tripped block resolver never affects the layout endpoint.

---

## Logging

Structured JSON via `log/slog`. Every log line includes:

```json
{
  "time":           "2026-06-20T10:00:00Z",
  "level":          "INFO",
  "msg":            "layout composed",
  "service":        "ms-home-liverpool",
  "request_id":     "e3b0c442-…",
  "correlation_id": "frontend-trace-abc",
  "total_blocks":   12,
  "static_blocks":  8,
  "dynamic_blocks": 4
}
```

**Log-once rule:** each failure is logged **exactly once** by the layer with the most context. Adapters log outbound failures; handlers do not re-log them.

**What is never logged:** `Authorization`, cookies, tokens, session IDs, or any PII.
Debug cURL commands are emitted at `DEBUG` level with all sensitive headers masked:
```
curl -X GET 'http://content-service:8090/content/page/es-mx' \
  -H 'x-brand-id: LP' \
  -H 'Authorization: ***MASKED***'
```

---

## OpenTelemetry

When `OTEL_EXPORTER_OTLP_ENDPOINT` is set, the service exports traces over OTLP HTTP.

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318 \
OTEL_SAMPLE_RATIO=1.0 \
go run ./cmd/home
```

If the variable is empty, tracing is a **no-op** — no overhead, no errors.

Each request creates a root span named after the service. The content-service call and each block resolve call create child spans. Breaker state transitions are recorded as span events.

---

## Docker build

```bash
# Build
docker build -f deployments/Dockerfile -t ms-home-liverpool:local .

# Run
docker run --rm -p 8080:8080 \
  -e CONTENT_SERVICE_URL=http://host.docker.internal:8090 \
  -e LOG_LEVEL=DEBUG \
  ms-home-liverpool:local

# Verify
curl -s http://localhost:8080/healthz | jq
```

The final image is **distroless/static-debian12:nonroot** (~5 MB). The binary runs as uid 65532 with no shell, no package manager, and no writable filesystem.

---

## Deployment (Cloud Run)

See [docs/deployment.md](docs/deployment.md) for the full guide. Short version:

```bash
IMAGE=REGION-docker.pkg.dev/PROJECT_ID/REPO/ms-home-liverpool

# Build and push
docker build -f deployments/Dockerfile -t $IMAGE:$TAG .
docker push $IMAGE:$TAG

# Deploy
gcloud run services replace deployments/service.yaml \
  --region REGION \
  --project PROJECT_ID
```

`CONTENT_SERVICE_URL` is injected from **Secret Manager** in `deployments/service.yaml` — it is never hardcoded or committed.

---

## Contributing

1. Branch from `develop`.
2. Run `go test -race ./...` and `go vet ./...` before opening a PR.
3. Run `govulncheck ./...` if you add or upgrade a dependency.
4. Each new block resolver must include its own unit tests and a handler-level integration test for the new `{blockType}` path.
5. Run `/security-review` on any PR that touches `adapters/`, `pkg/`, or `config/`.
