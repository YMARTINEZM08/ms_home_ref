# E2E Gap Analysis — ms_home_liverpool vs digital_bff

**Test date:** 2026-06-20
**BFF endpoint:** `GET http://localhost:3000/web-bff/content/page/es-mx/tienda/home`
**Go endpoint:** `GET http://localhost:8081/home`
**Content-service:** `https://ogcp-apigke-d.liverpool.com.mx/content-service`
**Session state:** Guest (not logged in — `me.isLoggedIn: false`, `me.isGuest: true`)

> **Gaps 1, 2, and 3 were re-confirmed live on 2026-06-20** using `CONTENT_SERVICE_URL`
> derived from the `SHARED_CONTENT_URL` found in the repo's `.env` file. All three
> critical blockers are still present — the Go service returns `NOT_FOUND` with the
> current configuration.
>
> **Security note:** The `.env` file committed to `ms_home_liverpool/` contains
> the `digital_bff` production secrets (API keys, auth headers, service URLs).
> It must be removed from git history and a `.gitignore` added immediately.
> See [`.gitignore`](../.gitignore) — already created.

---

## Summary

| # | Gap | Severity | Area |
|---|---|---|---|
| 1 | Go calls wrong content-service URL (no page identifier) | **CRITICAL** | `outbound/contentservice/client.go` |
| 2 | Mapper looks for blocks at wrong location in response | **CRITICAL** | `outbound/contentservice/mapper.go` |
| 3 | Normalize does not handle double-wrapped block shape | **CRITICAL** | `outbound/contentservice/normalize.go` |
| 4 | Real block types differ from Go domain constants | **HIGH** | `domain/home/block_type.go` |
| 5 | `product_list` vs `products_list` name mismatch | **HIGH** | `domain/home/block_type.go` |
| 6 | `container` vs `container_grid` name mismatch | **HIGH** | `domain/home/block_type.go` |
| 7 | Block count divergence (28 CS vs 16 BFF) | **MEDIUM** | mapper + audience filtering |
| 8 | Go response does not include page-level data | **MEDIUM** | `inbound/http/response.go` |
| 9 | Go has no user session enrichment (`me`, `shortcuts`) | **MEDIUM** | design scope |
| 10 | Dynamic block population vs placeholder design | **DESIGN** | intentional — frontend contract |

---

## Gap 1 — Wrong content-service URL path (CRITICAL)

### Observation
The Go service calls:
```
GET https://ogcp-apigke-d.liverpool.com.mx/content-service/content/page/es-mx
```
The content-service responds with **404**:
```json
{"statusCode": 404, "error": "Not Found", "message": "Cannot GET /content/page/es-mx"}
```

The BFF (and the correct URL) requires the **page identifier** appended to the path:
```
GET https://ogcp-apigke-d.liverpool.com.mx/content-service/content/page/es-mx/tienda/home
```
This call succeeds and returns the full page entry.

### Root cause
`contentservice/client.go → buildURL()` constructs the path as:
```go
base.Path = fmt.Sprintf("/content/%s/%s", contentType, req.Locale)
// produces: /content/page/es-mx
```
The page slug (`tienda/home`) is never included.

### Fix needed
`HomeRequest` must carry the page identifier and `buildURL()` must append it:
```go
base.Path = fmt.Sprintf("/content/%s/%s/%s", contentType, req.Locale, req.PageID)
// produces: /content/page/es-mx/tienda/home
```
`GET /home` handler must either accept a `page` query param or have the slug hardcoded as a config value (recommended: `HOME_PAGE_ID=tienda/home`).

---

## Gap 2 — Blocks at wrong location in the response (CRITICAL)

### Observation
The Go mapper (`contentServiceResponse`) looks for blocks at the **root** of the response:
```go
type contentServiceResponse struct {
    Layout    []any `json:"layout"`
    Blocks    []any `json:"blocks"`
    TopLayout []any `json:"top_layout"`
}
```
None of these fields exist at the root. The content-service actual response structure is:
```
root
├── uid
├── locale
├── template
│   ├── _content_type_uid = "flex"
│   └── layout
│       └── blocks: [ ...28 items... ]   ← ACTUAL LOCATION
├── header
├── footer
├── seo
└── ...
```

