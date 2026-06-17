# Business Rules (HOME) — to preserve exactly

Source of truth: `digital_bff` `content.service.ts` + the two `content.controller.ts`.
Status: ✅ ported | ⏳ pending.

1. ✅ **Path defaulting** — web `'' → '/'`; pocket `'' → 'home'`.
2. ✅ **Parallel fetch** — GLOBAL (web/csc only, not pocket) + page; page/global rejection is fatal.
   ✅ favorite-store fetch (personalization) via ATG cart header, tolerated on failure
   (`HomeService.loadSession`, runs after flag merge).
3. ✅ **Feature-flag merge** — `personalization = envGate AND cms.personalization`.
4. ✅ **Block normalization** — drop `template.content`; map `layout→blocks`,
   `top_layout→top_content`, `bottom_layout→bottom_content` (`NormalizeDoubleBlocks`,
   incl. `container_grid` flattening). `internal/content/normalize.go`.
5. ✅ **Content-type gating** — `ReturnWithoutChanges` / `TemplatesWithUid` /
   `NeedCategoryID`; `RenameKeys`/`DeleteKeys` per content type. `internal/content/`.
6. ✅ **Populate** `blocks`/`top_content`/`bottom_content`/`products` in parallel;
   a block that fails to populate is **dropped**, request still succeeds.
   `internal/populate/`. ⏳ category data (Category Indexer) still pending.
7. ✅ **Custom data events** — `processBffCustomDataEvents` at block-level (PopulateAll)
   and template-level (personalization). index→tag_index; selected_store.name/code from
   `RequestState.SelectedStore` (currently nil → null until favorite store lands).
8. ✅ **Legacy Android welcome container** — injected into `screen` blocks for logged-in
   users (personalization on), after `container_shortcuts`. `content.LegacyWelcomeContainer`.
9. 🟡 **Web personalization merge** — `shortcuts.shoppingAssistant` + `shortcuts.continueBuying`
   (ATG cart) attached (web/page, personalization). Deferred: `me` (User service + token claims).

## 12 populate strategies — ✅ 11 of 12 ported
- `container`, `countdown` (deterministic).
- `product_list-groupby` (GroupBy search + `FromGroupBySearch`; blacklist no-op; metrics deferred).
- `product_list-recently_viewed` (GroupBy recs + `FromGroupByRecomendation`; brand seller filter).
- `products_list` jewel (Jewel model + `FromJewel`; jewel+personalization flags).
- `container_guest` (personalization; guests only), `container_shortcuts` (logged-in; flatten items).
- `container_greeting` (personalization; birthday via Salesforce campaign check).
- `products_cards` (Salesforce INCREDIBLE_OFFERS), `recommendation_product_list` (CAN_LIKE),
  `product_list-salesforce` (`FromSalesfroce`; min/max; campaignName/title).

⏳ pending (1) — needs providers not yet read; do not invent shapes:
- `banner_products` — Middleware `getFavoriteStore` (ATG/profile + store service) +
  Search Facade `getMultiProductDetails` (`FromSearchFacadeProduct`) + GroupBy
  similar-items + `combineInformation` over `hotspots_manager` image groups.
- Salesforce: `products_cards`, `product_list-salesforce`, `recommendation_product_list`.
- Personalization/login: `container_greeting`, `container_guest`, `container_shortcuts`.

Framework preserves `supports(block)` + drop-on-failure; new strategies register via
`populate.DefaultStrategies()`.
