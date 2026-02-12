// Package http provides HTTP handlers for authentication and client management operations.
package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	validation "github.com/jellydator/validation"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
	authUseCase "github.com/allisson/secrets/internal/auth/usecase"
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

// CreateClientRequest contains the parameters for creating a new authentication client.
type CreateClientRequest struct {
	Name     string                      `json:"name"`
	IsActive bool                        `json:"is_active"`
	Policies []authDomain.PolicyDocument `json:"policies"`
}

// Validate checks if the create client request is valid.
func (r *CreateClientRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required,
			customValidation.NotBlank,
			validation.Length(1, 255),
		),
		validation.Field(&r.Policies,
			validation.Required,
			validation.Each(validation.By(validatePolicyDocument)),
		),
	)
}

// UpdateClientRequest contains the parameters for updating an existing client.
type UpdateClientRequest struct {
	Name     string                      `json:"name"`
	IsActive bool                        `json:"is_active"`
	Policies []authDomain.PolicyDocument `json:"policies"`
}

// Validate checks if the update client request is valid.
func (r *UpdateClientRequest) Validate() error {
	return validation.ValidateStruct(r,
		validation.Field(&r.Name,
			validation.Required,
			customValidation.NotBlank,
			validation.Length(1, 255),
		),
		validation.Field(&r.Policies,
			validation.Required,
			validation.Each(validation.By(validatePolicyDocument)),
		),
	)
}

// validatePolicyDocument validates a single policy document.
func validatePolicyDocument(value interface{}) error {
	policy, ok := value.(authDomain.PolicyDocument)
	if !ok {
		return validation.NewError("validation_policy_type", "must be a policy document")
	}

	return validation.ValidateStruct(&policy,
		validation.Field(&policy.Path,
			validation.Required,
			customValidation.NotBlank,
			validation.Length(1, 500),
		),
		validation.Field(&policy.Capabilities,
			validation.Required,
			validation.Length(1, 0), // At least one capability
		),
	)
}

// CreateClientResponse contains the result of creating a new client.
// SECURITY: The secret is only returned once and must be saved securely.
type CreateClientResponse struct {
	ID     string `json:"id"`
	Secret string `json:"secret"`
}

// ClientResponse represents a client in API responses (excludes secret).
type ClientResponse struct {
	ID        string                      `json:"id"`
	Name      string                      `json:"name"`
	IsActive  bool                        `json:"is_active"`
	Policies  []authDomain.PolicyDocument `json:"policies"`
	CreatedAt time.Time                   `json:"created_at"`
}

// mapClientToResponse converts a domain client to an API response.
func mapClientToResponse(client *authDomain.Client) ClientResponse {
	return ClientResponse{
		ID:        client.ID.String(),
		Name:      client.Name,
		IsActive:  client.IsActive,
		Policies:  client.Policies,
		CreatedAt: client.CreatedAt,
	}
}

// CreateHandler creates a new authentication client with policies.
// POST /v1/clients - Requires WriteCapability on path /v1/clients.
// Returns 201 Created with ID and plain text secret.
func (h *ClientHandler) CreateHandler(c *gin.Context) {
	var req CreateClientRequest

	// Parse and bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
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
	response := CreateClientResponse{
		ID:     output.ID.String(),
		Secret: output.PlainSecret,
	}

	c.JSON(http.StatusCreated, response)
}

// GetHandler retrieves a client by ID.
// GET /v1/clients/:id - Requires ReadCapability on path /v1/clients/:id.
// Returns 200 OK with client data (no secret).
func (h *ClientHandler) GetHandler(c *gin.Context) {
	// Parse and validate UUID
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleValidationErrorGin(c,
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
	c.JSON(http.StatusOK, mapClientToResponse(client))
}

// UpdateHandler updates an existing client's configuration.
// PUT /v1/clients/:id - Requires WriteCapability on path /v1/clients/:id.
// Returns 200 OK with updated client data (no secret).
func (h *ClientHandler) UpdateHandler(c *gin.Context) {
	// Parse and validate UUID
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("invalid client ID format: must be a valid UUID"),
			h.logger)
		return
	}

	var req UpdateClientRequest

	// Parse and bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.HandleValidationErrorGin(c, err, h.logger)
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
	c.JSON(http.StatusOK, mapClientToResponse(client))
}

// DeleteHandler soft deletes a client by setting IsActive to false.
// DELETE /v1/clients/:id - Requires DeleteCapability on path /v1/clients/:id.
// Returns 204 No Content.
func (h *ClientHandler) DeleteHandler(c *gin.Context) {
	// Parse and validate UUID
	clientID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httputil.HandleValidationErrorGin(c,
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
