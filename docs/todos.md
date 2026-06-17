# TODOs

## Phase-1a ✅ done
- Block normalization (rule #4) + content-type gating/rename/delete (rule #5).
- Populate framework (interface + registry + parallel + drop-on-failure).
- Pure strategies: `container`, `countdown`.

## Phase-1b ✅ mostly done
- ✅ GroupBy **search** adapter + `FromGroupBySearch` + `product_list-groupby`.
- ✅ GroupBy **recommendations** adapter + `FromGroupByRecomendation` +
  `product_list-recently_viewed` (brand seller filter, profileId/visitorId gating).
- ✅ **Jewel** adapter + `FromJewel` + `products_list` jewel strategy.
- ✅ Golden-contract harness skeleton (`test/contract`) — fixtures still to capture.

## Phase-1b remaining
- `banner_products` — needs Middleware `getFavoriteStore` (ATG/profile + store
  service → Phase 2) + Search Facade `getMultiProductDetails` + `FromSearchFacadeProduct`
  + GroupBy SIMILAR_ITEMS recommendations + `combineInformation`/`createStringSkuArray`
  over `hotspots_manager` image groups (`details`). Blocked on Search Facade
  multi-product request shape (not yet read) + favorite-store chain (TODO-2).
- **Blacklist**: wire Search Facade restricted-products endpoint (currently empty set,
  matching digital_bff fallback) + `BLACKLIST_REFRESH_MS` refresh.
- **AI metrics** (`pushMetric` → clientMetadataRecords) — deferred (Phase 2 metrics sink).
- Category Indexer outbound adapter; `categoryData` when `category_id` present
  (page-blp / clp) — not exercised by HOME page/screen.
- Capture golden-contract fixtures (web/pocket × anon/logged × preview × flags).
- Minor parity note: recomendation categories `id` serializes as `""` (TS `undefined`).

## Phase-2a ✅ done
- Salesforce adapter + `FromSalesfroce`; strategies `container_greeting`,
  `container_guest`, `container_shortcuts`, `recommendation_product_list`,
  `product_list-salesforce`, `products_cards`.
- Greeting de-dup (rule #7 partial); legacy Android welcome container (rule #8).

## Phase-2b ✅ done
- Custom-data events (rule #7) — block + template level; `RequestState` (tag_index + store).
- Web `shortcuts.shoppingAssistant` (rule #9 partial).

## Phase-2c ✅ done
- ATG cart-header adapter; favorite store → `RequestState.SelectedStore` (selected_store
  events resolve); `continueBuying` web shortcut.

## Phase-2d remaining
- **Login/auth**: real token-based `LoggedIn`/`ProfileID` (currently from `x-profile-id`
  header — assumes gateway-authenticated identity). Decide auth boundary. Also: digital_bff
  derives `isLoggedIn` from the cart header when unset — reconcile with the header source.
- **Web `me`** (rule #9): User service `getUserInformation` = ATG cart header details
  (already fetched in `loadSession`) + token claims (`decodeAccessToken`) projected via
  `CartHeaderDetailsDto` (excludeExtraneousValues). Needs the JWT claims + exact field list.
- **buy-again** (Apigee `getOrders`: POST `/generic/compras-services/orders`, `dyn-user-id`
  + `X-Impersonate-Profile-ID`; groupby flag), **wishlist** (Apigee2 `getWishlists`:
  GET `/digital_wish_list/wishList`, `dyn-user-id`) shortcuts.
- **`banner_products`**: Search Facade `getMultiProductDetails` (+ favorite store, now
  available) + GroupBy similar-items + `combineInformation` over hotspots image groups.
- **Set-Cookie favoriteStore** side effect (getFavoriteStore echoes the store cookie) — skipped.
- **Salesforce per-request memo**: dedupe identical action calls within one request
  (TS caches in reqContext.cache.salesforce) — minor efficiency, currently uncached.

## Phase-3
- Apigee/Apigee2 adapters: buy-again, wishlist shortcuts.

## Cross-cutting / unknowns (do not invent)
- **TODO-1** Direct-to-Contentstack adapter — needs Delivery/Preview keys + transform parity.
- **TODO-2** Per-strategy backend-call inventory (esp. async `supports`); GroupBy/Salesforce request shapes.
- **TODO-3** Confirm web/pocket clients tolerate JSON key ordering / absent optional fields.
- **TODO-4** `useDecommission` web semantics (bypasses personalization) — source + lifetime.
- **TODO-5** Caching ownership in ms_home (recommend none P1; evaluate per-request memo P2).
- **TODO-6** OTel tracing/metrics wiring (currently slog only).
- **TODO-7** Confirm Content Service response envelope (raw entry vs `{data:...}`) — adapter decodes raw.
