package domain

// Algorithm represents the cryptographic algorithm used for encryption.
// Supported algorithms provide AEAD (Authenticated Encryption with Associated Data).
type Algorithm string

const (
	// AESGCM represents AES-256-GCM authenticated encryption (optimal with AES-NI hardware).
	AESGCM Algorithm = "aes-gcm"

	// ChaCha20 represents ChaCha20-Poly1305 authenticated encryption (optimal without AES-NI).
	ChaCha20 Algorithm = "chacha20-poly1305"
)
