# Configuration & Runtime Feature Flags

How `ms_home` is configured (environment variables) and how behavior is controlled at
runtime (CMS feature flags). Two distinct layers:

- **Deploy-time** — environment variables read once at startup ([internal/config/config.go](../internal/config/config.go)).
- **Runtime** — Contentstack `feature_flags` (from the GLOBAL entry) evaluated **per request**,
  changeable without a redeploy.

---

## 1. How env vars are defined

All configuration is loaded once at boot by `config.Load()` (Twelve-Factor; no hardcoded
URLs/secrets/timeouts). Three typed helpers back every value:

| Helper | Parsing | Invalid / empty value |
|---|---|---|
| `getEnv(key, fallback)` | raw string | empty string is treated as **unset** → `fallback` |
| `getBool(key, fallback)` | `strconv.ParseBool` (`true/false/1/0/t/f`) | unparseable → `fallback` |
| `getDuration(key, fallback)` | `time.ParseDuration` (`5s`, `750ms`, `2m`) | unparseable → `fallback` |

Conventions:
- **Naming** mirrors digital_bff where a shared backend exists (`SHARED_*`, `SALESFORCE_MODULE_HTTP`)
  so ops can reuse known values; service-local concerns use plain names (`PORT`, `AUTH_*`, `OTEL_*`).
- **Defaults are dev-safe**: the service boots locally with almost nothing set.
- **Fail-fast**: exactly one var is required — `SHARED_CONTENT_SERVICE_URL`. If it is missing,
  `Load()` returns an error and the process exits before serving.
- **Presence = enablement**: every optional backend URL acts as a feature switch. An empty URL
  means the corresponding outbound adapter/strategy is **not registered** at startup
  (see [bootstrap.go](../internal/bootstrap/bootstrap.go)). This is the deploy-time toggle.

---

## 2. Full reference

### Core
| Variable | Default | Required | Effect |
|---|---|---|---|
| `SHARED_CONTENT_SERVICE_URL` | — | ✅ | Content Service proxy base URL (the only hard dependency) |
| `SHARED_CONTENT_SERVICE_TIMEOUT` | `5s` | | Content Service call timeout |
| `PORT` | `8080` | | HTTP listen port |
| `ENV` | `dev` | | `dev`/`qa`/`staging`/`prod` (label + trace attribute) |
| `LOG_LEVEL` | `info` | | `debug` enables outbound cURL logging |
| `BUILD_VERSION` | `dev` | | Revision id echoed by `/healthz` + trace `service.version` (canary id) |
| `DEFAULT_BRAND` | `LP` | | Brand used when the request omits `x-brand-id` |
| `PERSONALIZATION_ENABLED` | `false` | | **Env gate** ANDed with the CMS `personalization` flag (see §4) |

### Optional backends (empty URL disables the feature)
| Variable | Default | Enables |
|---|---|---|
| `SHARED_GROUPBY_SEARCH_URL` | — | `product_list-groupby` carousels |
| `SHARED_GROUPBY_RECOMMENDATIONS_URL` | — | `product_list-recently_viewed` + banner_products similar-items |
| `SHARED_GROUPBY_TIMEOUT` | `5s` | GroupBy call timeout |
| `SHARED_JEWEL_URL` | — | `products_list` jewel carousels |
| `SHARED_JEWEL_TIMEOUT` | `5s` | Jewel call timeout |
| `SALESFORCE_MODULE_HTTP` | — | `products_cards`, `recommendation_product_list`, `product_list-salesforce`, birthday `container_greeting` |
| `SALESFORCE_MODULE_TIMEOUT` | `5s` | Salesforce call timeout |
| `SHARED_ATG_CART_HEADER_URL` | — | favorite store (→ `selected_store` events) + `continueBuying` shortcut |
| `SHARED_ATG_TIMEOUT` | `5s` | ATG call timeout |
| `SHARED_SEARCH_FACADE_URL` | — | `banner_products` (`/getMultiProduct`) |
| `SHARED_SEARCH_FACADE_TIMEOUT` | `5s` | Search Facade call timeout |

> `container`, `countdown`, `container_guest`, `container_shortcuts`, and `container_greeting`
> (non-birthday) need no backend and are always registered.

### Auth (mode by precedence: opaque > jwt > dev)
| Variable | Default | Effect |
|---|---|---|
| `AUTH_OPAQUE_EXCHANGE_URL` | — | **Opaque mode** (digital_bff parity): exchange the session cookie at the Auth service |
| `AUTH_COOKIE_NAME` | `SessionId` | Opaque session cookie name |
| `AUTH_JWKS_URL` | — | **JWT mode**: validate RS256 Bearer tokens against this JWKS |
| `AUTH_ISSUER` / `AUTH_AUDIENCE` | — | JWT `iss`/`aud`, validated only when non-empty |
| `AUTH_PROFILE_CLAIM` | `prn` | JWT mode: claim holding the profile id |
| `AUTH_TIMEOUT` | `5s` | Auth call timeout |

