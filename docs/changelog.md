# Changelog

## Phase 3b — Cutover mechanics
- `BUILD_VERSION` stamp: echoed by `/healthz` `/readyz` (`{"status","version"}`), startup
  log, and the `service.version` trace attribute — identifies the canary revision.
- `docs/rollout.md`: cutover runbook (routing options, pre-cutover gates, canary stages
  0→1→5→25→50→100, gate signals, instant weight→0 rollback).
- `deployments/gateway-routing.example.yaml`: weighted HOME backend split (GCLB sketch).
- `deployments/cloudrun.yaml`: minScale 1 for canary, BUILD_VERSION, provider/auth/OTEL env stubs.

## Phase 3a — Observability (OTel tracing) & parity tooling
- `observability.InitTracing`: W3C propagation always; OTLP exporter when
  `OTEL_EXPORTER_OTLP_ENDPOINT` set (else propagation-only, no-op tracer). Graceful
  shutdown flush wired through `App.Shutdown` → main.
- Inbound `traceMiddleware`: server span per request (skips health), status attribute.
- `pkg/httpclient`: client span per outbound call + traceparent injection + attributes
  (method/url/server.address/status); spans no-op unless a provider is installed.
- Deps added: OpenTelemetry SDK + OTLP/HTTP exporter.
- `scripts/capture-fixtures.sh`: capture digital_bff vs ms_home responses for the
  golden-contract harness (web/pocket × anon/preview/logged).
- Tests: propagation extract/inject; e2e verified trace id flows to the downstream.

## Phase 2f — Auth boundary: service-side JWT validation
- `internal/auth.Verifier`: RS256 JWT validation via JWKS (golang-jwt/jwt/v5 + stdlib
  JWKS cache with throttled refresh). Rejects alg=none/HMAC, unknown kid, expired,
  wrong issuer/audience. `AUTH_*` config.
- Handler: when configured, identity comes only from a valid Bearer token (x-profile-id
  ignored); else dev fallback to x-profile-id. `RequestInfo.Claims` carries JWT claims.
- `me`: now merges token claims over the cart-header projection across the full
  CartHeaderDetailsDto `@Expose` allowlist (claims win) — token fields
  (lastPasswordReset, dateOfBirth, isSignUp, …) now populated.
- Dep added: `github.com/golang-jwt/jwt/v5`.
- Tests: verifier (valid/expired/wrong-issuer/alg-none/unknown-kid, in-memory JWKS),
  me claims merge, handler bearer→me + dev-header path; race-tested.

## Phase 2e — banner_products (12/12 strategies)
- `internal/product`: `FromSearchFacadeProduct` mapper.
- Outbound `searchfacade` adapter (`/getMultiProduct`: searchFacadeConfig
  `{dataCenter:SiteA, brand, channel}` + `productIds` + favoriteStore) + `SearchFacadePort`.
- `banner_products` strategy: `createStringSkuArray` (hotspot image groups),
  multi-product fetch, GroupBy similar-items fallback for missing skus,
  `combineInformation` (attaches `details` to image groups by baseId/productId).
- `product.ToMap` (Dto → block field). `loadSession` broadened to `personalization||groupby`
  so banner_products gets the favorite store. `SHARED_SEARCH_FACADE_URL` config.
- Tests: `FromSearchFacadeProduct`; banner_products (combine, sku dedupe, drop paths, null details).

