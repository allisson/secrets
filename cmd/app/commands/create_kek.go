package commands

import (
	"context"
	"fmt"
	"log/slog"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
)

// RunCreateKek creates a new Key Encryption Key using the specified algorithm.
// Should only be run once during initial system setup. The KEK is encrypted using
// the active master key from MASTER_KEYS environment variable.
//
// Requirements: Database must be migrated, MASTER_KEYS and ACTIVE_MASTER_KEY_ID must be set.
func RunCreateKek(
	ctx context.Context,
	kekUseCase cryptoUseCase.KekUseCase,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	logger *slog.Logger,
	algorithmStr string,
) error {
	logger.Info("creating new KEK", slog.String("algorithm", algorithmStr))

	// Parse algorithm
	algorithm, err := parseAlgorithm(algorithmStr)
	if err != nil {
		return err
	}

	logger.Info("master key chain loaded",
		slog.String("active_master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

	// Create the KEK
	if err := kekUseCase.Create(ctx, masterKeyChain, algorithm); err != nil {
		return fmt.Errorf("failed to create KEK: %w", err)
	}

	logger.Info("KEK created successfully",
		slog.String("algorithm", string(algorithm)),
		slog.String("master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

	return nil
}

// parseAlgorithm converts algorithm string to cryptoDomain.Algorithm type.
// Returns an error if the algorithm string is invalid.
func parseAlgorithm(algorithmStr string) (cryptoDomain.Algorithm, error) {
	switch algorithmStr {
	case "aes-gcm":
		return cryptoDomain.AESGCM, nil
	case "chacha20-poly1305":
		return cryptoDomain.ChaCha20, nil
	default:
		return "", fmt.Errorf(
			"invalid algorithm: %s (valid options: aes-gcm, chacha20-poly1305)",
			algorithmStr,
		)
	}
}
