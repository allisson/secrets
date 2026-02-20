# ADR 0005: Context-Based Transaction Management

> Status: accepted
> Date: 2026-01-29

## Context

The system requires atomic multi-step database operations to maintain consistency:

- **Atomic operations needed**: KEK rotation (update old KEK + create new KEK), client updates (update client + update policies)
- **Clean Architecture principle**: Repository layer should not control transaction boundaries (use cases orchestrate business logic)
- **Transparency requirement**: Repositories should work identically whether called within or outside a transaction
- **Database abstraction**: Transaction pattern must work with both PostgreSQL and MySQL
- **Error handling**: Automatic rollback on errors, commit on success

## Decision

Adopt context-based transaction propagation with the following pattern:

**Transaction storage:**

- Store active transaction in context using `context.WithValue(ctx, txKey{}, tx)`
- Use unexported `txKey{}` type to prevent external packages from accessing transaction directly

**Repository pattern:**

- Repositories call `database.GetTx(ctx, db)` to get either:
  - Active transaction from context (if present)
  - Database connection (if no transaction active)
- Single repository method signature works for both transactional and non-transactional calls

**Use case coordination:**

- Use cases coordinate transactions via `TxManager.WithTx(ctx, fn)` interface
- Transaction manager handles begin, commit, rollback automatically
- Automatic rollback on any error within transaction function
- No nested transaction support (single transaction per context)

**Example usage:**

```go
// Use case orchestrates transaction
return k.txManager.WithTx(ctx, func(ctx context.Context) error {
    if err := k.kekRepo.Update(ctx, oldKek); err != nil {
        return err  // Automatic rollback
    }
    return k.kekRepo.Create(ctx, newKek)  // Commit on success
})

// Repository transparently uses transaction or DB
func (r *Repository) Create(ctx context.Context, kek *Kek) error {
    querier := database.GetTx(ctx, r.db)  // Gets tx if active, else db
    _, err := querier.ExecContext(ctx, query, args...)
    return err
}
```

## Alternatives Considered

### 1. Explicit Transaction Passing

Pass transaction explicitly as parameter: `repository.Create(tx *sql.Tx, kek *Kek)`.

**Rejected because:**

- Forces dual signatures: methods need both `*sql.DB` and `*sql.Tx` versions
- Repositories have two versions of every method: `Create()` and `CreateTx()`
- Use case code becomes verbose (must check if transaction needed)
- Violates DRY principle (duplicate implementation)
- Harder to refactor (add/remove transaction changes all call sites)

### 2. Transaction in Repository Layer

Repositories call `db.Begin()` internally when transaction needed.

**Rejected because:**

- Violates Clean Architecture (repositories should not decide transaction boundaries)
- Business logic (which operations are atomic) leaks into repository layer
- Cannot compose multiple repository calls in single transaction from use case
- Hard to test transactional behavior in isolation
- Repository layer has too much responsibility

### 3. Unit of Work Pattern

Explicit transaction object passed around and accumulated changes.

**Rejected because:**

- More verbose than context-based approach (similar outcome with more boilerplate)
- Requires explicit `unitOfWork.Begin()`, `unitOfWork.Commit()` calls
- Still needs some form of propagation (context or explicit parameter)
- Added complexity without significant benefit over context pattern

### 4. No Transactions - Application-Level Idempotency

Rely on idempotency and retries instead of database transactions.

**Rejected because:**

- Unacceptable for financial and cryptographic operations requiring strict consistency
- Key rotation errors could leave system in inconsistent state
- Complex to implement correctly (requires careful state machine design)
- Performance overhead from retry logic
- Database transactions are proven, reliable primitive

## Consequences

**Benefits:**

- **Simple repository signatures**: Single method works for both transactional and non-transactional calls
- **No performance overhead**: Direct DB connection when transaction not needed
- **Clean Architecture compliance**: Use cases control transaction boundaries, repositories participate
- **Easy testing**: Repositories can be tested without transaction complexity
- **Automatic rollback**: Error handling simplified (any error triggers rollback)
- **Database agnostic**: Works identically with PostgreSQL and MySQL

**Trade-offs:**

- **Context pollution concern**: Using context for dependency injection (not just cancellation/deadlines)
  - Acceptable trade-off: Go community precedent (database/sql uses context for cancellation)
  - Transaction is request-scoped, fits context lifetime model
- **Implicit behavior**: Active transaction not visible in function signature
  - Mitigated by: Clear naming (`WithTx`), documentation, code review practices
- **Debugging difficulty**: Transaction state requires context inspection
  - Mitigated by: Logging at transaction boundaries, clear error messages

**Limitations:**

- **No nested transactions**: Context holds single transaction, no savepoint support
  - Acceptable: Use cases designed to avoid nested transaction needs
  - Could add savepoint support in future if needed
- **No transaction isolation control**: Uses database default isolation level
  - Acceptable: Default isolation sufficient for current use cases
  - Could add isolation level parameter to `WithTx()` if needed

**Future considerations:**

- Could add savepoint support for nested transaction semantics
- Could add transaction isolation level configuration
- Could add transaction timeout configuration
- Monitor for context pollution becoming problematic (no issues so far)

## See also

- [Architecture concepts](../concepts/architecture.md)
- [ADR 0004: Dual Database Support](0004-dual-database-support.md)
