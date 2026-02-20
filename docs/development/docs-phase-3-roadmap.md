# 🛣️ Docs Phase 3 Roadmap

> Last updated: 2026-02-20

This roadmap captures next-step documentation improvements after the `v0.7.0` release prep
and Phase 2 operator hardening updates.

## Objectives

- Reduce time-to-troubleshoot for operators
- Improve contract clarity for API consumers
- Lower long-term documentation drift risk

## Quick Wins (same-day)

1. Add an API contracts hub page (`docs/api/contracts.md`) and link from all endpoint pages
2. Add task-based operator navigation (`deploy`, `debug auth`, `debug 429`, `rotate keys`) in runbook index
3. Add negative examples in `docs/examples/*` for common `401/403/422/429` paths
4. Add glossary backlinks for core terms in security and operations pages

## Medium Scope (1 PR)

1. Create release cut companion checklist covering non-doc release actions:
   - tag creation validation
   - image publish and pull verification
   - rollback artifact verification
2. Add a docs ownership matrix by domain:
   - API docs ownership
   - operations/runbook ownership
   - release docs ownership

## Deeper Scope (1-2 PRs)

1. Build a canonical API contract invariants page with explicit guarantees:
   - ciphertext input/output contracts
   - error response structure guarantees
   - versioning and compatibility expectations
2. Add cross-page consistency guards for contract terms (light static checks)

## Suggested PR Breakdown

1. **PR A (Quick wins):** contracts hub + operator task nav + negative examples
2. **PR B (Governance):** release cut companion checklist + docs ownership matrix
3. **PR C (Contracts hardening):** invariants page + consistency checks

## Definition of Done (Phase 3)

- All endpoint docs link to shared contracts page
- Runbook index includes task-based operator entry points
- Examples include at least one negative flow per major API area
- Release docs include non-doc release validation links
- Docs ownership matrix is published and linked from docs architecture map

## Prioritized Backlog (S/M/L + dependencies)

| Priority | Initiative | Effort | Dependencies | Why now |
| --- | --- | --- | --- | --- |
| P0 | Docs decision tree for incident triage (`401/403/429/5xx`) | S | none | Fastest operator navigation win during incidents |
| P0 | First 15 minutes incident playbook with copy/paste commands | S | none | Reduces on-call ambiguity and response time |
| P1 | OpenAPI-to-doc coverage guard (endpoint reference consistency) | M | stable endpoint docs links | Prevents contract docs drift over time |
| P1 | Example parity checks across curl/python/js/go | M | examples folder conventions | Keeps multi-language guidance consistent |
| P2 | Docs ownership metadata (owner + review cadence) | S | team ownership agreement | Improves freshness accountability |
| P2 | Release audience diff pages (users/operators/security) | M | release template updates | Speeds release communication and change impact review |

## Suggested execution order

1. Deliver P0 items together in one quick PR
2. Deliver P1 checks with CI integration in a second PR
3. Deliver P2 governance/reporting items in a third PR

## Risks and mitigations

- Risk: extra static checks create noisy CI failures
  - Mitigation: start in warning mode locally, then enforce in CI after one release cycle
- Risk: ownership metadata becomes stale
  - Mitigation: include ownership review in release checklist cadence
- Risk: endpoint-doc mapping false positives for grouped docs pages
  - Mitigation: allow mapping config file for intentional many-to-one endpoint coverage

## Validation

Run before merge:

```bash
make docs-lint
make docs-check-examples
make docs-check-metadata
make docs-check-release-tags
```

## See also

- [Docs architecture map](docs-architecture-map.md)
- [Docs release checklist](docs-release-checklist.md)
- [Operator runbook index](../operations/runbook-index.md)
