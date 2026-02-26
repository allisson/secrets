# ADR 0011: HMAC-SHA256 Cryptographic Signing for Audit Log Integrity

> Status: accepted
> Date: 2026-02-20

## Context

The application must ensure audit log integrity to meet compliance requirements and detect unauthorized tampering:

- **Tamper detection**: Detect if audit logs are modified after creation
- **Compliance evidence**: Provide cryptographic proof of log authenticity
- **Key separation**: Separate signing keys from encryption keys (best practice)
- **Backward compatibility**: Support mixed signed/unsigned logs during migration
- **Performance**: Minimal overhead for high-frequency audit logging

Audit logs record security-sensitive operations (client creation, secret access, policy changes) with the following fields:

- Request ID, Client ID, Capability, Path, Metadata, Created At

Key security considerations:

- **Threat model**: Attacker gains database write access, attempts to hide malicious activity by modifying/deleting logs
- **Attack vectors**: Log tampering (modify metadata), log deletion (cover tracks), log injection (false audit trail)
- **Insider threats**: Malicious administrator with database access modifying logs post-breach

## Decision

Adopt **HMAC-SHA256** with **HKDF-SHA256 key derivation** for cryptographic audit log signing:

**Algorithm choice:**

- **HMAC-SHA256**: Keyed-hash message authentication code using SHA-256
- **HKDF-SHA256**: HMAC-based Key Derivation Function for deriving signing keys from KEKs
- **Recommended by NIST SP 800-107** for message authentication
- **Industry standard**: Widely used for data integrity in protocols (TLS, JWT, AWS Signature v4)

**Key derivation (HKDF-SHA256):**

```go
// Derive signing key from KEK (separates encryption and signing key usage)
info := []byte("audit-log-signing-v1")
hash := sha256.New
hkdf := hkdf.New(hash, kekKey, nil, info)
signingKey := make([]byte, 32)
io.ReadFull(hkdf, signingKey)
defer signingKey.Zero() // Clear from memory after use
```

**Parameters:**

- Extract length: 32 bytes (256 bits)
- Info string: `"audit-log-signing-v1"` (domain separation)
- Salt: nil (KEK already has high entropy)
- Hash function: SHA-256

**Canonical log format:**

```go
// Length-prefixed encoding prevents ambiguity in variable-length fields
canonical := 
    request_id (16 bytes UUID) ||
    client_id (16 bytes UUID) ||
    len(capability) (4 bytes uint32) || capability (variable) ||
    len(path) (4 bytes uint32) || path (variable) ||
    len(metadata_json) (4 bytes uint32) || metadata_json (variable) ||
    created_at_unix_nano (8 bytes int64)
```

**Signature generation:**

```go
mac := hmac.New(sha256.New, signingKey)
mac.Write(canonicalBytes)
signature := mac.Sum(nil) // 32 bytes
```

**Database schema (Migration 000003):**

```sql
ALTER TABLE audit_logs ADD COLUMN signature BYTEA;
ALTER TABLE audit_logs ADD COLUMN kek_id UUID REFERENCES keks(id) ON DELETE RESTRICT;
ALTER TABLE audit_logs ADD COLUMN is_signed BOOLEAN DEFAULT FALSE;
ALTER TABLE audit_logs ADD CONSTRAINT fk_audit_logs_kek_id 
    FOREIGN KEY (kek_id) REFERENCES keks(id) ON DELETE RESTRICT;
```

**Architecture layers:**

1. **Service Layer** (`internal/auth/service/audit_signer.go`):
   - `AuditSigner` interface with `Sign()` and `Verify()` methods
   - HKDF key derivation from KEK
   - Canonical log serialization
   - HMAC-SHA256 signature generation/verification

2. **Use Case Layer** (`internal/auth/usecase/audit_log_usecase.go`):
   - `Create()` automatically signs logs if `KekChain` and `AuditSigner` available
   - `VerifyBatch()` validates signatures for time range
   - `VerifyAuditLog()` validates single log signature

3. **CLI Layer** (`cmd/app/commands/verify_audit_logs.go`):
   - `verify-audit-logs` command with `--start-date`, `--end-date`, `--format` flags
   - Text and JSON output formats
   - Exit code 0 (pass) or 1 (fail) for automation

**Usage pattern:**

