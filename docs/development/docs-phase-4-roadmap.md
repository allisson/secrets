# 🧭 Docs Phase 4 Micro-Roadmap

> Last updated: 2026-02-20

This phase focuses on documentation process quality, freshness visibility, and guardrails.

## Scope

- Keep improvements small and enforceable
- Prefer CI-backed checks over manual reminders
- Ship in 3 focused PRs

## PR 1: Freshness SLA and Stale Page Guard

### PR 1 Goal

Detect stale docs pages before drift becomes operational risk.

### PR 1 Changes

- Add `docs/tools/check_docs_freshness.py`
- Add `make docs-check-freshness`
- Add CI step in `.github/workflows/ci.yml`
- Add freshness policy section in `docs/contributing.md`

### PR 1 Rule set (starter)

- Fail if `> Last updated:` is older than 120 days for:
  - `docs/api/*.md`
  - `docs/operations/*.md`
  - `docs/getting-started/*.md`
- Exclude historical release pages and ADR pages

## PR 2: Internal Anchor Integrity Guard

### PR 2 Goal

Catch broken section links when headings change in long docs.

### PR 2 Changes

- Add `docs/tools/check_internal_anchors.py`
- Add `make docs-check-anchors`
- Add CI step in `.github/workflows/ci.yml`
- Document anchor-link practices in `docs/development/docs-architecture-map.md`

### PR 2 Rule set (starter)

- Validate local markdown links with fragments (e.g., `file.md#section-heading`)
- Fail when target file exists but fragment no longer resolves

## PR 3: Command Validation Markers + Persona Entrypoints

### PR 3 Goal

Improve trust in copy/paste blocks and speed onboarding by audience.

### PR 3 Changes

- Add command validation markers to critical pages:
  - `docs/operations/production-rollout.md`
  - `docs/operations/production.md`
  - `docs/getting-started/troubleshooting.md`
  - `docs/getting-started/smoke-test.md`
- Add persona landing pages:
  - `docs/personas/operator.md`
  - `docs/personas/developer.md`
  - `docs/personas/security.md`
- Link persona pages from `docs/README.md`

### PR 3 Marker format (starter)

Use a compact marker above critical command blocks:

```text
> Command status: verified on YYYY-MM-DD
```

## Dependencies and Order

1. PR 1 (freshness) first
2. PR 2 (anchors) second
3. PR 3 (usability) third

## Success Criteria

- Freshness check runs in CI and fails stale high-risk pages
- Anchor check runs in CI and prevents broken section links
- Critical command blocks include validation markers
- Persona pages provide a shortest-path doc flow by role

## Validation Commands

```bash
make docs-lint
make docs-check-examples
make docs-check-metadata
make docs-check-release-tags
```

After PR 1 and PR 2:

```bash
make docs-check-freshness
make docs-check-anchors
```

## See also

- [Docs phase 3 roadmap](docs-phase-3-roadmap.md)
- [Docs release checklist](docs-release-checklist.md)
- [Docs architecture map](docs-architecture-map.md)
