# 🗒️ Documentation Changelog

> Last updated: 2026-02-14

## 2026-02-14 (docs v1)

- Split large root README into focused docs under `docs/`
- Made Docker image the default run path (`allisson/secrets:latest`)
- Added API references, examples (curl/python/javascript/go), and operations guides
- Added restart requirement after master key/KEK rotation
- Added troubleshooting guide, policy cookbook, and production deployment guide
- Added API quick flows and API v1 applicability notes
- Added docs CI checks (markdown lint + offline link checks)
- Added Make target `make docs-lint` for local docs checks
- Added quickstart copy block and shell compatibility notes for smoke tests
- Added API error payload examples and shared response-shapes reference page
- Added policy mistakes table and rolling restart runbook guidance
- Added supported platforms section and docs ownership/review cadence guidance
- Added clickable anchor links in long-document table of contents sections
- Added first-time operator path in docs index for onboarding flow
- Added docs release process checklist in `docs/contributing.md`
- Added CI docs-only PR guard requiring `docs/CHANGELOG.md` updates
- Added pull request template with documentation quality gate checklist
- Added baseline OpenAPI spec (`docs/openapi.yaml`) and linked it from API docs
