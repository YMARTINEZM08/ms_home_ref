# Migration Plan — `ms_home_liverpool` (Go Home service)

> Note: file references of the form `apps/...` and `libs/...` point at the legacy
> `digital_bff` repository (read-only reference), not this repo.

## Context

`ms_home_liverpool` is a brand‑new, essentially empty Go repo (only `go.mod` →
`module ms_home_ref`, `go 1.26.4`, plus the `go-code-quality` skill). The
`go-code-quality` skill (`golang-contentstack-hexagonal-architect` v3) prescribes
an **enterprise Go + Contentstack Hexagonal Architecture** service optimized for
Google Cloud Run, scoped to **Home logic + Contentstack interactions only**
(Rule 0).

The legacy `digital_bff` (Node/NestJS+Fastify monorepo) is the **read‑only
reference** (Rule 19): we may copy only the *external integration contracts*, never
its business logic, DTOs, package structure, or naming. Today the legacy Home lives
in the `content` domain:
- Web home: `GET /content/page/:locale` · Mobile: `GET /content/screen/:locale?channel=pocket`
- Core: `apps/web-bff/src/app/domain/content/service/content.service.ts`,
  `libs/providers/src/services/content.service.ts`,
  populate strategies in `libs/providers/src/services/strategies/populate.registry.ts`,
  block normalization in `libs/providers/src/utils/block.utils.ts`.
- Contentstack is **not** called directly; the BFF proxies an internal **content‑service**
  (`SHARED_CONTENT_URL`) over raw undici HTTP with an `x-brand-id` header.

**Goal of this migration:** stand up the new Go service that exposes a single
`GET /home` endpoint. It composes the page **layout** from the content‑service
proxy, returns **static blocks inline** and **dynamic blocks as placeholders**
(block id, type, resolve‑endpoint/path, fallback, feature‑flag id, enabled flag) so
the frontend can resolve them via dedicated endpoints. The service performs **no**
recommendation/personalization logic (Rule 18). A dedicated error contract tells the
frontend when a block path/endpoint is **temporarily disabled**.

### Decisions (confirmed with user)
1. **`GET /home` resolves only the layout** — static blocks inline + dynamic blocks as
   placeholders. **Independent per‑block resolve endpoints** (part of this service)
   resolve the dynamic details. This is deliberately modular: each dynamic block can be
   toggled, observed, and failure‑isolated on its own. Include a specific error for a
   temporarily‑disabled path/endpoint.
2. **CMS access:** reuse the existing **content‑service proxy** (not direct Contentstack CDA).
3. **Dynamic blocks:** `/home` returns placeholder + resolve‑endpoint only (no inline
   population); the detail comes from the block's own resolve endpoint.
4. **Framework:** **Chi + net/http**.
5. **Resilience:** **circuit breakers on every outbound call**, **no retries**, trip at a
   **5% failure threshold**. When a breaker is open, the fallback returns a custom error
   — `"service not available at this moment"` (machine code `SERVICE_UNAVAILABLE`).
6. **Error handling & logging** follow the `logger-handler` skill
   (`enterprise-error-handling-logging`): a centralized custom‑error model with
   `errorCode, category, status, message, detail, retryable, cause`; standardized
   frontend‑safe responses (never expose stack traces/framework errors); structured,
   runtime‑level‑configurable logging; **each failure logged exactly once by the highest
   responsible layer**; no middleware/filters introduced *solely* for error handling.
7. **Security is a first‑class, cross‑cutting concern** in every layer (see the Security
   section): strict input validation/allowlisting, SSRF‑safe outbound calls, secret
   hygiene, least‑privilege runtime, and no sensitive data in responses, logs, or traces.

---

## Architecture (Hexagonal — skill Rule 1 & 3)

Dependencies point inward; the domain never imports infrastructure. Proposed layout:

