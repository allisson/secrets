package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"

	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
)

// RotateClientSecretResult holds the result of the client secret rotation operation.
type RotateClientSecretResult struct {
	ID string `json:"client_id"`
	// #nosec G117
	PlainSecret string `json:"secret"`
}

// ToText returns a human-readable representation of the rotation result.
func (r *RotateClientSecretResult) ToText() string {
	output := "\nClient secret rotated successfully!\n"
	output += fmt.Sprintf("Client ID: %s\n", r.ID)
	output += fmt.Sprintf("New Secret: %s\n", r.PlainSecret)
	output += "\nIMPORTANT: The new secret is shown only once. Update your applications immediately."
	output += "\nNOTE: All existing tokens for this client have been revoked."
	return output
}

// ToJSON returns a JSON representation of the rotation result.
func (r *RotateClientSecretResult) ToJSON() string {
	// #nosec G117 - Intentionally marshaling secret for CLI output. This is shown once during client secret rotation.
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunRotateClientSecret generates a new secret for a client and revokes all its active tokens.
func RunRotateClientSecret(
	ctx context.Context,
	clientUseCase authUseCase.ClientUseCase,
	logger *slog.Logger,
	writer io.Writer,
	id string,
	format string,
) error {
	logger.Info("rotating client secret", slog.String("client_id", id))

	// Parse ID
	clientID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	// Call use case
	output, err := clientUseCase.RotateSecret(ctx, clientID)
	if err != nil {
		return fmt.Errorf("failed to rotate client secret: %w", err)
	}

	// Output result
	result := &RotateClientSecretResult{
		ID:          output.ID.String(),
		PlainSecret: output.PlainSecret,
	}
	WriteOutput(writer, format, result)

	logger.Info("client secret rotated successfully",
		slog.String("client_id", output.ID.String()),
	)

	return nil
}
