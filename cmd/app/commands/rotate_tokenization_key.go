package commands

import (
	"context"
	"fmt"
	"log/slog"

	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

// RunRotateTokenizationKey creates a new version of an existing tokenization key.
// Increments the version number and generates a new DEK while preserving old versions
// for detokenization of previously issued tokens.
//
// Requirements: Database must be migrated, named tokenization key must exist.
func RunRotateTokenizationKey(
	ctx context.Context,
	tokenizationKeyUseCase tokenizationUseCase.TokenizationKeyUseCase,
	logger *slog.Logger,
	name string,
	formatType string,
	isDeterministic bool,
	algorithmStr string,
) error {
	logger.Info("rotating tokenization key",
		slog.String("name", name),
		slog.String("format_type", formatType),
		slog.Bool("is_deterministic", isDeterministic),
		slog.String("algorithm", algorithmStr),
	)

	// Parse format type
	format, err := parseFormatType(formatType)
	if err != nil {
		return err
	}

	// Parse algorithm
	algorithm, err := parseAlgorithm(algorithmStr)
	if err != nil {
		return err
	}

	// Rotate the tokenization key
	key, err := tokenizationKeyUseCase.Rotate(ctx, name, format, isDeterministic, algorithm)
	if err != nil {
		return fmt.Errorf("failed to rotate tokenization key: %w", err)
	}

	logger.Info("tokenization key rotated successfully",
		slog.String("id", key.ID.String()),
		slog.String("name", key.Name),
		slog.String("format_type", string(key.FormatType)),
		slog.Uint64("version", uint64(key.Version)),
		slog.Bool("is_deterministic", key.IsDeterministic),
		slog.String("algorithm", string(algorithm)),
	)

	return nil
}
