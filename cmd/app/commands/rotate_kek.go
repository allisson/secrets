package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
)

// RunRotateKek rotates the existing Key Encryption Key using the specified algorithm.
// Creates a new KEK version and marks the previous active KEK as inactive. The new KEK is
// encrypted using the active master key. This operation is atomic and maintains backward
// compatibility - existing DEKs encrypted with the old KEK remain readable.
//
// Key rotation recommended every 90 days or when suspecting KEK compromise, changing encryption
// algorithms, or rotating master keys.
//
// Requirements: An active KEK must already exist, MASTER_KEYS and ACTIVE_MASTER_KEY_ID must be set.
func RunRotateKek(ctx context.Context, algorithmStr string) error {
	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("rotating KEK", slog.String("algorithm", algorithmStr))

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

	// Rotate the KEK
	if err := kekUseCase.Rotate(ctx, masterKeyChain, algorithm); err != nil {
		return fmt.Errorf("failed to rotate KEK: %w", err)
	}

	logger.Info("KEK rotated successfully",
		slog.String("algorithm", string(algorithm)),
		slog.String("master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

	return nil
}