```
cmd/home/main.go                      # thin entrypoint → bootstrap
internal/
  domain/home/                        # entities + ports, zero infra deps
    home.go                           # Home aggregate, Block, StaticBlock, DynamicBlock
    block_type.go                     # block type constants (Rule 20)
    errors.go                         # AppError model + categories (logger-handler skill)
    ports.go                          # inbound HomeUseCase + outbound ContentPort interfaces
  application/home/
    service.go                        # HomeService: orchestrates layout composition
    classify.go                       # static-vs-dynamic classification + placeholder build
  application/blocks/                  # one use case per dynamic block type (block-oriented, Rule 19.5+)
    resolve.go                        # BlockResolver interface + registry keyed by block type
    <blocktype>.go                    # per-block use case (orchestration only, no business logic)
  adapters/
    inbound/http/
      router.go                       # Chi router + middleware wiring
      home_handler.go                 # GET /home → layout use case → response DTO
      block_handler.go                # GET /home/blocks/{blockType} → block resolver → detail DTO
      response.go                     # domain → JSON contract mapping
      errors.go                       # domain error → HTTP status/code mapping
    outbound/contentservice/
      client.go                       # net/http client to content-service proxy (breaker-wrapped)
      mapper.go                       # content-service payload → domain blocks
      normalize.go                    # flatten double/grid/tabs blocks (ref Rule 17/legacy block.utils)
    outbound/<downstream>/            # per-block detail sources (breaker-wrapped), as needed
  config/config.go                    # env-driven config struct + validation
  bootstrap/app.go                    # DI wiring, server, graceful shutdown, OTel init
pkg/
  logger/                             # slog structured logger (request_id, correlation_id, trace_id)
  httpx/                              # reusable http.Client (keep-alive), header masking, debug cURL
  breaker/                            # circuit breaker wrapper (no retries, 5% trip), open→SERVICE_UNAVAILABLE
  observability/                      # OTel tracer + health helpers
configs/                             # env templates per DEV/QA/Staging/Prod
deployments/                         # Dockerfile, Cloud Run service yaml
docs/                                # architecture.md, decisions.md, integrations.md, deployment.md, changelog.md
test/                                # integration tests + content-service fixtures
```

> Optional: rename go.mod module from `ms_home_ref` to a stable path
> (e.g. `github.com/Servicios-Liverpool-Infraestructura/ms_home_liverpool`). Keep
> `ms_home_ref` if avoiding import churn is preferred — decide before writing imports.

---

## Domain model & contract

**`Home`** = ordered `[]Block` (ordering from content‑service is preserved verbatim;
never reorder — Rule 18).

**`Block`** (discriminated by `Type`, a constant from `block_type.go`):
- **StaticBlock** — no session dependency, eligible for caching; carries resolved
  Contentstack content inline.
- **DynamicBlock** (placeholder) — exposes the Rule 18 contract:
  `block_id`, `block_type`, `resolve_endpoint` (path the frontend calls),
  `fallback`, `feature_flag_id`, `enabled`.

**Static vs dynamic classification** lives in `application/home/classify.go`, driven by
block type + an `audience_filter`/`source_of_data`/`handle` signal mirrored from the
legacy block shape (`libs/providers/src/types/populate.types.ts`) —
any block that is session/runtime dependent (recommendations, greetings, shortcuts,
recently‑viewed, salesforce/groupby/jewel/lob sources) becomes a placeholder. We copy
only the *contract* (which types are dynamic), not the population logic.

**Temporarily‑disabled error contract:** a typed domain error
`ErrBlockDisabled` mapped by `adapters/inbound/http/errors.go` to a stable HTTP
status + machine code (e.g. `423` / `"BLOCK_TEMPORARILY_DISABLED"`). Two surfaces:
1. In `/home`, a dynamic block whose feature flag is off is returned with
   `enabled:false` + its `fallback` (page still renders — Rule 18 failure handling).
2. The shared error contract is reusable by the future resolve endpoints so a disabled
   path returns the same `BLOCK_TEMPORARILY_DISABLED` code consistently.

**Resilience:** one block (or the content‑service) failing must never blank the whole
page — partial failures degrade to fallback/placeholder, never panic (Rule 12).

---

## Outbound adapter — content‑service proxy

`adapters/outbound/contentservice/client.go` implements `domain.ContentPort`:
- Single reused `*http.Client` with keep‑alive + per‑request `context` timeout
  (Rule 4/5). Configurable, default ~30s to match legacy `SHARED_CONTENT_TIMEOUT`.
- Calls the legacy content endpoint shape (`GET {CONTENT_SERVICE_URL}/content/{type}/{locale}[/{path}]`),
  preserving the **only** required auth/integration header: `x-brand-id`
  (`{brand}` or `{brand}-PREVIEW` when preview) — ref
  `libs/providers/src/providers/content.provider.ts`.
- Per Rule 10: every outbound call logs request_id, correlation_id, latency,
  response status, execution time; masks `Authorization`/tokens/cookies; emits an
  equivalent **cURL** at DEBUG level.
- `mapper.go` + `normalize.go` translate the content‑service payload into domain
  blocks, flattening the legacy double/`container_grid`/`tabs_container` nesting
  (contract from `libs/providers/src/utils/block.utils.ts`). No SDK,
  no SQL/ORM (Rule 2).

