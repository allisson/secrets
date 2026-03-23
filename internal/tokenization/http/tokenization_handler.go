// Package http provides HTTP handlers for tokenization key management and token operations.
package http

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	"github.com/allisson/secrets/internal/httputil"
	"github.com/allisson/secrets/internal/tokenization/http/dto"
	tokenizationUseCase "github.com/allisson/secrets/internal/tokenization/usecase"
	customValidation "github.com/allisson/secrets/internal/validation"
)

// TokenizationHandler handles HTTP requests for tokenization operations.
// Coordinates tokenize, detokenize, validate, and revoke operations with TokenizationUseCase.
type TokenizationHandler struct {
	tokenizationUseCase tokenizationUseCase.TokenizationUseCase
	batchLimit          int
	logger              *slog.Logger
}

// NewTokenizationHandler creates a new tokenization handler with required dependencies.
func NewTokenizationHandler(
	tokenizationUseCase tokenizationUseCase.TokenizationUseCase,
	batchLimit int,
	logger *slog.Logger,
) *TokenizationHandler {
	return &TokenizationHandler{
		tokenizationUseCase: tokenizationUseCase,
		batchLimit:          batchLimit,
		logger:              logger,
	}
}

// TokenizeHandler generates a token for the given plaintext value using the named key.
// POST /v1/tokenization/keys/:name/tokenize - Requires EncryptCapability.
// In deterministic mode, returns existing token if the value has been tokenized before.
// Returns 201 Created with token and metadata.
func (h *TokenizationHandler) TokenizeHandler(c *gin.Context) {
	var req dto.TokenizeRequest

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

	// Get key name from URL parameter
	keyName := c.Param("name")
	if keyName == "" {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("key name is required in URL path"),
			h.logger)
		return
	}

	// Decode base64 plaintext
	plaintext, err := base64.StdEncoding.DecodeString(req.Plaintext)
	if err != nil {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("plaintext must be valid base64"),
			h.logger)
		return
	}

	// Calculate expiration time if TTL is provided
	var expiresAt *time.Time
	if req.TTL != nil {
		expiry := time.Now().UTC().Add(time.Duration(*req.TTL) * time.Second)
		expiresAt = &expiry
	}

	// Call use case
	token, err := h.tokenizationUseCase.Tokenize(
		c.Request.Context(),
		keyName,
		plaintext,
		req.Metadata,
		expiresAt,
	)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	response := dto.MapTokenToTokenizeResponse(token)
	c.JSON(http.StatusCreated, response)
}

// TokenizeBatchHandler generates tokens for multiple plaintext values using the named key.
// POST /v1/tokenization/keys/:name/tokenize-batch - Requires EncryptCapability.
// Wrapped in a transaction for atomicity.
// Returns 201 Created with a batch of tokens and metadata.
func (h *TokenizationHandler) TokenizeBatchHandler(c *gin.Context) {
	var req dto.TokenizeBatchRequest

	// Parse and bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	// Validate request
	if err := req.Validate(h.batchLimit); err != nil {
		httputil.HandleValidationErrorGin(c, customValidation.WrapValidationError(err), h.logger)
		return
	}

	// Get key name from URL parameter
	keyName := c.Param("name")
	if keyName == "" {
		httputil.HandleBadRequestGin(c,
			fmt.Errorf("key name is required in URL path"),
			h.logger)
		return
	}

	// Prepare data for use case
	plaintexts := make([][]byte, len(req.Items))
	metadatas := make([]map[string]any, len(req.Items))
	var commonExpiresAt *time.Time

	for i, item := range req.Items {
		// Decode base64 plaintext
		plaintext, err := base64.StdEncoding.DecodeString(item.Plaintext)
		if err != nil {
			httputil.HandleBadRequestGin(c,
				fmt.Errorf("item %d: plaintext must be valid base64", i),
				h.logger)
			return
		}
		plaintexts[i] = plaintext

		// Setup metadata
		metadatas[i] = item.Metadata

		// Note: The usecase currently takes a single expiresAt for the batch.
		// For simplicity, we'll use the TTL of the first item if provided.
		// A more advanced implementation could support individual TTLs if the usecase is updated.
		if i == 0 && item.TTL != nil {
			expiry := time.Now().UTC().Add(time.Duration(*item.TTL) * time.Second)
			commonExpiresAt = &expiry
		}
	}

	// SECURITY: Ensure plaintexts are zeroed after use
	defer func() {
		for _, p := range plaintexts {
			cryptoDomain.Zero(p)
		}
	}()

	// Call use case
	tokens, err := h.tokenizationUseCase.TokenizeBatch(
		c.Request.Context(),
		keyName,
		plaintexts,
		metadatas,
		commonExpiresAt,
	)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	response := dto.MapTokensToTokenizeBatchResponse(tokens)
	c.JSON(http.StatusCreated, response)
}