### Fix needed
`contentServiceResponse` must reflect the real nesting:
```go
type contentServiceResponse struct {
    UID      string          `json:"uid"`
    Template templateField   `json:"template"`
    Header   map[string]any  `json:"header"`
    Footer   map[string]any  `json:"footer"`
    SEO      map[string]any  `json:"seo"`
    PageTitle string         `json:"page_title"`
}

type templateField struct {
    Layout templateLayout `json:"layout"`
}

type templateLayout struct {
    Blocks []any `json:"blocks"`
}
```
`layoutItems()` must return `resp.Template.Layout.Blocks`.

---

## Gap 3 — Double-wrapped block shape not handled (CRITICAL)

### Observation
`normalize.go` was written to handle **single-wrapped** blocks:
```json
{ "banner": { "_content_type_uid": "banner", "uid": "...", ... } }
```

The real content-service uses a **double-wrapped** structure for every block:
```json
{ "hero_banner_slider": { "hero_banner_slider": [ item1, item2, ... ] } }
```
- Outer key = content type name
- Inner key = content type name (same)
- Inner value = **list** of actual content items

**All 28 blocks** in the real response follow this double-wrapper pattern. The current `unwrap()` function returns `nil` for these because `val.(map[string]any)` fails when the inner value is a list.

### Fix needed
`unwrap()` and `handleContainer()` must be extended to handle the case where the inner value is `[]any` (extract and normalize each item in that list):
```go
// pseudo-code
case inner value is []any:
    for each item in the list:
        normalize each item and append to output
```

---

## Gap 4 — Block type constants do not match real CMS types (HIGH)

### Observation
Types seen in the real content-service response vs types defined in `block_type.go`:

| Real `_content_type_uid` | Count | In Go domain? | Go constant |
|---|---|---|---|
| `hero_banner_slider` | 6 | ❌ No | — |
| `product_list` | 10 | ❌ No (see Gap 5) | `products_list` (different name) |
| `container` | 6 | ❌ No (see Gap 6) | `container_grid` (different name) |
| `band` | 1 | ❌ No | — |
| `card_slider` | 1 | ❌ No | — |
| `user_generated_content` | 1 | ❌ No | — |
| `container_greeting` | 2 | ✅ Yes | `BlockTypeGreeting` |
| `container_guest` | 1 | ✅ Yes | `BlockTypeGuestContainer` |
| `container_shortcuts` | 1 | ✅ Yes | `BlockTypeShortcuts` |

Go domain types not seen in any real block:
- `banner`, `carousel`, `hero_banner`, `promo_bar`, `static_content`, `form`, `comparepage`, `search_banners`, `countdown`
- `products_list`, `banner_products`, `recommendation_product_list`, `products_cards`

### Fix needed
Add missing static types to `block_type.go`:
```go
BlockTypeHeroBannerSlider  BlockType = "hero_banner_slider"
BlockTypeBand              BlockType = "band"
BlockTypeCardSlider        BlockType = "card_slider"
BlockTypeUGC               BlockType = "user_generated_content"
BlockTypeContainer         BlockType = "container"
```

---

## Gap 5 — `product_list` vs `products_list` name mismatch (HIGH)

### Observation
The real CMS content type is `product_list` (10 occurrences in the home layout).
Go domain defines `BlockTypeProductsList BlockType = "products_list"` — note the extra `s`.

This means the dynamic classification check `domain.IsDynamic(raw.Type)` will return `false` for every real `product_list` block, classifying it as static. The resolve endpoint `/home/blocks/products_list` will also never match a real block's type.

Additionally, `IsAllowedResolveType("product_list")` returns `false`, so a frontend request to `/home/blocks/product_list` would be rejected with 400.

