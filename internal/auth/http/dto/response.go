// Package dto provides data transfer objects for HTTP request and response handling.
package dto

import (
	"time"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

// CreateClientResponse contains the result of creating a new client.
// SECURITY: The secret is only returned once and must be saved securely.
type CreateClientResponse struct {
	ID     string `json:"id"`
	Secret string `json:"secret"` //nolint:gosec // returned once on creation
}

// ClientResponse represents a client in API responses (excludes secret).
type ClientResponse struct {
	ID        string                      `json:"id"`
	Name      string                      `json:"name"`
	IsActive  bool                        `json:"is_active"`
	Policies  []authDomain.PolicyDocument `json:"policies"`
	CreatedAt time.Time                   `json:"created_at"`
}

// MapClientToResponse converts a domain client to an API response.
func MapClientToResponse(client *authDomain.Client) ClientResponse {
	return ClientResponse{
		ID:        client.ID.String(),
		Name:      client.Name,
		IsActive:  client.IsActive,
		Policies:  client.Policies,
		CreatedAt: client.CreatedAt,
	}
}

// ListClientsResponse represents a paginated list of clients in API responses.
type ListClientsResponse struct {
	Data []ClientResponse `json:"data"`
}

// MapClientsToListResponse converts a slice of domain clients to a list API response.
func MapClientsToListResponse(clients []*authDomain.Client) ListClientsResponse {
	clientResponses := make([]ClientResponse, 0, len(clients))
	for _, client := range clients {
		clientResponses = append(clientResponses, MapClientToResponse(client))
	}
	return ListClientsResponse{
		Data: clientResponses,
	}
}

// IssueTokenResponse contains the result of issuing a token.
// SECURITY: The token is only returned once and must be saved securely.
type IssueTokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// AuditLogResponse represents an audit log entry in API responses.
type AuditLogResponse struct {
	ID         string         `json:"id"`
	RequestID  string         `json:"request_id"`
	ClientID   string         `json:"client_id"`
	Capability string         `json:"capability"`
	Path       string         `json:"path"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

// MapAuditLogToResponse converts a domain audit log to an API response.
func MapAuditLogToResponse(auditLog *authDomain.AuditLog) AuditLogResponse {
	return AuditLogResponse{
		ID:         auditLog.ID.String(),
		RequestID:  auditLog.RequestID.String(),
		ClientID:   auditLog.ClientID.String(),
		Capability: string(auditLog.Capability),
		Path:       auditLog.Path,
		Metadata:   auditLog.Metadata,
		CreatedAt:  auditLog.CreatedAt,
	}
}

// ListAuditLogsResponse represents a paginated list of audit logs in API responses.
type ListAuditLogsResponse struct {
	Data []AuditLogResponse `json:"data"`
}

// MapAuditLogsToListResponse converts a slice of domain audit logs to a list API response.
func MapAuditLogsToListResponse(auditLogs []*authDomain.AuditLog) ListAuditLogsResponse {
	auditLogResponses := make([]AuditLogResponse, 0, len(auditLogs))
	for _, auditLog := range auditLogs {
		auditLogResponses = append(auditLogResponses, MapAuditLogToResponse(auditLog))
	}
	return ListAuditLogsResponse{
		Data: auditLogResponses,
	}
}
