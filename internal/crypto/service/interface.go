// Package service provides cryptographic service interfaces and implementations.
//
// This package implements the service layer for envelope encryption, providing
// concrete implementations of authenticated encryption algorithms and key management.
//
// # Services Overview
//
// AEADManagerService: Factory for creating AEAD cipher instances.
// Supports AES-256-GCM and ChaCha20-Poly1305 algorithms.
//
// KeyManagerService: Manages the lifecycle of KEKs and DEKs in envelope encryption.
// Handles key generation, encryption, and decryption operations.
//
// AESGCMCipher: Implements AEAD using AES-256-GCM with hardware acceleration support.
//
// ChaCha20Poly1305Cipher: Implements AEAD using ChaCha20-Poly1305 for platforms
// without AES hardware acceleration.
//
// # Usage Example
//
//	// Create services
//	aeadManager := NewAEADManager()
//	keyManager := NewKeyManager(aeadManager)
//
//	// Load master keys
//	masterKeyChain, err := domain.LoadMasterKeyChainFromEnv()
//	if err != nil {
//	    return err
//	}
//	defer masterKeyChain.Close()
//
//	// Get active master key
//	activeMasterKey, _ := masterKeyChain.Get(masterKeyChain.ActiveMasterKeyID())
//
//	// Create KEK
//	kek, err := keyManager.CreateKek(activeMasterKey, domain.AESGCM)
//	if err != nil {
//	    return err
//	}
//
//	// Create DEK for encrypting data
//	dek, err := keyManager.CreateDek(kek, domain.AESGCM)
//	if err != nil {
//	    return err
//	}
//
//	// Create cipher and encrypt data
//	cipher, err := aeadManager.CreateCipher(kek.Key, domain.AESGCM)
//	if err != nil {
//	    return err
//	}
//	ciphertext, nonce, err := cipher.Encrypt(plaintext, nil)
//
// # Thread Safety
//
// All service implementations are stateless and thread-safe. Multiple goroutines
// can safely use the same service instances for concurrent operations.
//
// # Algorithm Selection
//
//   - Use AESGCM on servers and modern CPUs with AES-NI hardware acceleration
//   - Use ChaCha20 on mobile devices, embedded systems, or platforms without AES-NI
//   - Both provide equivalent 256-bit security when properly implemented
//
// # Dependencies
//
// The service layer depends on the crypto/domain package for models and errors,
// following Clean Architecture principles. Services should be injected as
// dependencies rather than instantiated directly in business logic.
package service

