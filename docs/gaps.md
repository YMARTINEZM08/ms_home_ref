# E2E Gap Analysis — ms_home_liverpool vs digital_bff

**Test date:** 2026-06-20
**BFF endpoint:** `GET http://localhost:3000/web-bff/content/page/es-mx/tienda/home`
**Go endpoint:** `GET http://localhost:8081/home` / `http://localhost:8083/home`
**Content-service:** `https://ogcp-apigke-d.liverpool.com.mx/content-service`

## Test runs

| Run | Date | Session | Go result | BFF blocks | Notes |
|---|---|---|---|---|---|
| 1 | 2026-06-20 | Guest (`me.isGuest: true`) | `NOT_FOUND` | 16 | Gaps 1-3 confirmed |
| 2 | 2026-06-20 | Logged-in cookies (expired) | `NOT_FOUND` | 16 | All gaps persist; session expired — see note |

### Run 2 — logged-in session findings

The session cookies (`LoggedInSession=TRUE`, `DYN_USER_ID=31005309570`) appear to have **expired**. Evidence:

- The BFF response has **no `me` key** (guest run had `me: {isLoggedIn, isGuest, firstName, …}`).
- The BFF response has **no `shortcuts` key** (guest run had `shortcuts: {}`).
- The block list is **byte-for-byte identical** to the guest run (16 blocks, same order, same types).
- `container_guest` is still present at position [01]; `container_greeting` is still absent — behaviour expected for an unauthenticated user.

With valid logged-in cookies the BFF would return a populated `me` object (`isLoggedIn: true`, `firstName`, `cartCount`, etc.) and is expected to swap `container_guest` for `container_greeting`.

> **All gaps from Run 1 remain unchanged in Run 2.** The Go service returns `NOT_FOUND` in both runs for the same root cause (Gap 1).

> **Security note:** The `.env` file in `ms_home_liverpool/` contains `digital_bff` production secrets
> (API keys, auth headers, service URLs). A `.gitignore` has been added — see [`.gitignore`](../.gitignore).

---

## Summary

| # | Gap | Severity | Status | Fix date |
|---|---|---|---|---|
| 1 | Go calls wrong content-service URL (no page identifier) | **CRITICAL** | ✅ Fixed | 2026-06-21 |
| 2 | Mapper looks for blocks at wrong location in response | **CRITICAL** | ✅ Fixed | 2026-06-21 |
| 3 | Normalize does not handle list-wrapped block shape | **CRITICAL** | ✅ Fixed | 2026-06-21 |
| 4 | Real block types differ from Go domain constants | **HIGH** | ✅ Fixed | 2026-06-21 |
| 5 | `product_list` vs `products_list` name mismatch | **HIGH** | ✅ Fixed | 2026-06-21 |
| 6 | `container` vs `container_grid` name mismatch | **HIGH** | ✅ Fixed | 2026-06-21 |
| 7 | No audience filtering (greeting vs guest block) | **MEDIUM** | ✅ Fixed | 2026-06-21 |
| 8 | Go response does not include page-level data | **MEDIUM** | 📋 Scoped out | see below |
| 9 | Go has no user session enrichment (`me`, `shortcuts`) | **MEDIUM** | 📋 Scoped out | see below |
| 10 | Dynamic block population vs placeholder design | **DESIGN** | ✅ Intentional | by design |

---

## Fix priority

| Priority | Gap | Effort | Status |
|---|---|---|---|
| P0 | Gap 1 — correct content-service URL | Small | ✅ Done |
| P0 | Gap 2 — fix response struct to read `template.layout.blocks` | Small | ✅ Done |
| P0 | Gap 3 — handle list-wrapped block shape in `normalize.go` | Medium | ✅ Done |
| P1 | Gap 5 — rename `products_list` → `product_list` | Small | ✅ Done |
| P1 | Gap 6 — add `container` handling in normalize | Small | ✅ Done |
| P1 | Gap 4 — add missing block types to domain | Small | ✅ Done |
| P2 | Gap 7 — audience-based block filtering | Medium | ✅ Done |
| P3 | Gap 8 — page-level data (header/footer/SEO) | Decision | 📋 Scoped out |
| P3 | Gap 9 — user session (`me`, `shortcuts`) | Decision | 📋 Scoped out |
| — | Gap 10 — placeholder vs inline population | No code change | ✅ By design |

