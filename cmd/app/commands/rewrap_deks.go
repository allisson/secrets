package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
)

// RunRewrapDeks finds all DEKs that are not encrypted with the specified KEK ID
// and re-encrypts them with the specified KEK in batches.
func RunRewrapDeks(
	ctx context.Context,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	kekUseCase cryptoUseCase.KekUseCase,
	dekUseCase cryptoUseCase.DekUseCase,
	logger *slog.Logger,
	kekIDStr string,
	batchSize int,
) error {
	// Parse KEK ID
	newKekID, err := uuid.Parse(kekIDStr)
	if err != nil {
		return fmt.Errorf("invalid kek-id: %w", err)
	}

	if batchSize <= 0 {
		return fmt.Errorf("batch-size must be greater than 0")
	}

	logger.Info("starting DEK rewrap process",
		slog.String("kek_id", kekIDStr),
		slog.Int("batch_size", batchSize),
	)

	kekChain, err := kekUseCase.Unwrap(ctx, masterKeyChain)
	if err != nil {
		return fmt.Errorf("failed to load and unwrap kek chain: %w", err)
	}
	defer kekChain.Close()

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