import (
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// AEAD defines the interface for Authenticated Encryption with Associated Data.
//
// AEAD encryption provides both confidentiality and authenticity guarantees,
// protecting against unauthorized access and tampering. Implementations ensure
// that any modification to the ciphertext or AAD will be detected during decryption.
//
// Security requirements:
//   - Nonces must be unique for each encryption with the same key
//   - Keys should be at least 256 bits for strong security
//   - The same AAD used during encryption must be provided during decryption
//
// Implementations: AESGCMCipher, ChaCha20Poly1305Cipher
type AEAD interface {
	// Encrypt encrypts plaintext with optional additional authenticated data (AAD).
	//
	// The AAD parameter allows binding the ciphertext to additional context
	// (e.g., user ID, record ID, metadata) without encrypting it. This prevents
	// ciphertext from being used in a different context even if intercepted.
	//
	// A unique nonce is automatically generated for each encryption operation.
	// The nonce must be stored alongside the ciphertext for later decryption.
	//
	// Parameters:
	//   - plaintext: The data to encrypt (can be empty)
	//   - aad: Additional data to authenticate but not encrypt (can be nil)
	//
	// Returns:
	//   - ciphertext: The encrypted data including authentication tag
	//   - nonce: The randomly generated nonce used for this encryption
	//   - err: Any error encountered during encryption or nonce generation
	Encrypt(plaintext, aad []byte) (ciphertext, nonce []byte, err error)

	// Decrypt decrypts ciphertext using the provided nonce and AAD.
	//
	// This method verifies the authentication tag before returning plaintext,
	// ensuring the ciphertext hasn't been tampered with. If authentication fails,
	// no plaintext is returned to prevent processing of modified data.
	//
	// Parameters:
	//   - ciphertext: The encrypted data to decrypt (including authentication tag)
	//   - nonce: The nonce that was used during encryption
	//   - aad: The same additional data provided during encryption (can be nil)
	//
	// Returns:
	//   - plaintext: The decrypted data
	//   - err: Authentication failure, invalid nonce, or other decryption errors
	Decrypt(ciphertext, nonce, aad []byte) ([]byte, error)
}

// AEADManager defines the interface for creating AEAD cipher instances.
//
// This interface acts as a factory for creating authenticated encryption cipher
// instances. It abstracts the cipher creation logic, allowing callers to obtain
// cipher instances without knowing the specific implementation details.
//
// The manager supports two algorithms:
//   - AESGCM: AES-256-GCM (best on hardware with AES-NI acceleration)
//   - ChaCha20: ChaCha20-Poly1305 (best on mobile/embedded systems)
//
// Both algorithms provide authenticated encryption with associated data (AEAD),
// ensuring confidentiality and authenticity of encrypted data.
//
// Usage pattern:
//  1. Create an AEADManager instance
//  2. Call CreateCipher with a 32-byte key and desired algorithm
//  3. Use the returned AEAD cipher to encrypt/decrypt data
//
// Example:
//
//	manager := NewAEADManager()
//	cipher, err := manager.CreateCipher(dekKey, cryptoDomain.AESGCM)
//	if err != nil {
//	    return err
//	}
//	ciphertext, nonce, err := cipher.Encrypt(plaintext, aad)
//
// Implementation: AEADManagerService
type AEADManager interface {
	// CreateCipher creates an AEAD cipher instance for the specified algorithm.
	//
	// This factory method instantiates the appropriate cipher implementation
	// based on the provided algorithm. The key must be exactly 32 bytes (256 bits)
	// for both supported algorithms.
	//
	// The returned cipher is stateless and thread-safe, allowing concurrent
	// encryption/decryption operations with the same cipher instance.
	//
	// Parameters:
	//   - key: The encryption key (must be exactly 32 bytes)
	//   - alg: The algorithm to use (AESGCM or ChaCha20)
	//
	// Returns:
	//   - An AEAD cipher instance ready for encryption/decryption operations
	//   - ErrInvalidKeySize if key is not 32 bytes
	//   - ErrUnsupportedAlgorithm if algorithm is not supported
	CreateCipher(key []byte, alg cryptoDomain.Algorithm) (AEAD, error)
}

// KeyManager defines the interface for managing cryptographic keys in an envelope encryption scheme.
//
// Envelope encryption is a multi-tier key hierarchy that provides efficient key management
// and rotation capabilities. The key hierarchy works as follows:
//
//	Master Key (stored in KMS or secure storage)
//	    ↓ encrypts
//	KEK (Key Encryption Key - stored in database)
//	    ↓ encrypts
//	DEK (Data Encryption Key - stored with encrypted data)
//	    ↓ encrypts
//	Application Data
//
// This design provides several benefits:
//   - Key rotation: Only the KEK needs to be rotated, not individual DEKs
//   - Performance: DEKs can be cached in memory after decryption
//   - Security: Master key is never used directly to encrypt application data
//   - Scalability: Each piece of data can have its own DEK
//
// Typical workflow:
//  1. Create a KEK encrypted with the master key (done once or during key rotation)
//  2. For each piece of data to encrypt, create a DEK encrypted with the KEK
//  3. Use the DEK to encrypt the actual data
//  4. Store the encrypted DEK alongside the encrypted data
//  5. To decrypt: retrieve KEK, decrypt DEK, then decrypt data
//
// Implementation: KeyManagerService
type KeyManager interface {
	// CreateKek creates a new Key Encryption Key encrypted with the master key.
	//
	// The KEK is generated as a random 32-byte (256-bit) key and encrypted using
	// the master key with the specified algorithm. The encrypted KEK should be
	// stored in a database for later use in encrypting/decrypting DEKs.
	//
	// The MasterKey pointer allows the service to track which master key was used
	// to encrypt each KEK, enabling proper key rotation and decryption when multiple
	// master keys are in use.
	//
	// Parameters:
	//   - masterKey: The MasterKey used to encrypt the KEK (contains both ID and 32-byte key)
	//   - alg: The encryption algorithm (AESGCM or ChaCha20)
	//
	// Returns:
	//   - A Kek struct with the encrypted key, nonce, master key ID, and metadata
	//   - An error if the algorithm is unsupported or encryption fails
	CreateKek(
		masterKey *cryptoDomain.MasterKey,
		alg cryptoDomain.Algorithm,
	) (cryptoDomain.Kek, error)

	// DecryptKek decrypts a Key Encryption Key using the master key.
	//
	// This method recovers the plaintext KEK from its encrypted form stored
	// in the database. The decrypted KEK is needed to encrypt and decrypt DEKs.
	// The decrypted KEK should be kept in memory only and never persisted.
	//
	// This is typically used when loading KEKs from the database at application
	// startup or when accessing a KEK that hasn't been cached yet. For performance,
	// decrypted KEKs can be cached in memory using a KekChain.
	//
	// Parameters:
	//   - kek: The encrypted Key Encryption Key to decrypt
	//   - masterKey: The MasterKey used to decrypt the KEK (must match kek.MasterKeyID)
	//
	// Returns:
	//   - The decrypted KEK as a 32-byte slice
	//   - ErrDecryptionFailed if decryption fails or ciphertext is tampered
	DecryptKek(kek *cryptoDomain.Kek, masterKey *cryptoDomain.MasterKey) ([]byte, error)

	// CreateDek creates a new Data Encryption Key encrypted with the KEK.
	//
	// The DEK is generated as a random 32-byte (256-bit) key and encrypted using
	// the KEK with the specified algorithm. The encrypted DEK should be stored
	// alongside the data it encrypts (e.g., in the same database record).
	//
	// Each piece of encrypted data should have its own DEK for maximum security
	// and to facilitate individual data deletion (crypto shredding).
	//
	// Parameters:
	//   - kek: The Key Encryption Key used to encrypt the DEK (must have Key field populated)
	//   - alg: The encryption algorithm for the DEK (AESGCM or ChaCha20)
	//
	// Returns:
	//   - A Dek struct with the encrypted key, nonce, and KEK reference
	//   - An error if the algorithm is unsupported or encryption fails
	CreateDek(kek *cryptoDomain.Kek, alg cryptoDomain.Algorithm) (cryptoDomain.Dek, error)

	// DecryptDek decrypts a Data Encryption Key using the KEK.
	//
	// This method recovers the plaintext DEK so it can be used to decrypt
	// application data. The decrypted DEK should only be kept in memory
	// and should never be persisted in plaintext form.
	//
	// For performance, decrypted DEKs can be cached in memory with an
	// appropriate expiration policy.
	//
	// Parameters:
	//   - dek: The encrypted Data Encryption Key to decrypt
	//   - kek: The Key Encryption Key used to decrypt the DEK (must have Key field populated)
	//
	// Returns:
	//   - The decrypted DEK as a 32-byte slice
	//   - ErrDecryptionFailed if decryption fails or ciphertext is tampered
	DecryptDek(dek *cryptoDomain.Dek, kek *cryptoDomain.Kek) ([]byte, error)
}
