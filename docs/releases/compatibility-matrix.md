# 🔁 Release Compatibility Matrix

> Last updated: 2026-02-20

Use this page to understand upgrade impact between recent releases.

## Coverage Policy

This matrix covers **recent releases only** (typically last 5-6 versions) to focus on relevant upgrade paths. Historical releases remain documented in [RELEASES.md](RELEASES.md) but are excluded here to avoid clutter.

If you need upgrade guidance for older versions, consult the full release history in [RELEASES.md](RELEASES.md) or reach out via GitHub issues.

## Matrix

| From -> To | Schema migration impact | Runtime/default changes | Required operator action |
| --- | --- | --- | --- |
| `v0.7.0 -> v0.8.0` | No changes | Documentation improvements only | None (backward compatible, no runtime changes) |
| `v0.6.0 -> v0.7.0` | No new mandatory migration | Added IP-based token endpoint rate limiting (`RATE_LIMIT_TOKEN_ENABLED`, `RATE_LIMIT_TOKEN_REQUESTS_PER_SEC`, `RATE_LIMIT_TOKEN_BURST`), token endpoint may return `429` with `Retry-After` | Add and tune `RATE_LIMIT_TOKEN_*`, validate token issuance under normal and burst load, review trusted proxy/IP behavior |
| `v0.5.1 -> v0.6.0` | No new mandatory migration | Added KMS-based master key support (`KMS_PROVIDER`, `KMS_KEY_URI`), new `rotate-master-key` CLI workflow | Decide KMS vs legacy mode, validate startup key loading, run key-dependent smoke checks |
| `v0.5.0 -> v0.5.1` | No new mandatory migration | Master key memory handling bugfix and teardown zeroing hardening | Deploy `v0.5.1` and verify key-dependent flows (token, secrets, transit) |
| `v0.4.x -> v0.5.1` | No new destructive schema migration required for core features | Token TTL default `24h -> 4h`; rate limiting enabled by default; CORS config introduced (disabled by default); includes `v0.5.1` master key memory handling hardening | Set explicit `AUTH_TOKEN_EXPIRATION_SECONDS`, review `RATE_LIMIT_*`, configure `CORS_*` only if browser access is required, then run key-dependent smoke checks |
| `v0.4.0 -> v0.4.1` | No new mandatory migration beyond v0.4.0 baseline | Policy matcher bugfix and docs alignment | Update image tag and validate policy wildcard behavior |
| `v0.4.x -> v0.5.0` | No new destructive schema migration required for core features | Token TTL default `24h -> 4h`; rate limiting enabled by default; CORS config introduced (disabled by default) | Set explicit `AUTH_TOKEN_EXPIRATION_SECONDS`, review `RATE_LIMIT_*`, configure `CORS_*` only if browser access is required |

## Upgrade verification by target

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