### Fix needed
Rename or alias the constant:
```go
BlockTypeProductList BlockType = "product_list"  // real CMS type
```
Update `dynamicBlockTypes`, `allowedBlockTypes`, and the `StubResolver` registration in `bootstrap/app.go`.

---

## Gap 6 — `container` vs `container_grid` name mismatch (HIGH)

### Observation
The real CMS content type for containers is `container` (6 occurrences). Go domain + `normalize.go` handle `container_grid` (flattens its `grid_items` children). The real `container` type has a `blocks` field (not `grid_items`) and its inner blocks are `card` type.

BFF after normalization shows the `container` blocks kept intact with their nested `card` children. The current `handleContainer()` switch in `normalize.go` only matches `"container_grid"` — real `container` blocks fall through to the default case and are returned as-is (without expanding children).

### Fix needed
1. Add `container` to `block_type.go` as a known static type.
2. Extend `handleContainer()` to handle `container` → expand its `blocks` field (not `grid_items`):
```go
case "container":
    return flattenContainer(block)  // reads block["blocks"]
```

---

## Gap 7 — Block count divergence: 28 (CS) vs 16 (BFF) (MEDIUM)

### Observation
The raw content-service returns **28 blocks** at `template.layout.blocks`.
The BFF response returns **16 blocks** after normalization and filtering.

**Blocks present in CS but absent in BFF response:**
- `container_greeting` (×2) — filtered because user is a guest (`me.isGuest: true`); greeting is for logged-in users
- `container_shortcuts` (×1) — filtered; shortcuts returned as a separate `shortcuts: {}` key (empty for guest)
- `product_list` entries (×10) — collapsed or filtered by the BFF's populate strategies

**Blocks in BFF but not visible in CS as independent entries:**
- `container` blocks in the BFF appear to aggregate/embed content that in the CS is split across multiple blocks

### Implication for Go service
The Go service must replicate audience-based filtering:
- `container_greeting` → only include when user is authenticated
- `container_guest` → only include when user is a guest
- `container_shortcuts` → surface separately, not inline in the blocks array (or filter per audience)

Currently `HomeRequest` has no `IsLoggedIn` or `AudienceType` field and the service applies no audience filtering. All blocks are returned to all users regardless of their session state.

---

## Gap 8 — Go response missing page-level data (MEDIUM)

### Observation
BFF response includes (beyond blocks):

| Field | BFF | Go |
|---|---|---|
| `header` | Full nav header | ❌ Absent |
| `footer` | Full footer with menu items | ❌ Absent |
| `seo` | `{meta_description, meta_keywords, canonical_url, ...}` | ❌ Absent |
| `page_title` | `"Liverpool \| Venta Especial \| Hasta 55% de descuento"` | ❌ Absent |
| `globalData` | Global CMS data | ❌ Absent |
| `me` | `{isLoggedIn, cartCount, firstName, ...}` | ❌ Absent |
| `shortcuts` | User shortcuts | ❌ Absent |
| `template.events` | Page events (GTM, analytics) | ❌ Absent |
| `template.json_ld_data` | Structured data (JSON-LD) | ❌ Absent |
| `template.live_bambuser` | Live shopping config | ❌ Absent |

Go response only returns:
```json
{ "blocks": [ ... ] }
```

### Decision required
The Go service was scoped to return the **block layout only**. If the frontend needs `header`, `footer`, `seo`, `page_title`, or `globalData`, three options exist:
1. **Expand Go response** — add these fields to `layoutResponse` (increases coupling to the full page entry).
2. **Separate endpoints** — add `GET /home/meta` and `GET /home/global` for non-block data.
3. **Frontend fetches directly** — header/footer/SEO are fetched independently by the frontend; Go only owns blocks.

Option 3 aligns with the microservice scope. This gap should be treated as an explicit scope decision, not a bug.

---

## Gap 9 — No user session enrichment (`me`, `shortcuts`) (MEDIUM)