> Pagination/retries/rate‑limiting do **not** exist in the legacy content path; add a
> simple bounded retry/backoff at the adapter only if needed — keep YAGNI (Rule 15).

---

## Inbound adapters — `/home` + per‑block resolve endpoints

- Chi router with middleware: request‑id/correlation‑id propagation, OTel tracing,
  structured access log, panic‑recovery (Rule 12), CORS.
- **`GET /home`** (`home_handler.go`) — composes the layout only. Inputs: `locale`,
  `brand` (from `x-brand-id`), optional `channel`, `preview`. Output: ordered layout
  with static blocks inline + dynamic placeholders. Each placeholder's `resolve_endpoint`
  points at the matching per‑block endpoint below.
- **`GET /home/blocks/{blockType}`** (`block_handler.go`) — independent, modular detail
  resolution for one dynamic block. Dispatches to the `BlockResolver` registered for that
  block type (`application/blocks`). Each block type is independently toggleable (feature
  flag) and independently failure‑isolated (its own breaker), so issues are easy to
  pinpoint and switch off. Orchestration/proxy only — **no** recommendation/personalization
  logic (Rule 18); the detail comes from the block's outbound adapter.
- `errors.go` maps domain errors → HTTP: `BLOCK_TEMPORARILY_DISABLED` (flag off) and
  `SERVICE_UNAVAILABLE` (breaker open / downstream down).
- Health: `GET /healthz` (liveness) + `GET /readyz` (readiness, checks config; does
  not hard‑fail on content‑service blips). Graceful shutdown on SIGTERM (Cloud Run).

---

## Resilience — circuit breakers (no retries)

`pkg/breaker` wraps **every** outbound call (content‑service in `/home`, and each
per‑block downstream in the resolve endpoints):
- **No retries** — a failed call fails fast; the breaker counts it.
- **Trip at 5% failure ratio** over a rolling window, with a minimum request volume so
  a tiny sample can't trip it; configurable (`BREAKER_FAILURE_RATIO=0.05`,
  `BREAKER_MIN_REQUESTS`, `BREAKER_OPEN_TIMEOUT`, half‑open probes). Implement with a
  small lib (e.g. `sony/gobreaker`) or a thin internal wrapper.
- **One breaker per outbound dependency** (keyed by adapter/block) so one failing
  downstream never trips another — matching the modular toggling goal.
- **Open‑state fallback** returns the custom error `SERVICE_UNAVAILABLE` →
  `"service not available at this moment"`. In `/home` this degrades the affected block
  to its `fallback`/placeholder and never blanks the page (Rule 18); in a resolve
  endpoint it is returned as the endpoint's error body.
- Breaker state transitions are logged + traced (OTel) and exposed for observability.

**Error‑code summary:** `BLOCK_TEMPORARILY_DISABLED` = intentionally toggled off;
`SERVICE_UNAVAILABLE` = breaker open / dependency unhealthy. Distinct, so the frontend
and operators can tell a deliberate toggle from an outage.

---

## Config (Rule 6 — all via env, nothing hardcoded)

`internal/config/config.go` loads & validates:
`PORT`, `SERVICE_NAME`, `ENVIRONMENT` (dev/qa/staging/prod), `LOG_LEVEL`,
`CONTENT_SERVICE_URL`, `CONTENT_SERVICE_TIMEOUT_MS`, `DEFAULT_BRAND`,
`BREAKER_FAILURE_RATIO` (0.05), `BREAKER_MIN_REQUESTS`, `BREAKER_OPEN_TIMEOUT`,
`OTEL_EXPORTER_OTLP_ENDPOINT` / sampling, feature‑flag source. Provide
`configs/.env.example` and per‑env templates. New names (do **not** copy legacy
`SHARED_*` naming — Rule 19), but documented mapping in `docs/integrations.md`.

---

## Error model & logging (`logger-handler` skill + Rules 9–12)

**Centralized error model** in `internal/domain/home/errors.go` — a single `AppError`
type carrying `errorCode, category, status, message (consumer‑safe), detail (developer),
retryable, cause (internal only)`, with a category hierarchy: `Validation, Business,
ResourceNotFound, ExternalService, Timeout, Configuration, Infrastructure, Unexpected`.
Each category defines default metadata to avoid duplication. Concrete cases used here:
- `BLOCK_TEMPORARILY_DISABLED` → category `Business`/`Configuration`, `retryable:false`,
  message ≈ "This section is currently turned off."
