// Package http provides HTTP handlers for transit key management and cryptographic operations.
package http

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/httputil"
	"github.com/allisson/secrets/internal/transit/http/dto"
	transitUseCase "github.com/allisson/secrets/internal/transit/usecase"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// CryptoHandler handles HTTP requests for transit encryption and decryption operations.
// It coordinates authentication, authorization, and audit logging with the TransitKeyUseCase.
type CryptoHandler struct {
	transitKeyUseCase transitUseCase.TransitKeyUseCase // Business logic for encryption and decryption operations
	logger            *slog.Logger                     // Structured logger for request handling and error reporting
}

// NewCryptoHandler creates a new crypto handler with required dependencies.
func NewCryptoHandler(
	transitKeyUseCase transitUseCase.TransitKeyUseCase,
	logger *slog.Logger,
) *CryptoHandler {
	return &CryptoHandler{
		transitKeyUseCase: transitKeyUseCase,
		logger:            logger,
	}
}

// EncryptHandler encrypts plaintext data using the specified transit key.
// POST /v1/transit/keys/:name/encrypt - Requires EncryptCapability.
// Returns 200 OK with ciphertext in format "version:base64-ciphertext".
func (h *CryptoHandler) EncryptHandler(c *gin.Context) {
	// Extract and validate name from URL parameter
	name := c.Param("name")
	if name == "" {
		httputil.HandleBadRequestGin(
			c,
			fmt.Errorf("transit key name cannot be empty"),
			h.logger,
		)
		return
	}

	var req dto.EncryptRequest

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

	// Decode base64 plaintext
	plaintext, err := base64.StdEncoding.DecodeString(req.Plaintext)
	if err != nil {
		httputil.HandleBadRequestGin(c, fmt.Errorf("invalid base64 plaintext: %w", err), h.logger)
		return
	}

	// Call use case
	encryptedBlob, err := h.transitKeyUseCase.Encrypt(c.Request.Context(), name, plaintext)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response with ciphertext string
	response := dto.EncryptResponse{
		Ciphertext: encryptedBlob.String(), // Format: "version:base64-ciphertext"
		Version:    encryptedBlob.Version,
	}
	c.JSON(http.StatusOK, response)
}

// DecryptHandler decrypts ciphertext using the version specified in the encrypted blob.
// POST /v1/transit/keys/:name/decrypt - Requires DecryptCapability.
// Returns 200 OK with plaintext bytes. SECURITY: Plaintext is zeroed after response.
func (h *CryptoHandler) DecryptHandler(c *gin.Context) {
	// Extract and validate name from URL parameter
	name := c.Param("name")
	if name == "" {
		httputil.HandleBadRequestGin(
			c,
			fmt.Errorf("transit key name cannot be empty"),
			h.logger,
		)
		return
	}

	var req dto.DecryptRequest

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

	// Call use case with ciphertext string (format: "version:base64-ciphertext")
	decryptedBlob, err := h.transitKeyUseCase.Decrypt(c.Request.Context(), name, req.Ciphertext)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// SECURITY: Zero plaintext after mapping to response
	defer cryptoDomain.Zero(decryptedBlob.Plaintext)

	// Return response with base64-encoded plaintext
	response := dto.MapDecryptResponse(decryptedBlob.Plaintext, decryptedBlob.Version)
	c.JSON(http.StatusOK, response)
}
