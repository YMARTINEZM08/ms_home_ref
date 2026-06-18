# Decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | Migrate **both** web + pocket HOME (phased) | Single owner for the HOME surface |
| D2 | Call the existing **Content Service proxy**, not Contentstack directly | Behavior parity; reuse proxy caching/transforms; defer direct CS (TODO-1) |
| D3 | **Full parity incl. personalization**, delivered incrementally behind flags | Preserve UX while migrating safely |
| D4 | **stdlib `net/http`** (1.22 ServeMux) | Skill preference; minimal deps; fast cold start |
| D5 | No third-party deps in Phase 0 | Cold-start + supply-chain hygiene; revisit if OTel SDK is added |
| D6 | Per-request `RequestInfo` in `context.Context` | Replace NestJS async-local `RequestContext`; stateless |
| D7 | `Document = map[string]any` for CMS payloads | Content is highly dynamic; avoid lossy rigid structs |
| D8 | **Auth boundary: service validates JWT** (RS256 via JWKS) | Self-contained, mirrors digital_bff in-process decode; identity from a verified Bearer token, not trusted headers |
| D9 | One dep: `golang-jwt/jwt/v5` for JWT | Vetted crypto over hand-rolled; JWKS cache stays stdlib |

## Auth boundary (D8)
When `AUTH_JWKS_URL` is set, identity comes **only** from a valid RS256 Bearer token
(`x-profile-id` is ignored); only RS256 is accepted (alg=none/HMAC rejected); issuer
and audience are validated when configured. An absent/invalid token → anonymous
request (HOME still served, personalization off). With `AUTH_JWKS_URL` empty the
service runs in **dev mode**: identity from the `x-profile-id` header. JWT claims feed
the `me` projection (claims override cart-header fields per the `@Expose` allowlist).

### Two auth modes (D8 update)
Auth mode is chosen by config (opaque > jwt > dev):
- **Opaque exchange (digital_bff parity)** — `AUTH_OPAQUE_EXCHANGE_URL` set. ms_home reads the
  session cookie (`AUTH_COOKIE_NAME`, default `SessionId`), calls
  `GET /v2/auth/exchange-token` with `x-brand-id=<brand>`, and uses `decodeAccessToken`:
  profileId = `prn`, isLoggedIn = `!isAnonymous`, claims feed `me`. Matches the incumbent's
  transport — no upstream JWT conversion needed.
- **Local JWT** — `AUTH_JWKS_URL` set. RS256 Bearer validated against JWKS; `AUTH_PROFILE_CLAIM`
  (default `prn`). Use when an upstream already issues JWTs.
- **Dev** — both empty: identity from the `x-profile-id` header.

Resolved: the earlier transport mismatch (opaque cookie vs Bearer JWT) — opaque mode now
matches digital_bff directly. Still **not ported** (out of HOME scope): per-brand Auth0 client
selection (uses `x-brand-id` as-is), CSC agent path (`x-cs-folio-id`), impersonation, and the
cookie-clear/400 on invalid token (ms_home → anonymous instead).

## Personalization flag rule (preserved)
Effective personalization = `PERSONALIZATION_ENABLED` (env gate) **AND** CMS
`feature_flags.personalization`. Implemented in `HomeService.mergeFeatureFlags`.
