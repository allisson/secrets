package commands

import (
	"context"
	"fmt"
	"log/slog"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
)

func RunRotateKek(
	ctx context.Context,
	kekUseCase cryptoUseCase.KekUseCase,
	masterKeyChain *cryptoDomain.MasterKeyChain,
	logger *slog.Logger,
	algorithmStr string,
) error {
	logger.Info("rotating KEK", slog.String("algorithm", algorithmStr))

	// Parse algorithm
	algorithm, err := parseAlgorithm(algorithmStr)
	if err != nil {
		return err
	}

	logger.Info("master key chain loaded",
		slog.String("active_master_key_id", masterKeyChain.ActiveMasterKeyID()),
	)

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
