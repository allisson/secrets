package commands

import (
	"context"
	"fmt"
	"log/slog"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
)

// RunRotateKek rotates the Key Encryption Key (KEK) for a specific algorithm.
// Generates a new KEK version and marks it as active. Existing secrets encrypted
// with old KEKs remain valid until rewrapped.
func RunRotateKek(
	ctx context.Context,
	kekUseCase cryptoUseCase.KekUseCase,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	logger *slog.Logger,
	algorithmStr string,
) error {
	logger.Info("rotating KEK", slog.String("algorithm", algorithmStr))

	// Parse algorithm
	algorithm, err := ParseAlgorithm(algorithmStr)
	if err != nil {
		return err
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
