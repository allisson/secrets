package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
)

// RunRotateTokenizationKey creates a new version of an existing tokenization key.
// Increments the version number and generates a new DEK while preserving old versions
// for detokenization of previously issued tokens.
//
// Requirements: Database must be migrated, named tokenization key must exist.
func RunRotateTokenizationKey(
	ctx context.Context,
	name string,
	formatType string,
	isDeterministic bool,
	algorithmStr string,
) error {
	// Load configuration
	cfg := config.Load()

	// Create DI container
	container := app.NewContainer(cfg)

	// Get logger from container
	logger := container.Logger()
	logger.Info("rotating tokenization key",
		slog.String("name", name),
		slog.String("format_type", formatType),
		slog.Bool("is_deterministic", isDeterministic),
		slog.String("algorithm", algorithmStr),
	)

	// Ensure cleanup on exit
	defer closeContainer(container, logger)

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

	// Get tokenization key use case from container
	tokenizationKeyUseCase, err := container.TokenizationKeyUseCase()
	if err != nil {
		return fmt.Errorf("failed to initialize tokenization key use case: %w", err)
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
