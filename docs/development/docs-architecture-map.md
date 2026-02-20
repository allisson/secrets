# 🗺️ Docs Architecture Map

> Last updated: 2026-02-20

This page defines canonical vs supporting docs to reduce duplication and drift.

## Canonical Sources

| Topic | Canonical document |
| --- | --- |
| Release and API label metadata | `docs/metadata.json` |
| API contract subset | `docs/openapi.yaml` |
| Capability-to-endpoint mapping | `docs/api/capability-matrix.md` |
| Authorization path matcher semantics | `docs/api/policies.md` |
| Runtime env configuration | `docs/configuration/environment-variables.md` |
| Production security posture | `docs/operations/security-hardening.md` |
| Release narrative | `docs/releases/vX.Y.Z.md` |

## Supporting Documents

| Area | Supporting docs |
| --- | --- |
| Onboarding | `docs/getting-started/*.md` |
| Endpoint behavior details | `docs/api/*.md` |
| Operations runbooks | `docs/operations/*.md` |
| Integration snippets | `docs/examples/*.md` |
| Docs process and governance | `docs/contributing.md`, `docs/development/*.md` |

## Sync Rules

1. Update canonical source first
2. Propagate essential deltas to supporting docs
3. Update `docs/CHANGELOG.md` for significant docs updates
4. Run docs checks before merge

Recommended local validation:

- `make docs-lint`
- `make docs-check-metadata`
- `make docs-check-release-tags`

## CI/Tooling Guards

- `docs/tools/check_docs_metadata.py`: release/API metadata and `Last updated` consistency
- `docs/tools/check_release_docs_links.py`: release docs link integrity in PRs
- `docs/tools/check_example_shapes.py`: JSON example structure sanity checks
- `docs/tools/check_release_image_tags.py`: pinned current-release Docker tag consistency

## Drift Signals

- Endpoint docs disagree with capability matrix
- Release references disagree with `docs/metadata.json`
- Examples use old response/error semantics
- Troubleshooting behavior diverges from runbooks

## See also

- [Documentation contributing guide](../contributing.md)
- [Docs release checklist](docs-release-checklist.md)
- [Documentation index](../README.md)
