# ms_home

Go service serving the **HOME** experience (web `page` + pocket `screen`), migrated
from the `digital_bff` (NestJS) monorepo. Hexagonal architecture, stateless,
optimized for Google Cloud Run.

> This is a **living document**. Each migration phase appends its run/test notes
> here and updates [docs/](docs/). See [docs/migration-roadmap.md](docs/migration-roadmap.md)
> for where we are and what's next.

---

## 1. Prerequisites
- **Go 1.26+** (`go version`)
- A reachable **Content Service** proxy URL (the existing `SHARED_CONTENT_SERVICE_URL`).
  For local work you can point at a dev proxy or a stub (see §5).
- No third-party Go dependencies — the service is stdlib-only.

## 2. Configure
Copy the sample env and adjust:
```sh
cp configs/.env.example .env
```
| Variable | Required | Default | Purpose |
|---|---|---|---|
| `SHARED_CONTENT_SERVICE_URL` | ✅ | — | Content Service proxy base URL |
| `SHARED_CONTENT_SERVICE_TIMEOUT` | | `5s` | Outbound call timeout |
| `SHARED_GROUPBY_SEARCH_URL` | | — | GroupBy search; empty disables `product_list-groupby` |
| `SHARED_GROUPBY_RECOMMENDATIONS_URL` | | — | GroupBy recs; empty disables `recently_viewed` |
| `SHARED_GROUPBY_TIMEOUT` | | `5s` | GroupBy call timeout |
| `SHARED_JEWEL_URL` | | — | Jewel service; empty disables `products_list` jewel |
| `SHARED_JEWEL_TIMEOUT` | | `5s` | Jewel call timeout |
| `SALESFORCE_MODULE_HTTP` | | — | Salesforce actions; empty disables Salesforce strategies |
| `SALESFORCE_MODULE_TIMEOUT` | | `5s` | Salesforce call timeout |
| `SHARED_ATG_CART_HEADER_URL` | | — | ATG cart header; empty disables favorite store + continue-buying |
| `SHARED_ATG_TIMEOUT` | | `5s` | ATG call timeout |
| `PORT` | | `8080` | HTTP listen port |
| `ENV` | | `dev` | `dev`/`qa`/`staging`/`prod` |
| `LOG_LEVEL` | | `info` | `debug` enables cURL logging of outbound calls |
| `DEFAULT_BRAND` | | `LP` | Brand used when `x-brand-id` is absent |
| `PERSONALIZATION_ENABLED` | | `false` | Env gate; ANDed with the CMS `personalization` flag |

## 3. Run locally
```sh
export $(grep -v '^#' .env | xargs)   # load .env (or use a dotenv tool)
go run ./cmd/server
```
The server logs `server starting` and listens on `:$PORT`. It shuts down gracefully
on `SIGINT`/`SIGTERM`.

### Endpoints
| Method | Path | Notes |
|---|---|---|
| GET | `/content/{contentType}/{locale}/{path...}` | HOME content |
| GET | `/healthz`, `/readyz` | Cloud Run probes |

Examples:
```sh
curl "localhost:8080/content/page/es-mx/"        # web HOME (path defaults to "/")
curl "localhost:8080/content/screen/es-mx/home"  # pocket HOME
curl -H "x-preview: 1" "localhost:8080/content/page/es-mx/"   # preview mode (brand -PREVIEW)
```

## 4. Test
```sh
go test ./...        # unit + table-driven tests
go vet ./...
gofmt -l .           # must print nothing
go test -bench=. ./... # benchmarks (when present)
```
Coverage today: flag-merge truth table, normalization (modular blocks + container_grid),
key rename/delete, content-type gating, populate framework (drop-on-failure, order),
strategies (`container`, `countdown`), and the HTTP client (secret masking,
context-cancellation).

## 5. Local stub for the Content Service (optional)
No proxy handy? Run a tiny stub and point the service at it:
```sh
python3 - <<'PY' &
import http.server, json
class H(http.server.BaseHTTPRequestHandler):
    def do_GET(self):
        self.send_response(200); self.send_header('Content-Type','application/json'); self.end_headers()
        body = {"feature_flags":{"personalization":True}} if self.path.startswith('/content/global/') \
               else {"_content_type_uid":"page","layout":{"blocks":[{"container":{"_metadata":{"uid":"c1"}}}]}}
        self.wfile.write(json.dumps(body).encode())
    def log_message(self,*a): pass
http.server.HTTPServer(('127.0.0.1',9099),H).serve_forever()
PY
SHARED_CONTENT_SERVICE_URL=http://127.0.0.1:9099 PERSONALIZATION_ENABLED=true go run ./cmd/server
```

## 6. Build & deploy
```sh
docker build -f deployments/Dockerfile -t ms-home .   # distroless, static binary
# Cloud Run: see deployments/cloudrun.yaml (set IMAGE + env per environment)
```

## 7. Architecture & docs
Hexagonal layout and rationale: [docs/architecture.md](docs/architecture.md).
Business rules being preserved (with ported/pending status):
[docs/business-rules.md](docs/business-rules.md). Open items: [docs/todos.md](docs/todos.md).

## 8. Status (what works today)
- ✅ Parallel page + GLOBAL fetch via the Content Service proxy; feature-flag merge
  (env ∧ CMS); `globalData` attach; path defaulting.
- ✅ Template normalization (drop `content`, `layout/top_layout/bottom_layout` →
  `blocks/top_content/bottom_content`, modular-block flattening).
- ✅ Content-type gating + per-type key rename/delete.
- ✅ Populate framework (parallel, drop-on-failure) with `container`, `countdown`,
  `product_list-groupby`, `product_list-recently_viewed`, `products_list` jewel,
  plus **personalization**: `container_guest`, `container_shortcuts`,
  `container_greeting` (birthday via Salesforce), `products_cards`,
  `recommendation_product_list`, `product_list-salesforce`. Greeting de-dup +
  legacy Android welcome container. Effective flags + identity via context.
- ✅ Golden-contract harness skeleton (`test/contract`, structural diff; see its README).
- ✅ Custom-data events (index + selected_store) at block + template level.
- ✅ ATG cart header → favorite store (resolves `selected_store` events) +
  `continueBuying` shortcut; web `shortcuts.shoppingAssistant`.
- ⏳ Next: `me` (User/token claims), buy-again/wishlist shortcuts (Apigee/Apigee2),
  `banner_products` (Search Facade multi-product), category data, OTel tracing,
  blacklist + AI metrics. See [docs/todos.md](docs/todos.md).
