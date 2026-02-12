package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
)

// RunCreateKek creates a new Key Encryption Key using the specified algorithm.
// Should only be run once during initial system setup. The KEK is encrypted using
// the active master key from MASTER_KEYS environment variable.
//
// Requirements: Database must be migrated, MASTER_KEYS and ACTIVE_MASTER_KEY_ID must be set.
func RunCreateKek(ctx context.Context, algorithmStr string) error {
	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("creating new KEK", slog.String("algorithm", algorithmStr))

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Parse algorithm
	algorithm, err := parseAlgorithm(algorithmStr)
	if err != nil {
		return err
	}

	// Get master key chain from container
	masterKeyChain, err := container.MasterKeyChain()
	if err != nil {
		return fmt.Errorf("failed to load master key chain: %w", err)
	}

	logger.Info("master key chain loaded",
		slog.String("active_master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

	// Get KEK use case from container
	kekUseCase, err := container.KekUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize KEK use case: %w", err)
	}

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
