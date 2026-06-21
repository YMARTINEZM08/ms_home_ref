# Architecture — ms_home_liverpool

## Overview

`ms_home_liverpool` is a Go microservice that exposes the Home page layout for Liverpool's digital channels. It replaces the Home logic from the `digital_bff` Node/NestJS monorepo and follows the **Hexagonal Architecture (Ports & Adapters)** pattern.

## Hexagonal layers

```
cmd/home/main.go          ← thin entrypoint; delegates to bootstrap
internal/
  domain/home/            ← entities, ports, error model (zero infra deps)
  application/home/       ← HomeService: layout composition + classification
  application/blocks/     ← BlockResolver registry; per-block use cases
  adapters/
    inbound/http/         ← Chi router, HTTP handlers, request parsing
    outbound/
      contentservice/     ← content-service proxy adapter (breaker-wrapped)
  config/                 ← env-driven config
  bootstrap/              ← DI wiring, server, OTel init, graceful shutdown
pkg/
  breaker/                ← generic circuit breaker wrapper (no retries)
  httpx/                  ← HTTP client, header masking, debug cURL
  logger/                 ← slog JSON structured logger
  observability/          ← OTel tracer init
```

**Dependency direction:** adapters → application → domain. Domain never imports infrastructure.

## API contract

### `GET /home`

Returns the ordered page layout. Static blocks carry their content inline; dynamic blocks carry only a placeholder so the frontend can resolve them independently.

```json
{
  "blocks": [
    { "kind": "static",  "id": "...", "type": "banner", "content": { ... } },
    { "kind": "dynamic", "id": "...", "type": "products_list",
      "resolve_endpoint": "/home/blocks/products_list",
      "fallback": "...", "feature_flag_id": "...", "enabled": true }
  ]
}
```

### `GET /home/blocks/{blockType}`

Resolves one dynamic block independently. Each block type has its own resolver and circuit breaker, so a single downstream failure only affects that block.

Accepted block types (allowlist): `products_list`, `banner_products`, `container_greeting`, `guest_container`, `shortcuts`, `recommendations`, `product_cards`.

### `GET /healthz` / `GET /readyz`

Liveness and readiness probes (200 JSON).

## Block classification

A block is classified as **dynamic** if any of the following is true:
- Its `_content_type_uid` is one of the seven dynamic types.
- Its `source_of_data` is `groupby`, `salesforce`, `recently_viewed`, `jewel`, or `lob`.
- Its `handle` is `"client-side"`.

All other blocks are **static**.

## Content normalization

The content-service returns nested structures that `normalize.go` flattens before mapping:
- **Key-wrapper:** `{ "banner": { ... } }` → unwrapped; `_content_type_uid` inferred from the key.
- **`container_grid`:** flattened to its `grid_items` children in order.
- **`tabs_container`:** kept as a single block (the tabs are rendered client-side).
