package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	"github.com/allisson/secrets/internal/ui"
)

// CreateClientResult holds the result of the client creation operation.
type CreateClientResult struct {
	ID string `json:"client_id"`
	// #nosec G117
	PlainSecret string `json:"secret"`
}

// ToText returns a human-readable representation of the creation result.
func (r *CreateClientResult) ToText() string {
	var sb fmt.Stringer
	output := "\nClient created successfully!\n"
	output += fmt.Sprintf("Client ID: %s\n", r.ID)
	output += fmt.Sprintf("Secret: %s\n", r.PlainSecret)
	output += "\nIMPORTANT: The secret is shown only once. Store it securely."
	_ = sb
	return output
}

// ToJSON returns a JSON representation of the creation result.
func (r *CreateClientResult) ToJSON() string {
	jsonBytes, _ := json.MarshalIndent(r, "", "  ")
	return string(jsonBytes)
}

// RunCreateClient creates a new authentication client with policies.
// Supports both interactive mode (when policiesJSON is empty) and non-interactive
// mode (when policiesJSON is provided).
func RunCreateClient(
	ctx context.Context,
	clientUseCase authUseCase.ClientUseCase,
	logger *slog.Logger,
	name string,
	isActive bool,
	policiesJSON string,
	format string,
	io IOTuple,
) error {
	logger.Info("creating new client", slog.String("name", name))

	// Parse or prompt for policies
	var policies []authDomain.PolicyDocument
	var err error

	if policiesJSON == "" {
		// Interactive mode
		policies, err = ui.PromptForPolicies(io.Reader, io.Writer)
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

	// Create input
	input := &authDomain.CreateClientInput{
		Name:     name,
		IsActive: isActive,
		Policies: policies,
	}

	// Create the client
	output, err := clientUseCase.Create(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Output result
	result := &CreateClientResult{
		ID:          output.ID.String(),
		PlainSecret: output.PlainSecret,
	}
	WriteOutput(io.Writer, format, result)

	logger.Info("client created successfully",
		slog.String("client_id", output.ID.String()),
		slog.String("name", name),
		slog.Bool("is_active", isActive),
	)

	return nil
}
