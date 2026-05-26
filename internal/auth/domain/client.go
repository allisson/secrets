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
	ID             uuid.UUID        // Unique identifier (UUIDv7)
	Secret         string           //nolint:gosec // hashed client secret (not plaintext)
	Name           string           // Human-readable client name
	IsActive       bool             // Whether the client can authenticate
	Policies       []PolicyDocument // Authorization policies for this client
	FailedAttempts int              // Number of consecutive failed authentication attempts
	LockedUntil    *time.Time       // Time until which the client is locked (nil if not locked)
	CreatedAt      time.Time
}

// matchPath checks if the request path matches the policy path pattern.
// Supports three types of wildcards:
//  1. Full wildcard: "*" matches any path
//  2. Trailing wildcard: "prefix/*" matches any path starting with "prefix/" (greedy)
//  3. Mid-path wildcard: "/v1/keys/*/rotate" matches paths with * as single segment
//
// Examples:
//   - "*" matches any path
//   - "/v1/secrets/*" matches "/v1/secrets/app/db" and "/v1/secrets/app/db/password"
//   - "/v1/keys/*/rotate" matches "/v1/keys/payment/rotate" but NOT "/v1/keys/rotate"
//   - "/v1/*/keys/*/rotate" matches "/v1/transit/keys/payment/rotate"
func matchPath(policyPath, requestPath string) bool {
	// Special case: full wildcard matches everything
	if policyPath == "*" {
		return true
	}

	// No wildcard: exact match required
	if !strings.Contains(policyPath, "*") {
		return policyPath == requestPath
	}

	// Trailing wildcard (/*): prefix match (greedy - matches remaining path)
	if strings.HasSuffix(policyPath, "/*") {
		prefix := strings.TrimSuffix(policyPath, "/*")
		return strings.HasPrefix(requestPath, prefix+"/")
	}

	// Mid-path wildcards: segment-by-segment matching
	// Each * matches exactly one segment
	policyParts := strings.Split(policyPath, "/")
	requestParts := strings.Split(requestPath, "/")

	// Must have same number of segments for mid-path wildcards
	if len(policyParts) != len(requestParts) {
		return false
	}

	// Compare each segment
	for i := 0; i < len(policyParts); i++ {
		if policyParts[i] == "*" {
			// Wildcard matches any single segment
			continue
		}
		if policyParts[i] != requestParts[i] {
			return false
		}
	}

	return true
}

// IsAllowed checks if the client's policies permit the given capability on the specified path.
// Uses case-sensitive path matching with wildcard support. Returns true if any policy
// matches the path and includes the capability.
//
// Wildcard patterns:
//   - "*" matches everything (admin mode)
//   - "secret/*" matches any path starting with "secret/" (trailing wildcard - greedy)
//   - "/v1/keys/*/rotate" matches "/v1/keys/payment/rotate" (single-segment wildcard)
//   - "/v1/*/keys/*/rotate" matches "/v1/transit/keys/payment/rotate" (multiple wildcards)
//
// Path matching rules:
//   - Exact match: "secret" matches only "secret"
//   - Trailing wildcard: "secret/*" matches "secret/app", "secret/app/db", etc.
//   - Mid-path wildcard: "/v1/keys/*/rotate" matches exactly 4 segments with 3rd being any value
//   - Case-sensitive: "Secret" does NOT match "secret"
func (c *Client) IsAllowed(path string, capability Capability) bool {
	// Edge case: empty path or capability
	if path == "" || capability == "" {
		return false
	}

	// Iterate through all policies
	for _, policy := range c.Policies {
		// Check if path matches using wildcard support
		if matchPath(policy.Path, path) {
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
	ID          uuid.UUID        // Unique identifier for the created client (UUIDv7)
	PlainSecret string           // Plain text secret for authentication (transmit securely, never log)
	Name        string           // Human-readable client name
	IsActive    bool             // Whether the client can authenticate
	Policies    []PolicyDocument // Authorization policies for this client
	CreatedAt   time.Time        // Creation timestamp
}

// UpdateClientInput contains the mutable fields for updating an existing client.
// The client ID and secret cannot be modified through updates.
type UpdateClientInput struct {
	Name     string           // Updated human-readable name
	IsActive bool             // Updated active status (false prevents authentication)
	Policies []PolicyDocument // Updated authorization policies
}

// Decision is the outcome variant of Client.AttemptLogin.
type Decision int

const (
	// DecisionAuthenticated indicates the secret matched on an active, unlocked client.
	DecisionAuthenticated Decision = iota
	// DecisionBadSecret indicates the client exists and is active, but the secret did not match.
	DecisionBadSecret
	// DecisionLocked indicates the client is within an active lockout window; no attempt was counted.
	DecisionLocked
	// DecisionInactive indicates the client has IsActive=false; no attempt was counted.
	DecisionInactive
)

// LockoutPolicy configures how Client.AttemptLogin reacts to failures.
// MaxAttempts of zero disables locking; the counter is still incremented for visibility.
type LockoutPolicy struct {
	MaxAttempts int
	Duration    time.Duration
}

// LoginOutcome is the result of Client.AttemptLogin. It carries the post-attempt
// state the caller must persist (FailedAttempts, LockedUntil); the Client receiver
// is not mutated.
type LoginOutcome struct {
	Decision       Decision
	FailedAttempts int
	LockedUntil    *time.Time
}

// AttemptLogin runs the login state machine for this client and returns the outcome.
//
// Order of checks:
//  1. If LockedUntil is in the future, return Locked (attempt is not counted).
//  2. If the client is not active, return Inactive (attempt is not counted).
//  3. If compare returns false, return BadSecret with incremented attempts and,
//     when MaxAttempts > 0 and the threshold is reached, a fresh LockedUntil.
//  4. Otherwise return Authenticated with FailedAttempts reset to 0 and LockedUntil nil.
//
// The receiver is not mutated; the outcome is the single source of truth for the
// new (FailedAttempts, LockedUntil) tuple. The caller persists it.
func (c *Client) AttemptLogin(
	plain string,
	compare func(plain, hashed string) bool,
	policy LockoutPolicy,
	now time.Time,
) LoginOutcome {
	if c.LockedUntil != nil && now.Before(*c.LockedUntil) {
		return LoginOutcome{
			Decision:       DecisionLocked,
			FailedAttempts: c.FailedAttempts,
			LockedUntil:    c.LockedUntil,
		}
	}

	if !c.IsActive {
		return LoginOutcome{
			Decision:       DecisionInactive,
			FailedAttempts: c.FailedAttempts,
			LockedUntil:    c.LockedUntil,
		}
	}

	if !compare(plain, c.Secret) {
		newAttempts := c.FailedAttempts + 1
		var lockedUntil *time.Time
		if policy.MaxAttempts > 0 && newAttempts >= policy.MaxAttempts {
			t := now.Add(policy.Duration)
			lockedUntil = &t
		}
		return LoginOutcome{
			Decision:       DecisionBadSecret,
			FailedAttempts: newAttempts,
			LockedUntil:    lockedUntil,
		}
	}

	return LoginOutcome{
		Decision:       DecisionAuthenticated,
		FailedAttempts: 0,
		LockedUntil:    nil,
	}
}
