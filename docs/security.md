# Security

## Threat model summary

| Threat | Control |
|---|---|
| Injection via `{blockType}` | Allowlist in `domain.IsAllowedResolveType()` — validated before any downstream call |
| Log injection via correlation ID | Regex `^[a-zA-Z0-9\-_]{1,64}$` validated before storing in context |
| SSRF via content-service URL | Host always from `CONTENT_SERVICE_URL` config; user input selects only allowlisted path segments |
| Secret leakage in logs/traces | `pkg/httpx.MaskSensitiveHeaders` masks Authorization, Cookie, Set-Cookie, x-api-key, x-auth-token, x-access-token |
| Internal detail in error responses | `writeError` serializes only `Code`, `Message`, `Retryable` — never `Detail` or `Cause` |
| Arbitrary header forwarding | content-service adapter forwards only `x-brand-id` and `Accept`; block handlers forward only locale/brand/channel |
| XSS via JSON content-type | All responses set `Content-Type: application/json` + `X-Content-Type-Options: nosniff` |
| Clickjacking | `X-Frame-Options: DENY` on all responses |
| Path traversal | Go's HTTP stack path-cleans URLs before routing; Chi never receives `../` segments |
| Oversized requests | `ReadTimeout: 10s` on the HTTP server; `LimitReader` (4 MB) on outbound response bodies |
| Privilege escalation at runtime | Distroless nonroot image (uid 65532); no extra Linux capabilities; read-only filesystem |
| Dependency vulnerabilities | `go.sum` pins all modules; run `govulncheck ./...` in CI; image scanning in Artifact Registry |

## Input validation

- `locale`: must match `^[a-z]{2}-[a-z]{2}$` (validated in content-service client before URL build)
- `brand` (x-brand-id): must match `^[A-Z0-9]{1,20}$`
- `channel`: allowlist (`web`, `mobile`, `pocket`)
- `{blockType}`: allowlist of 7 known types (see `domain.IsAllowedResolveType`)
- `x-correlation-id`: `^[a-zA-Z0-9\-_]{1,64}$`

Any input failing validation returns `400 BAD_REQUEST` without reaching the downstream.

## Secret management

All secrets (`CONTENT_SERVICE_URL`, downstream credentials) are injected via Cloud Run Secret Manager references — never hardcoded, never committed to source control, never logged.

## Cache safety

Only static blocks (no session dependency) are candidates for caching. Cache keys must include `brand + locale + channel + preview` to prevent cross-brand or cross-user content poisoning.

## Security review checklist

Before the first production PR:
- [ ] Run `/security-review` skill on implementation
- [ ] Run `govulncheck ./...` clean
- [ ] Confirm no secrets in `git log` / `.env` files committed
- [ ] Verify Cloud Run SA has only required IAM roles (least-privilege)
- [ ] Validate OTel attribute masking if enabling tracing in production
