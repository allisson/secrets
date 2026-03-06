// Package http provides HTTP handlers for authentication and client management operations.
package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	"github.com/allisson/secrets/internal/auth/http/dto"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
	apperrors "github.com/allisson/secrets/internal/errors"
	"github.com/allisson/secrets/internal/httputil"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// ClientHandler handles HTTP requests for client management operations.
// It coordinates authentication, authorization, and audit logging with the ClientUseCase.
type ClientHandler struct {
	clientUseCase   authUseCase.ClientUseCase
	auditLogUseCase authUseCase.AuditLogUseCase
	logger          *slog.Logger
}

// NewClientHandler creates a new client handler with required dependencies.
func NewClientHandler(
	clientUseCase authUseCase.ClientUseCase,
	auditLogUseCase authUseCase.AuditLogUseCase,
	logger *slog.Logger,
) *ClientHandler {
	return &ClientHandler{
		clientUseCase:   clientUseCase,
		auditLogUseCase: auditLogUseCase,
		logger:          logger,
	}
}

// CreateHandler creates a new authentication client with policies.
// POST /v1/clients - Requires WriteCapability on path /v1/clients.
// Returns 201 Created with ID and plain text secret.
func (h *ClientHandler) CreateHandler(c *gin.Context) {
	var req dto.CreateClientRequest

	// Parse and bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		httputil.HandleValidationErrorGin(c, customValidation.WrapValidationError(err), h.logger)
		return
	}

	// Create input for use case
	input := &authDomain.CreateClientInput{
		Name:     req.Name,
		IsActive: req.IsActive,
		Policies: req.Policies,
	}

	// Call use case
	output, err := h.clientUseCase.Create(c.Request.Context(), input)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response with plain secret
	c.JSON(http.StatusCreated, dto.MapCreateClientOutputToResponse(output))
}

// GetHandler retrieves a client by ID.
// GET /v1/clients/:id - Requires ReadCapability on path /v1/clients/:id.
// Returns 200 OK with client data (no secret).
func (h *ClientHandler) GetHandler(c *gin.Context) {
	// Parse and validate UUID
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("invalid client ID format: must be a valid UUID"),
			h.logger)
		return
	}

	// Call use case
	client, err := h.clientUseCase.Get(c.Request.Context(), clientID)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	c.JSON(http.StatusOK, dto.MapClientToResponse(client))
}

// UpdateHandler updates an existing client's configuration.
// PUT /v1/clients/:id - Requires WriteCapability on path /v1/clients/:id.
// Returns 200 OK with updated client data (no secret).
func (h *ClientHandler) UpdateHandler(c *gin.Context) {
	// Parse and validate UUID
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("invalid client ID format: must be a valid UUID"),
			h.logger)
		return
	}

	var req dto.UpdateClientRequest

	// Parse and bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		httputil.HandleValidationErrorGin(c, customValidation.WrapValidationError(err), h.logger)
		return
	}

	// Create input for use case
	input := &authDomain.UpdateClientInput{
		Name:     req.Name,
		IsActive: req.IsActive,
		Policies: req.Policies,
	}

	// Call use case
	if err := h.clientUseCase.Update(c.Request.Context(), clientID, input); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Get updated client to return
	client, err := h.clientUseCase.Get(c.Request.Context(), clientID)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	c.JSON(http.StatusOK, dto.MapClientToResponse(client))
}

// DeleteHandler soft deletes a client by setting IsActive to false.
// DELETE /v1/clients/:id - Requires DeleteCapability on path /v1/clients/:id.
// Returns 204 No Content.
func (h *ClientHandler) DeleteHandler(c *gin.Context) {
	// Parse and validate UUID
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("invalid client ID format: must be a valid UUID"),
			h.logger)
		return
	}

	// Call use case
	if err := h.clientUseCase.Delete(c.Request.Context(), clientID); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return 204 No Content with empty body
	c.Data(http.StatusNoContent, "application/json", nil)
}

// UnlockHandler resets the lockout state for a client.
// POST /v1/clients/:id/unlock - Requires WriteCapability.
// Returns 200 OK with updated client data.
func (h *ClientHandler) UnlockHandler(c *gin.Context) {
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("invalid client ID format: must be a valid UUID"),
			h.logger)
		return
	}
	if err := h.clientUseCase.Unlock(c.Request.Context(), clientID); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}
	client, err := h.clientUseCase.Get(c.Request.Context(), clientID)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}
	c.JSON(http.StatusOK, dto.MapClientToResponse(client))
}

// ListHandler retrieves clients with cursor-based pagination support.
// GET /v1/clients?after_id=<uuid>&limit=50 - Requires ReadCapability on path /v1/clients.
// Returns 200 OK with paginated client list ordered by ID descending. Uses cursor pagination
// with after_id parameter.
func (h *ClientHandler) ListHandler(c *gin.Context) {
	// Parse cursor and limit query parameters
	afterID, limit, err := httputil.ParseUUIDCursorPagination(c, "after_id")
	if err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	// Call use case with limit + 1 to detect if there are more results
	clients, err := h.clientUseCase.ListCursor(c.Request.Context(), afterID, limit+1)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Determine if there are more results and set next cursor
	var nextCursor *string
	if len(clients) > limit {
		// More results exist, use the last visible item's ID as cursor
		clients = clients[:limit]
		cursorValue := clients[len(clients)-1].ID.String()
		nextCursor = &cursorValue
	}

	// Map to response
	response := dto.MapClientsToListResponse(clients, nextCursor)
	c.JSON(http.StatusOK, response)
}

// RevokeTokensHandler revokes all active tokens for a client.
// DELETE /v1/clients/:id/tokens - Requires DeleteCapability.
// Returns 204 No Content.
func (h *ClientHandler) RevokeTokensHandler(c *gin.Context) {
	// Parse and validate UUID
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("invalid client ID format: must be a valid UUID"),
			h.logger)
		return
	}

	// Call use case
	if err := h.clientUseCase.RevokeTokens(c.Request.Context(), clientID); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return 204 No Content
	c.Data(http.StatusNoContent, "application/json", nil)
}

// RotateSecretHandler rotates the secret for a client.
// It supports two modes:
// 1. Self-service: POST /v1/auth/clients/self/rotate-secret (id from authenticated client)
// 2. Administrative: POST /v1/auth/clients/:id/rotate-secret (id from URL parameter)
// Both modes require RotateCapability on the respective path.
func (h *ClientHandler) RotateSecretHandler(c *gin.Context) {
	var clientID uuid.UUID
	var err error

	// Determine if this is a self-service or administrative request
	idParam := c.Param("id")
	if idParam == "self" {
		// Self-service mode: get ID from authenticated client
		client, ok := GetClient(c.Request.Context())
		if !ok || client == nil {
			httputil.HandleErrorGin(c, apperrors.ErrUnauthorized, h.logger)
			return
		}
		clientID = client.ID
	} else {
		// Administrative mode: get ID from URL parameter
		clientID, err = uuid.Parse(idParam)
		if err != nil {
			httputil.HandleBadRequestGin(c,
				fmt.Errorf("invalid client ID format: must be a valid UUID"),
				h.logger)
			return
		}
	}

	// Call use case
	output, err := h.clientUseCase.RotateSecret(c.Request.Context(), clientID)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response with new plain secret and metadata
	c.JSON(http.StatusOK, dto.MapCreateClientOutputToResponse(output))
}
