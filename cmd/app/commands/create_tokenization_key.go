package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
)

// RunCreateTokenizationKey creates a new tokenization key with the specified parameters.
// Should be run during initial setup or when adding new tokenization formats.
//
// Requirements: Database must be migrated, MASTER_KEYS and ACTIVE_MASTER_KEY_ID must be set.
func RunCreateTokenizationKey(
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
	logger.Info("creating new tokenization key",
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
