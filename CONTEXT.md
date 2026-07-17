# CONTEXT

Domain vocabulary for the `secrets` service. Use these terms exactly in code,
docs, commit messages, and conversations. Drift weakens the seams.

## Cryptography

### Envelope encryption
The pattern in which a user payload is encrypted under a **Data Encryption Key
(DEK)**, and the DEK itself is encrypted under a **Key Encryption Key (KEK)**,
which is in turn protected by a **Master Key** held by an external **KMS**.
See [ADR-0001](docs/adr/0001-envelope-encryption-model.md).

### Master Key
A symmetric key held outside this service in a KMS (AWS, GCP, Azure, or
`localsecrets://` for development). Never stored in the database. Loaded at
boot via `KMSKeeper.Decrypt` and held in a `MasterKeyChain` for the process
lifetime.

### KEK — Key Encryption Key
A symmetric key that exists only to encrypt DEKs. Persisted in the `keks`
table as ciphertext (wrapped by a Master Key). Rotates on demand via the
KEK rotation worker. The set of all loaded KEKs is the `KekChain`; the
newest is the *active* KEK.

### DEK — Data Encryption Key
A symmetric key that encrypts exactly one piece of user data:
- in `secrets` and `tokenization`: one DEK per stored row (fresh each call);
- in `transit`: one DEK per Transit Key, reused across user requests.

Persisted in the `deks` table as ciphertext (wrapped by a KEK). Identified
by a UUIDv7 (`DekID`).

### AEAD
Authenticated Encryption with Associated Data. The service supports
`aes-256-gcm` and `chacha20-poly1305`. Algorithm is chosen at DEK creation
and recorded on the envelope; ciphertext from one algorithm cannot be
decrypted by another.

### Rewrap
Re-encrypting an existing DEK under a newer KEK without changing the
underlying DEK key material. Used by the KEK rotation worker so old
ciphertexts remain decryptable without bulk re-encryption.

## Modules

### Keyring
**The single module that owns envelope encryption.** Exposes a small interface
to feature modules; hides KEK chain, DEK lifecycle, AEAD selection, and KMS
calls behind it. Call sites do not know KEK from DEK.

- `Encrypt(ctx, plaintext) → Envelope` — fresh-DEK envelope encryption.
  Used by `secrets` and `tokenization`.
- `Decrypt(ctx, envelope) → plaintext` — inverse of `Encrypt`.
- `AllocateDek(ctx, alg) → DekHandle` — persists a DEK and returns an
  opaque handle. Used by `transit` once per Transit Key.
- `EncryptWith(ctx, handle, plaintext, aad) → (ciphertext, nonce)` — encrypt
  under a previously-allocated DEK. Used by `transit` per request.
- `DecryptWith(ctx, handle, ciphertext, nonce, aad) → plaintext` — inverse.
- `Rewrap(ctx, dekID)` — rewrap a DEK under the active KEK. Used by the
  rotation worker.

### Envelope
The value returned by `Keyring.Encrypt`. Contains `DekID`, `Ciphertext`,
`Nonce`, and `Algorithm`. Features persist all four fields; nothing else
about the DEK or KEK is leaked to callers.

### DekHandle
An opaque reference to a persistent DEK held by Keyring. Returned by
`AllocateDek`, accepted by `EncryptWith` / `DecryptWith`. Features store
only the handle's `DekID` and reload it on demand. Used to model the
`transit` flow where many user requests share one DEK.

### Transit Key
A named, long-lived encryption key managed via the transit HTTP API.
Backed internally by a single DEK (a DekHandle). Users call `encrypt` and
`decrypt` against the name; the DEK never leaves Keyring.

### Tokenization Key
A named encryption key associated with a token format (UUID, numeric,
alphanumeric, Luhn) and a determinism flag. Each tokenize call still uses
a fresh DEK via `Keyring.Encrypt` — the Tokenization Key itself is
metadata + format rules, not a long-lived crypto key.

### Secret
A path-addressed, versioned encrypted payload. Each version has its own
DEK (fresh per write). The path is the lookup key; the latest version is
the default read.

### KEK rotation worker
A background job that calls `Keyring.Rewrap` for every DEK not yet
encrypted under the active KEK. Runs after `Keyring.RotateKek`. Idempotent.

