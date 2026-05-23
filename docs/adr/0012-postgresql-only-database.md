# ADR 0012: PostgreSQL-Only Database

> Status: accepted
> Date: 2026-05-23
> Supersedes: [ADR 0004: Dual Database Support](0004-dual-database-support.md)

## Context

ADR 0004 adopted dual PostgreSQL and MySQL support to broaden deployment options. That choice added meaningful maintenance cost:

- Repository implementations had to be kept in parity.
- Migrations had to be authored and tested twice.
- Integration testing needed database-specific services and cases.
- SQL features were constrained to the overlap between PostgreSQL and MySQL.

The project is still pre-1.0. The public README warns that `v0.x.y` releases are not recommended for production and that the API is not stable. There is no production compatibility promise to preserve for MySQL deployments.

## Decision

Secrets will support PostgreSQL only.

The implementation will:

- Remove MySQL runtime support, repositories, migrations, tests, Docker services, and development targets.
- Remove `DB_DRIVER`.
- Keep `DB_CONNECTION_STRING` as the database connection setting.
- Flatten migrations from `migrations/postgresql/` to `migrations/`.
- Rename remaining repository implementations to backend-neutral names.
- Run database-backed tests against PostgreSQL only.

## Alternatives Considered

### 1. Keep Dual Database Support

Continue supporting both PostgreSQL and MySQL.

**Rejected because:**

- The maintenance cost is high for an alpha project.
- Every persistence change requires duplicate SQL and tests.
- MySQL compatibility prevents PostgreSQL-specific schema and query improvements.
- Existing pre-1.0 documentation already warns that the project is not production-stable.

### 2. Deprecate MySQL Before Removal

Keep MySQL support temporarily while warning users.

**Rejected because:**

- The project has not reached a stable production release.
- A compatibility window would retain most complexity while delaying the simplification.
- Keeping `DB_DRIVER` or MySQL stubs would preserve an obsolete mental model.

### 3. Keep PostgreSQL-Labeled Paths

Keep `migrations/postgresql/` and `repository/postgresql/` after deleting MySQL.

**Rejected because:**

- With one database, vendor-labeled directories imply future database variants.
- Backend-neutral names better match the new architecture.

## Consequences

**Benefits:**

- Lower repository and migration maintenance burden.
- Faster and simpler database-backed tests.
- Smaller dependency surface.
- Freedom to use PostgreSQL-specific features when useful.
- Simpler configuration and documentation.

**Costs:**

- Users cannot deploy new versions against MySQL.
- Any existing MySQL data would need an external migration to PostgreSQL before upgrading.
- Future database support would require a new architectural decision.

## See also

- [ADR 0004: Dual Database Support](0004-dual-database-support.md)
- [Configuration](../configuration.md)