---

## Gap 1 — Wrong content-service URL path ✅ FIXED

### Fix applied
Added `HOME_PAGE_ID` env var (default `tienda/home`). `buildURL()` now produces:
```
/content/{type}/{locale}/{HOME_PAGE_ID}
→ /content/page/es-mx/tienda/home
```

**Files changed:**
- `internal/config/config.go` — `ContentServiceConfig.HomePageID` + `Load()` reads `HOME_PAGE_ID`
- `internal/adapters/outbound/contentservice/client.go` — `Config.HomePageID`; `buildURL()` 3-segment path
- `internal/bootstrap/app.go` — wires `HomePageID`
- `configs/.env.example` — documents `HOME_PAGE_ID=tienda/home`
- `internal/adapters/outbound/contentservice/client_test.go` — `newClient()` sets `HomePageID`

---

## Gap 2 — Blocks at wrong location in the response ✅ FIXED

### Fix applied
`contentServiceResponse` now reflects the real nesting:

```go
type contentServiceResponse struct {
    UID      string     `json:"uid"`
    Template csTemplate `json:"template"`
}
type csTemplate struct { Layout csLayout `json:"layout"` }
type csLayout   struct { Blocks []any    `json:"blocks"` }
```

`layoutItems()` returns `r.Template.Layout.Blocks`. Test payloads updated to match.

**Files changed:**
- `internal/adapters/outbound/contentservice/mapper.go`
- `internal/adapters/outbound/contentservice/client_test.go`

---

## Gap 3 — List-wrapped block shape not handled ✅ FIXED

### Root cause
CMS layout items use a list-wrapped shape for every block:
```json
{ "hero_banner_slider": [ item1, item2, ... ] }
```
The previous `unwrap()` attempted `val.(map[string]any)` and skipped on `[]any`, returning nil for all blocks.

### Fix applied
`unwrap()` now has a `case []any` branch that iterates each item, sets `_content_type_uid` from the outer key if missing, then calls `handleContainer` per item:

```go
case []any:
    for _, raw := range v {
        item, ok := raw.(map[string]any)
        ...
        item["_content_type_uid"] = key
        out = append(out, handleContainer(item)...)
    }
```

**Files changed:** `internal/adapters/outbound/contentservice/normalize.go`

---

## Gap 4 — Missing block type constants ✅ FIXED

### Fix applied
Added to `block_type.go` (static section):

| Constant | Value |
|---|---|
| `BlockTypeHeroBannerSlider` | `"hero_banner_slider"` |
| `BlockTypeBand` | `"band"` |
| `BlockTypeCardSlider` | `"card_slider"` |
| `BlockTypeUGC` | `"user_generated_content"` |
| `BlockTypeContainer` | `"container"` |

**Files changed:** `internal/domain/home/block_type.go`

---

## Gap 5 — `product_list` vs `products_list` name mismatch ✅ FIXED

### Fix applied
Renamed `BlockTypeProductsList` → `BlockTypeProductList`, value `"products_list"` → `"product_list"` across all files:

- `internal/domain/home/block_type.go` — constant + allowlists
- `internal/bootstrap/app.go` — StubResolver registration
- All `*_test.go` files with string literal `"products_list"` or the old constant name

---

## Gap 6 — `container` vs `container_grid` name mismatch ✅ FIXED

### Fix applied
1. `BlockTypeContainer BlockType = "container"` added as a static type.
2. `handleContainer()` now routes `"container"` to `flattenContainer()`:
   ```go
   func flattenContainer(block map[string]any) []map[string]any {
       sub, _ := block["blocks"].([]any)
       if len(sub) > 0 {
           block["blocks"] = normalize(sub)
       }
       return []map[string]any{block}
   }
   ```
   This normalises nested `card` children while keeping the container as a single top-level block — matching BFF output where containers appear intact with their nested data.

