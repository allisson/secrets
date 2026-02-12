package commands

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// RunCreateMasterKey generates a cryptographically secure 32-byte master key for envelope encryption.
// Creates the root key used to encrypt all KEKs. Key material is zeroed from memory after encoding.
// If keyID is empty, generates a default ID in format "master-key-YYYY-MM-DD".
//
// Output format: MASTER_KEYS="<keyID>:<base64-encoded-key>" and ACTIVE_MASTER_KEY_ID="<keyID>"
//
// Security: Store output securely (secrets manager/KMS), never commit to version control, rotate
// every 90 days. For production, consider using a proper KMS instead of environment variables.
func RunCreateMasterKey(keyID string) error {
	// Generate default key ID if not provided
	if keyID == "" {
		keyID = fmt.Sprintf("master-key-%s", time.Now().Format("2006-01-02"))
	}

	// Generate a cryptographically secure 32-byte master key
	masterKey := make([]byte, 32)
	if _, err := rand.Read(masterKey); err != nil {
		return fmt.Errorf("failed to generate master key: %w", err)
	}

	// Encode the master key to base64
	encodedKey := base64.StdEncoding.EncodeToString(masterKey)

	// Zero out the master key from memory for security
	for i := range masterKey {
		masterKey[i] = 0
	}

	// Print the environment variable configuration
	fmt.Println("# Master Key Configuration")
	fmt.Println("# Copy these environment variables to your .env file or secrets manager")
	fmt.Println()
	fmt.Printf("MASTER_KEYS=\"%s:%s\"\n", keyID, encodedKey)
	fmt.Printf("ACTIVE_MASTER_KEY_ID=\"%s\"\n", keyID)
	fmt.Println()
	fmt.Println("# For multiple master keys (key rotation), use comma-separated format:")
	fmt.Printf("# MASTER_KEYS=\"%s:%s,new-key:base64-encoded-new-key\"\n", keyID, encodedKey)
	fmt.Println("# ACTIVE_MASTER_KEY_ID=\"new-key\"")

	return nil
}
