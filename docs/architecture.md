# Architecture

`ms_home` is a stateless Go service serving the **HOME** experience (web `page` +
pocket `screen`), migrated from `digital_bff`. **Hexagonal (Ports & Adapters)** —
dependencies point inward; the domain/application never import infrastructure.

> Diagrams use Mermaid. View on GitHub or any Mermaid-aware Markdown viewer.

---

## 1. System context (all systems involved)

Every outbound dependency is **optional** (enabled by setting its URL — see
[configuration.md](configuration.md)); only the Content Service is required.

```mermaid
flowchart TB
    subgraph clients[Clients]
      web[Web app<br/>contentType=page]
      pocket[Pocket app<br/>contentType=screen]
    end

    gw[Edge / API gateway<br/>weighted HOME routing]

    subgraph svc[ms_home Cloud Run]
      ms[ms_home service]
    end

    web --> gw
    pocket --> gw
    gw -->|GET /content/...| ms

    ms -->|page + GLOBAL content REQUIRED| cs[(Content Service proxy<br/>→ Contentstack)]
    ms -->|session-cookie exchange| auth[(Auth service<br/>/v2/auth/exchange-token · Auth0)]
    ms -->|cart header / favorite store| atg[(ATG)]
    ms -->|product search| gbs[(GroupBy Search)]
    ms -->|recommendations / similar items| gbr[(GroupBy Recommendations)]
    ms -->|jewel model products| jw[(Jewel)]
    ms -->|personalized actions| sf[(Salesforce)]
    ms -->|multi-product details| sfac[(Search Facade)]
    ms -->|spans OTLP| otel[(OTel collector<br/>→ Cloud Trace)]

    style cs stroke-width:3px
    style ms fill:#e6f0ff
```

| System | Purpose | Enabled by |
|---|---|---|
| Content Service proxy | HOME page + GLOBAL (feature flags) — **required** | `SHARED_CONTENT_SERVICE_URL` |
| Auth service (Auth0) | opaque session → claims (`prn`, `isAnonymous`) | `AUTH_OPAQUE_EXCHANGE_URL` (or local JWT via `AUTH_JWKS_URL`) |
| ATG | cart header → favorite store + last cart item | `SHARED_ATG_CART_HEADER_URL` |
| GroupBy Search | `product_list-groupby` carousels | `SHARED_GROUPBY_SEARCH_URL` |
| GroupBy Recommendations | recently-viewed + banner similar-items | `SHARED_GROUPBY_RECOMMENDATIONS_URL` |
| Jewel | jewel-model carousels | `SHARED_JEWEL_URL` |
| Salesforce | greeting/offers/recommendation actions | `SALESFORCE_MODULE_HTTP` |
| Search Facade | `banner_products` multi-product details | `SHARED_SEARCH_FACADE_URL` |
| OTel collector | trace export | `OTEL_EXPORTER_OTLP_ENDPOINT` |

---

## 2. Hexagonal layers (dependencies point inward)

```mermaid
flowchart LR
    subgraph inbound[Inbound adapter]
      H[http.Handler + Router<br/>auth · tracing middleware]
    end

    subgraph app[Application]
      HS[HomeService<br/>orchestration]
      POP[populate.Service<br/>strategy registry]
    end

    subgraph dom[Domain - pure]
      D[ContentType · Document · Block<br/>RequestInfo · RequestState · errors]
    end

    subgraph ports[Ports - interfaces]
      P[ContentPort · GroupBy* · JewelPort<br/>SalesforcePort · CartHeaderPort · SearchFacadePort]
    end

    subgraph outbound[Outbound adapters]
      OA[contentservice · groupby · jewel<br/>salesforce · atg · searchfacade · auth]
    end

    H --> HS
    HS --> POP
    HS --> D
    POP --> D
    HS --> P
    POP --> P
    OA -. implements .-> P
    OA --> EXT[(External systems)]

    style dom fill:#fff5e6
    style ports fill:#eef9ee
```

Rule: arrows only point **toward** the domain. Adapters depend on ports (interfaces);
the application depends on ports, never on concrete adapters. SDK/HTTP types never leak
past an adapter.

---

## 3. Package layout & dependencies

```mermaid
flowchart TD
    main[cmd/server] --> boot[internal/bootstrap]
    boot --> cfg[internal/config]
    boot --> obs[internal/observability]
    boot --> inhttp[adapters/inbound/http]
    boot --> app[internal/application]
    boot --> outs[adapters/outbound/*]
    boot --> authp[internal/auth]

    inhttp --> app
    inhttp --> authp
    app --> ports[internal/ports]
    app --> pop[internal/populate]
    app --> content[internal/content]
    pop --> ports
    pop --> product[internal/product]
    outs --> ports
    outs --> hc[pkg/httpclient]
    authp --> hc
    ports --> domain[internal/domain]
    app --> domain
    pop --> domain
```

| Path | Responsibility |
|---|---|
| `cmd/server` | entrypoint, HTTP server, graceful shutdown (+ tracing flush) |
| `internal/domain` | pure types: `ContentType`, `Document`, `Block`, `RequestInfo`, `RequestState`, errors |
| `internal/ports` | outbound interfaces (`ContentPort`, `GroupBy*Port`, `JewelPort`, `SalesforcePort`, `CartHeaderPort`, `SearchFacadePort`) |
| `internal/application` | `HomeService` — orchestration (port of `content.service.ts`) |
| `internal/populate` | strategy framework + 12 block strategies + events |
| `internal/content` | pure CMS transforms (normalization, gating, welcome) |
| `internal/product` | `ProductDto` + mappers (GroupBy/Jewel/Salesforce/SearchFacade) |
| `internal/adapters/inbound/http` | handler, router, auth, tracing middleware |
| `internal/adapters/outbound/*` | one client per backend |
| `internal/auth` | JWT verifier + opaque-token exchange |
| `internal/config` · `internal/observability` | env config · slog + OTel |
| `internal/bootstrap` | compile-time wiring (no reflection) |
| `pkg/httpclient` | shared HTTP client (keep-alive, logging, cURL@debug, masking, client spans) |

