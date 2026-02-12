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
	"github.com/allisson/secrets/internal/httputil"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// TokenHandler handles HTTP requests for token operations.
// It coordinates token issuance with the TokenUseCase.
type TokenHandler struct {
	tokenUseCase authUseCase.TokenUseCase
	logger       *slog.Logger
}

// NewTokenHandler creates a new token handler with required dependencies.
func NewTokenHandler(
	tokenUseCase authUseCase.TokenUseCase,
	logger *slog.Logger,
) *TokenHandler {
	return &TokenHandler{
		tokenUseCase: tokenUseCase,
		logger:       logger,
	}
}

// IssueTokenHandler issues a new authentication token for a client.
// POST /v1/token - No authentication required (this is the authentication endpoint).
// Returns 201 Created with token and expiration time.
func (h *TokenHandler) IssueTokenHandler(c *gin.Context) {
	var req dto.IssueTokenRequest

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

	// Parse client ID as UUID
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		httputil.HandleValidationErrorGin(c,
			fmt.Errorf("invalid client_id format: must be a valid UUID"),
			h.logger)
		return
	}

	// Create input for use case
	input := &authDomain.IssueTokenInput{
		ClientID:     clientID,
		ClientSecret: req.ClientSecret,
	}

	// Call use case
	output, err := h.tokenUseCase.Issue(c.Request.Context(), input)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response with token and expiration
	response := dto.IssueTokenResponse{
		Token:     output.PlainToken,
		ExpiresAt: output.ExpiresAt,
	}

	c.JSON(http.StatusCreated, response)
}
