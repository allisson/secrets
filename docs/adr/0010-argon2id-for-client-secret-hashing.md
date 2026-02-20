# ADR 0010: Argon2id for Client Secret Hashing

> Status: accepted
> Date: 2026-02-10

## Context

The application requires secure password hashing for client authentication credentials with the following needs:

- **Resistance to offline attacks**: Protect hashed secrets if database is compromised
- **GPU/ASIC resistance**: Prevent brute-force attacks using specialized hardware
- **Configurable work factor**: Balance security and performance as hardware evolves
- **Standard compliance**: Use well-vetted algorithms recommended by security standards
- **Future-proof**: Algorithm should remain secure for 5+ years
- **Performance**: Hash verification must complete within acceptable latency (< 500ms)

Client secrets are used for Bearer token issuance via `POST /v1/token` endpoint. If the database is compromised, attackers gain access to hashed secrets and can mount offline brute-force attacks without rate limiting.

Key security considerations:

- **Password strength**: Client secrets are 32-character random alphanumeric strings (generated, not user-chosen)
- **Attack vectors**: Offline brute-force (post-breach), rainbow tables, GPU/ASIC cracking
- **Threat model**: Attacker gains read access to database, attempts to recover plaintext secrets

## Decision

Adopt **Argon2id** with **PolicyModerate** configuration using the `go-pwdhash` library:

**Algorithm choice:**

- **Argon2id**: Hybrid mode combining Argon2i (side-channel resistant) and Argon2d (GPU-resistant)
- **Winner of Password Hashing Competition (PHC) 2015**
- **Recommended by OWASP** for password storage as of 2024

**Configuration (PolicyModerate):**

```go
// via github.com/allisson/go-pwdhash v0.3.1
hasher, _ := pwdhash.NewArgon2idHasher(pwdhash.PolicyModerate)

// Internally uses:
// - Memory: 64 MiB (m=65536)
// - Iterations: 3 (t=3)
// - Parallelism: 4 threads (p=4)
// - Salt length: 16 bytes
// - Hash length: 32 bytes
```

**Output format (PHC string):**

```text
$argon2id$v=19$m=65536,t=3,p=4$<base64-salt>$<base64-hash>
```

**Usage pattern:**

```go
// Hash client secret on creation
hash, err := secretHasher.Hash(ctx, clientSecret)

// Verify client secret on authentication
valid, err := secretHasher.Verify(ctx, clientSecret, storedHash)
```

**Policy rationale:**

- **PolicyModerate** chosen over PolicyConservative/PolicyParanoid for balance:
  - Security: 64 MiB memory requirement resists GPU attacks
  - Performance: ~100-200ms verification time (acceptable for auth endpoint)
  - Hardware: Works on constrained environments (containers with 512 MiB+ memory)

## Alternatives Considered

### 1. bcrypt

Industry-standard algorithm based on Blowfish cipher.

**Rejected because:**

- **GPU-vulnerable**: Low memory usage (4 KiB) allows efficient GPU/ASIC implementation
- **Limited work factor**: Maximum cost factor 31 (2^31 iterations) may become insufficient
- **32-byte output limit**: Cannot increase hash length for future quantum resistance
- **No memory hardness**: Attackers can parallelize on GPUs with minimal memory per thread
- **Aging algorithm**: Designed in 1999, shows weakness against modern hardware (GPUs with 10,000+ cores)

**Performance comparison:**

- bcrypt cost 12: ~300ms CPU time, 4 KiB memory
- Argon2id (Moderate): ~150ms CPU time, 64 MiB memory → **16,000x more memory required**

### 2. scrypt

Memory-hard key derivation function designed for hardware resistance.

**Rejected because:**

- **No side-channel resistance**: Vulnerable to cache-timing attacks (Argon2i/Argon2id address this)
- **Less vetted**: Not winner of PHC, less cryptanalysis than Argon2
- **Complex parameter tuning**: Requires setting N (CPU/memory cost), r (block size), p (parallelism) independently
- **No standard format**: No PHC string format standardization (portability issues)
- **Older design**: Created in 2009, superseded by Argon2 (2015)

### 3. PBKDF2-SHA256

NIST-approved key derivation function.

**Rejected because:**

- **No memory hardness**: Trivially parallelizable on GPUs (same weakness as bcrypt)
- **High iteration count required**: 600,000+ iterations needed for equivalent security (slower than Argon2id)
- **No built-in salt handling**: Requires manual salt generation and storage
- **Designed for key derivation**: Not purpose-built for password hashing (Argon2 designed specifically for this)
- **GPU-vulnerable**: Attackers can run millions of parallel attempts on GPU

### 4. Argon2i (Data-Independent Mode)

