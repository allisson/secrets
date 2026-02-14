# 🤝 Documentation Contributing Guide

> Last updated: 2026-02-14

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

## Security Messaging

- Use this exact warning where base64 appears:
  - `⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.`
- Avoid security claims not backed by implementation
- Do not reintroduce removed features into docs

## Examples

- Prefer copy/paste-ready examples
- Include expected status/result where useful
- Avoid placeholder values that look like real secrets

## Local Docs Checks

Run the same style/link checks locally before opening a PR:

```bash
make docs-lint
```

This target runs markdown linting and offline markdown link validation.

## PR Checklist

1. Links are valid and relative paths resolve
2. API examples reflect current behavior
3. Security warnings are present where needed
4. Terminology is consistent across files
5. `docs/CHANGELOG.md` updated for significant documentation changes

## Ownership and Review Cadence

- Docs owners: project maintainers and reviewers for touched domain (`api`, `operations`, `security`)
- Every functional change PR should include corresponding docs updates when behavior changes
- Perform a monthly docs review for stale examples, outdated commands, and dead links
- During releases, verify `Last updated` metadata and append entries to `docs/CHANGELOG.md`

## Docs Release Process

1. Update `Last updated` in every changed docs file
2. Add or update relevant examples if behavior/commands changed
3. Append a concise entry in `docs/CHANGELOG.md` for significant docs changes
4. Run `make docs-lint` before opening or merging PRs
