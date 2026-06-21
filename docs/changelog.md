# Changelog

All notable changes to `ms_home_liverpool` are documented here.
Format: [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

---

## [Unreleased]

### Added
- `GET /home` — ordered page layout with static blocks inline and dynamic blocks as placeholders.
- `GET /home/blocks/{blockType}` — per-block resolve endpoint for 7 dynamic block types (stub resolvers; real adapters TBD per block).
- `GET /healthz` / `GET /readyz` health probes.
- Hexagonal architecture: domain, application, and adapter layers with clear dependency boundaries.
- `pkg/breaker` — generic circuit breaker wrapper (sony/gobreaker v2); no retries; 5% failure threshold.
- `pkg/httpx` — keep-alive HTTP client, sensitive header masking, debug cURL builder.
- `pkg/logger` — slog JSON structured logger with runtime-configurable level.
- `pkg/observability` — OpenTelemetry OTLP HTTP init with ParentBased+TraceIDRatioBased sampler.
- `AppError` centralized error model with consumer-safe responses.
- `BLOCK_TEMPORARILY_DISABLED` (423) and `SERVICE_UNAVAILABLE` (503) error codes.
- Content-service proxy adapter with breaker, SSRF-safe URL construction, and 4 MB response limit.
- Block normalization: key-wrapper unwrap, `container_grid` flattening, `tabs_container` passthrough.
- Static vs dynamic block classification by type, `source_of_data`, and `handle`.
- Multi-stage distroless Dockerfile (non-root, CGO_ENABLED=0).
- Cloud Run service descriptor (`deployments/service.yaml`).
- Full test suite: breaker, domain, classification, normalization, client, handler tests.
- Security: input allowlisting, log-injection prevention, SSRF prevention, header masking, no internal detail in responses.
