# 🔁 Release Compatibility Matrix

> Last updated: 2026-02-19

Use this page to understand upgrade impact between recent releases.

## Matrix

| From -> To | Schema migration impact | Runtime/default changes | Required operator action |
| --- | --- | --- | --- |
| `v0.4.0 -> v0.4.1` | No new mandatory migration beyond v0.4.0 baseline | Policy matcher bugfix and docs alignment | Update image tag and validate policy wildcard behavior |
| `v0.4.x -> v0.5.0` | No new destructive schema migration required for core features | Token TTL default `24h -> 4h`; rate limiting enabled by default; CORS config introduced (disabled by default) | Set explicit `AUTH_TOKEN_EXPIRATION_SECONDS`, review `RATE_LIMIT_*`, configure `CORS_*` only if browser access is required |

## Upgrade verification by target

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

- [v0.5.0 release notes](v0.5.0.md)
- [v0.5.0 upgrade guide](v0.5.0-upgrade.md)
- [Production rollout golden path](../operations/production-rollout.md)
