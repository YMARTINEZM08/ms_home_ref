# Risks

| Risk | Mitigation |
|---|---|
| Hidden coupling in digital_bff async-local `RequestContext` | Explicit `RequestInfo` via context; golden-contract diff catches divergence |
| Strategy `supports()` async side-effects | Audit each strategy before porting (TODO-2) |
| Flag-merge subtlety (env ∧ CMS) | Replicated + unit-tested truth table |
| Drop-on-failure block semantics | Port precisely; assert partial-success in tests |
| JSON key ordering / extra keys | Structural (not byte) parity; confirm client tolerance (TODO-3) |
| Content Service response envelope unknown | Adapter decodes raw map; verify (TODO-7) |
| **Auth transport mismatch**: digital_bff exchanges an opaque cookie; ms_home validates a Bearer JWT | Add upstream opaque→JWT exchange (or revisit D8); set `AUTH_PROFILE_CLAIM=prn`; verify logged-in in shadow. See decisions.md D8 / rollout.md |

## Rollback
Per-phase gateway route flag → flip back to digital_bff. Stateless, no data
migration, instant.