If both `AUTH_OPAQUE_EXCHANGE_URL` and `AUTH_JWKS_URL` are empty → **dev mode**: identity from
the `x-profile-id` header (local only — never enable in prod).

### Tracing
| Variable | Default | Effect |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | — | Set → export spans via OTLP; empty → propagation-only (W3C headers still flow) |
| `OTEL_SERVICE_NAME` | `ms_home` | Trace service name. Other standard `OTEL_*` vars are honored by the exporter |

---

## 3. How to use them

**Local (dev mode):**
```sh
cp configs/.env.example .env
# minimum:
export SHARED_CONTENT_SERVICE_URL=https://content-service.dev.example.com
go run ./cmd/server
# optional: turn on a feature by giving it a URL
export SHARED_SALESFORCE_URL=...   # (SALESFORCE_MODULE_HTTP)
```

**Production (Cloud Run):** set vars in [deployments/cloudrun.yaml](../deployments/cloudrun.yaml).
Recommended prod baseline: `ENV=prod`, `BUILD_VERSION=<tag>`, `PERSONALIZATION_ENABLED=true`,
the required Content Service URL, every backend URL you want active, `AUTH_OPAQUE_EXCHANGE_URL`
(parity), and `OTEL_EXPORTER_OTLP_ENDPOINT`.

**Enable / disable a capability** = add / remove its URL and redeploy. Example: rolling out
GroupBy carousels = set `SHARED_GROUPBY_SEARCH_URL`; instant disable = clear it.

---

## 4. Runtime feature flags (no redeploy)

There are **two control points** for behavior:

1. **Deploy-time switches** — whether an adapter/strategy exists at all (backend URL present).
2. **Runtime CMS flags** — `feature_flags` on the Contentstack **GLOBAL** entry, fetched per
   request via the Content Service proxy. Editing them in the CMS changes behavior immediately
   for the next request — no deploy.

### Flow of a CMS flag through a request
```
GLOBAL entry.feature_flags (CMS)
  → HomeService.fetch(): mergeFeatureFlags()         # personalization = envGate AND cms
  → withFlags(ctx, page): copy bool flags            # into RequestInfo.FeatureFlags (context)
  → strategies read ri.Flag("<name>")                # per-block gating during populate
```
- `mergeFeatureFlags` ([home_service.go](../internal/application/home_service.go)) forces
  `personalization` to `false` unless **both** `PERSONALIZATION_ENABLED` (env) **and** the CMS
  `personalization` flag are true. All other flags pass through from the CMS verbatim.
- `withFlags` stores the effective flags on `RequestInfo.FeatureFlags`; strategies call
  `ri.Flag(name)` ([request.go](../internal/domain/request.go)).

### Which flag gates what
| CMS flag | Gates |
|---|---|
| `personalization` | `container_guest`, `container_greeting`, jewel; custom-data events; legacy welcome; web `me`/shortcuts merge; favorite-store fetch |
| `groupby` | `product_list-groupby`, `product_list-recently_viewed`, `banner_products` |
| `salesforce` | `products_cards`, `recommendation_product_list`, `product_list-salesforce`, birthday `container_greeting` |
| `jewel` | `products_list` jewel (also requires `personalization`) |
| `shopping_assistant` | shopping-assistant shortcut |

A block is **dropped** (not rendered) when its gating flag is off — the populate framework
treats a strategy returning "drop" as removing that block, so toggling a CMS flag cleanly
adds/removes carousels at runtime.

### Truth table — effective personalization
| `PERSONALIZATION_ENABLED` (env) | CMS `personalization` | Effective |
|---|---|---|
| false | any | **false** |
| true | false | **false** |
| true | true | **true** |

This lets ops hard-disable personalization in an environment (env gate) regardless of CMS,
while the CMS remains the day-to-day switch.

### Notes & limits
- **Pocket (`screen`)** does not fetch GLOBAL, so it currently has no CMS flags in context →
  personalization-gated blocks are off for pocket until GLOBAL is wired for it (see
  [todos.md](todos.md)).
- Flags are read fresh per request from GLOBAL; there is no in-process flag cache, so CMS edits
  take effect on the next request (subject to any caching inside the Content Service proxy).
- Deploy-time and runtime layers combine: a strategy runs only if **both** its backend URL is
  configured **and** its CMS flag is on.
