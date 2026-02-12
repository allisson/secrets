// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"context"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// clientKey is a context key type for storing authenticated clients.
type clientKey struct{}

// pathKey is a context key type for storing authorized paths.
type pathKey struct{}

// capabilityKey is a context key type for storing authorized capabilities.
type capabilityKey struct{}

// WithClient stores an authenticated client in the context.
// This is typically called by the authentication middleware after successful token validation.
func WithClient(ctx context.Context, client *authDomain.Client) context.Context {
	return context.WithValue(ctx, clientKey{}, client)
}

// GetClient retrieves an authenticated client from the context.
// Returns (client, true) if a client is present, or (nil, false) if no client was set.
// This is typically called by handlers or subsequent middleware that need the authenticated client.
func GetClient(ctx context.Context) (*authDomain.Client, bool) {
	client, ok := ctx.Value(clientKey{}).(*authDomain.Client)
	return client, ok
}

// WithPath stores the authorized path in the context.
// This is typically called by the authorization middleware after successful authorization check.
func WithPath(ctx context.Context, path string) context.Context {
	return context.WithValue(ctx, pathKey{}, path)
}

// GetPath retrieves the authorized path from the context.
// Returns (path, true) if a path is present, or ("", false) if no path was set.
// This is typically called by handlers that need the authorized path for audit logging.
func GetPath(ctx context.Context) (string, bool) {
	path, ok := ctx.Value(pathKey{}).(string)
	return path, ok
}

// WithCapability stores the authorized capability in the context.
// This is typically called by the authorization middleware after successful authorization check.
func WithCapability(ctx context.Context, capability authDomain.Capability) context.Context {
	return context.WithValue(ctx, capabilityKey{}, capability)
}

// GetCapability retrieves the authorized capability from the context.
// Returns (capability, true) if present, or ("", false) if not set.
// This is typically called by handlers that need the authorized capability for audit logging.
func GetCapability(ctx context.Context) (authDomain.Capability, bool) {
	capability, ok := ctx.Value(capabilityKey{}).(authDomain.Capability)
	return capability, ok
}
