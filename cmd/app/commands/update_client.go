package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
)

// RunUpdateClient updates an existing authentication client's configuration.
// Supports both interactive mode (when policiesJSON is empty) and non-interactive
// mode (when policiesJSON is provided). Only Name, IsActive, and Policies can be
// updated. The client ID and secret remain unchanged.
//
// Requirements: Database must be migrated and the client must exist.
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
		policies, err = promptForPoliciesUpdate(io, existingClient.Policies)
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

	// Output result based on format
	if format == "json" {
		outputUpdateJSON(io.Writer, clientID, name, isActive)
	} else {
		outputUpdateText(io.Writer, clientID, name, isActive)
	}

	logger.Info("client updated successfully",
		slog.String("client_id", clientID.String()),
		slog.String("name", name),
		slog.Bool("is_active", isActive),
	)

	return nil
}

// promptForPoliciesUpdate interactively prompts the user to enter policy documents.
// Shows current policies and available capabilities. Accepts multiple policies until user declines.
func promptForPoliciesUpdate(
	io IOTuple,
	currentPolicies []authDomain.PolicyDocument,
) ([]authDomain.PolicyDocument, error) {
	reader := bufio.NewReader(io.Reader)
	var policies []authDomain.PolicyDocument

	_, _ = fmt.Fprintln(io.Writer, "\nCurrent policies:")
	for i, policy := range currentPolicies {
		capsStr := make([]string, len(policy.Capabilities))
		for j, cap := range policy.Capabilities {
			capsStr[j] = string(cap)
		}
		_, _ = fmt.Fprintf(
			io.Writer,
			"  %d. Path: %s, Capabilities: [%s]\n",
			i+1,
			policy.Path,
			strings.Join(capsStr, ", "),
		)
	}

	_, _ = fmt.Fprintln(io.Writer, "\nEnter new policies for the client")
	_, _ = fmt.Fprintln(io.Writer, "Available capabilities: read, write, delete, encrypt, decrypt, rotate")
	_, _ = fmt.Fprintln(io.Writer)

	policyNum := 1
	for {
		_, _ = fmt.Fprintf(io.Writer, "Policy #%d\n", policyNum)

		// Get path
		_, _ = fmt.Fprint(io.Writer, "Enter path pattern (e.g., 'secret/*' or '*'): ")
		path, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read path: %w", err)
		}
		path = strings.TrimSpace(path)

		if path == "" {
			return nil, fmt.Errorf("path cannot be empty")
		}

		// Get capabilities
		_, _ = fmt.Fprint(io.Writer, "Enter capabilities (comma-separated, e.g., 'read,write'): ")
		capsInput, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read capabilities: %w", err)
		}
		capsInput = strings.TrimSpace(capsInput)

		if capsInput == "" {
			return nil, fmt.Errorf("capabilities cannot be empty")
		}

		capabilities, err := parseCapabilities(capsInput)
		if err != nil {
			return nil, err
		}

		// Add policy
		policies = append(policies, authDomain.PolicyDocument{
			Path:         path,
			Capabilities: capabilities,
		})

		// Ask if user wants to add another
		_, _ = fmt.Fprint(io.Writer, "Add another policy? (y/n): ")
		addAnother, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}
		addAnother = strings.ToLower(strings.TrimSpace(addAnother))

		if addAnother != "y" && addAnother != "yes" {
			break
		}

		_, _ = fmt.Fprintln(io.Writer)
		policyNum++
	}

	return policies, nil
}

// outputUpdateText outputs the result in human-readable text format.
func outputUpdateText(writer io.Writer, clientID uuid.UUID, name string, isActive bool) {
	_, _ = fmt.Fprintln(writer, "\nClient updated successfully!")
	_, _ = fmt.Fprintf(writer, "Client ID: %s\n", clientID.String())
	_, _ = fmt.Fprintf(writer, "Name: %s\n", name)
	_, _ = fmt.Fprintf(writer, "Active: %t\n", isActive)
}

// outputUpdateJSON outputs the result in JSON format for machine consumption.
func outputUpdateJSON(writer io.Writer, clientID uuid.UUID, name string, isActive bool) {
	result := map[string]interface{}{
		"client_id": clientID.String(),
		"name":      name,
		"is_active": isActive,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}

	_, _ = fmt.Fprintln(writer, string(jsonBytes))
}