### Route Module
Each feature (`auth`, `secrets`, `transit`, `tokenization`) owns a `Module`
in its `http` package that registers that feature's HTTP routes next to its
handlers. Constructed by the composition root with its handlers, the
`Authorizer`, and business metrics already bound, it implements
`RouteRegistrar`. `internal/http` imports no feature package; the import
direction is feature → `internal/http`.

### RouteRegistrar
The seam `internal/http` exposes so it can mount features without knowing any
of them: `Register(v1 *gin.RouterGroup, mw RouteMiddlewares)`. `SetupRouter`
builds the global middleware chain and loops over a `[]RouteRegistrar`. Adding
an endpoint is a one-file change inside the owning feature.

### RouteMiddlewares
The shared per-route middleware bundle passed to every `Register`:
`Auth` and `RateLimit`, both plain `gin.HandlerFunc` so the seam carries no
feature type. `RateLimit` is nil when disabled; the auth module's IP-based
token rate limiter is captured by that module, not carried here.

## Authentication

### Client
An authentication principal. Holds a hashed `Secret`, an `IsActive` flag,
authorization `Policies`, and the lockout state (`FailedAttempts`,
`LockedUntil`). Owns the login state machine via `Client.AttemptLogin`.

### LoginOutcome
The value returned by `Client.AttemptLogin`. Carries the `Decision` and
the new `(FailedAttempts, LockedUntil)` tuple the caller must persist.
The Client itself is not mutated by `AttemptLogin`; the outcome is the
single source of truth for the post-attempt state.

### Decision
The enum variant of `LoginOutcome`. One of:
- `Authenticated` — secret matched on an active, unlocked client.
- `BadSecret` — client exists and is active, but the secret didn't match.
- `Locked` — `LockedUntil` is in the future; no attempt was counted.
- `Inactive` — `IsActive == false`; no attempt was counted.

### LockoutPolicy
The configuration passed to `Client.AttemptLogin`: `MaxAttempts` (zero
disables lockout) and `Duration` (how long a fresh lock lasts). Sourced
from `config.Config.LockoutMaxAttempts` / `LockoutDuration` at the seam.

## Storage

### `keks` table
Wrapped KEK material, ordered by `version`. The highest-version, non-revoked
row is the active KEK.

### `deks` table
Wrapped DEK material, joined to the `keks` row that wrapped them. Indexed
by `kek_id` to support the rotation worker's batch query.

### Transactions
All multi-row writes (creating a DEK + the row that references it) happen
inside a `database.TxManager` transaction propagated via `context.Context`
(per [ADR-0005](docs/adr/0005-context-based-transaction-management.md)).
`Keyring.Encrypt` and `Keyring.AllocateDek` join the caller's transaction
when one is present.

## Operations

### Retention sweep
The age-based deletion shared by six CLI commands (`purge-secrets`,
`purge-transit-keys`, `purge-tokenization-keys`, `clean-expired-tokens`,
`clean-audit-logs`, `purge-auth-tokens`). Each deletes rows older than a
`--days` threshold. The umbrella term covers both the soft-delete *purges*
and the expiry-based *cleans*; use "retention sweep" for the shared concept
and keep the per-command verb (`purge` / `clean`) in user-facing text.

A single deep module, `RunRetentionSweep` in `cmd/app/commands`, owns the
shape: validate `days` → log → `metrics.Track(module, op)` → run the
feature's sweep func (dry-run aware where supported) → format output as
text or JSON. Each command supplies a `SweepSpec`:

- `Verb` / `Subject` — the wording for output (e.g. `purge` /
  `"expired/revoked authentication token(s)"`).
- `MetricModule` / `MetricOp` — the `metrics.Track` labels.
- `SupportsDryRun` — `false` only for the auth-token sweep, whose
  `TokenUseCase.PurgeExpiredAndRevoked` takes no `dryRun`; the module then
  emits a "dry-run not supported" notice and deletes nothing.
- `Sweep` — a closure adapting the feature usecase's sweep method
  (`PurgeDeleted`, `CleanupExpired`, `DeleteOlderThan`,
  `PurgeExpiredAndRevoked`), which have no shared interface.
