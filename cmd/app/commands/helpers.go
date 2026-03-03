// Package commands contains CLI command implementations for the application.
package commands

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/allisson/secrets/internal/app"
	"github.com/allisson/secrets/internal/config"
	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// IOTuple holds reader and writer for commands, allowing for testing.
type IOTuple struct {
	Reader io.Reader
	Writer io.Writer
}

// DefaultIO returns an IOTuple with os.Stdin and os.Stdout.
func DefaultIO() IOTuple {
	return IOTuple{
		Reader: os.Stdin,
		Writer: os.Stdout,
	}
}

// Formatter defines the interface for data that can be output in multiple formats.
type Formatter interface {
	ToText() string
	ToJSON() string
}

// WriteOutput writes the formatted data to the provided writer based on the specified format.
func WriteOutput(writer io.Writer, format string, data Formatter) {
	if format == "json" {
		_, _ = fmt.Fprintln(writer, data.ToJSON())
	} else {
		_, _ = fmt.Fprintln(writer, data.ToText())
	}
}

// ExecuteWithContainer encapsulates the standard CLI command execution pattern:
// loading configuration, initializing the DI container, and ensuring graceful shutdown.
func ExecuteWithContainer(
	ctx context.Context,
	fn func(ctx context.Context, container *app.Container) error,
) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	container := app.NewContainer(cfg)
	defer func() {
		if err := container.Shutdown(ctx); err != nil {
			container.Logger().Error("failed to shutdown container", slog.Any("error", err))
		}
	}()

	return fn(ctx, container)
}

// CloseContainer closes all resources in the container and logs any errors.
func CloseContainer(container *app.Container, logger *slog.Logger) {
	if err := container.Shutdown(context.Background()); err != nil {
		logger.Error("failed to shutdown container", slog.Any("error", err))
	}
}

// CloseMigrate closes the migration instance and logs any errors.
func CloseMigrate(migrate *migrate.Migrate, logger *slog.Logger) {
	sourceError, databaseError := migrate.Close()
	if sourceError != nil || databaseError != nil {
		logger.Error(
			"failed to close the migrate",
			slog.Any("source_error", sourceError),
			slog.Any("database_error", databaseError),
		)
	}
}

// ParseAlgorithm converts algorithm string to cryptoDomain.Algorithm type.
func ParseAlgorithm(algorithmStr string) (cryptoDomain.Algorithm, error) {
	switch algorithmStr {
	case "aes-gcm":
		return cryptoDomain.AESGCM, nil
	case "chacha20-poly1305":
		return cryptoDomain.ChaCha20, nil
	default:
		return "", fmt.Errorf(
			"invalid algorithm: %s (valid options: aes-gcm, chacha20-poly1305)",
			algorithmStr,
		)
	}
}

// ParseFormatType converts format type string to tokenizationDomain.FormatType.
func ParseFormatType(formatType string) (tokenizationDomain.FormatType, error) {
	switch formatType {
	case "uuid":
		return tokenizationDomain.FormatUUID, nil
	case "numeric":
		return tokenizationDomain.FormatNumeric, nil
	case "luhn-preserving":
		return tokenizationDomain.FormatLuhnPreserving, nil
	case "alphanumeric":
		return tokenizationDomain.FormatAlphanumeric, nil
	default:
		return "", fmt.Errorf(
			"invalid format type: %s (valid options: uuid, numeric, luhn-preserving, alphanumeric)",
			formatType,
		)
	}
}
