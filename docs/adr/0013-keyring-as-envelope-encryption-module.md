# ADR 0013: Keyring as Envelope Encryption Module

> Status: accepted
> Date: 2026-05-23

## Context

[ADR 0001](0001-envelope-encryption-model.md) defines the cryptographic
hierarchy (Master Key → KEK → DEK → data) but does not say where the
implementation lives. Initially the model was scattered across:

- `internal/crypto/domain` — types for KEK, DEK, MasterKeyChain
- `internal/crypto/service` — AEAD, AEADManager, KeyManager
- `internal/crypto/usecase` — KekUseCase, DekUseCase
- `internal/crypto/repository` — KEK and DEK persistence

Each feature module (`secrets`, `transit`, `tokenization`) imported
four of these and reimplemented the same six-step envelope dance per
operation:

```
get active KEK → CreateDek under KEK → persist DEK → DecryptDek →
CreateCipher with DEK key → Encrypt plaintext under cipher
```

This had three concrete costs:

1. **The interface is the pattern.** ADR-0001 was enforced by
   convention, not by a module boundary. Three near-identical copies of
   the dance existed (secrets, transit, tokenization), each with its
   own subtle differences.
2. **Single-adapter interfaces.** Every layer of the dance exposed
   interfaces (DekRepository, AEADManager, KeyManager) with exactly one
   production adapter and one generated mock. The mocks were the
   second "adapter," but they only existed for test isolation — they
   were not a real seam.
3. **Constructor bloat.** Feature usecases took 6–8 dependencies, most
   of them crypto plumbing rather than feature business logic.

## Decision

Introduce a single deep module, `internal/keyring`, that owns envelope
encryption end-to-end. Features depend only on the `Keyring`
interface and the small `Envelope` / `DekHandle` value types.

The interface exposes two encryption shapes:

```go
// Fresh-DEK envelope: used by secrets and tokenization where each
// stored item gets its own DEK.
Encrypt(ctx, plaintext) → Envelope
Decrypt(ctx, env)        → plaintext

// Persistent DEK: used by transit and tokenization-keys where a single
// DEK wraps many plaintexts over its lifetime.
AllocateDek(ctx, alg)                              → DekHandle
EncryptWith(ctx, handle, plaintext, aad)           → (ciphertext, nonce)
DecryptWith(ctx, handle, ciphertext, nonce, aad)   → plaintext

// KEK rotation:
Rewrap(ctx, dekID)              // single DEK
RewrapAll(ctx, batchSize)       // batch worker
ActiveKekID()                   // safety check for the rotation CLI
```

`Envelope` is `{DekID, Ciphertext, Nonce}`. Features persist these
three fields and nothing else about crypto state. `DekHandle` is an
opaque `{DekID}` returned by `AllocateDek` and consumed by the
EncryptWith/DecryptWith pair.

Keyring is constructed once at boot from a loaded `KekChain`, the
shared `KeyManager` and `AEADManager`, and a concrete
`*cryptoRepository.DekRepository`. The KEK chain, AEAD selection, DEK
persistence, and key zeroing all live inside the module.

A `keyring.Fake` is shipped alongside the production implementation. It
is deterministic, in-memory, and gives features a real second adapter
— making the seam a genuine seam (per the "one adapter = hypothetical
seam, two adapters = real seam" principle) rather than only mock
scaffolding.

`internal/crypto/usecase/DekUseCase` and the per-feature
`DekRepository` interfaces are deleted; their behaviour moves into the
Keyring. `internal/crypto/usecase/KekUseCase` remains for the KEK
lifecycle CLI (Create, Rotate, Unwrap), which runs outside the
request-time path Keyring serves.

## Consequences

### Positive

- ADR-0001's envelope-encryption hierarchy is enforced by a module
  boundary, not by per-feature convention.
- Feature usecases shed 4–5 of their crypto dependencies. Constructor
  signatures shrink dramatically (secrets: 8→4, tokenization key: 5→3,
  transit: 6→3).
- `keyring.Fake` is the first real second adapter in the codebase.
  Feature unit tests now exercise behaviour ("a secret round-trips")
  rather than asserting the call sequence against five mocks. The
  full feature test suite shrank by ~5,000 lines across secrets,
  tokenization, and transit.
- Adding a new feature that needs envelope encryption is now trivial:
  inject Keyring, call `Encrypt`/`Decrypt`. No knowledge of KEK, DEK,
  AEAD, or KMS required at the call site.
- Memory zeroing of DEK plaintext material is centralised in one
  place. Features cannot leak a DEK key by forgetting to call
  `Zero()`.

### Negative

- Keyring takes a concrete `*cryptoRepository.DekRepository` rather
  than the narrower `internal/crypto/usecase.DekRepository` interface.
  This is fine — Keyring needs `Create + Get + Update +
  GetBatchNotKekID` and only the concrete struct provides all four
  today.
- The boot-time `KekChain` is loaded once and cached on the running
  Keyring. The rewrap CLI must therefore run in a freshly-booted
  process after KEK rotation; this is enforced by
  `Keyring.ActiveKekID()` matching the operator-provided
  `--kek-id` argument.
- The `nonceSize` used in transit's wire format (12 bytes) is
  hardcoded in `internal/transit/usecase`. Both currently-supported
  algorithms (AES-256-GCM, ChaCha20-Poly1305) use 12-byte nonces; a
  future algorithm with a different nonce size would need either a
  size accessor on Keyring or a small refactor of the transit blob
  format.

## See also

- [ADR 0001: Envelope Encryption Model](0001-envelope-encryption-model.md)
- [ADR 0002: Transit Versioned Ciphertext Contract](0002-transit-versioned-ciphertext-contract.md) - Transit wire format that sits on top of Keyring's EncryptWith/DecryptWith
- [ADR 0005: Context-Based Transaction Management](0005-context-based-transaction-management.md) - Keyring's Encrypt/AllocateDek join the caller's transaction via ctx
- [`CONTEXT.md`](../../CONTEXT.md) - Keyring, Envelope, DekHandle, Rewrap vocabulary
