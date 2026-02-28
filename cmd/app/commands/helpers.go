// Package commands contains CLI command implementations for the application.
package commands

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"

	"io"
	"os"

	"github.com/allisson/secrets/internal/app"
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

// closeContainer closes all resources in the container and logs any errors.
func closeContainer(container *app.Container, logger *slog.Logger) {
	if err := container.Shutdown(context.Background()); err != nil {
		logger.Error("failed to shutdown container", slog.Any("error", err))
	}
}

// closeMigrate closes the migration instance and logs any errors.
func closeMigrate(migrate *migrate.Migrate, logger *slog.Logger) {
	sourceError, databaseError := migrate.Close()
	if sourceError != nil || databaseError != nil {
		logger.Error(
			"failed to close the migrate",
			slog.Any("source_error", sourceError),
			slog.Any("database_error", databaseError),
		)
	}
}

// parseFormatType converts format type string to tokenizationDomain.FormatType.
// Returns an error if the format type string is invalid.
func parseFormatType(formatType string) (tokenizationDomain.FormatType, error) {
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
