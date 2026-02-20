# 🧾 Docs Release Checklist

> Last updated: 2026-02-20

Use this checklist for each release (`vX.Y.Z`) to keep docs consistent and navigable.

## 1) Metadata and release labels

- Update `docs/metadata.json`:
  - `current_release`
  - `last_docs_refresh`
- Ensure `README.md` and `docs/README.md` reflect the same current release

## 2) Release pages

- Add release notes: `docs/releases/vX.Y.Z.md`
- Add upgrade guide when behavior/defaults change: `docs/releases/vX.Y.Z-upgrade.md`
- Start from templates:
  - `docs/releases/_template.md`
  - `docs/releases/_upgrade-template.md`
- Update release compatibility matrix: `docs/releases/compatibility-matrix.md`
- Promote new release links in docs indexes and operator runbooks

## 3) API contract and examples

- Update endpoint docs under `docs/api/*.md` for behavior/status changes
- Update `docs/openapi.yaml` for request/response changes
- Include `429` + `Retry-After` contract where protected routes can throttle
- Update at least curl plus one SDK/runtime example (`python`, `javascript`, or `go`)

## 4) Operations and runbooks

- Update `docs/getting-started/*` for default/config changes
- Update `docs/getting-started/troubleshooting.md` for new failure modes
- Update `docs/operations/*` guidance for production impact

## 5) Changelogs and navigation

- Update project changelog (`CHANGELOG.md`) for release-level behavior
- Update docs changelog (`docs/CHANGELOG.md`) for docs scope/process updates
- Verify links from:
  - `README.md`
  - `docs/README.md`
  - `docs/operations/runbook-index.md`

### Docker tag consistency rule

- Use pinned image tags (`allisson/secrets:vX.Y.Z`) in release guides, rollout runbooks, and copy/paste commands
  intended for reproducible operations.
- Use `allisson/secrets:latest` only in explicitly marked fast-iteration/dev-only examples.
- In one document, avoid mixing pinned and `latest` tags unless the distinction is explicitly explained.
- Ensure current-release pinned tag consistency guard passes (`docs/tools/check_release_image_tags.py`).

## 6) Validation before merge

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

## See also

- [Documentation contributing guide](../contributing.md)
- [Documentation changelog](../CHANGELOG.md)
- [API compatibility policy](../api/versioning-policy.md)
- [Production rollout golden path](../operations/production-rollout.md)
