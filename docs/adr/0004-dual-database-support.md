# ADR 0004: Dual Database Support

> Status: accepted
> Date: 2026-01-31

## Context

The system must support diverse deployment environments with different database preferences and constraints:

- **Customer requirements**: Organizations have existing database infrastructure and expertise (PostgreSQL or MySQL)
- **Operational constraints**: Teams may have existing database monitoring, backup, and operational tooling specific to one database
- **Cloud provider considerations**: Different cloud providers offer better support for different databases (AWS RDS, Google Cloud SQL, Azure Database)
- **Migration challenges**: Some teams cannot easily switch databases due to compliance, licensing, or operational constraints
- **Licensing preferences**: PostgreSQL (fully open source) vs MySQL (dual licensing, Oracle ownership concerns)

## Decision

Support both PostgreSQL 12+ and MySQL 8.0+ with parallel repository implementations:

**Repository structure:**

- Each domain has database-specific repository files: `postgresql_*.go` and `mysql_*.go`
- Single repository interface per domain (e.g., `KekRepository`, `SecretRepository`)
- Factory pattern selects correct implementation based on `DB_DRIVER` configuration

**Transaction pattern:**

- Unified transaction management using `database.GetTx(ctx, db)` works with both databases
- No database-specific transaction handling at use case layer

**Migration management:**

- Separate migration files per database: `migrations/postgres/*` and `migrations/mysql/*`
- Migration tool supports both databases (golang-migrate)
- Each migration must be authored twice (SQL syntax differences)

**Testing strategy:**

- Integration test suite runs against both databases in CI
- Every repository test executes twice (once per database)
- Ensures behavioral parity across databases

## Alternatives Considered

### 1. PostgreSQL Only

Support only PostgreSQL to simplify implementation and maintenance.

**Rejected because:**

- Customer feedback indicated hard requirement for MySQL in specific deployment scenarios
- Would limit adoption in organizations with standardized MySQL infrastructure
- MySQL expertise more common in some operational teams
- Would force migration for teams with existing MySQL deployments

### 2. ORM Abstraction (GORM)

Use an ORM like GORM to abstract database differences.

**Rejected because:**

- Performance overhead from ORM layer (reflection, query building)
- Less control over SQL optimization and query plans
- Additional dependency and version management complexity
- GORM-specific bugs and behaviors (leaky abstraction)
- Complex queries still require raw SQL or database-specific features

### 3. Query Builder (squirrel, goqu)

Use programmatic SQL generation to abstract database differences.

**Rejected because:**

- Still requires database-specific handling for type differences (BYTEA vs BLOB, RETURNING vs dual query)
- Adds abstraction layer complexity without eliminating dual implementation
- Learning curve for query builder syntax
- Less readable than raw SQL for complex queries

### 4. Database-Agnostic SQL Subset

Constrain all SQL to features common to both databases.

**Partially adopted:**

- We do constrain to common SQL subset (standard SELECT, INSERT, UPDATE, DELETE)
- Accept maintenance cost of dual implementation for broader compatibility

## Consequences

**Costs:**

- **Doubles maintenance burden**: Every repository change requires updates to both `postgresql_*.go` and `mysql_*.go`
- **Constrains SQL features**: Cannot use PostgreSQL-specific features:
  - `RETURNING *` clause (must do separate SELECT in MySQL)
  - Array types
  - JSONB operators and indexing
  - Advanced window functions
- **Migration complexity**:
  - Separate migration files with different SQL syntax
  - Type mapping differences (BYTEA vs BLOB, TEXT vs LONGTEXT)
  - Different default behaviors (auto_increment vs SERIAL)
- **Test complexity**: Integration tests run twice, doubling CI time
- **Future database cost**: Adding third database (SQLite, CockroachDB) would triple repository implementations

**Benefits:**

- **Broader adoption**: Organizations can use existing database infrastructure
- **Operational familiarity**: Teams use databases they already know and monitor
- **Cloud flexibility**: Choose database based on cloud provider strengths
- **Migration path**: Teams can switch databases without application changes
- **Licensing options**: PostgreSQL for fully open source, MySQL for Oracle ecosystem

**Implementation notes:**

- Raw SQL in repositories (no ORM) for maximum control and performance
- Database-specific optimizations possible where needed
- Common interface ensures use case layer is database-agnostic
- Factory pattern makes database selection transparent to business logic

**Future considerations:**

- If third database needed, reconsider ORM or query builder abstraction
- Could implement database-specific optimization branches (e.g., bulk inserts)
- Monitor for SQL feature divergence requiring major refactoring

## See also

- [Configuration](../configuration.md#db_driver)
- [Local development](../getting-started/local-development.md)
- [Production deployment](../operations/deployment/production.md)
- [ADR 0001: Envelope Encryption Model](0001-envelope-encryption-model.md)
- [ADR 0005: Context-Based Transaction Management](0005-context-based-transaction-management.md)
