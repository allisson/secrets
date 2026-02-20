# 📈 Docs Quality KPIs

> Last updated: 2026-02-20

Use these KPIs to track documentation reliability and operational usefulness.

## Core KPIs

| KPI | Target | Source |
| --- | --- | --- |
| Docs lint/link pass rate | 100% on main and PRs | CI (`make docs-lint`) |
| Stale high-risk pages (API/ops/getting-started) | 0 pages older than SLA | freshness check (Phase 4 PR 1) |
| Incident triage time-to-first-runbook | <= 5 minutes | on-call postmortems |
| Docs-related incident follow-up completion | 100% for Sev incidents | incident action tracker |
| Broken internal anchor count | 0 | anchor guard (Phase 4 PR 2) |

## Review cadence

- Weekly: CI quality metrics (lint/link/check failures)
- Monthly: freshness + ownership review
- After Sev incidents: triage path clarity and runbook updates

## Escalation triggers

- Repeated docs-check CI failures for 2+ weeks
- 2+ incidents in a month citing missing/unclear docs guidance
- Freshness SLA misses in API/operations docs

## See also

- [Documentation contributing guide](../contributing.md)
- [Postmortem to docs feedback loop](postmortem-doc-loop.md)
- [Docs phase 4 roadmap](docs-phase-4-roadmap.md)
