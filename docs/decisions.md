# Architecture Decisions

## ADR-001 — Single `/home` layout endpoint + independent per-block resolve endpoints

**Decision:** `GET /home` composes only the layout (static blocks inline, dynamic blocks as placeholders). Each dynamic block type has its own `GET /home/blocks/{blockType}` endpoint.

**Reason:** Decouples layout availability from downstream data source availability. One failing personalization service cannot blank the entire page. Each endpoint is independently toggleable and observable.

## ADR-002 — Circuit breakers on every outbound call; no retries; 5% failure threshold

**Decision:** `pkg/breaker` wraps every outbound call with a `gobreaker` circuit breaker. No retry logic. Breaker trips at 5% failure ratio over a minimum of 20 requests.

**Reason:** Retries amplify load on a struggling downstream during an incident. The breaker provides fast-fail behavior. The 5% threshold is low enough to catch real degradation without tripping on normal error rates.

**Open-state fallback:** returns `SERVICE_UNAVAILABLE` / "service not available at this moment". In `/home` the affected block degrades to its `fallback`; the page still renders.

## ADR-003 — Content-service proxy, not direct Contentstack CDA

**Decision:** All CMS content goes through the existing internal `content-service` proxy (`CONTENT_SERVICE_URL`). No direct Contentstack SDK usage.

**Reason:** Centralizes CMS auth and caching; avoids duplicating CDN/preview logic; matches the contract already in production.

## ADR-004 — Chi + net/http

**Decision:** Chi v5 router with stdlib `net/http`. No framework like Gin or Echo.

**Reason:** Minimal dependency surface; stdlib-compatible middleware; easy to test with `httptest`. Chi adds only routing and `chi.URLParam` — nothing that locks in the framework.

## ADR-005 — `AppError` centralized error model

**Decision:** All errors are typed as `*domain.AppError` carrying `Code, Category, Status, Message, Detail, Retryable, Cause`. Adapters return errors; only the handler layer logs them (exactly once).

**Reason:** Consistent, frontend-safe responses. Separates developer detail (`Detail`, `Cause`) from consumer message (`Message`). Prevents duplicate log lines across layers.

## ADR-006 — Package named `handler`, not `http`

**Decision:** `internal/adapters/inbound/http/` package is declared `package handler`.

**Reason:** Avoids shadowing `net/http` which would require every file in the package to import the stdlib under an alias.

## ADR-007 — Dynamic blocks as placeholders only in `/home`

**Decision:** `/home` never populates dynamic block data inline. It returns `resolve_endpoint` + `enabled` + `fallback` + `feature_flag_id`.

**Reason:** Personalized/session-dependent data (recommendations, greetings, shortcuts) must not be cached at the page level. Keeping dynamic data behind separate endpoints lets the frontend apply per-user caching independently.

## ADR-008 — `BLOCK_TEMPORARILY_DISABLED` (423) vs `SERVICE_UNAVAILABLE` (503)

**Decision:** Two distinct error codes:
- `BLOCK_TEMPORARILY_DISABLED` (423): feature flag is intentionally off.
- `SERVICE_UNAVAILABLE` (503): circuit breaker is open / downstream is unhealthy.

**Reason:** Frontend and on-call engineers need to distinguish a deliberate toggle from an outage. 423 is non-retryable; 503 is retryable.
