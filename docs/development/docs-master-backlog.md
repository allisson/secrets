# 🗂️ Docs Master Backlog

> Last updated: 2026-02-20

This page consolidates Phase 3, Phase 4, and maturity follow-ups into one prioritized execution sequence.

## P0 (Immediate)

| Item | Effort | Dependency |
| --- | --- | --- |
| Incident decision tree and first-15-minutes playbook | S | none |
| Operator/developer day-0 walkthrough paths | S | none |
| Known limitations page for ops/security expectations | S | none |

## P1 (Near-term)

| Item | Effort | Dependency |
| --- | --- | --- |
| Freshness SLA check + CI | M | policy alignment |
| Internal anchor integrity check + CI | M | docs tooling baseline |
| OpenAPI-to-doc coverage guard | M | endpoint mapping config |
| Example parity checks across runtimes | M | examples conventions |

## P2 (Governance)

| Item | Effort | Dependency |
| --- | --- | --- |
| Docs ownership matrix and review cadence page | S | team owner mapping |
| Postmortem-to-doc feedback loop policy | S | incident process agreement |
| Docs KPI reporting page and monthly review process | S | CI metrics visibility |

## P3 (Maturity)

| Item | Effort | Dependency |
| --- | --- | --- |
| API contracts/invariants canonical page | M | API doc harmonization |
| Release audience diff summaries (users/operators/security) | M | release template update |
| Search vocabulary normalization pass | S | page owners for key docs |

## Suggested execution sequence

1. Complete P0 content and navigation updates
2. Implement P1 checks in CI with low-noise defaults
3. Formalize P2 governance and cadence
4. Deliver P3 consistency and release communication upgrades

## See also

- [Docs phase 3 roadmap](docs-phase-3-roadmap.md)
- [Docs phase 4 roadmap](docs-phase-4-roadmap.md)
- [Docs release checklist](docs-release-checklist.md)
