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
func RunCreateMasterKey(
	ctx context.Context,
	kmsService cryptoService.KMSService,
	logger *slog.Logger,
	writer io.Writer,
	keyID string,
	kmsProvider string,
	kmsKeyURI string,
) error {
	// Validate required KMS parameters
	if kmsProvider == "" || kmsKeyURI == "" {
		return fmt.Errorf(
			"--kms-provider and --kms-key-uri are required\n\nFor local development, use:\n  --kms-provider=localsecrets --kms-key-uri=\"base64key://<32-byte-base64-key>\"\n\nFor production, use cloud KMS providers:\n  --kms-provider=gcpkms --kms-key-uri=\"gcpkms://projects/.../cryptoKeys/...\"\n  --kms-provider=awskms --kms-key-uri=\"awskms:///alias/...\"\n  --kms-provider=azurekeyvault --kms-key-uri=\"azurekeyvault://...\"",
		)
	}

	logger.Info("creating new master key",
		slog.String("kms_provider", kmsProvider),
		slog.String("kms_key_uri", kmsKeyURI),
	)

	// Generate default key ID if not provided
	if keyID == "" {
		keyID = fmt.Sprintf("master-key-%s", time.Now().Format("2006-01-02"))
	}

	// Generate a cryptographically secure 32-byte master key
	masterKey := make([]byte, 32)
	if _, err := rand.Read(masterKey); err != nil {
		return fmt.Errorf("failed to generate master key: %w", err)
	}

	_, _ = fmt.Fprintln(writer, "# KMS Mode: Encrypting master key with KMS")
	_, _ = fmt.Fprintf(writer, "# KMS Provider: %s\n", kmsProvider)
	_, _ = fmt.Fprintln(writer)

	// Open keeper
	keeperInterface, err := kmsService.OpenKeeper(ctx, kmsKeyURI)
	if err != nil {
		return fmt.Errorf("failed to open KMS keeper: %w", err)
	}
	defer func() {
		if closeErr := keeperInterface.Close(); closeErr != nil {
			_, _ = fmt.Fprintf(writer, "Warning: failed to close KMS keeper: %v\n", closeErr)
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
	_, _ = fmt.Fprintln(writer, "# Master Key Configuration (KMS Mode)")
	_, _ = fmt.Fprintln(writer, "# Copy these environment variables to your .env file or secrets manager")
	_, _ = fmt.Fprintln(writer)
	_, _ = fmt.Fprintf(writer, "KMS_PROVIDER=\"%s\"\n", kmsProvider)
	_, _ = fmt.Fprintf(writer, "KMS_KEY_URI=\"%s\"\n", kmsKeyURI)
	_, _ = fmt.Fprintf(writer, "MASTER_KEYS=\"%s:%s\"\n", keyID, encodedKey)
	_, _ = fmt.Fprintf(writer, "ACTIVE_MASTER_KEY_ID=\"%s\"\n", keyID)
	_, _ = fmt.Fprintln(writer)
	_, _ = fmt.Fprintln(
		writer,
		"# For multiple master keys (key rotation), encrypt each key with the same KMS key:",
	)
	_, _ = fmt.Fprintf(
		writer,
		"# MASTER_KEYS=\"%s:%s,new-key:base64-encoded-kms-ciphertext\"\n",
		keyID,
		encodedKey,
	)
	_, _ = fmt.Fprintln(writer, "# ACTIVE_MASTER_KEY_ID=\"new-key\"")

	// Zero out the master key from memory for security
	for i := range masterKey {
		masterKey[i] = 0
	}

	return nil
}
