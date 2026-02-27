package commands

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	cryptoService "github.com/allisson/secrets/internal/crypto/service"
)

// RunCreateMasterKey generates a cryptographically secure 32-byte master key for envelope encryption.
// Creates the root key used to encrypt all KEKs. Key material is zeroed from memory after encoding.
// If keyID is empty, generates a default ID in format "master-key-YYYY-MM-DD".
//
// KMS parameters (kmsProvider and kmsKeyURI) are required. The master key is encrypted with KMS before output.
// For local development, use kmsProvider="localsecrets" with kmsKeyURI="base64key://...".
//
// Output format:
//   - MASTER_KEYS="<keyID>:<base64-encoded-kms-ciphertext>"
//   - KMS_PROVIDER="<provider>"
//   - KMS_KEY_URI="<uri>"
//
// Security: Never use localsecrets provider in production. Use cloud KMS providers (gcpkms, awskms, azurekeyvault).
func RunCreateMasterKey(keyID, kmsProvider, kmsKeyURI string) error {
	ctx := context.Background()

	// Validate required KMS parameters
	if kmsProvider == "" || kmsKeyURI == "" {
		return fmt.Errorf(
			"--kms-provider and --kms-key-uri are required\n\nFor local development, use:\n  --kms-provider=localsecrets --kms-key-uri=\"base64key://<32-byte-base64-key>\"\n\nFor production, use cloud KMS providers:\n  --kms-provider=gcpkms --kms-key-uri=\"gcpkms://projects/.../cryptoKeys/...\"\n  --kms-provider=awskms --kms-key-uri=\"awskms:///alias/...\"\n  --kms-provider=azurekeyvault --kms-key-uri=\"azurekeyvault://...\"",
		)
	}

	// Generate default key ID if not provided
	if keyID == "" {
		keyID = fmt.Sprintf("master-key-%s", time.Now().Format("2006-01-02"))
	}

	// Generate a cryptographically secure 32-byte master key
	masterKey := make([]byte, 32)
	if _, err := rand.Read(masterKey); err != nil {
		return fmt.Errorf("failed to generate master key: %w", err)
	}

	fmt.Println("# KMS Mode: Encrypting master key with KMS")
	fmt.Printf("# KMS Provider: %s\n", kmsProvider)
	fmt.Println()

	// Create KMS service and open keeper
	kmsService := cryptoService.NewKMSService()
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	if err != nil {
		return fmt.Errorf("failed to open KMS keeper: %w", err)
	}
	defer func() {
		if closeErr := keeperInterface.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close KMS keeper: %v\n", closeErr)
		}
	}()

	// Type assert to get Encrypt method (needed for encryption)
	keeper, ok := keeperInterface.(interface {
		Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
	})
	if !ok {
		return fmt.Errorf("KMS keeper does not support encryption")
	}

	// Encrypt master key with KMS
	ciphertext, err := keeper.Encrypt(ctx, masterKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt master key with KMS: %w", err)
	}

	// Encode the ciphertext to base64
	encodedKey := base64.StdEncoding.EncodeToString(ciphertext)

	// Print KMS configuration
	fmt.Println("# Master Key Configuration (KMS Mode)")
	fmt.Println("# Copy these environment variables to your .env file or secrets manager")
	fmt.Println()
	fmt.Printf("KMS_PROVIDER=\"%s\"\n", kmsProvider)
	fmt.Printf("KMS_KEY_URI=\"%s\"\n", kmsKeyURI)
	fmt.Printf("MASTER_KEYS=\"%s:%s\"\n", keyID, encodedKey)
	fmt.Printf("ACTIVE_MASTER_KEY_ID=\"%s\"\n", keyID)
	fmt.Println()
	fmt.Println("# For multiple master keys (key rotation), encrypt each key with the same KMS key:")
	fmt.Printf("# MASTER_KEYS=\"%s:%s,new-key:base64-encoded-kms-ciphertext\"\n", keyID, encodedKey)
	fmt.Println("# ACTIVE_MASTER_KEY_ID=\"new-key\"")

	// Zero out the master key from memory for security
	for i := range masterKey {
		masterKey[i] = 0
	}

	return nil
}
