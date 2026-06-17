# Changelog

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
