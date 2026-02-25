# 🔁 Release Compatibility Matrix

> Last updated: 2026-02-25

Use this page to understand upgrade impact between recent releases.

## Coverage Policy

This matrix covers **recent releases only** (typically last 5-6 versions) to focus on relevant upgrade paths. Historical releases remain documented in [RELEASES.md](RELEASES.md) but are excluded here to avoid clutter.

If you need upgrade guidance for older versions, consult the full release history in [RELEASES.md](RELEASES.md) or reach out via GitHub issues.

## Matrix

| From -> To | Schema migration impact | Runtime/default changes | Required operator action |
| --- | --- | --- | --- |
| `v0.14.1 -> v0.15.0` | No schema migration required | Added Goreleaser support for automated builds | None (backward compatible, no runtime changes) |
| `v0.14.0 -> v0.14.1` | No schema migration required | Fixed empty KEK chain bug | None (backward compatible, no runtime changes) |
| `v0.13.0 -> v0.14.0` | No schema migration required | `/metrics` endpoint moved from port `8080` to `8081`. New `METRICS_PORT` env var (default: `8081`). | Update Prometheus/monitoring scrape configs to use port `8081`. Expose port `8081` in container orchestration if necessary. |
| `v0.12.0 -> v0.13.0` | No schema migration required | Aggressive documentation refactor for conciseness and maintainability | None (documentation only) |
| `v0.11.0 -> v0.12.0` | No schema migration required | Added `rewrap-deks` CLI command to bulk re-encrypt DEKs, removed localized "What's New" section from root README | None (backward compatible, no runtime changes) |
| `v0.10.0 -> v0.11.0` | Migration 000004 required (adds `failed_attempts INT NOT NULL DEFAULT 0` and `locked_until TIMESTAMPTZ` columns to `clients` table) | Account lockout enabled by default (10 attempts, 30 min). `POST /v1/token` may return `423 Locked`. Two new env vars: `LOCKOUT_MAX_ATTEMPTS`, `LOCKOUT_DURATION_MINUTES` | Run migration 000004, optionally configure `LOCKOUT_*` vars, verify token issuance still works, monitor for unexpected 423 responses |
| `v0.9.0 -> v0.10.0` | No schema migration required | Docker base image changed (scratch → distroless), container runs as non-root (UID 65532), read-only filesystem support, multi-arch builds (amd64/arm64) | Update health check patterns ([guide](../operations/observability/health-checks.md)), verify rollback to v0.9.0 works |
| `v0.8.0 -> v0.9.0` | Migration 000003 required (adds `signature`, `kek_id`, `is_signed` columns + FK constraints) | Audit logs automatically signed on creation, FK constraints prevent client/KEK deletion with audit logs | Run migration 000003, verify no orphaned client references, validate signing working, confirm FK constraint behavior |
| `v0.7.0 -> v0.8.0` | No changes | Documentation improvements only | None (backward compatible, no runtime changes) |
| `v0.6.0 -> v0.7.0` | No new mandatory migration | Added IP-based token endpoint rate limiting (`RATE_LIMIT_TOKEN_ENABLED`, `RATE_LIMIT_TOKEN_REQUESTS_PER_SEC`, `RATE_LIMIT_TOKEN_BURST`), token endpoint may return `429` with `Retry-After` | Add and tune `RATE_LIMIT_TOKEN_*`, validate token issuance under normal and burst load, review trusted proxy/IP behavior |
| `v0.5.1 -> v0.6.0` | No new mandatory migration | Added KMS-based master key support (`KMS_PROVIDER`, `KMS_KEY_URI`), new `rotate-master-key` CLI workflow | Decide KMS vs legacy mode, validate startup key loading, run key-dependent smoke checks |
| `v0.5.0 -> v0.5.1` | No new mandatory migration | Master key memory handling bugfix and teardown zeroing hardening | Deploy `v0.5.1` and verify key-dependent flows (token, secrets, transit) |
| `v0.4.x -> v0.5.1` | No new destructive schema migration required for core features | Token TTL default `24h -> 4h`; rate limiting enabled by default; CORS config introduced (disabled by default); includes `v0.5.1` master key memory handling hardening | Set explicit `AUTH_TOKEN_EXPIRATION_SECONDS`, review `RATE_LIMIT_*`, configure `CORS_*` only if browser access is required, then run key-dependent smoke checks |
| `v0.4.0 -> v0.4.1` | No new mandatory migration beyond v0.4.0 baseline | Policy matcher bugfix and docs alignment | Update image tag and validate policy wildcard behavior |
| `v0.4.x -> v0.5.0` | No new destructive schema migration required for core features | Token TTL default `24h -> 4h`; rate limiting enabled by default; CORS config introduced (disabled by default) | Set explicit `AUTH_TOKEN_EXPIRATION_SECONDS`, review `RATE_LIMIT_*`, configure `CORS_*` only if browser access is required |

