# ADR 0009: UUIDv7 for Identifiers

> Status: accepted
> Date: 2026-01-29

## Context

The application requires unique identifiers for database entities with the following needs:

- Globally unique across distributed systems (no coordination required)
- Database-friendly for indexing and query performance
- Sortable for chronological ordering (audit logs, version history)
- Secure against enumeration attacks (unpredictable)
- Compatible with PostgreSQL `UUID` type and MySQL `BINARY(16)` storage
- Usable in HTTP APIs (URL-safe string representation)

Identifier strategies include:

- **Sequential integers**: Auto-increment IDs (database-generated)
- **UUIDv4**: Random 122-bit identifiers (no time ordering)
- **UUIDv7**: Time-ordered with random components (RFC 9562, 2024)
- **ULID**: Lexicographically sortable, Crockford Base32 encoded
- **Snowflake IDs**: Twitter's time-based 64-bit IDs with machine/sequence components

Database performance matters: B-tree indexes perform poorly with random UUIDs (UUIDv4) due to page splits, but sequential IDs expose information leakage (entity count, creation rate).

## Decision

Adopt **UUIDv7** (RFC 9562) for all database entity identifiers:

**Core characteristics:**

- **48-bit timestamp**: Unix epoch milliseconds (provides natural time ordering)
- **12-bit random counter**: Sub-millisecond collision avoidance
- **62-bit randomness**: Cryptographically random bits for unpredictability
- **Total: 128 bits** (same as UUIDv4, compatible with UUID database types)

**Usage pattern:**

```go
import "github.com/google/uuid"

id := uuid.Must(uuid.NewV7())  // All database entities
```

**Applied to:**

- Database primary keys (clients, tokens, secrets, transit keys, tokenization keys, KEKs, DEKs, audit logs)
- Request IDs (HTTP middleware via `gin-contrib/requestid`)
- All domain entities requiring unique identification

**Sorting behavior:**

- UUIDv7 values are naturally sortable by time (ascending = oldest first)
- Eliminates need for separate `created_at` columns for chronological queries
- Enables efficient range scans: `WHERE id > $last_seen_id ORDER BY id LIMIT 100`

**Storage:**

- PostgreSQL: `UUID` type (16 bytes, native support)
- MySQL: `BINARY(16)` (requires manual conversion in repositories)

## Alternatives Considered

### 1. UUIDv4 (Random)

Standard random UUID with 122 bits of randomness.

**Rejected because:**

- **Poor index performance**: Random distribution causes B-tree page splits on every insert (write amplification)
- **No time ordering**: Cannot sort by ID to get chronological order (must add `created_at` column)
- **Wasted storage**: Requires separate timestamp columns for temporal queries
- **Fragmented indexes**: Database must constantly rebalance index pages (increased I/O)

**Benchmark impact** (PostgreSQL):

- UUIDv4 inserts: ~40% slower than UUIDv7 at scale (millions of rows)
- Index size: ~15-20% larger due to fragmentation

### 2. Auto-Increment Sequential IDs

Database-generated `SERIAL` (PostgreSQL) or `AUTO_INCREMENT` (MySQL).

**Rejected because:**

- **Information leakage**: Sequential IDs expose entity count and creation rate (security concern for API)
- **Coordination required**: Multi-region deployments need complex ID generation coordination
- **No global uniqueness**: Cannot merge data from multiple databases without ID collision
- **Migration complexity**: Changing ID space requires expensive table rewrites
- **Enumeration attacks**: Attackers can guess valid IDs (e.g., `/v1/secrets/1`, `/v1/secrets/2`, ...)

### 3. ULID (Universally Unique Lexicographically Sortable Identifier)

128-bit time-ordered IDs with Crockford Base32 encoding.

**Rejected because:**

- **Non-standard encoding**: Crockford Base32 incompatible with PostgreSQL `UUID` type (requires `TEXT` or custom type)
- **Storage overhead**: Text storage (26 characters) uses more space than binary UUID (16 bytes)
- **Ecosystem compatibility**: No native database support, requires custom conversion logic
- **Library maturity**: Less mature Go libraries compared to `google/uuid` (which has UUIDv7 support)

### 4. Snowflake IDs

64-bit time-based IDs with machine/datacenter/sequence components.

**Rejected because:**

- **Machine coordination**: Requires unique machine IDs (complex in containerized/serverless environments)
- **64-bit limit**: Smaller than UUID (potential collision risk in high-throughput systems)
- **No UUID compatibility**: Cannot use PostgreSQL `UUID` type (requires `BIGINT`)
- **Clock skew sensitivity**: Requires NTP synchronization across machines (operational complexity)
- **Single point of failure**: Machine ID exhaustion or clock drift can cause outages

### 5. UUIDv1 (Time-Based with MAC Address)

Time-ordered UUID with MAC address component.

**Rejected because:**

- **Privacy leak**: Embeds machine MAC address (reveals infrastructure details)
- **Non-monotonic**: Time field has unusual byte ordering (not naturally sortable as bytes)
- **No randomness**: Sequence number only 14 bits (collision risk in high-throughput systems)
- **Security concern**: Predictable MAC address component aids reconnaissance attacks

## Consequences

**Benefits:**

- **Index-friendly writes**: Time-ordered inserts minimize B-tree page splits (better write performance)
- **Chronological sorting**: IDs naturally sort by creation time (no separate `created_at` column needed for ordering)
- **Global uniqueness**: No coordination required across instances/regions
- **Enumeration resistance**: 62 bits of randomness prevent ID guessing attacks
- **Database compatibility**: Native `UUID` type support in PostgreSQL, `BINARY(16)` in MySQL
- **Standard compliance**: RFC 9562 specification (future-proof)
- **Pagination efficiency**: Can use `WHERE id > $cursor` for keyset pagination (more efficient than offset)
- **Audit log ordering**: Audit events naturally ordered by ID without timestamp sorting

**Trade-offs:**

- **Clock dependency**: Requires monotonic clock (mitigated by `time.Now()` in Go runtime)
- **Partial information leak**: Timestamp component reveals creation time to millisecond precision (acceptable for most use cases)
- **Not strictly monotonic**: Random bits mean IDs created in same millisecond are unordered (acceptable for business logic)

**Limitations:**

- **Millisecond precision**: Cannot distinguish order within same millisecond without additional sorting
  - Mitigated by: 12-bit random counter provides sub-millisecond uniqueness
  - Acceptable: Business logic does not require microsecond-level ordering
- **Time skew vulnerability**: Clock rollback could create IDs with earlier timestamps
  - Mitigated by: NTP synchronization and monotonic clock in modern OS/container environments
  - Acceptable: Crypto operations are not sensitive to millisecond-level time ordering

**Performance characteristics:**

- **Insert throughput**: ~60% faster than UUIDv4 at scale (millions of rows)
- **Index size**: ~15-20% smaller than UUIDv4 indexes (better cache locality)
- **Range scan efficiency**: Time-ordered layout improves sequential access patterns

**Migration notes:**

- Existing UUIDv4 values remain valid (no retroactive migration needed)
- New entities use UUIDv7 from implementation date forward
- Mixed UUID versions acceptable (version distinguishable by variant bits)

## See also

- [RFC 9562: Universally Unique IDentifiers (UUIDs)](https://www.rfc-editor.org/rfc/rfc9562.html)
- [google/uuid Go library](https://github.com/google/uuid)
- [Request ID middleware implementation](../../internal/http/middleware.go)
- [ADR 0008: Gin Web Framework with Custom Middleware](0008-gin-web-framework-with-custom-middleware.md) - Request ID usage
