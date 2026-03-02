package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/ui"
)

// UpdateClientResult holds the result of the client update operation.
type UpdateClientResult struct {
	ID       string `json:"client_id"`
	Name     string `json:"name"`
	IsActive bool   `json:"is_active"`
}

// ToText returns a human-readable representation of the update result.
func (r *UpdateClientResult) ToText() string {
	output := "\nClient updated successfully!\n"
	output += fmt.Sprintf("Client ID: %s\n", r.ID)
	output += fmt.Sprintf("Name: %s\n", r.Name)
	output += fmt.Sprintf("Active: %t", r.IsActive)
	return output
}

// ToJSON returns a JSON representation of the update result.
func (r *UpdateClientResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunUpdateClient updates an existing authentication client's configuration.
// Supports both interactive mode (when policiesJSON is empty) and non-interactive
// mode (when policiesJSON is provided).
func RunUpdateClient(
	ctx context.Context,
	clientUseCase authUseCase.ClientUseCase,
	logger *slog.Logger,
	io IOTuple,
	clientIDStr string,
	name string,
	isActive bool,
	policiesJSON string,
	format string,
) error {
	logger.Info("updating client", slog.String("client_id", clientIDStr))

	// Parse client ID
	clientID, err := uuid.Parse(clientIDStr)
	if err != nil {
		return fmt.Errorf("invalid client ID format: %w", err)
	}

	// Get existing client to display current values if in interactive mode
	existingClient, err := clientUseCase.Get(ctx, clientID)
	if err != nil {
		return fmt.Errorf("failed to get existing client: %w", err)
	}

	// Parse or prompt for policies
	var policies []authDomain.PolicyDocument

	if policiesJSON == "" {
		// Interactive mode - show current policies and prompt for new ones
		policies, err = ui.PromptForPoliciesUpdate(io.Reader, io.Writer, existingClient.Policies)
		if err != nil {
			return fmt.Errorf("failed to get policies: %w", err)
		}
	} else {
		// Non-interactive mode: parse JSON
		if err := json.Unmarshal([]byte(policiesJSON), &policies); err != nil {
			return fmt.Errorf("failed to parse policies JSON: %w", err)
		}
	}

	// Validate that at least one policy was provided
	if len(policies) == 0 {
		return fmt.Errorf("at least one policy is required")
	}

	// Create update input
	input := &authDomain.UpdateClientInput{
		Name:     name,
		IsActive: isActive,
		Policies: policies,
	}

	// Update the client
	if err := clientUseCase.Update(ctx, clientID, input); err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}

	// Output result
	result := &UpdateClientResult{
		ID:       clientID.String(),
		Name:     name,
		IsActive: isActive,
	}
	WriteOutput(io.Writer, format, result)

	logger.Info("client updated successfully",
		slog.String("client_id", clientID.String()),
		slog.String("name", name),
		slog.Bool("is_active", isActive),
	)

	return nil
}