**Files changed:** `internal/adapters/outbound/contentservice/normalize.go`, `internal/domain/home/block_type.go`

---

## Gap 7 — No audience-based block filtering ✅ FIXED

### Observation
`container_greeting` is for logged-in users only; `container_guest` is for unauthenticated users only. The BFF filters based on session state. The Go service had no audience concept.

### Fix applied
1. Added `IsLoggedIn bool` to `domain.HomeRequest`.
2. `filterByAudience()` in `application/home/classify.go` removes audience-gated blocks before classification:
   - Guest (`IsLoggedIn=false`): drops `container_greeting`
   - Logged-in (`IsLoggedIn=true`): drops `container_guest`
3. `home_handler.go` reads `x-authenticated: true` header (set by the API gateway after token validation — the service trusts this header within the internal network and never validates tokens itself).
4. Tests: `TestFilterByAudience_Guest` and `TestFilterByAudience_LoggedIn` cover both paths.

**Files changed:**
- `internal/domain/home/home.go`
- `internal/application/home/classify.go`
- `internal/application/home/service.go`
- `internal/application/home/export_test.go`
- `internal/application/home/classify_test.go`
- `internal/adapters/inbound/http/home_handler.go`

---

## Gap 8 — Go response missing page-level data 📋 SCOPED OUT

### Decision: frontend fetches independently (Option 3)

The BFF aggregates `header`, `footer`, `seo`, `page_title`, `globalData`, `template.events`, `template.json_ld_data`, and `template.live_bambuser` into one response.

**Chosen approach:** these are out of scope for the Go layout service. The microservice contract is `{ "blocks": [...] }`. Header, footer, SEO, and global config are either:
- Fetched by the frontend from a dedicated endpoint (CDN-cached), or
- Provided by a separate CMS adapter service.

This is deliberate — mixing layout blocks with page metadata would couple the service to the full page entry shape and make caching harder (blocks are static; metadata changes separately).

**No code change needed.** Document the explicit contract: `GET /home` returns layout blocks only.

---

## Gap 9 — No user session enrichment (`me`, `shortcuts`) 📋 SCOPED OUT

### Decision: session data is not a layout concern

The BFF injects `me: {isLoggedIn, cartCount, firstName, ...}` and `shortcuts: {...}` into the home response. The Go service has no user context beyond `IsLoggedIn` (which it uses for audience filtering only).

**Chosen approach:** the Go service does not return `me` or `shortcuts`. These are session-layer concerns:
- `me` is owned by a user/auth service; the frontend should fetch it from there.
- `shortcuts` are user-specific dynamic data; the `container_shortcuts` block is already returned as a dynamic placeholder pointing at `/home/blocks/container_shortcuts`.

Mixing session enrichment into the layout endpoint would require the layout service to make an authenticated downstream call, breaking failure isolation (a session service blip would blank the entire home page layout).

**No code change needed.** The `IsLoggedIn` field in `HomeRequest` is sufficient for audience filtering. Full session data stays out of scope.

---

## Gap 10 — Dynamic block population vs placeholder design ✅ Intentional

This is an **intentional design difference**, not a bug. The Go service returns layout placeholders for dynamic blocks; the frontend calls `/home/blocks/{blockType}` to resolve each one independently. This is the agreed architectural contract (see `docs/decisions.md` ADR-001 and ADR-007).

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

### Go service — guest
```bash
CONTENT_SERVICE_URL=https://ogcp-apigke-d.liverpool.com.mx/content-service \
HOME_PAGE_ID=tienda/home \
go run ./cmd/home &

curl -s http://localhost:8080/home \
  -H 'x-brand-id: LP' \
  -H 'x-authenticated: false' \
  | jq '{total: (.blocks | length), types: [.blocks[].type]}'
```

### Go service — logged-in
```bash
curl -s http://localhost:8080/home \
  -H 'x-brand-id: LP' \
  -H 'x-authenticated: true' \
  | jq '{total: (.blocks | length), types: [.blocks[].type]}'
```