## Upgrade verification by target

For `v0.15.0`:

1. `./bin/app --version` shows `v0.15.0`
2. Goreleaser configurations successfully execute platform builds (no runtime impact)

For `v0.14.1`:

1. `./bin/app --version` shows `v0.14.1`
2. Starting the application with an empty KEK chain correctly fails without panicking

For `v0.14.0`:

1. `GET :8081/metrics` succeeds and returns Prometheus exposition format
2. `GET :8080/metrics` returns `404 Not Found`
3. `./bin/app --version` shows `v0.14.0`

For `v0.13.0`:

1. All documentation is significantly more concise and reference-oriented
2. `make docs-lint` passes with no errors
3. Code examples are centralized in `docs/examples/`

For `v0.12.0`:

1. `./bin/app --version` shows `v0.12.0`
2. `rewrap-deks --help` is available as a CLI command
3. `GET /health` and `GET /ready` pass

For `v0.11.0`:

1. `GET /health` and `GET /ready` pass
2. Migration 000004 applied successfully (`SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1` returns `4`)
3. `POST /v1/token` with correct credentials issues a token normally
4. After 10 wrong-secret attempts, `POST /v1/token` returns `423 Locked`
5. `locked_until` and `failed_attempts` columns exist on the `clients` table
6. Successful auth after failures resets `failed_attempts` to 0 and clears `locked_until`

For `v0.10.0`:

1. `GET /health` and `GET /ready` pass
2. Container starts as non-root user (UID 65532, GID 65532)
3. `./bin/app --version` shows `v0.10.0` with "v" prefix
4. Multi-arch image works on both amd64 and arm64
5. Rollback to v0.9.0 completes without data loss
6. Security scanning passes (Trivy/Grype show expected base image)

For `v0.9.0`:

1. `GET /health` and `GET /ready` pass
2. Migration 000003 applied successfully (check `SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1` returns `3`)
3. New audit logs have `is_signed=true`, `signature` populated, and `kek_id` set
4. `verify-audit-logs` reports valid signatures for today's logs
5. FK constraints prevent client deletion with audit logs (DELETE returns FK violation error)
6. Legacy audit logs marked as `is_signed=false` (no signature)

For `v0.8.0`:

No upgrade verification needed - documentation-only release with no runtime changes.

For `v0.7.0`:

1. `GET /health` and `GET /ready` pass
2. `POST /v1/token` issues tokens at normal traffic levels
3. Controlled token burst returns `429` with `Retry-After`
4. Secrets and transit round-trip flows succeed

For `v0.6.0`:

1. `GET /health` and `GET /ready` pass
2. Startup logs confirm intended key mode (KMS or legacy)
3. `POST /v1/token` issues tokens successfully
4. Secrets and transit round-trip flows succeed

For `v0.5.1`:

1. `GET /health` and `GET /ready` pass
2. `POST /v1/token` issues tokens successfully
3. Secrets and transit round-trip flows succeed without key configuration errors

For `v0.5.0`:

1. `GET /health` and `GET /ready` pass
2. `POST /v1/token` issues token with expected expiration behavior
3. Protected endpoints behave correctly under normal load and return controlled `429` with `Retry-After` under bursts
4. CORS behavior matches deployment mode (server-to-server vs browser)

## Notes

- Keep migrations additive and avoid destructive rollback in production unless fully validated
- Pin release tags in automation for reproducible rollouts
- Preserve historical release notes; promote only the current release in operator navigation

## See also

- [All release notes](RELEASES.md)
- [Production rollout golden path](../operations/deployment/production-rollout.md)
