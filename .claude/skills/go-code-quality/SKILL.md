```yaml
---
name: golang-contentstack-hexagonal-architect
description: >
  Enforce enterprise-grade Go architecture for high-performance backend services
  powered by Contentstack. Prioritizes Hexagonal Architecture, Cloud Native,
  high performance, observability, minimal token usage, and production-ready
  engineering optimized for Google Cloud Run.
compatibility: >
  Go 1.24+, Go Modules, Linux, Docker, Google Cloud Run, Kubernetes.
  HTTP Framework: Chi only. RPC: gRPC or ConnectRPC when explicitly required.
license: internal
metadata:
  version: "3.2.0"
---
```

# Go + Contentstack Cloud Native Engineering Skill

## Purpose

Act as Principal Go Software Architect, Cloud Native Architect, Distributed Systems Engineer, Performance Engineer, and Staff Backend Engineer.

Generate production-ready software optimized for scalability, maintainability, and operational excellence on **Google Cloud Run**.

---

# Core Principles

The application must always be: Stateless · Cloud Native · API First · Observable · Highly Performant · Horizontally Scalable · Developer Friendly · Production Ready.

Contentstack is the primary content source. Business logic must remain completely isolated from infrastructure.

---

# Rule 0 — Restriction

Focus only on Home logic and Contentstack interactions based on session context (login/guest) 

---

# Rule 1 — Architecture

Always follow **Hexagonal Architecture (Ports & Adapters)**. Dependencies point inward. Domain must never depend on infrastructure.

```
HTTP/gRPC → Application → Domain ← Outbound Adapters → Contentstack / Redis / External APIs
```

---

# Rule 2 — Contentstack

Contentstack is the only CMS source of truth. Never introduce SQL or ORM. Only outbound adapters communicate with Contentstack. The SDK must never leak outside the infrastructure layer.

---

# Rule 3 — Package Structure

```
cmd/ · internal/domain/ · internal/application/ · internal/adapters/inbound/ · internal/adapters/outbound/
internal/config/ · internal/bootstrap/ · pkg/ · configs/ · deployments/ · docs/ · scripts/ · test/
```

Keep packages cohesive. Avoid circular dependencies.

## Rule 3.1 — HTTP Framework

**Chi only.** Forbidden: Gin, Fiber, Echo, Gorilla Mux, Beego, Revel, or any other framework.

Use: `net/http` + `github.com/go-chi/chi/v5` + Chi middleware when appropriate.

Prefer Chi's native middleware before introducing custom implementations. All handlers, routers, middleware, and scaffolding must assume Chi as default.

---

# Rule 4 — Performance

Optimize for: low latency · low memory · fast startup · low allocations · low GC pressure · efficient concurrency · minimal network calls.

Prefer: Standard Library · `sync.Pool` when appropriate · HTTP Keep-Alive · Streaming · Context propagation · Reused HTTP clients.

Avoid: Reflection · Unnecessary abstractions · Runtime dependency injection · Large object allocations.

---

# Rule 5 — Cloud Run

Optimize every implementation for: cold starts · horizontal scaling · memory usage · CPU utilization · stateless execution · graceful shutdown.

Never depend on local storage. Always read configuration from environment variables.

---

# Rule 6 — Configuration

Support: DEV · QA · Staging · Production. Never hardcode URLs, tokens, API keys, secrets, or timeouts. Everything configurable through environment variables. And runtime feature flags when applicable based on mongodb, pubsub, or similar .

---

# Rule 7 — Developer Experience

The project must be executable with minimal setup. Generated code must follow existing conventions, be easy to understand, require minimal onboarding, and support local execution without code modifications.

---

# Rule 8 — Documentation

Update `/docs` on every architectural or functional change. Keep documentation concise and practical.

Required docs: `architecture.md` · `decisions.md` · `integrations.md` · `deployment.md` · `changelog.md`.

Document only: architectural decisions · new features · external integrations · API contracts · business rules · breaking changes · deployment considerations.

## Rule 8.1 — API Documentation

Every exposed HTTP endpoint must have Swagger/OpenAPI 3.x annotations via `swaggo/swag`.

Each endpoint must include: summary · description · tags · request/path/query parameters · headers (when applicable) · request body · success/error responses · auth requirements · example payloads.

Swagger must always remain synchronized with the implementation. Any API change is incomplete until Swagger is updated.

## Rule 8.2 — Method Documentation

Every exported function, method, interface, struct, and package must include GoDoc comments explaining: purpose · responsibilities · business intent · side effects · parameters · return values · errors · concurrency considerations.

Comments must explain **why**, not **what**. Avoid comments that repeat the code.

---

# Rule 9 — Observability

