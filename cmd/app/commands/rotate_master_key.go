package commands

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/allisson/secrets/internal/config"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
)

// RunRotateMasterKey generates a new master key and combines it with existing keys for rotation.
// Reads current MASTER_KEYS from environment, generates a new key, and outputs the combined
// configuration with the new key set as active. The old keys remain accessible for decrypting
// existing KEKs.
//
// Mode Detection:
//   - KMS Mode: If KMS_PROVIDER and KMS_KEY_URI are set, encrypts new key with KMS
//   - Legacy Mode: If KMS variables are empty, generates plaintext base64-encoded key
//
// Key Rotation Workflow:
//  1. Run this command to generate new master key configuration
//  2. Update environment variables (MASTER_KEYS, ACTIVE_MASTER_KEY_ID)
//  3. Restart application (automatically decrypts KEKs with new master key chain)
//  4. Rotate KEKs: `app rotate-kek --algorithm aes-gcm`
//  5. After all KEKs rotated, remove old master key from MASTER_KEYS
//
// Requirements: MASTER_KEYS and ACTIVE_MASTER_KEY_ID must be set in environment.
func RunRotateMasterKey(ctx context.Context, keyID string) error {
	// Load configuration to get KMS settings
	cfg := config.Load()

	// Get existing master keys from environment
	existingMasterKeys := os.Getenv("MASTER_KEYS")
	existingActiveKeyID := os.Getenv("ACTIVE_MASTER_KEY_ID")

	// Validate existing configuration
	if existingMasterKeys == "" {
		return fmt.Errorf("MASTER_KEYS environment variable is not set - cannot rotate without existing keys")
	}
	if existingActiveKeyID == "" {
		return fmt.Errorf("ACTIVE_MASTER_KEY_ID environment variable is not set")
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
	defer func() {
		// Zero out the master key from memory for security
		for i := range masterKey {
			masterKey[i] = 0
		}
	}()

	var encodedKey string
	var newMasterKeys string

	// Determine mode based on KMS configuration
	if cfg.KMSProvider != "" && cfg.KMSKeyURI != "" {
		// KMS mode: encrypt new key
		fmt.Println("# KMS Mode: Encrypting new master key with KMS")
		fmt.Printf("# KMS Provider: %s\n", cfg.KMSProvider)
		fmt.Println()

		// Create KMS service and open keeper
		kmsService := cryptoService.NewKMSService()
		keeperInterface, err := kmsService.OpenKeeper(ctx, cfg.KMSKeyURI)
		if err != nil {
			return fmt.Errorf("failed to open KMS keeper: %w", err)
		}
		defer func() {
			if closeErr := keeperInterface.Close(); closeErr != nil {
				fmt.Printf("Warning: failed to close KMS keeper: %v\n", closeErr)
			}
		}()

		// Type assert to get Encrypt method
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
		encodedKey = base64.StdEncoding.EncodeToString(ciphertext)

		// Combine with existing keys (new key last, will be set as active)
		newMasterKeys = fmt.Sprintf("%s,%s:%s", existingMasterKeys, keyID, encodedKey)

		// Print KMS configuration
		fmt.Println("# Master Key Rotation (KMS Mode)")
		fmt.Println("# Update these environment variables in your .env file or secrets manager")
		fmt.Println()
		fmt.Printf("KMS_PROVIDER=\"%s\"\n", cfg.KMSProvider)
		fmt.Printf("KMS_KEY_URI=\"%s\"\n", cfg.KMSKeyURI)
		fmt.Printf("MASTER_KEYS=\"%s\"\n", newMasterKeys)
		fmt.Printf("ACTIVE_MASTER_KEY_ID=\"%s\"\n", keyID)
		fmt.Println()
		fmt.Println("# Rotation Workflow:")
		fmt.Println("# 1. Update the above environment variables")
		fmt.Println("# 2. Restart the application")
		fmt.Println("# 3. Rotate KEKs: app rotate-kek --algorithm aes-gcm")
		fmt.Printf(
			"# 4. After all KEKs rotated, remove old master key: MASTER_KEYS=\"%s:%s\"\n",
			keyID,
			encodedKey,
		)
	} else {
		// Legacy mode: plaintext base64 encoding
		fmt.Println("# Legacy Mode: Generating plaintext master key")
		fmt.Println("# WARNING: For production, use KMS mode (set KMS_PROVIDER and KMS_KEY_URI)")
		fmt.Println()

		// Encode the master key to base64
		encodedKey = base64.StdEncoding.EncodeToString(masterKey)

		// Combine with existing keys (new key last, will be set as active)
		newMasterKeys = fmt.Sprintf("%s,%s:%s", existingMasterKeys, keyID, encodedKey)

		// Print legacy configuration
		fmt.Println("# Master Key Rotation (Legacy Mode - Plaintext)")
		fmt.Println("# Update these environment variables in your .env file or secrets manager")
		fmt.Println()
		fmt.Printf("MASTER_KEYS=\"%s\"\n", newMasterKeys)
		fmt.Printf("ACTIVE_MASTER_KEY_ID=\"%s\"\n", keyID)
		fmt.Println()
		fmt.Println("# Rotation Workflow:")
		fmt.Println("# 1. Update the above environment variables")
		fmt.Println("# 2. Restart the application")
		fmt.Println("# 3. Rotate KEKs: app rotate-kek --algorithm aes-gcm")
		fmt.Printf(
			"# 4. After all KEKs rotated, remove old master key: MASTER_KEYS=\"%s:%s\"\n",
			keyID,
			encodedKey,
		)
		fmt.Println()
		fmt.Println("# For production, consider using KMS mode:")
		fmt.Println("# app create-master-key --kms-provider=localsecrets --kms-key-uri=base64key://...")
	}

	return nil
}
