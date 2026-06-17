# Contentstack / Content Service Analysis

ms_home does **not** call Contentstack directly. It calls the existing **Content
Service proxy** (`SHARED_CONTENT_SERVICE_URL`), which owns Delivery/Preview API
access, transforms, caching, retries/backoff, rate limiting, and auth (preserved
per skill Rule 17 — do not move that logic).

## Proxy contract used by HOME
- `GET /content/{contentType}/{locale}/{id}`
- Header `x-brand-id: {brand}[-PREVIEW]` (preview toggled by inbound `x-preview`).
- Content types (HOME-relevant): `page` (web), `screen` (pocket), `global`
  (nav/footer/**feature flags**).

## Caching
- Caching currently lives in the proxy. ms_home caches nothing in Phase 0/1.
  Per-request memoization to be evaluated in Phase 2 (TODO-5).

## Direct-to-Contentstack (deferred — TODO-1)
The skill prefers a direct outbound adapter. Deferred until Delivery/Preview keys
and parity for the proxy's transforms/caching are available.