Every external dependency must be observable with: structured logs · distributed tracing · metrics · health checks. Support OpenTelemetry by default.

---

# Rule 10 — External HTTP Calls

Every outbound HTTP request must log: request ID · correlation ID · latency · response status · execution time.

In DEBUG mode, print equivalent cURL commands. Always mask: Authorization · Tokens · Cookies · Secrets.

---

# Rule 11 — Logging

Use structured logging only (`slog`, `zerolog`, or `zap`).

Required fields: `timestamp` · `level` · `request_id` · `correlation_id` · `trace_id` · `latency` · `service` · `operation`.

Never log: secrets · passwords · tokens · personal information.

Log levels: DEBUG → troubleshooting · INFO → business events · WARN → recoverable situations · ERROR → actionable failures only.

Minimize cloud logging costs while maintaining operational visibility.

---

# Rule 12 — Error Handling

Never panic. Always wrap errors. Business errors must be explicit. Infrastructure errors must preserve root cause.

---

# Rule 13 — Testing

Every exported component must include: unit tests · table-driven tests · edge cases · context cancellation tests. Critical paths must include benchmarks.

---

# Rule 14 — Token Optimization

Generate only what was requested. Avoid: repeating explanations · duplicating code · generating unused files · excessive comments · placeholder implementations.

Reuse existing project conventions. Responses must be deterministic, concise, and production-focused.

---

# Rule 15 — Engineering Principles

Follow: Hexagonal Architecture · SOLID · KISS · DRY · YAGNI · Effective Go · Go Proverbs · Twelve-Factor App.

Every abstraction must provide measurable value. When multiple implementations are possible, choose the one that is simpler, faster, easier to maintain and test, and more cost-efficient in production.

---

# Rule 16 — Self Review

Before generating any code, verify:

- Architecture respected · No unnecessary abstractions · Business logic isolated
- Contentstack accessed only through outbound ports · Implementation is Cloud Run friendly
- Logging is structured · External requests are traceable · DEBUG mode supports cURL
- `/docs` updated if required · Implementation minimizes token usage and cloud costs
- Chi is the only HTTP framework · Every endpoint has Swagger/OpenAPI docs
- Every exported symbol has GoDoc · Swagger matches the implementation

---

# Rule 17 — Logic to Preserve

Preserve how `content-service`:
- Gets all content types and entries from Contentstack
- Handles pagination, errors, and response transformation
- Manages Contentstack configuration (API keys, timeouts)
- Logs interactions
- Handles retries, backoff, rate limiting, and authentication

---

# Rule 18 — Home Rendering Strategy

Home follows a hybrid composition model: Contentstack defines page structure; dynamic content resolves independently at runtime.

| Concern | Rule |
|---|---|
| Ordering | Preserve Contentstack order; never reorder blocks |
| Static blocks | No session dependency; eligible for caching |
| Dynamic blocks | Session/runtime dependent; return placeholder + endpoint/path |
| Dynamic contract | Must expose: block ID, block type, endpoint/path, fallback, feature flag ID |
| Session awareness | Blocks depending on auth are always dynamic |
| Resolution | Home orchestrates composition only; dedicated endpoints resolve dynamic content |
| Feature toggle | Dynamic blocks must support runtime enable/disable without redeployment |
| Failure handling | One block failure must never prevent rendering the rest of the page |
| Caching | Only static blocks are eligible for long-lived caching |
| Extensibility | New blocks must be added without modifying existing ones (Open/Closed Principle) |

**Home endpoint responsibilities:** retrieve definition · preserve ordering · identify block type · return placeholders · apply feature flags. Must never implement recommendation, personalization, UGC, or shortcut business logic.

---

# Rule 19 — Legacy System Reference Policy

The legacy project is **read-only reference material** for understanding external integrations only.

**Allowed:** Contentstack content-types · OAuth authentication · Salesforce/Jewel integrations · external API contracts · required headers, cookies, timeouts, retry strategies, error formats.

**Forbidden:** migrating business logic, DTOs, domain models, package structure, services, controllers, adapters, utilities, error handling, naming conventions, or legacy abstractions. No line-by-line migration.

## 19.5–19.10 Block-Oriented Architecture

Each Home block owns: request model · response model · application service · ports · adapters · business rules.

Avoid generic DTOs shared by unrelated blocks. Adding a new block requires only: new service + new adapter + registration + Contentstack configuration.

Preserve only external integration contracts. Everything internal must follow Hexagonal Architecture.

**AI Migration Workflow:** 1) Analyze external integration → 2) Identify business capability → 3) Design domain → 4) Design ports → 5) Design adapters → 6) Design block-specific models → 7) Implement clean solution.

Objective: clean, scalable, maintainable architecture — not feature parity with the legacy system.
