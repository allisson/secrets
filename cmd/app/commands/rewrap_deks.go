package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
)

// RunRewrapDeks finds all DEKs that are not encrypted with the specified KEK ID
// and re-encrypts them with the specified KEK in batches.
func RunRewrapDeks(ctx context.Context, kekIDStr string, batchSize int) error {
	// Parse KEK ID
	newKekID, err := uuid.Parse(kekIDStr)
	if err != nil {
		return fmt.Errorf("invalid kek-id: %w", err)
	}

	if batchSize <= 0 {
		return fmt.Errorf("batch-size must be greater than 0")
	}

	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("starting DEK rewrap process",
		slog.String("kek_id", kekIDStr),
		slog.Int("batch_size", batchSize),
	)

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

	// Get master key chain from container
	masterKeyChain, err := container.MasterKeyChain()
	if err != nil {
		return fmt.Errorf("failed to load master key chain: %w", err)
	}

	// Get KEK use case to unwrap KEKs into KekChain
	kekUseCase, err := container.KekUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize KEK use case: %w", err)
	}

	kekChain, err := kekUseCase.Unwrap(ctx, masterKeyChain)
	if err != nil {
		return fmt.Errorf("failed to load and unwrap kek chain: %w", err)
	}
	defer kekChain.Close()

	// Get CryptoDekUseCase
	dekUseCase, err := container.CryptoDekUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize CryptoDekUseCase: %w", err)
	}

	totalRewrapped := 0

	for {
		rewrappedCount, err := dekUseCase.Rewrap(ctx, kekChain, newKekID, batchSize)
		if err != nil {
			return fmt.Errorf("failed to rewrap DEKs in batch: %w", err)
		}

		if rewrappedCount == 0 {
			break
		}

		totalRewrapped += rewrappedCount
		logger.Info("rewrapped batch of DEKs",
			slog.Int("rewrapped_in_batch", rewrappedCount),
			slog.Int("total_rewrapped", totalRewrapped),
		)
	}

	logger.Info("DEK rewrap process completed",
		slog.Int("total_rewrapped", totalRewrapped),
		slog.String("target_kek_id", kekIDStr),
	)

	return nil
}
