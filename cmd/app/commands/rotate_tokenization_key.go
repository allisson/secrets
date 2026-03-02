package commands

import (
	"context"
	"fmt"
	"log/slog"

	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
)

// RunRotateTokenizationKey rotates an existing tokenization key to a new version.
// Updates format and deterministic settings. Existing tokens remain valid until rotated.
func RunRotateTokenizationKey(
	ctx context.Context,
	tokenizationKeyUseCase tokenizationUseCase.TokenizationKeyUseCase,
	logger *slog.Logger,
	name string,
	formatTypeStr string,
	isDeterministic bool,
	algorithmStr string,
) error {
	logger.Info("rotating tokenization key",
		slog.String("name", name),
		slog.String("format_type", formatTypeStr),
		slog.Bool("is_deterministic", isDeterministic),
		slog.String("algorithm", algorithmStr),
	)

	// Parse format type
	format, err := ParseFormatType(formatTypeStr)
	if err != nil {
		return err
	}

	// Parse algorithm
	algorithm, err := ParseAlgorithm(algorithmStr)
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
