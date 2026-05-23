package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/allisson/secrets/internal/keyring"
)

// RunRewrapDeks finds DEKs not encrypted with the keyring's active KEK and
// rewraps them in batches. The kekIDStr argument is a safety check: it must
// match the keyring's currently-active KEK, so an operator cannot accidentally
// rewrap DEKs against a stale chain.
func RunRewrapDeks(
	ctx context.Context,
	kr keyring.Keyring,
	logger *slog.Logger,
	kekIDStr string,
	batchSize int,
) error {
	wantedKekID, err := uuid.Parse(kekIDStr)
	if err != nil {
		return fmt.Errorf("invalid kek-id: %w", err)
	}

	if batchSize <= 0 {
		return fmt.Errorf("batch-size must be greater than 0")
	}

	activeKekID := kr.ActiveKekID()
	if activeKekID != wantedKekID {
		return fmt.Errorf(
			"requested kek-id %s does not match keyring active KEK %s; "+
				"restart the rewrap process after KEK rotation so the latest chain is loaded",
			wantedKekID, activeKekID,
		)
	}

	logger.Info("starting DEK rewrap process",
		slog.String("kek_id", kekIDStr),
		slog.Int("batch_size", batchSize),
	)

	total, err := kr.RewrapAll(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to rewrap DEKs: %w", err)
	}

	logger.Info("DEK rewrap process completed",
		slog.Int("total_rewrapped", total),
		slog.String("target_kek_id", kekIDStr),
	)

	return nil
}
