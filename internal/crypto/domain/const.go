package domain

// Algorithm represents the cryptographic algorithm used for encryption.
//
// All supported algorithms provide Authenticated Encryption with Associated Data (AEAD),
// ensuring both confidentiality and authenticity of encrypted data. AEAD prevents both
// unauthorized reading and tampering with encrypted data.
//
// Algorithm selection guidelines:
//   - Use AESGCM on modern CPUs with AES-NI hardware acceleration
//   - Use ChaCha20 on mobile devices or systems without AES-NI
//   - Both provide equivalent 256-bit security when used correctly
type Algorithm string

const (
	// AESGCM represents the AES-256-GCM authenticated encryption algorithm.
	//
	// AES-GCM (Advanced Encryption Standard in Galois/Counter Mode) combines
	// AES encryption with GMAC authentication. It uses a 256-bit key and
	// provides excellent performance on hardware with AES-NI acceleration.
	//
	// Key features:
	//   - 256-bit key size for maximum security
	//   - 12-byte nonce (96 bits)
	//   - 16-byte authentication tag
	//   - Hardware acceleration on modern CPUs
	AESGCM Algorithm = "aes-gcm"

	// ChaCha20 represents the ChaCha20-Poly1305 authenticated encryption algorithm.
	//
	// ChaCha20-Poly1305 combines the ChaCha20 stream cipher with the Poly1305 MAC
	// for authentication. It's designed for high performance on platforms without
	// AES hardware acceleration and is resistant to timing attacks.
	//
	// Key features:
	//   - 256-bit key size
	//   - 12-byte nonce (96 bits)
	//   - 16-byte authentication tag
	//   - Constant-time implementation
	//   - Excellent software performance
	ChaCha20 Algorithm = "chacha20-poly1305"
)
