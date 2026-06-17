# Architecture

`ms_home` is a stateless Go service serving the **HOME** experience (web `page` +
pocket `screen`), migrated from `digital_bff`. Hexagonal (Ports & Adapters);
dependencies point inward.

```
inbound/http ──> application ──> domain
                     │
                     └─ ports (interfaces) ◄── adapters/outbound ──> Content Service proxy
```

## Layout
| Path | Responsibility |
|---|---|
| `cmd/server` | entrypoint, HTTP server, graceful shutdown |
| `internal/domain` | pure types: `ContentType`, `Document`, `RequestInfo`, errors |
| `internal/ports` | outbound interfaces (`ContentPort`) |
| `internal/application` | `HomeService` — orchestration (port of `content.service.ts`) |
| `internal/adapters/inbound/http` | handler + router (stdlib ServeMux) |
| `internal/adapters/outbound/contentservice` | Content Service proxy client |
| `internal/config` | env-driven configuration |
| `internal/observability` | slog logger (OTel tracing: TODO) |
| `internal/bootstrap` | compile-time dependency wiring |
| `pkg/httpclient` | shared HTTP client (keep-alive, logging, cURL@debug, masking) |

## Rules honored
- Domain imports no infrastructure; SDK/HTTP never leak past adapters.
- Per-request state via `context.Context` (`domain.RequestInfo`), no global mutable state.
- stdlib only — no third-party deps (fast cold start).

See [decisions.md](decisions.md), [current-state.md](current-state.md),
[target-state.md](target-state.md).
