# ADR 0003: Capability-Based Authorization Model

> Status: accepted
> Date: 2026-02-11

## Context

The system requires fine-grained access control to protect cryptographic operations and sensitive data. Traditional authorization models present several challenges:

- **Security requirement**: Separate encrypt/decrypt permissions to support split-role workflows (e.g., a backup service can encrypt but never decrypt data)
- **Operational flexibility**: Different clients need different levels of access to specific resource paths
- **Complexity constraint**: Avoid over-engineering authorization for a pre-1.0 system while maintaining security
- **Path-specific control**: Authorization decisions must consider both the operation type and the specific resource path

## Decision

Adopt a capability-based authorization model with the following characteristics:

**Six capabilities:**

- `read` - List or inspect metadata/state without decrypting payload values
- `write` - Create or update non-cryptographic resources and key definitions
- `delete` - Delete resources or revoke token lifecycle entries
- `encrypt` - Create encrypted outputs (secrets writes, transit encrypt, tokenization tokenize)
- `decrypt` - Resolve encrypted/tokenized values back to plaintext
- `rotate` - Create new key versions

**Policy evaluation:**

- Per-client policy documents attached to authentication clients
- Authorization check via `Client.IsAllowed(path, capability)` method
- Policies evaluated on every authenticated request after successful authentication

**Path matching semantics:**

- Exact match: No wildcard means full exact match (`/v1/audit-logs` matches only `/v1/audit-logs`)
- Full wildcard: `*` matches any request path
- Trailing wildcard: `prefix/*` matches any path starting with `prefix/` (greedy for deeper paths)
- Mid-path wildcard: `/v1/keys/*/rotate` matches paths with `*` as exactly one segment

**Examples:**

- `/v1/secrets/*` matches `/v1/secrets/app`, `/v1/secrets/app/db`, and `/v1/secrets/app/db/password`
- `/v1/transit/keys/*/rotate` matches `/v1/transit/keys/payment/rotate`
- `/v1/*/keys/*/rotate` matches `/v1/transit/keys/payment/rotate`

## Alternatives Considered

### 1. Role-Based Access Control (RBAC)

Predefined roles like "admin", "operator", "viewer" with fixed permission sets.

**Rejected because:**

- Too coarse-grained for cryptographic operations (can't separate encrypt/decrypt within a role)
- No support for path-specific permissions (admin has all access, viewer has read-only everywhere)
- Difficult to model split-role security requirements (backup service needs encrypt-only)
- Role proliferation problem (would need many roles: "read-only-secrets", "encrypt-only-transit", etc.)

### 2. Attribute-Based Access Control (ABAC)

Complex attribute evaluation with conditions like `user.department == "finance" AND resource.environment == "production"`.

**Rejected because:**

- Over-engineered for pre-1.0 system requirements
- Higher implementation complexity and maintenance burden
- Steeper learning curve for operators authoring policies
- Performance overhead from complex condition evaluation
- Current requirements satisfied by simpler capability + path model

### 3. Access Control Lists (ACLs) Per Resource

Each secret/key/resource has its own ACL defining who can access it.

**Rejected because:**

- Management complexity (policies scattered across many resources)
- No centralized view of a client's permissions
- Difficult to audit "what can this client access?"
- Harder to implement (requires ACL storage and evaluation per resource)

## Consequences

**Benefits:**

- **Simpler than ABAC**: Easier to understand and implement
- **More flexible than RBAC**: Supports path-specific and operation-specific permissions
- **Security flexibility**: Enables split-role workflows (encrypt-only, decrypt-only services)
- **Clear audit trail**: Policy evaluation logged per request for forensic analysis
- **Performance acceptable**: Per-request evaluation with simple pattern matching (no database lookups)

**Trade-offs:**

- **Policy authoring complexity**: Operators must understand path matching semantics (wildcards, exact match)
- **No hierarchical roles**: Can't define "base operator" role and inherit from it
- **Per-request evaluation**: Authorization check on every request (acceptable overhead for current scale)
- **Security training required**: Teams must understand wildcard implications (e.g., `*` grants full access)
- **No policy composition**: Can't combine multiple policy documents (single policy document per client)

**Future considerations:**

- Could add policy conditions (time windows, IP restrictions) without changing core capability model
- Could add policy inheritance or composition if complexity becomes warranted
- Policy evaluation performance can be optimized with caching if needed at scale

## See also

- [Policies cookbook](../api/auth/policies.md)
- [Capability matrix](../api/fundamentals.md#capability-matrix)
- [Security model](../concepts/security-model.md)
- [ADR 0007: Path-Based API Versioning](0007-path-based-api-versioning.md)