- `SERVICE_UNAVAILABLE` (breaker open / downstream down) → category `ExternalService`/
  `Infrastructure`, `retryable:true`, message = "service not available at this moment",
  `detail` explains the open breaker + dependency.
- Content‑service timeouts → `Timeout`; bad input → `Validation`.

**Responses** are standardized and frontend‑safe — never leak stack traces, framework
errors, or `cause`/`detail` internals. `adapters/inbound/http/errors.go` is a thin,
framework‑native translation from `AppError` → HTTP status + JSON body (`errorCode`,
`message`, `retryable`); **no heavy filter/interceptor introduced solely for error
handling** (skill rule) — errors carry their own status so mapping stays trivial.

**Logging discipline** (cloud‑cost optimized): `slog` structured JSON only, parameterized
(never concatenated, never whole‑object dumps). Each **failure is logged exactly once by
the highest responsible layer** — outbound adapters return errors without logging; the
use case / handler logs once with full context. Runtime‑configurable level
(`OFF/ERROR/WARN/INFO/DEBUG/TRACE` via `LOG_LEVEL`, changeable without redeploy); DEBUG/
TRACE only during incidents; skip evaluation when disabled. Log only valuable events
(startup/shutdown, config/external/unexpected failures, breaker state transitions) — not
per‑method calls or hot paths. Context fields: `timestamp, service, operation, class,
method, level, elapsedTime, errorCode, request_id, correlation_id, trace_id`. Never log
secrets/tokens/cookies/PII.

## Observability (Rules 9–11)

- OpenTelemetry tracing by default; spans around the outbound content‑service call and
  each per‑block resolve; breaker transitions recorded as span events.
- `/healthz`, `/readyz`. Cloud Run‑friendly: stateless, fast startup, no local storage.

---

## Security (cross‑cutting — applied in every layer)

- **Input validation & allowlisting (inbound):** validate `locale`, `brand`, `channel`,
  `preview`, and especially `{blockType}` against a **fixed allowlist** of known block
  types — never use raw path/query input to build downstream URLs or select adapters.
  Enforce max request size, request timeout, and reject unknown params.
- **SSRF prevention (outbound):** the content‑service base URL and every downstream URL
  come from **config only**; user input may select an allowlisted *path/identifier*, never
  a host. No user‑controlled redirects; cap redirects; TLS with certificate verification on
  all outbound calls.
- **Header trust boundary:** treat inbound headers as untrusted — allowlist and sanitize
  `x-brand-id`, `x-correlation-id`, `x-preview`; sanitize/validate correlation IDs before
  logging to prevent **log injection**; do not forward arbitrary client headers downstream.
- **AuthN/AuthZ & session:** `/home` composes anonymous/static layout. Any block whose
  detail is session‑dependent must have its **resolve endpoint validate the session/token**
  (propagated, not trusted blindly) before returning data — personalization stays behind
  authenticated, per‑block endpoints (aligns with Rule 18 + legacy `auth-guard`).
- **Secret hygiene:** all secrets via env / GCP Secret Manager; never hardcoded, committed,
  logged, or placed in traces. Mask `Authorization`/tokens/cookies in logs, cURL debug, and
  OTel attributes (reuse `pkg/httpx` masking).
- **No sensitive data in responses:** frontend‑safe errors only (no stack traces, internal
  detail, or upstream bodies leaked); JSON content‑type with proper output encoding; set
  baseline security response headers.
- **Cache safety:** only static, non‑session blocks are cacheable; cache keys include
  `brand+locale+preview` so content can never leak across brands/users (no cache poisoning).
- **Least‑privilege runtime:** distroless, non‑root, read‑only filesystem, no extra
  capabilities; minimal dependency surface; run `govulncheck` + image/dependency scanning in
  CI; pin/verify modules (`go.sum`). Rate‑limit / load‑shed friendly for DoS resistance.
- **Documentation:** capture the threat model + controls in `docs/security.md`; a security
  review (the `/security-review` skill) should gate the first implementation PR.

## Documentation (Rule 8)

Seed `docs/`: `architecture.md` (hexagonal + block contract), `decisions.md` (the
decisions above), `integrations.md` (content‑service contract + header + legacy env
mapping), `deployment.md` (Cloud Run), `changelog.md`, **`error-handling.md`**
(per `logger-handler` skill: exception hierarchy, error model, logging strategy,
runtime log configuration, operational + cloud‑cost recommendations), and
**`security.md`** (threat model + controls from the Security section).

---

## Testing (Rule 13)

