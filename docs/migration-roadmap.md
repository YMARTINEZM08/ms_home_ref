# Migration Roadmap

Evolutionary, never big-bang. Each phase is independently deployable and reversible
(gateway route flag / per-request personalization flag → instant rollback to digital_bff).

- **Phase 0 — Scaffold ✅.** Hexagonal skeleton, config, httpclient (cURL@debug, masking),
  slog, health, graceful shutdown, CS-proxy adapter, parallel fetch + flag merge.
- **Phase 1 — Read-only HOME ✅.** Normalization, content-type gating, populate framework;
  GroupBy (search/recs) + Jewel strategies (1a/1b). Golden-contract harness scaffolded.
- **Phase 2 — Personalization ✅.** container_guest/shortcuts/greeting, Salesforce strategies
  (2a); custom-data events + welcome container (2b); ATG favorite store + continue-buying (2c);
  `me` + Salesforce memo (2d); banner_products → 12/12 strategies (2e); service-side JWT auth (2f).
- **Phase 3 — Observability & cutover ✅ (code).** OTel tracing + parity tooling (3a);
  build-version stamp, rollout runbook, gateway-routing + Cloud Run manifests (3b).
- **Phase 4 — Cutover execution (ops).** Capture golden fixtures vs QA, shadow + canary ramp
  per [rollout.md](rollout.md), tune autoscaling/benchmarks, decommission HOME in digital_bff.

Business rules #1–#9 ported; all 12 populate strategies ported. Web/pocket shift independently.

See [rollout.md](rollout.md), [todos.md](todos.md), [risks.md](risks.md), [business-rules.md](business-rules.md).
