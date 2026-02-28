package commands

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"time"

	cryptoService "github.com/allisson/secrets/internal/crypto/service"
)

func RunRotateMasterKey(
	ctx context.Context,
	kmsService cryptoService.KMSService,
	logger *slog.Logger,
	writer io.Writer,
	keyID, kmsProvider, kmsKeyURI, existingMasterKeys, existingActiveKeyID string,
) error {
	// Validate required KMS parameters
	if kmsProvider == "" || kmsKeyURI == "" {
		return fmt.Errorf(
			"KMS_PROVIDER and KMS_KEY_URI are required for master key rotation\n\nFor local development, use:\n  KMS_PROVIDER=localsecrets\n  KMS_KEY_URI=\"base64key://<32-byte-base64-key>\"",
		)
	}

	// Validate existing configuration
	if existingMasterKeys == "" {
		return fmt.Errorf("MASTER_KEYS is not set - cannot rotate without existing keys")
	}
	if existingActiveKeyID == "" {
		return fmt.Errorf("ACTIVE_MASTER_KEY_ID is not set")
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

	// KMS mode: encrypt new key
	_, _ = fmt.Fprintln(writer, "# KMS Mode: Encrypting new master key with KMS")
	_, _ = fmt.Fprintf(writer, "# KMS Provider: %s\n", kmsProvider)
	_, _ = fmt.Fprintln(writer)

	// Create KMS service and open keeper
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	if err != nil {
		return fmt.Errorf("failed to open KMS keeper: %w", err)
	}
	defer func() {
		if closeErr := keeperInterface.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(writer, "Warning: failed to close KMS keeper: %v\n", closeErr)
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
	_, _ = fmt.Fprintln(writer, "# Master Key Rotation (KMS Mode)")
	_, _ = fmt.Fprintln(writer, "# Update these environment variables in your .env file or secrets manager")
	_, _ = fmt.Fprintln(writer)
	_, _ = fmt.Fprintf(writer, "KMS_PROVIDER=\"%s\"\n", kmsProvider)
	_, _ = fmt.Fprintf(writer, "KMS_KEY_URI=\"%s\"\n", kmsKeyURI)
	_, _ = fmt.Fprintf(writer, "MASTER_KEYS=\"%s\"\n", newMasterKeys)
	_, _ = fmt.Fprintf(writer, "ACTIVE_MASTER_KEY_ID=\"%s\"\n", keyID)
	_, _ = fmt.Fprintln(writer)
	_, _ = fmt.Fprintln(writer, "# Rotation Workflow:")
	_, _ = fmt.Fprintln(writer, "# 1. Update the above environment variables")
	_, _ = fmt.Fprintln(writer, "# 2. Restart the application")
	_, _ = fmt.Fprintln(writer, "# 3. Rotate KEKs: app rotate-kek --algorithm aes-gcm")
	_, _ = fmt.Fprintf(writer,
		"# 4. After all KEKs rotated, remove old master key: MASTER_KEYS=\"%s:%s\"\n",
		keyID,
		encodedKey,
	)

	return nil
}
