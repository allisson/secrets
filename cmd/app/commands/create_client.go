package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
)

// RunCreateClient creates a new authentication client with policies.
// Supports both interactive mode (when policiesJSON is empty) and non-interactive
// mode (when policiesJSON is provided). Outputs client ID and plain secret in
// either text or JSON format.
//
// Requirements: Database must be migrated and accessible.
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
		policies, err = promptForPolicies(io)
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

	// Output result based on format
	if format == "json" {
		outputJSON(output, io.Writer)
	} else {
		outputText(output, io.Writer)
	}

	logger.Info("client created successfully",
		slog.String("client_id", output.ID.String()),
		slog.String("name", name),
		slog.Bool("is_active", isActive),
	)

	return nil
}

// promptForPolicies interactively prompts the user to enter policy documents.
// Shows available capabilities and accepts multiple policies until user declines.
func promptForPolicies(io IOTuple) ([]authDomain.PolicyDocument, error) {
	reader := bufio.NewReader(io.Reader)
	writer := io.Writer
	var policies []authDomain.PolicyDocument

	_, _ = fmt.Fprintln(writer, "\nEnter policies for the client")
	_, _ = fmt.Fprintln(writer, "Available capabilities: read, write, delete, encrypt, decrypt, rotate")
	_, _ = fmt.Fprintln(writer)

	policyNum := 1
	for {
		_, _ = fmt.Fprintf(writer, "Policy #%d\n", policyNum)

		// Get path
		_, _ = fmt.Fprint(writer, "Enter path pattern (e.g., 'secret/*' or '*'): ")
		path, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read path: %w", err)
		}
		path = strings.TrimSpace(path)

		if path == "" {
			return nil, fmt.Errorf("path cannot be empty")
		}

		// Get capabilities
		_, _ = fmt.Fprint(writer, "Enter capabilities (comma-separated, e.g., 'read,write'): ")
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
		_, _ = fmt.Fprint(writer, "Add another policy? (y/n): ")
		addAnother, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}
		addAnother = strings.ToLower(strings.TrimSpace(addAnother))

		if addAnother != "y" && addAnother != "yes" {
			break
		}

		_, _ = fmt.Fprintln(writer)
		policyNum++
	}

	return policies, nil
}

// parseCapabilities converts a comma-separated string into a slice of Capability.
func parseCapabilities(input string) ([]authDomain.Capability, error) {
	parts := strings.Split(input, ",")
	capabilities := make([]authDomain.Capability, 0, len(parts))

	for _, part := range parts {
		cap := authDomain.Capability(strings.TrimSpace(part))
		if cap != "" {
			capabilities = append(capabilities, cap)
		}
	}

	if len(capabilities) == 0 {
		return nil, fmt.Errorf("at least one capability is required")
	}

	return capabilities, nil
}

// outputText outputs the result in human-readable text format.
func outputText(output *authDomain.CreateClientOutput, writer io.Writer) {
	_, _ = fmt.Fprintln(writer, "\nClient created successfully!")
	_, _ = fmt.Fprintf(writer, "Client ID: %s\n", output.ID.String())
	_, _ = fmt.Fprintf(writer, "Secret: %s\n", output.PlainSecret)
	_, _ = fmt.Fprintln(writer, "\nIMPORTANT: The secret is shown only once. Store it securely.")
}

// outputJSON outputs the result in JSON format for machine consumption.
func outputJSON(output *authDomain.CreateClientOutput, writer io.Writer) {
	result := map[string]string{
		"client_id": output.ID.String(),
		"secret":    output.PlainSecret,
	}

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to marshal JSON: %v\n", err)
		return
	}

	_, _ = fmt.Fprintln(writer, string(jsonBytes))
}
