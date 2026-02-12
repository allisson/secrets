// Package http provides HTTP middleware and utilities for authentication.
package http

import (
	"context"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// clientKey is a context key type for storing authenticated clients.
type clientKey struct{}

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
