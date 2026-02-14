# 🗒️ Documentation Changelog

> Last updated: 2026-02-14

## 2026-02-14 (docs v3 - v0.2.0 release prep)

- Added `clean-audit-logs` command documentation with dry-run and JSON/text output examples
- Added audit-log retention cleanup runbook to production operations guide
- Clarified audit log retention is a CLI cleanup workflow, while API remains list/query (`GET /v1/audit-logs`)
- Updated pinned Docker image tags and release references from `v0.1.0` to `v0.2.0`
- Added release notes page: `docs/releases/v0.2.0.md` and kept `v0.1.0` as historical

## 2026-02-14 (docs v2 - v0.1.0 release prep)

- Added first-client bootstrap flow to Docker and local development guides using `create-client`
- Added CLI reference page with runtime, key management, and client management commands
- Linked CLI docs and release notes from root README and docs index
- Switched Docker release guide examples to pinned image tag `allisson/secrets:v0.1.0`
- Added explicit OpenAPI coverage note: `docs/openapi.yaml` is baseline subset for common flows
- Clarified API v1 compatibility expectations relative to pre-1.0 app releases
- Added release notes page: `docs/releases/v0.1.0.md`

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
- Added cross-linking across all docs pages via `See also` sections for faster navigation
- Converted docs path references in `README.md` and `docs/README.md` into clickable Markdown links
- Clarified transit decrypt contract: callers must pass versioned ciphertext (`<version>:<base64-ciphertext>`) exactly as returned by encrypt
- Documented transit decrypt validation behavior: malformed ciphertext now returns `422 Unprocessable Entity`
- Added transit decrypt input contract examples (valid/invalid) and representative `422` payloads
- Added OpenAPI decrypt request examples and explicit `401`/`403`/`404` responses
- Added 422 troubleshooting matrix and transit round-trip verification/decode notes in examples
- Clarified transit key create behavior: duplicate key names now documented as `409 Conflict` and rotate is required for new versions
- Added transit create-vs-rotate guidance, idempotency notes, endpoint error matrix, and representative `409` conflict payload examples
- Added transit automation runbook note for handling create `409` by rotating keys
- Added API status code quick-reference tables to clients, secrets, transit, and audit docs
- Added glossary page and cross-links from API/reference documentation
- Added example-page common mistakes sections (curl, python, javascript, go)
- Added docs contribution policy for breaking vs non-breaking documentation updates
- Added docs freshness SLA table in docs index
- Added failure playbooks for 401/403/409 incident triage
- Added API compatibility/versioning policy page for breaking/non-breaking expectations
- Added ADRs for envelope encryption model and transit versioned ciphertext contract
- Added executable example shape checks (`make docs-check-examples`) and CI integration
- Added environment bootstrap sections to all examples pages

## See also

- [Documentation index](README.md)
- [Contributing guide](contributing.md)
- [Testing guide](development/testing.md)
