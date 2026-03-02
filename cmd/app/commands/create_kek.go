package commands

import (
	"context"
	"fmt"
	"log/slog"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
)

// RunCreateKek creates a new Key Encryption Key (KEK) and encrypts it with the master key.
// The new KEK will be stored in the database and marked as active for its algorithm.
func RunCreateKek(
	ctx context.Context,
	kekUseCase cryptoUseCase.KekUseCase,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	logger *slog.Logger,
	algorithmStr string,
) error {
	logger.Info("creating new KEK", slog.String("algorithm", algorithmStr))

	// Parse algorithm
	algorithm, err := ParseAlgorithm(algorithmStr)
	if err != nil {
		return err
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