### Observation
The BFF injects user session data into the home response:
```json
"me": {
  "isLoggedIn": false,
  "cartCount": 0,
  "firstName": "",
  "email": "",
  "profileId": "",
  "isGuest": true
}
```
This data is used by the frontend to personalize the UI (show/hide greeting, shortcuts, cart badge, etc.) and by the BFF itself to filter blocks by audience.

The Go service has no user context. `HomeRequest` contains only `Locale`, `Brand`, `Channel`, and `Preview`.

### Implication
Without knowing whether the caller is logged in or a guest, the Go service cannot:
- Filter `container_greeting` (logged-in only) vs `container_guest` (guest only)
- Populate `shortcuts` separately
- Return the correct `me` object

### Decision required
Two options:
1. **Accept an auth token / session header** — Go validates/introspects it and adds `IsLoggedIn`/`AudienceType` to `HomeRequest`. Audience filtering happens in the application layer.
2. **Return all blocks; let frontend filter** — Include `audience_filter` in the placeholder contract so the frontend hides irrelevant blocks client-side.

Option 2 requires no auth dependency in the layout endpoint and is lower risk. Option 1 gives server-side filtering and matches the BFF behavior.

---

## Gap 10 — Dynamic block population vs placeholder design (DESIGN)

### Observation
This is an **intentional design difference**, not a bug.

| Behavior | BFF | Go |
|---|---|---|
| `products_list` blocks | Fully populated inline with product data from GroupBy/Salesforce | Returned as placeholder; frontend calls `/home/blocks/products_list` |
| `container_greeting` | Populated inline with personalized greeting text | Returned as placeholder |
| Recommendations | Fetched and embedded inline | Placeholder pointing to resolve endpoint |

The BFF aggregates all data into one response. The Go service separates layout from data — the frontend calls resolve endpoints to get block data independently. This is the agreed architectural contract (see `docs/decisions.md` ADR-001 and ADR-007).

No code change is needed, but the **frontend must be updated** to call resolve endpoints for dynamic blocks instead of expecting inline data.

---

## Reproduce commands

### BFF (baseline)
```bash
curl -s 'http://localhost:3000/web-bff/content/page/es-mx/tienda/home' \
  -H 'x-brand-id: LP' \
  -H 'Cookie: JSESSIONID=...' \
  | jq '{blocks: [.template.blocks[] | {type: ._content_type_uid, enabled}]}'
```

### Content-service (raw)
```bash
curl -s 'https://ogcp-apigke-d.liverpool.com.mx/content-service/content/page/es-mx/tienda/home' \
  -H 'x-brand-id: LP' -H 'Accept: application/json' \
  | jq '.template.layout.blocks | length'
# → 28
```

### Go service (current — returns NOT_FOUND due to Gap 1)
```bash
curl -s http://localhost:8081/home -H 'x-brand-id: LP' | jq
# → {"error_code":"NOT_FOUND","message":"home layout could not be found.","retryable":false}
```

---

## Fix priority

| Priority | Gap | Effort |
|---|---|---|
| P0 | Gap 1 — correct content-service URL | Small — add `HOME_PAGE_ID` env var + update `buildURL` |
| P0 | Gap 2 — fix response struct to read `template.layout.blocks` | Small — update `contentServiceResponse` |
| P0 | Gap 3 — handle double-wrapped block shape in `normalize.go` | Medium — extend `unwrap` for `[]any` inner values |
| P1 | Gap 5 — rename `products_list` → `product_list` | Small — rename constant + update allowlist |
| P1 | Gap 6 — add `container` handling in normalize | Small — add case to `handleContainer` |
| P1 | Gap 4 — add missing block types to domain | Small — add constants |
| P2 | Gap 7 — audience-based block filtering | Medium — add audience field to `HomeRequest` |
| P3 | Gap 8 — page-level data (header/footer/SEO) | Decision — see options above |
| P3 | Gap 9 — user session (`me`, `shortcuts`) | Decision — see options above |
| — | Gap 10 — placeholder vs inline population | No code change; frontend contract |
