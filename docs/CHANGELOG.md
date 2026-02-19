# 🗒️ Documentation Changelog

> Last updated: 2026-02-19

## 2026-02-19 (docs v9 - v0.4.1 bugfix release prep)

- Added release notes page: `docs/releases/v0.4.1.md` and promoted it as current in docs indexes
- Updated docs metadata source (`docs/metadata.json`) to `current_release: v0.4.1`
- Updated pinned Docker examples from `allisson/secrets:v0.4.0` to `allisson/secrets:v0.4.1`
- Documented policy path-matching behavior with mid-path wildcard support in `docs/api/policies.md`
- Updated troubleshooting and failure playbooks to include exact, trailing wildcard, and mid-path wildcard matching
- Corrected Clients API policy examples to use `decrypt` for `/v1/secrets/*` reads
- Added transit rotate smoke-test step for `/v1/transit/keys/*/rotate` wildcard validation
- Added malformed rotate path-shape smoke check and explicit unsupported wildcard pattern notes
- Added policy matcher quick-reference table to `docs/api/capability-matrix.md`
- Linked `v0.4.1` release notes from production and smoke-test operator guides
- Added route-shape vs policy-shape guidance and cross-links between policies and smoke tests
- Added copy-safe split-role policy snippets for transit rotate-only and secrets read/write separation
- Added operator quick checklist to `docs/releases/v0.4.1.md` and policy matcher FAQ in troubleshooting
- Added pre-deploy policy review checklist to `docs/api/policies.md`
- Added `v0.4.1` documentation migration map with direct section links for operators
- Added strict CI mode snippet for policy smoke checks and 403-vs-404 false-positive guidance
- Added canonical wildcard matcher semantics links in auth, clients, secrets, and transit API docs
- Converted Clients API related references to clickable links for navigation consistency
- Added policy triage cross-links in Audit Logs API and refreshed stale page update stamps
- Added docs metadata guard to require `> Last updated: YYYY-MM-DD` marker on all docs pages
- Added optional strict metadata freshness check via `DOCS_CHANGED_FILES` for changed docs pages
- Added Docs QA checklist and style baseline guidance to `docs/contributing.md`
- Added unified operator runbook hub: `docs/operations/runbook-index.md` and linked it from docs indexes

## 2026-02-18 (docs v8 - docs QA and operations polish)

- Added docs metadata source file `docs/metadata.json` and metadata consistency checker
- Added `make docs-check-metadata` and integrated it into `make docs-lint`
- Added CI docs metadata check and API/docs consistency guard for PRs
- Added policy verification runbook: `docs/operations/policy-smoke-tests.md`
- Added retention defaults table to production guide and linked policy smoke tests
- Added tokenization lifecycle sequence diagram in architecture docs
- Added copy-safe examples policy and release PR docs QA guard guidance in contributing docs

## 2026-02-18 (docs v7 - final v0.4.0 hardening)

- Added canonical capability reference page: `docs/api/capability-matrix.md`
- Linked capability matrix from API endpoint docs, policy cookbook, and docs indexes
- Expanded OpenAPI description and monitoring docs with route-template notes (`{name}` vs `:name`/`*path`)
- Added tokenization deterministic-mode caveats in curl, Python, JavaScript, and Go examples
- Expanded tokenization API guidance with metadata data-classification rules
- Added rollback guidance for additive tokenization schema migration in `docs/releases/v0.4.0.md`
- Added migration-focused troubleshooting for tokenization rollout and expanded smoke test coverage

## 2026-02-18 (docs v6 - v0.4.0 release prep)

- Added release notes page: `docs/releases/v0.4.0.md` and promoted it as current in docs indexes
- Updated pinned Docker examples from `allisson/secrets:v0.3.0` to `allisson/secrets:v0.4.0`
- Updated root `README.md` with `What's New in v0.4.0`, tokenization API overview, and release links
- Added tokenization endpoints and corrected request/response contracts in `docs/api/tokenization.md`
- Added tokenization CLI command docs in `docs/cli/commands.md`
- Added tokenization monitoring operations and retention workflow updates in production docs
- Added explicit OpenAPI-coverage gap notes for tokenization rollout docs
- Added tokenization snippets to Python, JavaScript, and Go examples for cross-language parity
- Added tokenization incident runbooks and policy mapping clarifications
- Added `v0.4.0` upgrade checklist (migrate, verify, tokenization smoke checks, retention cleanup)
- Expanded OpenAPI baseline with tokenization endpoint and schema coverage
- Added canonical capability matrix reference and cross-linked API docs to reduce policy drift
- Expanded smoke test script/docs with tokenization round-trip + revoke validation
- Added tokenization migration verification troubleshooting section

## 2026-02-16 (docs v5 - documentation quality improvements)

- Added `What's New in v0.3.0` section to root `README.md`
- Added Prometheus + Grafana quickstart and a metrics naming contract to `docs/operations/monitoring.md`
- Added production hardening guidance for securing `/metrics` exposure
- Added feature PR docs consistency checklist to `docs/contributing.md`
- Added metrics troubleshooting matrix to `docs/getting-started/troubleshooting.md`
- Added local and Docker command parity examples in `docs/cli/commands.md`
- Added telemetry breaking vs non-breaking examples in `docs/api/versioning-policy.md`

## 2026-02-16 (docs v4 - v0.3.0 release prep)

- Added release notes page: `docs/releases/v0.3.0.md` and set it as the current release in docs indexes
- Updated pinned Docker examples from `allisson/secrets:v0.2.0` to `allisson/secrets:v0.3.0`
- Added monitoring links to root README and expanded API overview with `GET /metrics`
- Aligned monitoring operations with implementation (`secret_create`, `secret_get_version`, `audit_log_delete`, `transit_key_rotate`)
- Clarified metrics disable behavior (`METRICS_ENABLED=false` removes metrics middleware and `/metrics` route)

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
