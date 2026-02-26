# 🤝 Documentation Contributing Guide

> Last updated: 2026-02-26

Use this guide when adding or editing project documentation.

## Table of Contents

- [Scope and Structure](#scope-and-structure)
- [Writing Style](#writing-style)
- [Technical Accuracy](#technical-accuracy)
- [Breaking vs Non-Breaking Docs Changes](#breaking-vs-non-breaking-docs-changes)
- [Security Messaging](#security-messaging)
- [Examples](#examples)
- [Metadata Source of Truth](#metadata-source-of-truth)
- [Local Docs Checks](#local-docs-checks)
- [PR Checklist](#pr-checklist)
- [Docs QA Checklist](#docs-qa-checklist)
- [Feature PR Docs Consistency Checklist](#feature-pr-docs-consistency-checklist)
- [Ownership and Review Cadence](#ownership-and-review-cadence)
- [Docs Release Process](#docs-release-process)
- [Release PR Docs QA Guard](#release-pr-docs-qa-guard)
- [Development and Testing](#development-and-testing)
- [Docs Architecture Map](#docs-architecture-map)
- [Docs Release Checklist](#docs-release-checklist)
- [Documentation Management](#documentation-management)
- [See Also](#see-also)

## Scope and Structure

- Keep root `README.md` concise and navigational
- Place detailed content under `docs/` in focused files
- Prefer adding a new focused page over extending a large page

## Writing Style

- Use short, direct sentences
- Use active voice and practical wording
- Keep headings in Title Case
- Use emojis for scanability, but keep usage moderate
- Keep list items concise and without trailing periods

Documentation style baseline:

- Prefer short sections with clear headings over long uninterrupted blocks
- Prefer plain bullet lists and tables over heavily emphasized text blocks
- Keep cross-links clickable (Markdown links) rather than inline code path references
- Keep operational steps copy/paste-ready and include expected status/result when useful

## Technical Accuracy

- Match implemented API paths exactly (`/v1/*`)
- Use capabilities consistently: `read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`
- Include `Last updated` metadata in new docs
- For API docs, include `Applies to: API v1`

## Breaking vs Non-Breaking Docs Changes

- Treat endpoint path changes, request/response contract changes, and status code behavior changes as breaking docs updates
- Breaking docs updates must include: updated API page, updated examples, and `releases/RELEASES.md` entry
- Treat wording clarifications, formatting, and cross-links as non-breaking docs updates
- Non-breaking docs updates should still run `make docs-lint` and keep links accurate

## Security Messaging

- Use this exact warning where base64 appears:
  - `⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.`
- Avoid security claims not backed by implementation
- Do not reintroduce removed features into docs

## Examples

- Prefer copy/paste-ready examples
- Include expected status/result where useful
- Avoid placeholder values that look like real secrets

Copy-safe examples policy:

- Use clearly synthetic values (`<client-id>`, `tok_sample`, `example.com`)
- Never include real keys, tokens, credentials, or production hostnames
- For sensitive domains (payments/PII), prefer redacted fragments (for example `last_four`)

## Metadata Source of Truth

- `docs/metadata.json` is the canonical source for current release and API version labels
- Keep `README.md`, `docs/README.md`, and API applies-to markers aligned with this file
- Validate with `make docs-check-metadata`

## Local Docs Checks

Run the same style/link checks locally before opening a PR:

```bash
make docs-lint
make docs-check-examples
make docs-check-metadata
```

This target runs markdown linting and offline markdown link validation.

`make docs-check-examples` validates representative JSON response shapes used in docs.

`make docs-check-metadata` validates release/API metadata alignment across docs entry points.

Optional strict freshness check for changed files:

```bash
DOCS_CHANGED_FILES="docs/api/auth/clients.md docs/api/policies.md" make docs-check-metadata
```

When `DOCS_CHANGED_FILES` is set, changed docs pages must refresh `Last updated` to
`docs/metadata.json:last_docs_refresh` (excluding `docs/adr/*` and `docs/releases/*`).

## PR Checklist

1. Links are valid and relative paths resolve
2. API examples reflect current behavior
3. Security warnings are present where needed
4. Terminology is consistent across files
5. `releases/RELEASES.md` updated for significant documentation changes

## Docs QA Checklist

1. Capability and endpoint mappings are consistent across `docs/api/*.md`
2. Route-shape (`404`) and policy-shape (`403`) behavior is validated for authorization changes
3. Release links and current release references match `docs/metadata.json`
4. `Last updated` markers are refreshed in changed docs pages
5. `make docs-lint` passes locally

## Feature PR Docs Consistency Checklist

For behavior changes, update all relevant docs in the same PR:

1. API endpoint page (`docs/api/<area>.md`) plus capability mapping references
2. OpenAPI contract updates (`docs/openapi.yaml`) for new/changed request and response shapes
3. Examples parity (`docs/examples/*.md`) for at least curl and one SDK/runtime path
4. Monitoring/query updates (`docs/operations/observability/monitoring.md`) when new operations/metrics are introduced
5. Runbook updates (`docs/operations/*.md` or `docs/operations/troubleshooting/index.md`) for incident impact
6. Release notes and changelog (consolidated in `releases/RELEASES.md`)
7. Entry-point navigation updates (`README.md`, `docs/README.md`) when docs scope expands

## Ownership and Review Cadence

- Docs owners: project maintainers and reviewers for touched domain (`api`, `operations`, `security`)
- Every functional change PR should include corresponding docs updates when behavior changes
- Perform a monthly docs review for stale examples, outdated commands, and dead links
- During releases, verify `Last updated` metadata and append entries to `releases/RELEASES.md`

Incident feedback policy:

- For Sev incidents, apply the [Postmortem to docs feedback loop](#postmortem-feedback-loop)
- Incident remediations should either update docs or record explicit no-doc-change rationale

Quality KPIs:

- Track baseline docs quality via [Docs quality KPIs](#quality-kpis)

## Docs Release Process

1. Update `Last updated` in every changed docs file
2. Update `docs/metadata.json` when release/API labels change
3. Add or update relevant examples if behavior/commands changed
4. Append a concise entry in `releases/RELEASES.md` for significant docs changes
5. Run `make docs-lint` before opening or merging PRs

## Release PR Docs QA Guard

CI includes an API/docs guard for pull requests:

- If API-facing code changes (`internal/*/http/*.go`, `cmd/app/commands/*.go`, `migrations/*`),
  PRs must include corresponding docs changes in at least one relevant docs area
- This guard helps ensure API/runtime changes ship with docs, examples, and/or runbook updates

## Development and Testing

### Useful Commands

```bash
make build
make run-server
make run-migrate
make lint
make test
make test-with-db
make mocks
make docs-check-examples
```

### Run Specific Tests

```bash
go test -v -race -run TestKekUseCase_Create ./internal/crypto/usecase
go test -v -race -run "TestKekUseCase_Create/Success" ./internal/crypto/usecase
```

### Test Databases

```bash
make test-db-up
make test
make test-db-down
```

### Local Development Loop

1. Update code
2. Run `make lint`
3. Run targeted tests
4. Run full `make test`

## Docs Architecture Map

This section defines canonical vs supporting docs to reduce duplication and drift.

### Canonical Sources

| Topic | Canonical document |
| --- | --- |
| Release and API label metadata | `docs/metadata.json` |
| API contract subset | `docs/openapi.yaml` |
| Capability-to-endpoint mapping | `docs/api/fundamentals.md#capability-matrix` |
| Authorization path matcher semantics | `docs/api/auth/policies.md` (see [ADR 0003](adr/0003-capability-based-authorization-model.md)) |
| Rate limiting strategy | `docs/api/fundamentals.md#rate-limiting` (see [ADR 0006](adr/0006-dual-scope-rate-limiting-strategy.md)) |
| API versioning approach | `docs/api/fundamentals.md#compatibility-and-versioning-policy` (see [ADR 0007](adr/0007-path-based-api-versioning.md)) |
| Database support | `docs/configuration.md#database-configuration` (see [ADR 0004](adr/0004-dual-database-support.md)) |
| Transaction management | `docs/concepts/architecture.md` (see [ADR 0005](adr/0005-context-based-transaction-management.md)) |
| Runtime env configuration | `docs/configuration.md` |
| Production security posture | `docs/operations/deployment/docker-hardened.md` |
| Release narrative | `docs/releases/vX.Y.Z.md` |
| Architectural decisions | `docs/adr/*.md` |

### Supporting Documents

| Area | Supporting docs |
| --- | --- |
| Onboarding | `docs/getting-started/*.md` |
| Endpoint behavior details | `docs/api/*.md` |
| Operations runbooks | `docs/operations/*.md` |
| Integration snippets | `docs/examples/*.md` |
| Docs process and governance | `docs/contributing.md` |

### Sync Rules

1. Update canonical source first
2. Propagate essential deltas to supporting docs
3. Update `CHANGELOG.md` for significant docs updates
4. Run docs checks before merge

Recommended local validation:

- `make docs-lint`
- `make docs-check-metadata`
- `make docs-check-release-tags`

### CI/Tooling Guards

- `docs/tools/check_docs_metadata.py`: release/API metadata and `Last updated` consistency
- `docs/tools/check_release_docs_links.py`: release docs link integrity in PRs
- `docs/tools/check_example_shapes.py`: JSON example structure sanity checks
- `docs/tools/check_release_image_tags.py`: pinned current-release Docker tag consistency

### Drift Signals

- Endpoint docs disagree with capability matrix
- Release references disagree with `docs/metadata.json`
- Examples use old response/error semantics
- Troubleshooting behavior diverges from runbooks

## Docs Release Checklist

Use this checklist for each release (`vX.Y.Z`) to keep docs consistent and navigable.

### 1) Metadata and Release Labels

- Update `docs/metadata.json`:
  - `current_release`
  - `last_docs_refresh`
- Ensure `README.md` and `docs/README.md` reflect the same current release

### 2) Release Pages

- Add release notes: `docs/releases/vX.Y.Z.md`
- Start from templates:
  - `docs/releases/_template.md`
- Promote new release links in docs indexes and operator runbooks

### 3) API Contract and Examples

- Update endpoint docs under `docs/api/*.md` for behavior/status changes
- Update `docs/openapi.yaml` for request/response changes
- Include `429` + `Retry-After` contract where protected routes can throttle
- Update at least curl plus one SDK/runtime example (`python`, `javascript`, or `go`)

### 4) Operations and Runbooks

- Update `docs/getting-started/*` for default/config changes
- Update `docs/operations/troubleshooting/index.md` for new failure modes
- Update `docs/operations/*` guidance for production impact

### 5) Changelogs and Navigation

- Update project changelog (`releases/RELEASES.md`) for release behavior and docs changes
- Verify links from:
  - `README.md`
  - `docs/README.md`
  - `docs/operations/runbooks/README.md`

#### Docker Tag Consistency Rule

- Use unpinned image tag (`allisson/secrets`) in all documentation for simplicity and to avoid repeated version updates.
- Historical note: Prior to v0.8.0, pinned tags (`allisson/secrets:vX.Y.Z`) were used. This was changed to reduce maintenance overhead.
- Ensure Docker image reference consistency guard passes (`docs/tools/check_release_image_tags.py`).
- The validation script allows either current pinned tags or unpinned references, but flags outdated version pins.

### 6) Validation Before Merge

Run:

```bash
make docs-lint
make docs-check-examples
make docs-check-metadata
make docs-check-release-tags
```

CI should also validate:

- markdown lint and link checks
- docs metadata consistency
- OpenAPI validity
- release docs link guard for new `docs/releases/vX.Y.Z.md` additions

## Documentation Management

This section consolidates documentation quality, incident feedback loops, and backlog management into one operational guide.

### Quality KPIs

Use these KPIs to track documentation reliability and operational usefulness.

#### Core KPIs

| KPI | Target | Source |
| --- | --- | --- |
| Docs lint/link pass rate | 100% on main and PRs | CI (`make docs-lint`) |
| Stale high-risk pages (API/ops/getting-started) | 0 pages older than SLA | freshness check (Phase 4 PR 1) |
| Incident triage time-to-first-runbook | <= 5 minutes | on-call postmortems |
| Docs-related incident follow-up completion | 100% for Sev incidents | incident action tracker |
| Broken internal anchor count | 0 | anchor guard (Phase 4 PR 2) |

#### Review Cadence

- Weekly: CI quality metrics (lint/link/check failures)
- Monthly: freshness + ownership review
- After Sev incidents: triage path clarity and runbook updates

#### Escalation Triggers

- Repeated docs-check CI failures for 2+ weeks
- 2+ incidents in a month citing missing/unclear docs guidance
- Freshness SLA misses in API/operations docs

### Postmortem Feedback Loop

Use this process to ensure incidents continuously improve operational documentation.

#### Policy

For every Sev incident, include one of the following outcomes in the postmortem:

1. Docs updated in the same remediation PR
2. Explicit note: "No documentation change needed" with rationale

#### Required Fields in Postmortem

- Runbook used first
- Time to first useful doc reference
- Missing/ambiguous docs sections
- Docs updates created (path + PR link)

#### Minimal Workflow

1. Incident is resolved
2. Owner identifies doc gaps from timeline
3. Patch docs or record no-change rationale
4. Update `releases/RELEASES.md` if docs changed
5. Confirm docs checks pass before merge

#### Suggested SLA

- Sev 1-2 incidents: docs follow-up within 2 business days
- Sev 3 incidents: docs follow-up within 5 business days

### Master Backlog

This section consolidates all documentation improvement initiatives into one prioritized execution sequence.

#### P0 (Immediate)

| Item | Effort | Dependency |
| --- | --- | --- |
| Incident decision tree and first-15-minutes playbook | S | none |
| Operator/developer day-0 walkthrough paths | S | none |
| Known limitations page for ops/security expectations | S | none |

#### P1 (Near-term)

| Item | Effort | Dependency |
| --- | --- | --- |
| Freshness SLA check + CI | M | policy alignment |
| Internal anchor integrity check + CI | M | docs tooling baseline |
| OpenAPI-to-doc coverage guard | M | endpoint mapping config |
| Example parity checks across runtimes | M | examples conventions |

#### P2 (Governance)

| Item | Effort | Dependency |
| --- | --- | --- |
| Docs ownership matrix and review cadence page | S | team owner mapping |
| Postmortem-to-doc feedback loop policy | S | incident process agreement |
| Docs KPI reporting page and monthly review process | S | CI metrics visibility |

#### P3 (Maturity)

| Item | Effort | Dependency |
| --- | --- | --- |
| API contracts/invariants canonical page | M | API doc harmonization |
| Release audience diff summaries (users/operators/security) | M | release template update |
| Search vocabulary normalization pass | S | page owners for key docs |

#### Suggested Execution Sequence

1. Complete P0 content and navigation updates
2. Implement P1 checks in CI with low-noise defaults
3. Formalize P2 governance and cadence
4. Deliver P3 consistency and release communication upgrades

## See also

- [Documentation index](README.md)
- [Changelog](releases/RELEASES.md)
- [Local development](getting-started/local-development.md)
- [Smoke test](getting-started/smoke-test.md)
- [Troubleshooting](operations/troubleshooting/index.md)
- [Incident response guide](operations/observability/incident-response.md)
- [API compatibility policy](api/fundamentals.md#compatibility-and-versioning-policy)
- [Production rollout golden path](operations/deployment/production-rollout.md)
