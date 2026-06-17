# Current State (Phase 0 + 1a + 1b + 2a + 2b + 2c complete)

Runnable vertical slice: HTTP → HomeService (normalize + gate + session + populate +
events + welcome + web shortcuts) → Content Service proxy (+ GroupBy, Jewel, Salesforce, ATG).

## Implemented (Phase 2c)
- `ports.CartHeaderPort` + outbound `atg` adapter (cart header).
- `HomeService.loadSession`: favorite store → `RequestState.SelectedStore` (selected_store
  events resolve); `RequestInfo.Cookie`; cart-header memo.
- `continueBuying` web shortcut (ATG last cart item).

## Implemented (Phase 2b)
- `domain.RequestState`: per-request tag_index counter + selected store (concurrency-safe).
- Custom-data events (`processBffCustomDataEvents`): block-level (PopulateAll) + template-level.
- `shortcuts.shoppingAssistant` web merge (self-contained).

Pending (Phase 2d): login/auth, `me`, buy-again/wishlist shortcuts, `banner_products`.

## Implemented (Phase 2a)
- `internal/product`: `FromSalesfroce`. Outbound `salesforce` adapter + `ports.SalesforcePort`.
- Strategies: `container_guest`, `container_shortcuts` (pure); `container_greeting`,
  `products_cards`, `recommendation_product_list`, `product_list-salesforce` (Salesforce).
- `PopulateAll` greeting de-dup; `HomeService` legacy Android welcome container.
- 11 of 12 populate strategies ported (only `banner_products` pending).
- Pending: login/auth, favorite store, custom-data events, web `me`/`shortcuts`, `banner_products`.

## Implemented (Phase 1b)
- `internal/product`: pure `ProductDto` + `FromGroupBySearch` / `FromGroupByRecomendation`
  / `FromJewel`.
- Outbound adapters: `groupby` (search + recommendations), `jewel`.
- `internal/populate` strategies: `product_list-groupby`, `product_list-recently_viewed`
  (brand seller filter; profileId/visitorId gating), `products_list` jewel
  (jewel+personalization flags; user/device-id gating). Blacklist no-op; AI metrics deferred.
- `domain.RequestInfo`: identity (ProfileID/VisitorID/Jewel ids) + client headers +
  effective `FeatureFlags`; flags threaded into context. Each strategy registers only
  when its URL is configured.
- `test/contract`: golden-contract harness skeleton.
- Pending: `banner_products` (Search Facade multi-product + favorite store).

## Implemented (Phase 1a)
- `internal/content`: `NormalizeDoubleBlocks` (+ container_grid flatten), content-type
  gating constants, `RenameKeys`/`DeleteKeys`.
- `internal/populate`: Strategy/Registry/Service (parallel, drop-on-failure) +
  `container`, `countdown` strategies.
- `HomeService.GetHome`: template extraction → drop `content` → layout mapping →
  UID/category gating → rename/delete → parallel populate.

## Implemented (Phase 0)
- Hexagonal skeleton (cmd / domain / ports / application / adapters / config / bootstrap / pkg).
- Env config (`internal/config`) — `SHARED_CONTENT_SERVICE_URL` required.
- `pkg/httpclient`: reused client, keep-alive, context propagation, structured slog
  logs (method/url/status/latency), **cURL emission at DEBUG**, sensitive-header masking.
- Content Service adapter: `GET /content/{ct}/{locale}/{id}`, `x-brand-id` (+`-PREVIEW`).
- `HomeService.GetHome`: parallel page + GLOBAL fetch (GLOBAL skipped for pocket),
  fatal page/global errors, feature-flag merge (env ∧ CMS), `globalData` attach.
- Inbound handler + router: `GET /content/{contentType}/{locale}/{path...}`,
  path defaulting (screen→`home`, else→`/`), health `/healthz` `/readyz`.
- `cmd/server`: tuned `http.Server`, SIGTERM graceful shutdown.
- Tests: flag-merge truth table, pocket-skips-global, fatal-error paths, httpclient
  masking + context-cancellation + success.
- Dockerfile (distroless), Cloud Run manifest, `.env.example`.

## Not yet implemented (Phase 1+)
See [todos.md](todos.md). Block normalization, the 12 populate strategies,
personalization merge (`me`/`shortcuts`), category data, custom events, legacy
Android welcome container, OTel tracing, golden-contract harness.