Side-channel resistant variant of Argon2.

**Rejected because:**

- **Weaker GPU resistance**: Slightly more vulnerable to GPU attacks than Argon2d/Argon2id
- **No hybrid benefit**: Argon2id provides side-channel resistance AND GPU resistance
- **Same implementation complexity**: No simplicity advantage over Argon2id
- **Not recommended**: PHC recommends Argon2id for password hashing (Argon2i for key derivation)

### 5. Argon2d (Data-Dependent Mode)

Maximum GPU-resistant variant of Argon2.

**Rejected because:**

- **Side-channel vulnerable**: Data-dependent memory access patterns leak information via cache timing
- **Not recommended for passwords**: PHC discourages Argon2d for password hashing (use Argon2id instead)
- **Same memory cost**: No performance benefit over Argon2id
- **Weaker security model**: Side-channel attacks more practical than brute-force in some scenarios

## Consequences

**Benefits:**

- **GPU/ASIC resistance**: 64 MiB memory requirement makes GPU attacks economically infeasible
- **Side-channel resistance**: Argon2id's hybrid mode protects against cache-timing attacks
- **Future-proof**: Winner of PHC, recommended by OWASP, designed for long-term security
- **Configurable security**: Can upgrade to PolicyConservative (128 MiB) or PolicyParanoid (256 MiB) if needed
- **Standard format**: PHC string format enables portability and version detection
- **Salt handling**: Library manages salt generation (16 bytes, cryptographically random)
- **Strong secrets**: Generated 32-character alphanumeric secrets have ~190 bits entropy (brute-force infeasible)

**Trade-offs:**

- **Memory usage**: 64 MiB per hash operation (vs 4 KiB for bcrypt)
  - Mitigated by: Hashing only happens during client creation and authentication (low frequency)
  - Acceptable: Modern containers/VMs have ample memory for this workload
- **Verification latency**: ~100-200ms per hash verification (vs ~300ms for bcrypt cost 12)
  - Mitigated by: Rate limiting on `/v1/token` endpoint prevents high-frequency verification
  - Acceptable: Authentication latency acceptable for server-to-server OAuth2-style flows
- **CPU usage**: Higher CPU cost than bcrypt for equivalent security
  - Mitigated by: Low authentication request volume (infrequent token issuance)
  - Acceptable: CPU cost justified by superior security properties

**Limitations:**

- **No hardware acceleration**: No AES-NI or other CPU instruction support (pure software implementation)
  - Acceptable: Memory hardness more important than CPU optimization for password hashing
- **Container memory requirements**: Minimum 512 MiB memory required per container (64 MiB × ~8 concurrent requests)
  - Acceptable: Production deployments already provision 1+ GiB per container
- **Incompatible with legacy bcrypt hashes**: Cannot verify existing bcrypt hashes without migration
  - Not applicable: New project, no legacy hashes to migrate

**Security characteristics:**

- **Offline attack cost**: ~$100,000 per hash cracked with 1000-GPU cluster (vs ~$1,000 for bcrypt)
- **Time to crack (96-bit secret)**: ~10^29 years at 1 million GPU-years compute (infeasible)
- **Side-channel resistance**: Cache-timing attacks mitigated by Argon2id's hybrid mode
- **Quantum resistance**: 256-bit output exceeds Grover's algorithm requirement (128 bits post-quantum)

**Configuration policy rationale:**

| Policy       | Memory  | Time   | Use Case                                                 |
|--------------|---------|--------|----------------------------------------------------------|
| Moderate     | 64 MiB  | ~150ms | **Production default** (balance security/performance)    |
| Conservative | 128 MiB | ~300ms | High-security environments (financial, healthcare)       |
| Paranoid     | 256 MiB | ~600ms | Maximum security (government, defense)                   |

**Current choice: Moderate** because:

- Client secrets are 32-character random strings (not weak user passwords)
- Rate limiting on auth endpoint limits attack surface
- 64 MiB memory cost already 16,000x harder than bcrypt
- Performance acceptable for API authentication latency requirements

**Migration path:**

- Can increase to Conservative/Paranoid by rehashing on next authentication
- PHC string format embeds parameters (enables gradual migration)
- Library supports policy detection from hash string

## See also

- [Argon2 RFC 9106](https://www.rfc-editor.org/rfc/rfc9106.html)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [Password Hashing Competition](https://www.password-hashing.net/)
- [go-pwdhash library](https://github.com/allisson/go-pwdhash)
- [Token authentication implementation](../../internal/auth/service/secret_service.go)
- [ADR 0006: Dual-Scope Rate Limiting Strategy](0006-dual-scope-rate-limiting-strategy.md) - Rate limiting on auth endpoint
