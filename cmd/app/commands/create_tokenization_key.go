package commands

import (
	"context"
	"fmt"
	"log/slog"

	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

// RunCreateTokenizationKey creates a new tokenization key with the specified parameters.
// Should be run during initial setup or when adding new tokenization formats.
//
// Requirements: Database must be migrated, MASTER_KEYS and ACTIVE_MASTER_KEY_ID must be set.
func RunCreateTokenizationKey(
	ctx context.Context,
	tokenizationKeyUseCase tokenizationUseCase.TokenizationKeyUseCase,
	logger *slog.Logger,
	name string,
	formatType string,
	isDeterministic bool,
	algorithmStr string,
) error {
	logger.Info("creating new tokenization key",
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

	// Create the tokenization key
	key, err := tokenizationKeyUseCase.Create(ctx, name, format, isDeterministic, algorithm)
	if err != nil {
		return fmt.Errorf("failed to create tokenization key: %w", err)
	}

	logger.Info("tokenization key created successfully",
		slog.String("id", key.ID.String()),
		slog.String("name", key.Name),
		slog.String("format_type", string(key.FormatType)),
		slog.Uint64("version", uint64(key.Version)),
		slog.Bool("is_deterministic", key.IsDeterministic),
		slog.String("algorithm", string(algorithm)),
	)

	return nil
}
