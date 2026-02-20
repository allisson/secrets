# 🔁 Postmortem to Docs Feedback Loop

> Last updated: 2026-02-20

Use this process to ensure incidents continuously improve operational documentation.

## Policy

For every Sev incident, include one of the following outcomes in the postmortem:

1. Docs updated in the same remediation PR
2. Explicit note: "No documentation change needed" with rationale

## Required fields in postmortem

- Runbook used first
- Time to first useful doc reference
- Missing/ambiguous docs sections
- Docs updates created (path + PR link)

## Minimal workflow

1. Incident is resolved
2. Owner identifies doc gaps from timeline
3. Patch docs or record no-change rationale
4. Update `docs/CHANGELOG.md` if docs changed
5. Confirm docs checks pass before merge

## Suggested SLA

- Sev 1-2 incidents: docs follow-up within 2 business days
- Sev 3 incidents: docs follow-up within 5 business days

## See also

- [Failure playbooks](../operations/failure-playbooks.md)
- [Incident decision tree](../operations/incident-decision-tree.md)
- [Docs quality KPIs](docs-quality-kpis.md)
