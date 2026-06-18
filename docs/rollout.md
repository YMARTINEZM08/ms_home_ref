# Rollout & Cutover Runbook

Goal: shift HOME traffic from `digital_bff` to `ms_home` gradually, with objective gates
and instant rollback. The migration plan requires every step to be independently
deployable and reversible (no data migration → rollback is stateless).

## Routing mechanism (pick one)
The HOME path (`GET /content/{page|screen}/{locale}/...`) is split between the two
backends at the edge. Recommended options, most-isolated first:

1. **Edge load balancer weighted backends (recommended)** — e.g. GCLB URL map with two
   backend services (`digital-bff`, `ms-home`) and a traffic weight on the HOME route.
   App-agnostic; weight is the single rollback lever. See
   [deployments/gateway-routing.example.yaml](../deployments/gateway-routing.example.yaml).
2. **API gateway route** (Apigee/etc.) — a route rule on the HOME path with a percentage
   split + a kill-switch flag.
3. **digital_bff proxy flag** — digital_bff's HOME controller proxies to `ms_home` for a
   configurable %/cohort. Keeps control in one codebase but couples rollback to a deploy.

Identify which backend served a response via `GET /healthz` → `{"status":"ok","version":"<rev>"}`
(set `BUILD_VERSION`) and the `service.version` trace attribute.

## Auth mode (set before traffic)
For digital_bff parity, run **opaque mode**: set `AUTH_OPAQUE_EXCHANGE_URL` to the Auth
service base (cookie `AUTH_COOKIE_NAME`, default `SessionId`). ms_home then exchanges the
session cookie exactly like the incumbent — no JWT conversion needed. (Alternative: local
JWT mode via `AUTH_JWKS_URL` if an upstream already issues JWTs.) Verify a logged-in request
authenticates end-to-end against the real Auth service in shadow. See [decisions.md](decisions.md) D8.

## Pre-cutover gates (must pass before any user traffic)
- **Golden-contract parity**: `scripts/capture-fixtures.sh` against QA, then
  `go test ./test/contract -run HomeParity` = 0 diffs across web/pocket × anon/preview/logged.
- **Shadow / mirror** HOME traffic to `ms_home` (responses discarded); compare latency and
  error rate to `digital_bff` for ≥24h. No 5xx regressions, p95 ≤ baseline.
- Auth verified against the real IdP (valid/expired/invalid → expected behavior).
- Dashboards live: request rate, error rate, p50/p95/p99, outbound span latency per
  dependency (Content Service, GroupBy, Salesforce, ATG, Search Facade).

## Canary stages
Advance only when the gate holds for the dwell time; otherwise roll back.

| Stage | Weight to ms_home | Dwell | Promote when |
|---|---|---|---|
| Shadow | 0% (mirror) | 24h | parity + no error/latency regression |
| Canary | 1% | 2h | error rate ≤ baseline, no new 5xx classes |
| Ramp | 5% → 25% → 50% | 2–4h each | gates hold; spot-check rendered HOME |
| Full | 100% | 24h | stable; then decommission HOME in digital_bff |

Segment the canary by surface if useful (web before pocket) — they shift independently.

## Rollback (instant)
Set the HOME weight for `ms_home` back to **0%** (or flip the kill-switch flag). No
redeploy of `ms_home` is needed; no data to revert. Confirm via `/healthz` version on the
served traffic. Capture the failing request + trace id before rolling forward again.

## Gate signals & thresholds (tune per SLO)
- Error rate: `ms_home` 5xx ≤ `digital_bff` 5xx (no new error classes).
- Latency: p95 ≤ `digital_bff` p95 + 10%.
- Downstream: no spike in Content Service / provider span error ratios.
- Logs: no unexpected ERROR-level entries (secrets must stay masked).

## Cloud Run notes
See [deployments/cloudrun.yaml](../deployments/cloudrun.yaml): `minScale` ≥ 1 during ramp to
avoid cold starts on the canary; `containerConcurrency` 80; CPU 1 / 256Mi (tune from load
test); startup probe on `/readyz`. Keep both services deployed until 100% + soak completes.
