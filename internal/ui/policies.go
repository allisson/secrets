// Package ui provides interactive CLI components and input validation for the application.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// PromptForPolicies interactively prompts the user to enter policy documents.
// Shows available capabilities and accepts multiple policies until user declines.
func PromptForPolicies(input io.Reader, output io.Writer) ([]authDomain.PolicyDocument, error) {
	reader := bufio.NewReader(input)
	var policies []authDomain.PolicyDocument

	_, _ = fmt.Fprintln(output, "\nEnter policies for the client")
	_, _ = fmt.Fprintln(output, "Available capabilities: read, write, delete, encrypt, decrypt, rotate")
	_, _ = fmt.Fprintln(output)

	policyNum := 1
	for {
		_, _ = fmt.Fprintf(output, "Policy #%d\n", policyNum)

		// Get path
		_, _ = fmt.Fprint(output, "Enter path pattern (e.g., 'secret/*' or '*'): ")
		path, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read path: %w", err)
		}
		path = strings.TrimSpace(path)

		if path == "" {
			return nil, fmt.Errorf("path cannot be empty")
		}

		// Get capabilities
		_, _ = fmt.Fprint(output, "Enter capabilities (comma-separated, e.g., 'read,write'): ")
		capsInput, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read capabilities: %w", err)
		}
		capsInput = strings.TrimSpace(capsInput)

		if capsInput == "" {
			return nil, fmt.Errorf("capabilities cannot be empty")
		}

		capabilities, err := ParseCapabilities(capsInput)
		if err != nil {
			return nil, err
		}

		// Add policy
		policies = append(policies, authDomain.PolicyDocument{
			Path:         path,
			Capabilities: capabilities,
		})

		// Ask if user wants to add another
		_, _ = fmt.Fprint(output, "Add another policy? (y/n): ")
		addAnother, err := reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("failed to read input: %w", err)
		}
		addAnother = strings.ToLower(strings.TrimSpace(addAnother))

		if addAnother != "y" && addAnother != "yes" {
			break
		}

		_, _ = fmt.Fprintln(output)
		policyNum++
	}

	return policies, nil
}

// PromptForPoliciesUpdate interactively prompts the user to enter policy documents during an update.
// Shows current policies and available capabilities.
func PromptForPoliciesUpdate(
	input io.Reader,
	output io.Writer,
	currentPolicies []authDomain.PolicyDocument,
) ([]authDomain.PolicyDocument, error) {
	_, _ = fmt.Fprintln(output, "\nCurrent policies:")
	for i, policy := range currentPolicies {
		capsStr := make([]string, len(policy.Capabilities))
		for j, cap := range policy.Capabilities {
			capsStr[j] = string(cap)
		}
		_, _ = fmt.Fprintf(
			output,
			"  %d. Path: %s, Capabilities: [%s]\n",
			i+1,
			policy.Path,
			strings.Join(capsStr, ", "),
		)
	}

	return PromptForPolicies(input, output)
}

// ParseCapabilities converts a comma-separated string into a slice of Capability.
func ParseCapabilities(input string) ([]authDomain.Capability, error) {
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