## Phase 2d — `me` projection & Salesforce request memo
- `me` (rule #9): `projectMe` builds it from the memoized ATG cart header — the
  CartHeaderDetailsDto `@Expose` subset, `email = login` remap, favoriteStore →
  `{storeName, id}`. Attached to the web HOME merge (personalization). Token-claim
  `@Expose` fields gateway-forwarded → currently omitted (see todos / auth boundary).
- `RequestState.SalesforceAction`: per-action, per-request memo (sync.Once) so repeated
  blocks share one Salesforce call (mirrors reqContext.cache.salesforce). Race-tested.
- Tests: `projectMe` (remap, subset, drop non-expose, omit absent); memo dedup + error caching.

## Phase 2c — ATG session: favorite store & continue-buying
- `ports.CartHeaderPort` + outbound `atg` adapter (`getCartHeaderDetails`:
  body fromBuyNow/rearrange; headers brand/channel/cookie; returns cartHeaderDetails).
- `domain.RequestInfo.Cookie`; `RequestState` cart-header memo.
- `HomeService.loadSession`: one cart-header fetch per request (personalization),
  resolves `favoriteStore.{id,storeName}` → `RequestState.SelectedStore` (so
  `selected_store` events resolve) — tolerated on failure.
- `continueBuyingShortcut` (ATG `lastCartAddedItem`, logged-in) added to the web merge.
- `config.ATG` (`SHARED_ATG_CART_HEADER_URL`); strategy/adapter registered only when set.
- Tests: continue-buying shortcut; GetHome favorite-store + selected_store resolution.

## Phase 2b — Custom-data events & web shortcuts
- `domain.RequestState` (per-request, concurrency-safe): tag_index counter (starts 1)
  + selected store; carried on `RequestInfo.State`, set in the handler.
- `internal/populate/events.go`: `processBffCustomDataEvents` port (button_events/events/
  dot_events; index → tag_index; selected_store.name/code; non-array list → []).
  Wired into `PopulateAll` (block-level) + `HomeService` (template-level, personalization).
- `HomeService.attachWebShortcuts` + `shoppingAssistantShortcut` (web/page,
  personalization): self-contained shopping-assistant shortcut.
- Tests: events (index, store, non-array, non-bff, nil-state); shopping-assistant.
- Deferred (need ATG/Apigee/User/Search Facade): favorite store (→ selected store),
  `me`, continue-buying/buy-again/wishlist shortcuts, `banner_products`.

## Phase 2a — Personalization core (Salesforce + pure)
- `internal/product`: `FromSalesfroce` mapper.
- Outbound `salesforce` adapter (`getActionFromUser`: action + source.application +
  user.ID_ATG1) + `ports.SalesforcePort` + `SALESFORCE_MODULE_HTTP` config.
- Strategies: `container_guest`, `container_shortcuts` (pure, in DefaultStrategies);
  `container_greeting` (personalization + birthday-via-Salesforce campaign check),
  `products_cards` (INCREDIBLE_OFFERS), `recommendation_product_list` (CAN_LIKE),
  `product_list-salesforce` (carousel; min/max slice; campaignName/title).
- `PopulateAll`: container_greeting de-duplication (keep birthday one).
- `HomeService`: legacy Android `container_welcome` injection for logged-in screen.
- Bootstrap: greeting always registered (Salesforce optional); other Salesforce
  strategies registered only when `SALESFORCE_MODULE_HTTP` is set.
- Tests: `FromSalesfroce`, all 6 strategies, greeting de-dup, welcome container.

## Phase 1b (cont.) — recently_viewed & jewel carousels
- `internal/product`: `FromGroupByRecomendation` (flat dotted keys) + `FromJewel`
  (+ `DiscountLabel`); shared int/number helpers.
- Adapters: GroupBy `RecommendationsAdapter` (POST + `groupByGetRecordsFields`);
  new `jewel` adapter (GET model; array or `{products}` envelope).
- Strategies: `product_list-recently_viewed` (groupby flag + profileId/visitorId
  gating + brand seller filter + min threshold + `records.modelId`); `products_list`
  `jewel` (jewel+personalization flags + user/device-id gating + min default 3/15).
- `domain.RequestInfo`: `JewelUserID`/`JewelDeviceID` (x-jml-* headers).
- `config`: `SHARED_GROUPBY_RECOMMENDATIONS_URL`, `SHARED_JEWEL_URL`
  (+ `SHARED_GROUPBY_TIMEOUT`); strategies registered only when their URL is set.
- Tests: both mappers + both strategies (gating, filter, min, error).

## Phase 1b — GroupBy product_list carousels
- `internal/product`: pure `ProductDto` + `FromGroupBySearch` mapper.
- `internal/adapters/outbound/groupby`: search adapter (price/availability refinements,
  loginId/visitorId, clientMetadata) implementing `ports.GroupBySearchPort`.
- `internal/populate`: `product_list-groupby` strategy (flag/category/audience/surface
  gating, min-products threshold, indexed products, `productsListId`). Blacklist no-ops
  to empty set (digital_bff fallback); AI metrics deferred.
- `domain.RequestInfo`: ProfileID/VisitorID/client headers + effective `FeatureFlags`
  (`Flag()`); flags threaded into context before populate; identity read in handler.
- `config`: `SHARED_GROUPBY_SEARCH_URL`/`_TIMEOUT`; strategy registered only when set.
- `test/contract`: golden-contract harness skeleton (structural JSON diff + capture docs).
- Tests: ProductDto mapping, groupby strategy (flag/category/min/audience/error), slug.

## Phase 1a — Normalization, gating & populate framework
- `internal/content`: `NormalizeDoubleBlocks` (+ `container_grid` flatten), content-type
  constants (`ReturnWithoutChanges`/`TemplatesWithUid`/`NeedCategoryID`/`AvailableLayouts`),
  `RenameKeys`/`DeleteKeys`.
- `internal/populate`: `Strategy` interface, `Registry`, `Service` (parallel populate,
  drop-on-failure) + pure strategies `container`, `countdown`.
- `HomeService.GetHome`: template extraction, `content` drop, layout mapping, UID/category
  gating, key rename/delete, parallel container populate.
- Tests: normalization, rename/delete, populate framework (order, drop-on-error),
  full GetHome normalize+populate, missing-UID error.

## Phase 0 — Scaffold & vertical slice
- Hexagonal skeleton (cmd/internal/pkg), stdlib-only.
- Env config; required `SHARED_CONTENT_SERVICE_URL`.
- `pkg/httpclient` with keep-alive, structured logs, cURL@debug, secret masking.
- Content Service proxy adapter (`GET /content/{ct}/{locale}/{id}`, `x-brand-id`).
- `HomeService`: parallel page+GLOBAL fetch, env∧CMS flag merge, `globalData` attach.
- Inbound `GET /content/{contentType}/{locale}/{path...}` + path defaulting; health probes.
- Graceful shutdown; Dockerfile (distroless); Cloud Run manifest.
- Unit tests: flag merge, error paths, httpclient masking/cancellation.
