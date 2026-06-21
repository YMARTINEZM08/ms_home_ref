# Architecture — ms_home_liverpool

## Table of contents

1. [Overview](#overview)
2. [Hexagonal Architecture](#hexagonal-architecture)
3. [Package dependency graph](#package-dependency-graph)
4. [Startup and DI wiring](#startup-and-di-wiring)
5. [Use case: GET /home — layout composition](#use-case-get-home--layout-composition)
6. [Use case: GET /home/blocks/{blockType} — block resolution](#use-case-get-homeblocksblocktype--block-resolution)
7. [Block classification flow](#block-classification-flow)
8. [Content normalization flow](#content-normalization-flow)
9. [Circuit breaker state machine](#circuit-breaker-state-machine)
10. [Request middleware chain](#request-middleware-chain)
11. [Error propagation model](#error-propagation-model)
12. [Graceful shutdown sequence](#graceful-shutdown-sequence)
13. [Block type reference](#block-type-reference)

---

## Overview

`ms_home_liverpool` is a Go microservice that exposes the **Home page layout** for Liverpool's digital channels. It replaces the Home logic from the `digital_bff` Node/NestJS monorepo and runs on **Google Cloud Run**.

**Core responsibilities:**
- Fetch the ordered block list from the internal **content-service proxy**.
- **Classify** each block as static (inline content) or dynamic (placeholder + resolve endpoint).
- Return the layout to the frontend via `GET /home`.
- Expose **independent resolve endpoints** (`GET /home/blocks/{blockType}`) for each dynamic block type, each with its own circuit breaker.

**What it explicitly does NOT do:**
- Call Contentstack directly (all CMS access goes through the content-service proxy).
- Perform personalization or recommendation logic (this stays in the downstream resolvers).
- Re-order or filter blocks (ordering from the content-service is preserved verbatim).

---

## Hexagonal Architecture

The service follows **Ports & Adapters (Hexagonal Architecture)**. Dependencies point inward — the domain never imports infrastructure.

```
┌────────────────────────────────────────────────────────────────────────────┐
│                           ms_home_liverpool                                 │
│                                                                             │
│  ┌──────────────┐      ┌──────────────────────────┐      ┌───────────────┐ │
│  │   INBOUND    │      │       APPLICATION         │      │   OUTBOUND    │ │
│  │   ADAPTERS   │      │                           │      │   ADAPTERS    │ │
│  │              │      │  ┌────────────────────┐   │      │               │ │
│  │  GET /home   │─────▶│  │   HomeService      │   │      │  content-     │ │
│  │              │      │  │  (classify blocks) │   │─────▶│  service      │ │
│  │  GET /home/  │      │  └────────────────────┘   │      │  proxy        │ │
│  │  blocks/     │      │                           │      │  [breaker]    │ │
│  │  {blockType} │─────▶│  ┌────────────────────┐   │      │               │ │
│  │              │      │  │  BlockRegistry     │   │      │  StubResolver │ │
│  │  GET /healthz│      │  │  (dispatch by type)│   │─────▶│  ×7 (TBD)    │ │
│  │  GET /readyz │      │  └────────────────────┘   │      │  [breaker]    │ │
│  └──────────────┘      │                           │      └───────────────┘ │
│                        │  ┌────────────────────────────────────────────┐    │
│                        │  │                 DOMAIN                      │    │
│                        │  │  Layout · Block · AppError                  │    │
│                        │  │  HomeUseCase (port) · ContentPort (port)    │    │
│                        │  └────────────────────────────────────────────┘    │
│                        └──────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────┘
```

**Dependency direction rule:** `adapters → application → domain`. No arrow ever points outward from the domain.

---

## Package dependency graph

```
cmd/home/main.go
    └── internal/bootstrap/app.go          ← composition root (only place that knows everything)
            ├── internal/config/config.go
            ├── pkg/logger/logger.go
            ├── pkg/observability/otel.go
            ├── pkg/httpx/client.go
            │
            ├── internal/adapters/outbound/contentservice/
            │       ├── client.go          ← implements domain.ContentPort
            │       ├── mapper.go
            │       ├── normalize.go
            │       └── pkg/breaker/       ← one Breaker[[]byte] per client
            │
            ├── internal/application/home/
            │       ├── service.go         ← implements domain.HomeUseCase
            │       └── classify.go
            │
            ├── internal/application/blocks/
            │       └── resolve.go         ← Registry + StubResolver
            │
            └── internal/adapters/inbound/http/   (package handler)
                    ├── router.go
                    ├── home_handler.go
                    ├── block_handler.go
                    ├── middleware.go
                    ├── response.go
                    ├── errors.go
                    └── health.go

internal/domain/home/          ← imported by everyone, imports nobody
    ├── home.go                (Layout, Block, StaticBlock, DynamicBlock, HomeRequest)
    ├── block_type.go          (BlockType constants, IsDynamic, IsAllowedResolveType)
    ├── errors.go              (AppError + all named constructors)
    └── ports.go               (HomeUseCase, ContentPort, BlockResolverPort, RawBlock)

pkg/                           ← shared utilities; no internal/ imports
    ├── breaker/breaker.go     (Breaker[T], Settings, IsOpen)
    ├── httpx/
    │   ├── client.go          (keep-alive http.Client factory)
    │   ├── context.go         (RequestID / CorrelationID typed context keys)
    │   ├── mask.go            (MaskSensitiveHeaders)
    │   └── curl.go            (BuildCurlCommand)
    ├── logger/logger.go       (slog JSON + LevelVar)
    └── observability/otel.go  (OTel OTLP HTTP init)
```

---

## Startup and DI wiring

`bootstrap.Run()` is the **composition root** — the only place in the codebase that instantiates and wires all dependencies together.

```
main()
  │
  └── bootstrap.Run()
        │
        ├── 1. config.Load()
        │       reads all env vars, validates required fields
        │       exits on missing CONTENT_SERVICE_URL or invalid values
        │
        ├── 2. logger.New(cfg.LogLevel)
        │       JSON slog with runtime-configurable LevelVar
        │
        ├── 3. observability.Init(...)
        │       OTel OTLP HTTP exporter
        │       noop if OTEL_EXPORTER_OTLP_ENDPOINT is empty
        │
        ├── 4. httpx.NewClient(timeout)
        │       keep-alive *http.Client shared by all outbound calls
        │
        ├── 5. contentservice.NewClient(cfg, httpClient, log)
        │       wraps with breaker.New[[]byte]("content-service", cfg.BreakerSettings)
        │       implements domain.ContentPort
        │
        ├── 6. apphome.NewService(csClient, log)
        │       implements domain.HomeUseCase
        │
        ├── 7. blocks.NewRegistry(log)
        │       + Register StubResolver for each of the 7 dynamic block types
        │       implements blocks.ResolveUseCase
        │
        ├── 8. inbound.NewRouter(homeService, blockRegistry, log, serviceName)
        │       Chi router + middleware chain + all routes
        │
        ├── 9. http.Server{ReadTimeout:10s, WriteTimeout:30s, IdleTimeout:120s}
        │
        └── 10. SIGTERM/SIGINT → graceful shutdown (30s drain) → OTel flush
```

---

## Use case: GET /home — layout composition

Full request lifecycle from the frontend to the JSON response.

```
Frontend / Client
      │
      │  GET /home?locale=es-mx&channel=pocket
      │  x-brand-id: LP
      │  x-correlation-id: abc-123
      │
      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        Middleware chain (Chi)                            │
│                                                                          │
│  requestIDMiddleware                                                     │
│    ├── read x-request-id header (or generate UUID)                       │
│    └── store in ctx + echo in response header                            │
│                                                                          │
│  correlationIDMiddleware                                                  │
│    ├── read x-correlation-id, validate ^[a-zA-Z0-9\-_]{1,64}$           │
│    │   (log injection prevention)                                         │
│    └── store in ctx + echo in response header                            │
│                                                                          │
│  otelhttp.NewMiddleware(serviceName)                                     │
│    └── start root OTel span for this request                             │
│                                                                          │
│  middleware.Recoverer                                                     │
│    └── catch any handler panic → 500 (never crashes the process)         │
│                                                                          │
│  accessLogMiddleware                                                      │
│    └── measure latency, log method/path/status/latency_ms on exit        │
└─────────────────────────────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         homeHandler.ServeHTTP                            │
│                                                                          │
│  parseRequest(r)                                                         │
│    ├── locale  → lowercase, default "es-mx"                              │
│    ├── brand   → uppercase, strip -PREVIEW, validate ^[A-Z0-9]{1,20}$   │
│    ├── channel → allowlist: "" | pocket | kiosk | mpos                   │
│    └── preview → x-preview header or -PREVIEW suffix on brand           │
│                                                                          │
│    BAD INPUT → writeError(400 BAD_REQUEST) + WARN log  ─────────────▶  │
│    (no downstream call made)                             response        │
└─────────────────────────────────────────────────────────────────────────┘
      │  HomeRequest{Locale, Brand, Channel, Preview}
      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                        HomeService.GetLayout                             │
│                                                                          │
│  ContentPort.FetchLayout(ctx, req)  ──────────────────────────────────▶ │
│      │                                                                   │
│      │         ┌────────────────────────────────────────────────────┐   │
│      │         │           contentservice.Client.FetchLayout         │   │
│      │         │                                                      │   │
│      │         │  1. validate locale ^[a-z]{2}-[a-z]{2}$            │   │
│      │         │  2. buildURL (host from config — SSRF prevention)   │   │
│      │         │     GET {CONTENT_SERVICE_URL}/content/page/es-mx   │   │
│      │         │  3. buildHeaders: only x-brand-id + Accept         │   │
│      │         │  4. emit cURL at DEBUG (headers masked)            │   │
│      │         │  5. breaker.Execute(do)                            │   │
│      │         │       do(): http.Client.Do → LimitReader(4MB)      │   │
│      │         │       non-2xx → httpStatusError                     │   │
│      │         │  6. mapError if fail:                               │   │
│      │         │       breaker open  → SERVICE_UNAVAILABLE (log 1×) │   │
│      │         │       404          → NOT_FOUND          (log 1×)   │   │
│      │         │       5xx          → SERVICE_UNAVAILABLE (log 1×)  │   │
│      │         │       timeout      → TIMEOUT            (log 1×)   │   │
│      │         │       other        → UNEXPECTED_ERROR   (log 1×)   │   │
│      │         │  7. json.Unmarshal → contentServiceResponse        │   │
│      │         │  8. mapToRawBlocks → normalize → []RawBlock        │   │
│      │         └────────────────────────────────────────────────────┘   │
│      │                                                                   │
│  ◀── []RawBlock (or *AppError)                                           │
│                                                                          │
│  for each RawBlock:                                                      │
│    classify(raw)  ──────────────────────────────────────────────────▶   │
│        isDynamic?                                                        │
│          type in dynamicBlockTypes?  → DynamicBlock (placeholder)        │
│          source_of_data ∈ {groupby, salesforce, ...}? → DynamicBlock    │
│          handle == "client-side"?    → DynamicBlock                      │
│          else                        → StaticBlock (inline content)      │
│                                                                          │
│  log INFO: "home layout composed" {total, static, dynamic, locale, brand}│
└─────────────────────────────────────────────────────────────────────────┘
      │  *Layout{Blocks: []}
      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                   homeHandler → writeJSON(200, layoutResponse)           │
│                                                                          │
│  toLayoutResponse(layout)                                                │
│    static  block → {kind, id, type, content}                             │
│    dynamic block → {kind, id, type, resolve_endpoint,                    │
│                     fallback, feature_flag_id, enabled}                  │
│                                                                          │
│  response headers:                                                       │
│    Content-Type: application/json                                        │
│    X-Content-Type-Options: nosniff                                       │
│    X-Frame-Options: DENY                                                 │
│    X-Request-Id: <uuid>                                                  │
│    X-Correlation-Id: abc-123                                             │
└─────────────────────────────────────────────────────────────────────────┘
      │
      ▼
  200 OK  {"blocks": [...]}
```

---

## Use case: GET /home/blocks/{blockType} — block resolution

```
Frontend / Client
      │
      │  GET /home/blocks/products_list?locale=es-mx
      │  x-brand-id: LP
      │
      ▼
  [Middleware chain — same as above: requestID, correlationID, OTel, Recoverer, accessLog]
      │
      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                       blockHandler.ServeHTTP                             │
│                                                                          │
│  chi.URLParam(r, "blockType")  → "products_list"                        │
│                                                                          │
│  domain.IsAllowedResolveType("products_list")                           │
│    ├── checks against fixed allowedBlockTypes map (7 entries)            │
│    ├── UNKNOWN → writeError(400 BAD_REQUEST) + WARN log  ─────────────▶ │
│    │   (no downstream call — security boundary)           response       │
│    └── KNOWN  → BlockType("products_list")                              │
│                                                                          │
│  extractParams(r)                                                        │
│    ├── locale:  from query (default "es-mx")                             │
│    ├── brand:   from x-brand-id header (default "LP", strip -PREVIEW)   │
│    └── channel: from query                                               │
│    (arbitrary headers are NEVER forwarded)                               │
└─────────────────────────────────────────────────────────────────────────┘
      │  blockType, params{locale, brand, channel}
      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    blocks.Registry.ResolveBlock                          │
│                                                                          │
│  lookup resolver by blockType                                            │
│    NOT FOUND → ErrNotFound (should not happen: allowlist guards this)   │
│                                                                          │
│  resolver.Resolve(ctx, params)                                           │
│    (currently StubResolver — returns stub payload)                       │
│    (future: real outbound adapter, breaker-wrapped)                      │
│                                                                          │
│  log INFO: "block resolved" {block_type, locale, brand}                 │
└─────────────────────────────────────────────────────────────────────────┘
      │  map[string]any
      ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              blockHandler → writeJSON(200, result)                       │
│  response headers: Content-Type, X-Content-Type-Options, X-Frame-Options│
└─────────────────────────────────────────────────────────────────────────┘
      │
      ▼
  200 OK  {"block_type": "products_list", "stub": true, ...}
```

---

## Block classification flow

`internal/application/home/classify.go` — pure in-memory, never errors, never panics.

```
RawBlock{Type, SourceOfData, Handle, Fields, Enabled, FeatureFlagID}
      │
      ▼
  isDynamic(raw)?
      │
      ├── domain.IsDynamic(raw.Type)?
      │     checks dynamicBlockTypes map:
      │       products_list           → YES
      │       banner_products         → YES
      │       container_greeting      → YES
      │       container_guest         → YES
      │       container_shortcuts     → YES
      │       recommendation_product_list → YES
      │       products_cards          → YES
      │       banner, carousel, hero_banner,
      │       promo_bar, static_content,
      │       form, comparepage,
      │       search_banners, countdown → NO
      │
      ├── raw.SourceOfData ∈ {groupby, salesforce,
      │                        recently_viewed, jewel, lob}?  → YES
      │
      └── raw.Handle == "client-side"?  → YES
      │
      ▼
  ┌─────────────────────────────────┬──────────────────────────────────────┐
  │          STATIC BLOCK            │           DYNAMIC BLOCK              │
  │                                  │                                      │
  │  Block{                          │  Block{                              │
  │    Kind: "static",               │    Kind: "dynamic",                  │
  │    Static: &StaticBlock{         │    Dynamic: &DynamicBlock{           │
  │      ID:      raw.ID,            │      ID:              raw.ID,        │
  │      Type:    raw.Type,          │      Type:            raw.Type,      │
  │      Content: raw.Fields,        │      ResolveEndpoint: "/home/blocks/ │
  │    },                            │                        {raw.Type}",  │
  │  }                               │      Fallback:    from raw.Fields,   │
  │                                  │      FeatureFlagID: raw.FeatureFlagID│
  │  → inline content in /home       │      Enabled:     raw.Enabled,       │
  │  → safe to cache                 │    },                                │
  │                                  │  }                                   │
  │                                  │  → placeholder in /home              │
  │                                  │  → frontend calls resolve_endpoint   │
  └─────────────────────────────────┴──────────────────────────────────────┘
```

---

## Content normalization flow

`internal/adapters/outbound/contentservice/normalize.go`

The content-service returns blocks in one of three shapes that must be flattened before classification.

```
Content-service response payload
      │
      │  layout: [ item, item, item, ... ]
      │
      ▼
normalize(items []any)
      │
      ├── item is not map[string]any?  → skip (invalid, log nothing)
      │
      └── unwrap(m)
            │
            ├── Has "_content_type_uid" at top level?
            │     → already normalized → handleContainer(m)
            │
            └── Wrapper shape: { "banner": { ...fields } }
                  → extract inner map
                  → set inner["_content_type_uid"] = "banner" if absent
                  → handleContainer(inner)

handleContainer(block)
      │
      ├── _content_type_uid == "container_grid"
      │     → flattenGrid(block)
      │           grid_items: [ item, item ]
      │           → normalize(grid_items)   ← recursive
      │           → produces N blocks from one container
      │
      ├── _content_type_uid == "tabs_container"
      │     → flattenTabs(block)
      │           normalize each tab's content[] in place
      │           → keep tabs_container as ONE block (rendered client-side)
      │
      └── anything else
            → [ block ]   ← single-element slice, pass through


Example — container_grid with 2 children:

  Input:
  {
    "_content_type_uid": "container_grid",
    "grid_items": [
      { "banner":        { "uid": "b1", "_content_type_uid": "banner" } },
      { "products_list": { "uid": "p1", "_content_type_uid": "products_list" } }
    ]
  }

  Output (2 flat blocks):
  [
    { "uid": "b1", "_content_type_uid": "banner" },
    { "uid": "p1", "_content_type_uid": "products_list" }
  ]
```

---

## Circuit breaker state machine

`pkg/breaker/breaker.go` wraps `sony/gobreaker/v2`. One `Breaker[T]` instance per outbound dependency.

```
                      ┌─────────────────────────────────────┐
                      │                                       │
          requests pass through                               │
                      │                                       │
          ┌───────────▼──────────┐                           │
          │       CLOSED          │                           │
          │   (normal operation)  │                           │
          └───────────┬──────────┘                           │
                      │                                       │
          failures/requests >= BREAKER_FAILURE_RATIO         │
          AND requests >= BREAKER_MIN_REQUESTS               │
                      │                                       │
                      ▼                                       │
          ┌───────────────────────┐                          │
          │        OPEN           │                          │
          │  (fast-fail; no call) │                          │
          │                       │                          │
          │  Execute() returns:   │                          │
          │  ErrOpenState         │                          │
          │  → SERVICE_UNAVAILABLE│                          │
          └───────────┬──────────┘                          │
                      │                                      │
          after BREAKER_OPEN_TIMEOUT_S seconds              │
                      │                                      │
                      ▼                                      │
          ┌───────────────────────┐                         │
          │      HALF-OPEN        │                         │
          │   (one probe allowed) │                         │
          └───────┬───────┬───────┘                         │
                  │       │                                  │
          probe   │       │  probe                          │
          succeeds│       │  fails                          │
                  │       │                                  │
                  ▼       └──────────────────────────────▶  │
              CLOSED                                     OPEN│
                                                            │
                                                            └─────────────────┘

Configuration (env vars):
  BREAKER_FAILURE_RATIO  = 0.05   # trip at 5% failure rate
  BREAKER_MIN_REQUESTS   = 20     # minimum window before ratio is evaluated
  BREAKER_OPEN_TIMEOUT_S = 30     # seconds open before probing

One breaker per dependency:
  content-service breaker ──▶ breaker.New[[]byte]("content-service", settings)
  products_list breaker   ──▶ breaker.New[...]("products_list", settings)    [future]
  ... each block type independently

A tripped content-service breaker does NOT affect block resolve endpoints.
A tripped block resolver breaker does NOT affect /home layout.
```

---

## Request middleware chain

Applied to every request in declaration order. Evaluated as a stack (outermost wraps innermost).

```
Inbound HTTP request
      │
      ▼
┌─────────────────────────────────────────────────────────────┐
│  1. requestIDMiddleware                                      │
│     Read x-request-id header (or UUID v4)                   │
│     → store in ctx as CtxKeyRequestID                       │
│     → set X-Request-Id response header                      │
├─────────────────────────────────────────────────────────────┤
│  2. correlationIDMiddleware                                   │
│     Read x-correlation-id header                            │
│     validate: ^[a-zA-Z0-9\-_]{1,64}$  (log injection guard) │
│     fallback to chi request ID or new UUID                  │
│     → store in ctx as CtxKeyCorrelationID                   │
│     → set X-Correlation-Id response header                  │
├─────────────────────────────────────────────────────────────┤
│  3. otelhttp.NewMiddleware(serviceName)                      │
│     Start root OTel span                                    │
│     Propagate W3C traceparent if present                    │
│     Record span: method, path, status, latency              │
├─────────────────────────────────────────────────────────────┤
│  4. middleware.Recoverer  (chi)                              │
│     Catch any handler panic                                 │
│     → log stack trace + return 500                          │
│     → process never crashes                                 │
├─────────────────────────────────────────────────────────────┤
│  5. accessLogMiddleware                                      │
│     Skip /healthz and /readyz (high-frequency, no value)    │
│     Wrap ResponseWriter to capture status code              │
│     After handler returns:                                  │
│     → log INFO: method, path, status, latency_ms,          │
│                  request_id, correlation_id                  │
└─────────────────────────────────────────────────────────────┘
      │
      ▼
  Route handler (homeHandler / blockHandler / healthzHandler)
```

---

## Error propagation model

The **log-once rule**: each failure is logged exactly once by the layer with the most context. No layer re-logs an error it received from a layer below.

```
contentservice.Client.mapError()         ← logs once (has url, latency, dep)
      │
      │ returns *AppError
      ▼
HomeService.GetLayout()                  ← does NOT log (returns error as-is)
      │
      │ returns *AppError
      ▼
homeHandler.ServeHTTP()                  ← does NOT log (writes response only)
      │
      │ calls writeError(w, appErr)
      ▼
┌─────────────────────────────────────────┐
│  errorResponse (never leaks internals)  │
│  {                                       │
│    "error_code": "SERVICE_UNAVAILABLE", │
│    "message":    "service not available │
│                   at this moment",      │
│    "retryable":  true                   │
│  }                                       │
│                                          │
│  AppError.Detail  → NOT serialized      │
│  AppError.Cause   → NOT serialized      │
│  AppError.Category → NOT serialized     │
└─────────────────────────────────────────┘

AppError fields and their audience:
┌─────────────┬──────────────────────────────────┬───────────┐
│ Field        │ Audience                          │ In resp?  │
├─────────────┼──────────────────────────────────┼───────────┤
│ Code         │ Frontend (machine-readable)        │ YES       │
│ Message      │ Frontend (human-readable, safe)    │ YES       │
│ Retryable    │ Frontend (retry hint)              │ YES       │
│ Status       │ HTTP layer only                    │ as status │
│ Category     │ Operators / logs only              │ NO        │
│ Detail       │ Developers / logs only             │ NO        │
│ Cause        │ Internal / logs only               │ NO        │
└─────────────┴──────────────────────────────────┴───────────┘

Logging levels by error type:
  Bad input (parseRequest fails)  → WARN  (caller mistake, not a service fault)
  Unknown blockType               → WARN  (security rejection at boundary)
  Content not found (404)         → WARN  (expected operational state)
  Circuit breaker open            → ERROR (dependency degraded)
  Downstream 5xx                  → ERROR
  Timeout                         → ERROR
  Unexpected error                → ERROR
  Access log (every request)      → INFO
  Layout composed successfully     → INFO
  Block resolved successfully      → INFO
  OTel init failed (non-fatal)    → WARN
```

---

## Graceful shutdown sequence

Cloud Run sends `SIGTERM` before terminating an instance. The service drains in-flight requests before exiting.

```
SIGTERM or SIGINT received
      │
      ▼
signal.Notify channel fires
      │
      ▼
log INFO: "shutdown signal received — draining connections"
      │
      ▼
context.WithTimeout(30 * time.Second)
      │
      ├── srv.Shutdown(shutdownCtx)
      │     ├── stops accepting new connections
      │     ├── waits for in-flight handlers to complete
      │     └── returns when all done or timeout exceeded
      │
      └── shutdownOTEL(shutdownCtx)
            └── flushes pending OTel spans to the collector
      │
      ▼
log INFO: "server stopped"
      │
      ▼
process exits 0
```

---

## Block type reference

### Static block types (resolved inline in `/home`)

| Constant | `_content_type_uid` | Description |
|---|---|---|
| `BlockTypeBanner` | `banner` | Hero/promotional banner |
| `BlockTypeCarousel` | `carousel` | Image or content carousel |
| `BlockTypeHeroBanner` | `hero_banner` | Full-width hero image |
| `BlockTypePromoBar` | `promo_bar` | Promotion announcement bar |
| `BlockTypeStaticContent` | `static_content` | Rich text or HTML block |
| `BlockTypeForm` | `form` | Embedded form |
| `BlockTypeComparePage` | `comparepage` | Product comparison widget |
| `BlockTypeSearchBanners` | `search_banners` | Banners tied to search results |
| `BlockTypeCountdown` | `countdown` | Sale countdown timer |

### Dynamic block types (placeholders in `/home`, resolved via `GET /home/blocks/{type}`)

| Constant | `_content_type_uid` / resolve path | Description |
|---|---|---|
| `BlockTypeProductsList` | `products_list` | Product carousel (groupby/salesforce) |
| `BlockTypeBannerProducts` | `banner_products` | Banner with embedded product tiles |
| `BlockTypeGreeting` | `container_greeting` | Personalised greeting (authenticated) |
| `BlockTypeGuestContainer` | `container_guest` | Content for unauthenticated users |
| `BlockTypeShortcuts` | `container_shortcuts` | Quick-action shortcuts bar |
| `BlockTypeRecommendations` | `recommendation_product_list` | ML recommendations |
| `BlockTypeProductCards` | `products_cards` | Product card grid |

### Dynamic classification signals (any one is sufficient)

| Signal | Source | Values that trigger dynamic |
|---|---|---|
| Block type | `_content_type_uid` | Any of the 7 dynamic types above |
| Data source | `source_of_data` field | `groupby`, `salesforce`, `recently_viewed`, `jewel`, `lob` |
| Render handle | `handle` field | `"client-side"` |
