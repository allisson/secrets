/*
Package tokenization provides secure, format-preserving tokenization for sensitive data.

The tokenization module enables replacing sensitive values (credit cards, SSNs, PII) with
non-sensitive tokens while maintaining format compatibility with existing systems.

# Architecture

The module follows Clean Architecture principles:
  - domain: Core entities (TokenizationKey, Token) and business rules
  - usecase: Business logic orchestration
  - service: Token generation algorithms (UUID, Numeric, Luhn, Alphanumeric)
  - repository: Data persistence (MySQL, PostgreSQL)
  - http: HTTP handlers and DTOs

# Security Model

Uses a three-tier key hierarchy:
  - Master Key (MK): Root key stored in HSM/KMS
  - Key Encryption Key (KEK): Encrypts DEKs, derived from MK
  - Data Encryption Key (DEK): Encrypts plaintexts, encrypted with KEK

Plaintext is encrypted with AES-GCM or ChaCha20-Poly1305 AEAD ciphers.

# Token Formats

  - UUID: UUIDv7 tokens (standard format)
  - Numeric: Numeric-only tokens (configurable length)
  - Luhn-Preserving: Maintains Luhn check digit (for credit cards)
  - Alphanumeric: Alphanumeric tokens (A-Z, 0-9)

# Deterministic vs Non-Deterministic

Deterministic Mode (IsDeterministic: true):
  - Same plaintext always produces same token
  - Enables consistent token reuse
  - Risk: Frequency analysis possible

Non-Deterministic Mode (IsDeterministic: false):
  - Same plaintext produces different tokens each time
  - Maximum security (recommended)
  - Prevents frequency analysis

# Basic Usage

Create a tokenization key:

	key, err := tokenizationKeyUseCase.Create(
	    ctx,
	    "credit-card-key",
	    domain.FormatNumeric,
	    false, // non-deterministic
	    cryptoDomain.AESGCM,
	)

Tokenize sensitive data:

	plaintext := []byte("4532123456789012")
	metadata := map[string]any{"last4": "9012"}
	expiresAt := time.Now().Add(24 * time.Hour)

	token, err := tokenizationUseCase.Tokenize(
	    ctx,
	    "credit-card-key",
	    plaintext,
	    metadata,
	    &expiresAt,
	)

Detokenize to retrieve original value:

	plaintext, metadata, err := tokenizationUseCase.Detokenize(ctx, token.Token)
	defer cryptoDomain.Zero(plaintext) // CRITICAL: Zero plaintext after use

# Key Rotation

Create a new version of an existing key:

	newKey, err := tokenizationKeyUseCase.Rotate(
	    ctx,
	    "credit-card-key",
	    domain.FormatNumeric,
	    false,
	    cryptoDomain.AESGCM,
	)
	// newKey.Version = 2 (old tokens still work with version 1)

# Token Lifecycle

Validate token:

	isValid, err := tokenizationUseCase.Validate(ctx, "1234567890123456")

Revoke token:

	err = tokenizationUseCase.Revoke(ctx, "1234567890123456")

Cleanup expired tokens:

	count, err := tokenizationUseCase.CleanupExpired(ctx, 30, false)

# Security Best Practices

1. Always zero plaintext after use:

	plaintext, _, err := tokenizationUseCase.Detokenize(ctx, token)
	defer cryptoDomain.Zero(plaintext)

2. Never store sensitive data in metadata:

	// ✅ Good: Only display data
	metadata := map[string]any{"last4": "9012", "exp": "12/25"}

	// ❌ Bad: Sensitive data in metadata
	metadata := map[string]any{"full_number": "4532123456789012"}

3. Implement rate limiting on Tokenize():

	// Recommended: 100 requests/minute per user

4. Use appropriate determinism:

	// Use deterministic for analytics/joins
	// Use non-deterministic for maximum security (default)

# Constraints

  - Maximum plaintext size: 64 KB (enforced automatically)
  - Maximum token length: 255 characters (format-preserving only)
  - Minimum Luhn token length: 2 characters

# Compliance

Supports compliance with:
  - PCI DSS 4.0 (Requirement 3)
  - GDPR (Article 4, Article 25)
  - HIPAA (PHI de-identification)
  - CCPA (Consumer data protection)

For complete documentation, see README.md.
*/
package tokenization