---

## 4. HOME request — communication sequence

End-to-end flow of `GET /content/page/es-mx/` for a logged-in web user
(personalization + groupby + salesforce on). Optional calls run only when their flag
+ backend are enabled.

```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant H as inbound/http Handler
    participant A as Authenticator
    participant HS as HomeService
    participant CS as Content Service
    participant ATG as ATG
    participant POP as populate.Service
    participant EXT as GroupBy / Salesforce / Search Facade

    C->>H: GET /content/page/es-mx/ (Cookie / Bearer)
    H->>A: Authenticate(request)
    A->>EXT: (opaque) exchange-token  /  (jwt) verify via JWKS
    A-->>H: profileID, loggedIn, claims
    H->>HS: GetHome(ctx, page, locale, path)

    par parallel fetch
        HS->>CS: GET page content
    and
        HS->>CS: GET GLOBAL (feature_flags)
    end
    CS-->>HS: page + global
    HS->>HS: mergeFeatureFlags (env ∧ CMS) → ctx flags

    opt personalization || groupby
        HS->>ATG: cart header → favorite store (RequestState)
    end

    HS->>HS: normalize template (layout→blocks, drop content, gating)
    HS->>POP: PopulateAll(blocks)  (parallel, drop-on-failure)
    POP->>EXT: per-strategy calls (flag-gated)
    EXT-->>POP: products / actions
    POP-->>HS: populated blocks (+ events, greeting de-dup)
    HS->>HS: welcome container (screen) · attach me + shortcuts (web)
    HS-->>H: HOME document
    H-->>C: 200 JSON
```

---

## 5. Authentication modes

Mode is chosen at startup by which env var is set (precedence shown).

```mermaid
flowchart TD
    R[Inbound request] --> Q{AUTH_OPAQUE_EXCHANGE_URL set?}
    Q -- yes --> O[Opaque mode]
    O --> O1[read session cookie 'SessionId']
    O1 --> O2[GET /v2/auth/exchange-token<br/>x-brand-id=brand]
    O2 --> O3[decodeAccessToken:<br/>profileID=prn · loggedIn=!isAnonymous]

    Q -- no --> Q2{AUTH_JWKS_URL set?}
    Q2 -- yes --> J[JWT mode]
    J --> J1[Bearer token]
    J1 --> J2[verify RS256 via JWKS<br/>check iss/aud/exp]
    J2 --> J3[profileID=AUTH_PROFILE_CLAIM]

    Q2 -- no --> Dv[Dev mode<br/>x-profile-id header]

    O3 --> RI[RequestInfo: ProfileID, LoggedIn, Claims]
    J3 --> RI
    Dv --> RI
```

Invalid/absent credentials → **anonymous** request (HOME still served; personalization off).
Claims feed the `me` projection. See [decisions.md](decisions.md) D8.

---

## 6. Populate framework & feature-flag gating

Each block is routed to the strategy that `Supports()` it; a strategy may keep, mutate,
or **drop** the block (drop-on-failure). Flags come from the GLOBAL CMS entry, merged into
the per-request context.

```mermaid
flowchart TD
    BL[normalized blocks] --> RG{registry: which strategy Supports block?}
    RG --> S1[container / countdown<br/>deterministic]
    RG --> S2[product_list-groupby · recently_viewed · banner_products<br/>flag: groupby]
    RG --> S3[products_cards · recommendation · product_list-salesforce<br/>flag: salesforce]
    RG --> S4[jewel<br/>flags: jewel + personalization]
    RG --> S5[container_guest · container_shortcuts · container_greeting<br/>flag: personalization]
    S1 & S2 & S3 & S4 & S5 --> K{flag on AND backend configured?}
    K -- no --> DROP[drop block]
    K -- yes --> KEEP[populate block]
    KEEP --> EV[events: index + selected_store] --> OUT[final blocks]
    DROP --> OUT
```

Flag flow: `GLOBAL.feature_flags → mergeFeatureFlags (env ∧ CMS for personalization) →
withFlags → ctx → ri.Flag()`. Full table in [configuration.md](configuration.md) §4.

---

## 7. Actors & use cases

```mermaid
flowchart LR
    anon([Anonymous user])
    user([Logged-in user])
    ops([Ops / SRE])
    cms([Content editor])

    subgraph ms_home
      uc1((Get HOME page/screen))
      uc2((Personalized blocks:<br/>greeting · shortcuts · me))
      uc3((Product carousels:<br/>groupby · jewel · salesforce))
      uc4((Health / version probe))
      uc5((Toggle features at runtime))
    end

    anon --> uc1
    anon --> uc3
    user --> uc1
    user --> uc2
    user --> uc3
    ops --> uc4
    cms --> uc5
    uc5 -. CMS feature_flags .-> uc2
    uc5 -. CMS feature_flags .-> uc3
```

---

## Rules honored
- Domain imports no infrastructure; SDK/HTTP never leak past adapters.
- Per-request state via `context.Context` (`domain.RequestInfo` / `RequestState`); no global mutable state.
- Stateless, Cloud Run friendly; stdlib + vetted deps (`golang-jwt/jwt/v5`, OpenTelemetry).

See [configuration.md](configuration.md), [decisions.md](decisions.md),
[current-state.md](current-state.md), [rollout.md](rollout.md).