- Domain + application: table‑driven unit tests for classification, ordering
  preservation, disabled‑block → fallback, partial‑failure resilience, normalization.
- Outbound adapter: `httptest` server returning content‑service fixtures (capture a
  real legacy payload as a golden file); assert mapping + header + cURL/masking.
- Inbound handlers: end‑to‑end `httptest` on `GET /home` and `GET /home/blocks/{blockType}`,
  incl. `BLOCK_TEMPORARILY_DISABLED` and `SERVICE_UNAVAILABLE` (breaker open) → frontend‑safe
  body shape (`errorCode`/`message`/`retryable`, no leaks).
- Breaker: unit test that the 5% threshold opens the breaker, no retries occur, and the
  open‑state fallback returns `SERVICE_UNAVAILABLE`.
- Error/logging: assert each failure is logged exactly once at the highest layer.
- Security: tests for `{blockType}` allowlist rejection, oversized/invalid input,
  SSRF‑safe URL construction (host always from config), secret masking in logs/cURL/OTel,
  and unauthorized access to session‑dependent resolve endpoints.
- Context‑cancellation tests on the outbound client. Optional benchmark on the
  compose path (critical path).

---

## Build / deploy

`deployments/Dockerfile` — multi‑stage, distroless, non‑root, `CGO_ENABLED=0`,
graceful shutdown. Cloud Run service descriptor. (CI can mirror the legacy
`.github/workflows` later; out of scope for this slice.)

---

## Suggested implementation order

1. Scaffold module + package skeleton; pick/confirm module path.
2. Domain: `home.go`, `block_type.go`, `errors.go` (centralized `AppError` model +
   categories incl. `SERVICE_UNAVAILABLE`, `BLOCK_TEMPORARILY_DISABLED`), `ports.go`.
3. `pkg/breaker` (no retries, 5% trip, open→`SERVICE_UNAVAILABLE`).
4. Outbound content‑service adapter (breaker‑wrapped) + mapper + normalize (+ golden fixture).
5. Application `HomeService` (compose, classify, placeholders, resilience).
6. `application/blocks` resolver registry + per‑block use cases + their outbound adapters.
7. Inbound Chi handlers (`/home` + `/home/blocks/{blockType}`) + response/error mapping + health + middleware.
8. Config + logger + OTel + bootstrap wiring + graceful shutdown.
9. Tests across layers (incl. breaker open → fallback).
10. Dockerfile + Cloud Run descriptor + `docs/`.

---

## Verification (end‑to‑end)

1. `go build ./...` and `go vet ./...` clean.
2. `go test ./...` green (unit + adapter + handler).
3. Run locally: `CONTENT_SERVICE_URL=<proxy> PORT=8080 go run ./cmd/home`, then
   `curl -s localhost:8080/home -H 'x-brand-id: LP' -H 'x-correlation-id: test' | jq`
   → ordered layout, static blocks inline, dynamic blocks as placeholders carrying
   `resolve_endpoint`, `fallback`, `feature_flag_id`, `enabled`.
4. Per‑block resolve: `curl localhost:8080/home/blocks/<blockType>` returns that block's
   detail independently of `/home`.
5. Disabled path: with the relevant feature flag off, confirm the block returns
   `enabled:false` + fallback in `/home`, and the resolve endpoint returns
   `BLOCK_TEMPORARILY_DISABLED`.
6. Breaker: point a downstream at a failing stub; after the 5% threshold trips, confirm
   the breaker opens, `/home` degrades that block to its fallback (page still renders),
   and the resolve endpoint returns `SERVICE_UNAVAILABLE` /
   `"service not available at this moment"` — with no retry attempts in the logs.
7. Error model: failure responses are frontend‑safe — carry `errorCode`, `message`,
   `retryable`; never leak stack traces, `detail`, or `cause`. Confirm each failure is
   logged exactly once (no duplicate log lines from adapter + use case).
8. `curl localhost:8080/healthz` and `/readyz` → 200.
9. DEBUG logs show a masked, replayable cURL for the content‑service call; logs carry
   `correlation_id`/`trace_id`/`errorCode`; breaker state transitions are logged/traced.
   Changing `LOG_LEVEL` takes effect without redeploy.
10. Compare the `/home` layout order against the legacy `GET /content/page/:locale`
    response to confirm ordering parity (structure only — dynamic data intentionally differs).
11. Security: send an invalid/unknown `{blockType}` and malformed inputs → rejected
    (no downstream call); confirm no secrets/PII in logs or traces; run `govulncheck ./...`
    clean; run the `/security-review` skill on the implementation PR.
