# 🗒️ Documentation Changelog

> Last updated: 2026-02-20

## 2026-02-20 (docs v13 - v0.7.0 release prep)

- Added release notes page: `docs/releases/v0.7.0.md`
- Added upgrade guide: `docs/releases/v0.7.0-upgrade.md`
- Updated docs metadata source (`docs/metadata.json`) to `current_release: v0.7.0`
- Updated root README and docs index to promote `v0.7.0` release links
- Updated compatibility matrix with `v0.6.0 -> v0.7.0` upgrade path
- Updated API docs to document token endpoint rate limiting and `POST /v1/token` `429` behavior
- Updated environment variable docs for `RATE_LIMIT_TOKEN_ENABLED`, `RATE_LIMIT_TOKEN_REQUESTS_PER_SEC`, and `RATE_LIMIT_TOKEN_BURST`
- Updated troubleshooting and security hardening docs with token endpoint throttling guidance
- Updated pinned Docker image examples from `allisson/secrets:v0.6.0` to `allisson/secrets:v0.7.0`
- Added token endpoint throttling runbook section to production deployment guide
- Added token-endpoint-specific `429` response example and optional smoke test verification flow
- Expanded monitoring queries and alert starters for `/v1/token` throttling signals
- Added docs CI guard for current-release pinned image tag consistency
- Added operator quick card runbook (`docs/operations/operator-quick-card.md`) for rollout/incident triage
- Added trusted proxy reference guide (`docs/operations/trusted-proxy-reference.md`) for source-IP safety checks
- Added release note and upgrade guide templates (`docs/releases/_template.md`, `docs/releases/_upgrade-template.md`)
- Added auth docs retry handling snippets for token endpoint `429` and `Retry-After`
- Added docs architecture map updates for CI docs guards and local validation workflow
- Added Phase 3 planning roadmap (`docs/development/docs-phase-3-roadmap.md`)
- Expanded Phase 3 roadmap with prioritized backlog (`S/M/L`), dependencies, and execution order
- Added Phase 4 micro-roadmap (`docs/development/docs-phase-4-roadmap.md`) with 3 PR plan and CI guard proposals
- Added incident decision tree and first-15-minutes incident playbook runbooks
- Added known limitations page for rate limiting, proxy trust, and KMS startup tradeoffs
- Added versioned examples index by release (`docs/examples/versioned-by-release.md`)
- Added day-0 onboarding walkthroughs for operator and developer personas
- Added persona landing pages (`docs/personas/operator.md`, `docs/personas/developer.md`, `docs/personas/security.md`)
- Added docs KPI page and postmortem-to-doc feedback loop guidance
- Added consolidated docs master backlog (`docs/development/docs-master-backlog.md`)
- Added search alias shortcuts in docs index for faster incident/runbook discovery
- Added command verification markers to key rollout/troubleshooting/smoke docs

## 2026-02-19 (docs v12 - v0.6.0 release prep)

- Added release notes page: `docs/releases/v0.6.0.md`
- Added upgrade guide: `docs/releases/v0.6.0-upgrade.md`
- Updated docs metadata source (`docs/metadata.json`) to `current_release: v0.6.0`
- Updated root README and docs index to promote `v0.6.0` release links
- Updated operator runbook and production rollout references to `v0.6.0`
- Updated compatibility matrix with `v0.5.1 -> v0.6.0` upgrade path
- Updated pinned Docker image examples from `allisson/secrets:v0.5.1` to `allisson/secrets:v0.6.0`
- Updated CLI command docs for KMS mode flags and new `rotate-master-key` command
- Updated environment variable docs for `KMS_PROVIDER` and `KMS_KEY_URI` configuration
- Updated key management and troubleshooting guides with KMS rotation and failure-mode guidance

## 2026-02-19 (docs v11 - v0.5.1 patch release prep)

- Added release notes page: `docs/releases/v0.5.1.md`
- Added upgrade guide: `docs/releases/v0.5.1-upgrade.md`
- Updated docs metadata source (`docs/metadata.json`) to `current_release: v0.5.1`
- Updated root README and docs index to promote `v0.5.1` release links
- Updated operator runbook index and production runbooks to reference `v0.5.1`
- Updated compatibility matrix with `v0.5.0 -> v0.5.1` patch upgrade path
- Added direct `v0.4.x -> v0.5.1` compatibility path for skip-upgrade operators
- Updated pinned Docker image examples from `allisson/secrets:v0.5.0` to `allisson/secrets:v0.5.1`
- Updated API docs release labels to `v0.5.1` where current-release references are shown
- Reduced patch-version churn in OpenAPI coverage notes by using current-release wording
- Added v0.5.1-specific master-key regression triage note to troubleshooting
- Added copy/paste quick verification commands to `docs/releases/v0.5.1-upgrade.md`
- Added patch-release safety note to `docs/releases/v0.5.1.md`
- Added release history quick links in root `README.md`
- Added runtime version fingerprint checks for mixed deployment triage

## 2026-02-19 (docs v10 - v0.5.0 security hardening release prep)

- Added comprehensive security hardening guide: `docs/operations/security-hardening.md`
- Updated docs metadata source (`docs/metadata.json`) to `current_release: v0.5.0`
- Added release notes page: `docs/releases/v0.5.0.md` and promoted it as current in docs indexes
- Updated environment variables documentation with rate limiting and CORS configuration
- Added security warnings for database SSL/TLS requirements (production vs development)
- Added migration note for token expiration default change (24h → 4h)
- Updated `.env.example` with new configuration options and security warnings
- Added security warnings to Docker and local development getting-started guides
- Updated production deployment guide with security hardening reference
- Updated security model with comprehensive production recommendations
- Added security hardening link to root README and docs indexes
- Updated current-release references from v0.4.1 to v0.5.0 while preserving historical links
- Added upgrade guide: `docs/releases/v0.5.0-upgrade.md`
- Added API rate limiting reference: `docs/api/rate-limiting.md`
- Updated API endpoint docs with `429` behavior and rate-limiting cross-links
- Expanded troubleshooting with `429` and CORS/preflight diagnostics
- Added retry/backoff examples for `429` handling in curl, Python, JavaScript, and Go example docs
- Added rate-limiting production presets in environment variables documentation
- Added docs release checklist: `docs/development/docs-release-checklist.md`
- Added OpenAPI validation step in CI workflow
- Added production rollout golden path runbook: `docs/operations/production-rollout.md`
- Added API error decision matrix: `docs/api/error-decision-matrix.md`
- Added release compatibility matrix: `docs/releases/compatibility-matrix.md`
- Added persona-oriented policy templates and references in `docs/api/policies.md`
- Expanded monitoring guide with rate-limit Prometheus queries and alert examples
- Added CORS smoke checks (copy/paste) to troubleshooting guide
- Added quarterly operator drills runbook: `docs/operations/operator-drills.md`
- Added dashboard artifact templates under `docs/operations/dashboards/`
- Added docs architecture map: `docs/development/docs-architecture-map.md`
- Added release docs CI guard: `docs/tools/check_release_docs_links.py` + workflow integration
- Expanded policy smoke tests with pre-deploy automation wrapper pattern

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
