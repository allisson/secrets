// Package domain defines authentication and authorization domain models and business logic.
//
// It provides client-based authentication with policy-based authorization. Clients authenticate
// using secrets and are authorized via capability-based policies that control access to resource paths.
package domain

import (
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PolicyDocument defines access control rules for a specific resource path.
// Policies use prefix matching with wildcard support for flexible authorization.
type PolicyDocument struct {
	Path         string       `json:"path"`         // Resource path pattern (supports "*" and "/*" wildcards)
	Capabilities []Capability `json:"capabilities"` // List of allowed operations on the resource
}

// Client represents an authentication client with associated authorization policies.
// Clients are used to authenticate API requests and enforce access control.
type Client struct {
	ID        uuid.UUID        // Unique identifier (UUIDv7)
	Secret    string           // Hashed client secret for authentication
	Name      string           // Human-readable client name
	IsActive  bool             // Whether the client can authenticate
	Policies  []PolicyDocument // Authorization policies for this client
	CreatedAt time.Time
}

// IsAllowed checks if the client's policies permit the given capability on the specified path.
// Uses case-sensitive prefix matching with wildcard support. Returns true if any policy
// matches the path and includes the capability.
//
// Wildcard patterns:
//   - "*" matches everything (admin mode)
//   - "secret/*" matches any path starting with "secret/"
//   - "secret" matches exactly "secret"
func (c *Client) IsAllowed(path string, capability Capability) bool {
	// Edge case: empty path or capability
	if path == "" || capability == "" {
		return false
	}

	// Iterate through all policies
	for _, policy := range c.Policies {
		var pathMatches bool

		// Check path matching based on pattern type
		switch {
		case policy.Path == "*":
			// Wildcard matches everything
			pathMatches = true
		case strings.HasSuffix(policy.Path, "/*"):
			// Prefix matching: strip "/*" and check if path starts with "prefix/"
			prefix := strings.TrimSuffix(policy.Path, "/*")
			pathMatches = strings.HasPrefix(path, prefix+"/")
		default:
			// Exact match
			pathMatches = policy.Path == path
		}

		// If path matches, check if capability is in the list
		if pathMatches {
			if slices.Contains(policy.Capabilities, capability) {
				return true
			}
		}
	}

	// No matching policy found
	return false
}

// CreateClientInput contains the parameters for creating a new authentication client.
// The client secret will be automatically generated and cannot be specified by the caller.
type CreateClientInput struct {
	Name     string           // Human-readable name for identifying the client
	IsActive bool             // Whether the client can authenticate immediately after creation
	Policies []PolicyDocument // Authorization policies defining resource access permissions
}

// CreateClientOutput contains the result of creating a new client.
// SECURITY: The PlainSecret is only returned once and must be securely transmitted
// to the client. It will never be retrievable again after this response.
type CreateClientOutput struct {
	ID          uuid.UUID // Unique identifier for the created client (UUIDv7)
	PlainSecret string    // Plain text secret for authentication (transmit securely, never log)
}

// UpdateClientInput contains the mutable fields for updating an existing client.
// The client ID and secret cannot be modified through updates.
type UpdateClientInput struct {
	Name     string           // Updated human-readable name
	IsActive bool             // Updated active status (false prevents authentication)
	Policies []PolicyDocument // Updated authorization policies
}
