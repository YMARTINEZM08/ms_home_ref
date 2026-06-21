# Integrations

## content-service proxy

The only outbound dependency for `/home` layout resolution.

| Attribute | Value |
|---|---|
| Env var | `CONTENT_SERVICE_URL` (required) |
| Auth header | `x-brand-id: {brand}` (or `{brand}-PREVIEW` when `x-preview: true`) |
| Timeout | `CONTENT_SERVICE_TIMEOUT_MS` (default: 8000 ms) |
| Circuit breaker | `BREAKER_FAILURE_RATIO`, `BREAKER_MIN_REQUESTS`, `BREAKER_OPEN_TIMEOUT_S` |

### Legacy env var mapping

| Legacy (`digital_bff`) | New (`ms_home_liverpool`) |
|---|---|
| `SHARED_CONTENT_URL` | `CONTENT_SERVICE_URL` |
| `SHARED_CONTENT_TIMEOUT` | `CONTENT_SERVICE_TIMEOUT_MS` |
| `DEFAULT_BRAND` | `DEFAULT_BRAND` (same) |

### Request shape

```
GET {CONTENT_SERVICE_URL}/content/page/{locale}[?channel={channel}]
Headers:
  x-brand-id: LP            # or LP-PREVIEW
  Accept: application/json
```

The service forwards **only** `x-brand-id` and `Accept` — no arbitrary client headers are passed through.

### Response shape (normalized)

The content-service returns a nested entry. `normalize.go` unwraps key-wrappers and flattens `container_grid` before mapping to `[]RawBlock`.

## OpenTelemetry

Optional OTLP HTTP export. If `OTEL_EXPORTER_OTLP_ENDPOINT` is empty, tracing is a no-op.

| Env var | Description |
|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `host:port` of OTLP collector (no scheme) |
| `OTEL_SAMPLE_RATIO` | Sampling ratio 0.0–1.0 (default 0.1 in production) |

## Future per-block resolvers

When real outbound adapters replace `StubResolver`, each must:
1. Be registered in `bootstrap/app.go` via `blockRegistry.Register(...)`.
2. Wrap every HTTP call with `pkg/breaker.New(...)`.
3. Never forward arbitrary headers; only allowlisted params.
4. Log errors exactly once at the adapter layer; return `*domain.AppError`.
