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

## Personalization flag rule (preserved)
Effective personalization = `PERSONALIZATION_ENABLED` (env gate) **AND** CMS
`feature_flags.personalization`. Implemented in `HomeService.mergeFeatureFlags`.