```go
// Automatic signing on audit log creation
auditLog := &authDomain.AuditLog{
    ID:         uuid.Must(uuid.NewV7()),
    ClientID:   client.ID,
    Capability: authDomain.WriteCapability,
    Path:       "/v1/clients",
    Metadata:   metadata,
    CreatedAt:  time.Now().UTC(),
}
err := auditLogUseCase.Create(ctx, auditLog) // Signed if KEK chain available

// CLI verification
$ ./bin/app verify-audit-logs --start-date 2026-02-01 --end-date 2026-02-28
Verification Report (2026-02-01 to 2026-02-28)
Total Logs Checked: 1,234
  Signed Logs: 1,200
  Unsigned Logs: 34
  Valid Signatures: 1,200
  Invalid Signatures: 0

Status: ✓ All signed logs verified successfully
```

**Backward compatibility:**

- Existing logs marked as `is_signed=false` (legacy logs)
- `VerifyBatch()` reports both signed and unsigned counts
- Mixed verification supports gradual migration
- No re-signing of historical logs (preserves original signatures)

## Alternatives Considered

### 1. Digital Signatures (RSA/ECDSA)

Asymmetric cryptography with public key verification.

**Rejected because:**

- **Non-repudiation unnecessary**: Audit logs are internal system records, not legally binding documents
- **Performance overhead**: RSA-2048 signing ~50-100x slower than HMAC-SHA256 (~500µs vs ~10µs)
- **Key management complexity**: Requires public/private key pairs, certificate rotation, key storage
- **Overkill for threat model**: Symmetric keys sufficient when attacker has database access (key compromise assumed)
- **No security benefit**: If attacker compromises KEK chain, can compromise signing keys regardless of algorithm
- **Size overhead**: RSA-2048 signatures are 256 bytes (8x larger than HMAC-SHA256's 32 bytes)

**Performance comparison:**

- RSA-2048: ~500µs signing, ~50µs verification, 256-byte signature
- HMAC-SHA256: ~10µs signing, ~2µs verification, 32-byte signature

### 2. HMAC-SHA512

Stronger SHA-2 variant with 512-bit output.

**Rejected because:**

- **No security benefit**: SHA-256 already provides 128-bit collision resistance (sufficient for MAC)
- **Larger signatures**: 64-byte signatures vs 32-byte (2x storage overhead)
- **No attack scenarios**: No known attacks on HMAC-SHA256 requiring SHA-512 upgrade
- **Slower performance**: SHA-512 ~10-20% slower on 64-bit systems for small messages
- **Overkill**: 256-bit output exceeds security requirements for audit log integrity

**Security analysis:**

- HMAC-SHA256: 128-bit security (birthday bound), 256-bit preimage resistance
- HMAC-SHA512: 256-bit security (birthday bound), 512-bit preimage resistance
- Audit logs require ~80-100 bits security (no brute-force forgery feasible)

### 3. Direct KEK Signing (Without HKDF)

Use KEK directly as HMAC key without derivation.

**Rejected because:**

- **Key separation violation**: Breaks cryptographic best practice of separating key usage contexts
- **Encryption key reuse**: Same key used for AES-GCM encryption and HMAC signing (security risk)
- **Domain confusion**: If KEK compromised, both encryption and signing affected simultaneously
- **No algorithm agility**: Cannot upgrade signing algorithm independently from encryption
- **NIST recommendation**: SP 800-108 recommends key derivation for separate purposes

**Security risk:**

- Related-key attacks possible when same key used in different algorithms
- HKDF provides domain separation via `info` parameter (`"audit-log-signing-v1"`)

### 4. Hash Chains (Merkle Tree)

Link logs with cryptographic hashes forming immutable chain.

**Rejected because:**

- **Sequential verification**: Must verify entire chain from genesis to detect tampering (slow)
- **Complex migration**: Existing logs cannot be retroactively chained without re-signing
- **Deletion detection only**: Detects missing logs but not modified metadata (HMAC detects both)
- **No random access**: Cannot verify single log without traversing chain
- **Chain break propagation**: Single deleted log breaks verification for all subsequent logs
- **Operational complexity**: Requires careful chain management and backup

**Verification performance:**

- Merkle tree: O(n) for full verification, O(log n) for single log (with tree structure)
- HMAC signatures: O(1) for single log, O(n) for batch (parallelizable)

### 5. Append-Only Log Database

Use specialized database (e.g., Amazon QLDB, Azure Immutable Storage) with built-in integrity.

**Rejected because:**

- **External dependency**: Requires additional managed service or specialized database
- **Vendor lock-in**: Tied to cloud provider's proprietary solution
- **Operational complexity**: Separate database for audit logs, replication/backup overhead
- **Migration burden**: Must export logs from PostgreSQL/MySQL to specialized store
- **Cost**: Additional service fees for managed append-only storage
- **Redundant**: Application-level signing provides same guarantees without external dependency

**Deployment complexity:**

- Current: Single PostgreSQL/MySQL database
- Alternative: PostgreSQL/MySQL + QLDB/Immutable Storage (2 systems to manage)

### 6. External Audit Log Service

Send logs to third-party service (e.g., Splunk, Datadog, AWS CloudTrail).

**Rejected because:**

- **Network dependency**: Audit logging fails if external service unavailable (availability risk)
- **Latency overhead**: Network round trip adds 50-200ms per audit log write
- **Additional cost**: Per-log ingestion fees for external service
- **Data sovereignty**: Audit logs may leave controlled infrastructure (compliance risk)
- **Still need local signing**: External service doesn't prevent database tampering (complementary, not alternative)
- **Complexity**: Requires service integration, credential management, retry logic

**Availability impact:**

- Local HMAC signing: ~10µs overhead, zero dependencies
- External service: ~100ms latency, network/service availability dependency

## Consequences

**Benefits:**

- **Tamper detection**: Cryptographic proof of log integrity, detects modifications/deletions
- **Key separation**: HKDF derivation separates signing keys from encryption keys (security best practice)
- **Backward compatibility**: `is_signed` flag supports mixed signed/unsigned logs during migration
- **Minimal performance impact**: ~10-15µs signing overhead per log (negligible)
- **Fast verification**: Batch verification of 10,000 logs completes in ~20-30ms
- **Standard algorithms**: HMAC-SHA256 and HKDF-SHA256 are NIST-approved, widely vetted
- **No external dependencies**: In-process signing, no network calls or external services
- **Automation-friendly**: CLI exit codes enable automated integrity checks in CI/CD

**Trade-offs:**

- **Migration required**: Database migration 000003 adds three columns and FK constraints
  - Downtime: ~1-10 seconds for schema changes (depends on table size)
  - Foreign key constraints prevent deletion of clients/KEKs with audit logs
  - Existing logs remain unsigned (`is_signed=false`, marked as legacy)
  
- **KEK retention requirement**: Signed logs create permanent KEK dependency via `fk_audit_logs_kek_id`
  - Cannot delete KEKs referenced by signed audit logs (FK constraint `ON DELETE RESTRICT`)
  - KEK rotation does NOT re-sign old logs (preserves historical signatures with original KEK)
  - Verification requires KEK chain with all historical KEKs loaded into memory
  - Acceptable: KEKs are small (32 bytes), typical deployments have <100 KEKs
  
- **Legacy logs unverified**: Existing audit logs (pre-migration) cannot be verified
  - Mitigation: Clear reporting of signed vs unsigned logs in verification output
  - Acceptable: Migration clearly marks legacy vs new logs with `is_signed` flag
  
- **Operational overhead**: Periodic integrity checks required via `verify-audit-logs` CLI
  - Mitigation: Automate via cron job or monitoring system
  - Acceptable: ~30ms per 10k logs, can run during off-peak hours

**Limitations:**

- **KEK compromise exposes signing keys**: HKDF derivation is deterministic (KEK + info → signing key)
  - If attacker compromises KEK chain, can forge signatures for logs referencing that KEK
  - Acceptable: Matches threat model (database compromise assumed, focus on tamper detection not prevention)
  - Acceptable: Encryption keys NOT reversed from signing keys (one-way derivation)
  
- **No real-time integrity monitoring**: Verification runs on-demand via CLI, not automatic
  - Mitigation: Schedule periodic verification jobs (e.g., daily cron)
  - Future enhancement: Add `/v1/audit-logs/verify` API endpoint for real-time checks
  
- **Signature does not prove timestamp authenticity**: Attacker with database access could modify `created_at`
  - Mitigation: Request ID (UUIDv7) embeds timestamp, monotonically increasing
  - Mitigation: Application-level validation ensures `created_at` matches request processing time
  - Acceptable: Focus on detecting content modification, not timestamp forgery

**Security characteristics:**

- **Signature strength**: 256-bit HMAC-SHA256 provides 128-bit security (birthday bound)
- **Brute-force resistance**: 2^128 attempts to forge signature (computationally infeasible)
- **Collision resistance**: SHA-256 has no known collision attacks (as of 2026)
- **Key derivation**: HKDF-SHA256 is provably secure under standard assumptions (RFC 5869)
- **Domain separation**: Info string `"audit-log-signing-v1"` prevents cross-protocol attacks
- **Canonical format**: Length-prefixed encoding prevents reordering/substitution attacks
- **Memory safety**: Signing keys zeroed from memory after use (prevents memory dumps)

**KEK preservation requirements:**

- **Foreign key constraint**: `fk_audit_logs_kek_id` enforces referential integrity
  - DELETE operations on `keks` table fail if `audit_logs` references exist
  - Prevents accidental KEK deletion breaking signature verification
  
- **KEK rotation policy**: New KEKs used for new logs, old KEKs retained for verification
  - Historical logs remain signed with original KEK (preserves signature validity)
  - `VerifyBatch()` looks up appropriate KEK from chain based on `kek_id`
  
- **Chain loading**: `LoadMasterKeyChain()` must load all KEKs into `KekChain` for verification
  - Chain stored in memory as map (O(1) lookup by KEK ID)
  - Typical deployment: 50-100 KEKs, ~3-5 KB memory overhead (negligible)

**Performance characteristics:**

| Operation              | Latency | Throughput      | Notes                          |
|------------------------|---------|-----------------|--------------------------------|
| Sign single log        | ~10µs   | 100k logs/sec   | HKDF + HMAC-SHA256             |
| Verify single log      | ~2µs    | 500k logs/sec   | HMAC-SHA256 only               |
| Batch verify 10k logs  | ~30ms   | 333k logs/sec   | Parallelizable verification    |
| KEK lookup             | ~0.1µs  | N/A             | O(1) map lookup from chain     |

**Configuration:**

No configuration required - signing is automatic when KEK chain available:

```go
// Automatic behavior based on dependencies
if kekChain != nil && auditSigner != nil {
    // Sign new audit logs automatically
    signature, kekID, err := auditSigner.Sign(ctx, auditLog, kekChain)
    auditLog.Signature = signature
    auditLog.KekID = &kekID
    auditLog.IsSigned = true
} else {
    // Legacy mode (no signing)
    auditLog.IsSigned = false
}
```

**Future enhancements:**

- **HSM integration**: Store signing keys in hardware security module for tamper-proof key storage
  - Would require KMS integration for signing key retrieval (see ADR 0010 for KMS patterns)
  - Benefit: Prevents signing key extraction even with database compromise
  
- **Batch verification API endpoint**: Add `POST /v1/audit-logs/verify` for programmatic integrity checks
  - Input: `{"start_date": "2026-02-01", "end_date": "2026-02-28"}`
  - Output: `{"total": 1234, "signed": 1200, "valid": 1200, "invalid": 0}`
  - Use case: Real-time integrity monitoring from external tools
  
- **Real-time integrity monitoring**: Continuous verification with alerting on signature failures
  - Periodic background job verifies recent logs (e.g., last 24 hours)
  - Alert on invalid signatures via email/Slack/PagerDuty
  - Use case: Detect tampering within hours instead of manual verification

## See also

- [Audit Logs API Documentation](../api/observability/audit-logs.md) - API schema with signature fields
- [CLI Commands - verify-audit-logs](../cli-commands.md#verify-audit-logs) - Verification command usage
- [AuditSigner Service Implementation](../../internal/auth/service/audit_signer.go) - HKDF + HMAC-SHA256 implementation
- [AuditLogUseCase Implementation](../../internal/auth/usecase/audit_log_usecase.go) - Automatic signing logic
- [verify-audit-logs CLI Command](../../cmd/app/commands/verify_audit_logs.go) - CLI verification implementation
- [Migration 000003 (PostgreSQL)](../../migrations/postgresql/000003_add_audit_log_signature.up.sql) - Schema changes
- [Migration 000003 (MySQL)](../../migrations/mysql/000003_add_audit_log_signature.up.sql) - Schema changes
- [ADR 0009: UUIDv7 for Identifiers](0009-uuidv7-for-identifiers.md) - Request ID embedded timestamps
- [NIST SP 800-107](https://csrc.nist.gov/pubs/sp/800/107/r1/final) - Recommendation for HMAC
- [RFC 5869 (HKDF)](https://www.rfc-editor.org/rfc/rfc5869.html) - HKDF specification
