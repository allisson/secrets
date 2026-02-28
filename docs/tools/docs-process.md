# ⚙️ Documentation Process

> Last updated: 2026-02-28

This guide defines the processes for managing and releasing documentation for Secrets.

## Local Docs Checks

Run these checks locally before opening a PR:

```bash
make docs-lint
make docs-check-examples
make docs-check-metadata
```

- `make docs-lint`: Runs markdown linting and link validation.
- `make docs-check-examples`: Validates JSON response shapes.
- `make docs-check-metadata`: Validates release/API metadata alignment.

## PR Checklists

### Standard PR Checklist

1. Links are valid and relative paths resolve.
2. API examples reflect current behavior.
3. Security warnings are present where needed.
4. Terminology is consistent across files.
5. `releases/RELEASES.md` updated for significant changes.

### Feature PR Consistency Checklist

For behavior changes, update:

1. API endpoint page (`docs/api/<area>.md`).
2. OpenAPI contract (`docs/openapi.yaml`).
3. Examples parity (`docs/examples/*.md`).
4. Monitoring/query updates (`docs/operations/observability/monitoring.md`).
5. Runbook updates (`docs/operations/*.md`).
6. Release notes (`releases/RELEASES.md`).

## Docs Release Process

1. Update `Last updated` in every changed docs file.
2. Update `docs/metadata.json` when release/API labels change.
3. Add or update relevant examples.
4. Append an entry in `releases/RELEASES.md`.
5. Run `make docs-lint` before merge.

## Documentation Management

### Quality KPIs

| KPI | Target | Source |
| --- | --- | --- |
| Docs lint/link pass rate | 100% | CI (`make docs-lint`) |
| Stale high-risk pages | 0 | freshness check |
| Incident triage time | <= 5 mins | postmortems |

### Postmortem Feedback Loop

For every Sev incident:

1. Update docs in the same remediation PR OR
2. Record "No documentation change needed" with rationale.

## See Also

- [Contributing Guide](../contributing.md)
- [Documentation Style Guide](docs-style-guide.md)
