# Docs, Changelog, And Releases

- For API or runtime behavior changes, update `docs/openapi.yaml`, examples, and relevant docs.
- For CLI command changes, update `docs/cli-commands.md` and relevant examples.
- For configuration changes, update `.env.example`, `docs/configuration.md`, and relevant deployment examples.
- For migrations, update related operations or deployment docs when behavior changes.
- Docs-only PRs must update `CHANGELOG.md`.
- Releases must update root `CHANGELOG.md` and `version` in `cmd/app/main.go`.
- Keep docs concise and reference-oriented: prefer bullets, tables, and centralized examples over long prose.
