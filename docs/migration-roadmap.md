# Migration Roadmap

Evolutionary, never big-bang. Each phase is independently deployable and reversible
(gateway route flag / per-request personalization flag → instant rollback to digital_bff).

- **Phase 0 — Scaffold ✅ (current).** Hexagonal skeleton, config, httpclient (cURL@debug,
  masking), slog, health checks, graceful shutdown, CS-proxy adapter, parallel fetch +
  flag merge, tests, Dockerfile, Cloud Run manifest. *Golden-contract harness: pending.*
- **Phase 1 — Read-only HOME content (no personalization).** Normalization, content-type
  gating, non-personalized populate strategies, GroupBy + Category Indexer adapters.
  Anonymous traffic behind a flag; diff vs golden fixtures.
- **Phase 2 — Personalization core.** Login + Middleware/Salesforce/ATG/User adapters;
  greeting/guest/shortcuts/recommendation/salesforce strategies; custom events; web
  `me`/`shortcuts`; legacy Android welcome. Logged-in canary.
- **Phase 3 — Shortcuts breadth.** Apigee/Apigee2: buy-again, wishlist. Full parity.
- **Phase 4 — Cutover & hardening.** Ramp to 100%, benchmarks, tune timeouts/pools,
  autoscaling, decommission HOME path in digital_bff.

Web ships before pocket within each phase.

See [todos.md](todos.md), [risks.md](risks.md), [business-rules.md](business-rules.md).