// DetokenizeHandler retrieves the original plaintext value for a given token.
// POST /v1/tokenization/detokenize - Requires DecryptCapability.
// Returns 200 OK with base64-encoded plaintext and metadata.
func (h *TokenizationHandler) DetokenizeHandler(c *gin.Context) {
	var req dto.DetokenizeRequest

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

	// Call use case
	plaintext, metadata, err := h.tokenizationUseCase.Detokenize(
		c.Request.Context(),
		req.Token,
	)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}
	// SECURITY: Zero plaintext from memory after encoding
	defer cryptoDomain.Zero(plaintext)

	// Encode plaintext as base64 for JSON response
	plaintextB64 := base64.StdEncoding.EncodeToString(plaintext)

	// Return response
	response := dto.DetokenizeResponse{
		Plaintext: plaintextB64,
		Metadata:  metadata,
	}
	c.JSON(http.StatusOK, response)
}

// DetokenizeBatchHandler retrieves original plaintext values for multiple tokens.
// POST /v1/tokenization/detokenize-batch - Requires DecryptCapability.
// Wrapped in a transaction for atomicity.
// Returns 200 OK with a batch of base64-encoded plaintexts and metadata.
func (h *TokenizationHandler) DetokenizeBatchHandler(c *gin.Context) {
	var req dto.DetokenizeBatchRequest

	// Parse and bind JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.HandleBadRequestGin(c, err, h.logger)
		return
	}

	// Validate request
	if err := req.Validate(h.batchLimit); err != nil {
		httputil.HandleValidationErrorGin(c, customValidation.WrapValidationError(err), h.logger)
		return
	}

	// Call use case
	plaintexts, metadatas, err := h.tokenizationUseCase.DetokenizeBatch(
		c.Request.Context(),
		req.Tokens,
	)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// SECURITY: Ensure plaintexts are zeroed after encoding
	defer func() {
		for _, p := range plaintexts {
			cryptoDomain.Zero(p)
		}
	}()

	// Encode plaintexts as base64 for JSON response
	plaintextB64s := make([]string, len(plaintexts))
	for i, p := range plaintexts {
		plaintextB64s[i] = base64.StdEncoding.EncodeToString(p)
	}

	// Return response
	response := dto.MapPlaintextsToDetokenizeBatchResponse(plaintextB64s, metadatas)
	c.JSON(http.StatusOK, response)
}

// ValidateHandler checks if a token exists and is valid (not expired or revoked).
// POST /v1/tokenization/validate - Requires ReadCapability.
// Returns 200 OK with validation result.
func (h *TokenizationHandler) ValidateHandler(c *gin.Context) {
	var req dto.ValidateTokenRequest

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

	// Call use case
	isValid, err := h.tokenizationUseCase.Validate(
		c.Request.Context(),
		req.Token,
	)
	if err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return response
	response := dto.ValidateTokenResponse{
		Valid: isValid,
	}
	c.JSON(http.StatusOK, response)
}

// RevokeHandler marks a token as revoked, preventing further detokenization.
// POST /v1/tokenization/revoke - Requires DeleteCapability.
// Returns 204 No Content on success.
func (h *TokenizationHandler) RevokeHandler(c *gin.Context) {
	var req dto.RevokeTokenRequest

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

	// Call use case
	if err := h.tokenizationUseCase.Revoke(c.Request.Context(), req.Token); err != nil {
		httputil.HandleErrorGin(c, err, h.logger)
		return
	}

	// Return 204 No Content
	c.Data(http.StatusNoContent, "application/json", nil)
}
