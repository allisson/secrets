# 🤝 Documentation Contributing Guide

> Last updated: 2026-02-18

Use this guide when adding or editing project documentation.

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

## Technical Accuracy

- Match implemented API paths exactly (`/v1/*`)
- Use capabilities consistently: `read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`
- Include `Last updated` metadata in new docs
- For API docs, include `Applies to: API v1`

## Breaking vs Non-Breaking Docs Changes

- Treat endpoint path changes, request/response contract changes, and status code behavior changes as breaking docs updates
- Breaking docs updates must include: updated API page, updated examples, and `docs/CHANGELOG.md` entry
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

## PR Checklist

1. Links are valid and relative paths resolve
2. API examples reflect current behavior
3. Security warnings are present where needed
4. Terminology is consistent across files
5. `docs/CHANGELOG.md` updated for significant documentation changes

## Feature PR Docs Consistency Checklist

For behavior changes, update all relevant docs in the same PR:

1. API endpoint page (`docs/api/<area>.md`) plus capability mapping references
2. OpenAPI contract updates (`docs/openapi.yaml`) for new/changed request and response shapes
3. Examples parity (`docs/examples/*.md`) for at least curl and one SDK/runtime path
4. Monitoring/query updates (`docs/operations/monitoring.md`) when new operations/metrics are introduced
5. Runbook updates (`docs/operations/*.md` or `docs/getting-started/troubleshooting.md`) for incident/upgrade impact
6. Release notes and changelog (`docs/releases/vX.Y.Z.md`, `docs/CHANGELOG.md`)
7. Entry-point navigation updates (`README.md`, `docs/README.md`) when docs scope expands

## Ownership and Review Cadence

- Docs owners: project maintainers and reviewers for touched domain (`api`, `operations`, `security`)
- Every functional change PR should include corresponding docs updates when behavior changes
- Perform a monthly docs review for stale examples, outdated commands, and dead links
- During releases, verify `Last updated` metadata and append entries to `docs/CHANGELOG.md`

## Docs Release Process

1. Update `Last updated` in every changed docs file
2. Update `docs/metadata.json` when release/API labels change
3. Add or update relevant examples if behavior/commands changed
4. Append a concise entry in `docs/CHANGELOG.md` for significant docs changes
5. Run `make docs-lint` before opening or merging PRs

## Release PR Docs QA Guard

CI includes an API/docs guard for pull requests:

- If API-facing code changes (`internal/*/http/*.go`, `cmd/app/commands/*.go`, `migrations/*`),
  PRs must include corresponding docs changes in at least one relevant docs area
- This guard helps ensure API/runtime changes ship with docs, examples, and/or runbook updates

## See also

- [Documentation index](README.md)
- [Testing guide](development/testing.md)
- [Changelog](CHANGELOG.md)
- [Local development](getting-started/local-development.md)
