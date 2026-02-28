// Package httputil provides HTTP utility functions for request and response handling.
package httputil

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	apperrors "github.com/allisson/secrets/internal/errors"
)

// ErrorResponse represents a structured error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// HandleErrorGin maps domain errors to HTTP status codes and returns a JSON response using Gin.
// This is an adapter for Gin's context that maintains the same error handling logic.
func HandleErrorGin(c *gin.Context, err error, logger *slog.Logger) {
	if err == nil {
		return
	}

	var statusCode int
	var errorResponse ErrorResponse

	// Map domain errors to HTTP status codes (same logic as HandleError)
	switch {
	case apperrors.Is(err, apperrors.ErrNotFound):
		statusCode = http.StatusNotFound
		errorResponse = ErrorResponse{
			Error:   "not_found",
			Message: "The requested resource was not found",
		}

	case apperrors.Is(err, apperrors.ErrConflict):
		statusCode = http.StatusConflict
		errorResponse = ErrorResponse{
			Error:   "conflict",
			Message: "A conflict occurred with existing data",
		}

	case apperrors.Is(err, apperrors.ErrInvalidInput):
		statusCode = http.StatusUnprocessableEntity
		errorResponse = ErrorResponse{
			Error:   "invalid_input",
			Message: err.Error(),
		}

	case apperrors.Is(err, apperrors.ErrUnauthorized):
		statusCode = http.StatusUnauthorized
		errorResponse = ErrorResponse{
			Error:   "unauthorized",
			Message: "Authentication is required",
		}

	case apperrors.Is(err, apperrors.ErrLocked):
		statusCode = http.StatusLocked
		errorResponse = ErrorResponse{
			Error:   "client_locked",
			Message: "Account is locked due to too many failed authentication attempts",
		}

	case apperrors.Is(err, apperrors.ErrForbidden):
		statusCode = http.StatusForbidden
		errorResponse = ErrorResponse{
			Error:   "forbidden",
			Message: "You don't have permission to access this resource",
		}

	default:
		// For unknown/internal errors, don't expose details to the client
		statusCode = http.StatusInternalServerError
		errorResponse = ErrorResponse{
			Error:   "internal_error",
			Message: "An internal error occurred",
		}
	}

	// Log the full error details (including wrapped errors)
	if logger != nil {
		logger.Error("request failed",
			slog.Int("status_code", statusCode),
			slog.String("error_code", errorResponse.Error),
			slog.Any("error", err),
		)
	}

	c.JSON(statusCode, errorResponse)
}

// HandleBadRequestGin writes a 400 Bad Request response for malformed JSON or parameters using Gin.
func HandleBadRequestGin(c *gin.Context, err error, logger *slog.Logger) {
	if logger != nil {
		logger.Warn("bad request", slog.Any("error", err))
	}

	errorResponse := ErrorResponse{
		Error:   "bad_request",
		Message: err.Error(),
	}

	c.JSON(http.StatusBadRequest, errorResponse)
}

// HandleValidationErrorGin writes a 422 Unprocessable Entity response for validation errors using Gin.
func HandleValidationErrorGin(c *gin.Context, err error, logger *slog.Logger) {
	if logger != nil {
		logger.Warn("validation failed", slog.Any("error", err))
	}

	errorResponse := ErrorResponse{
		Error:   "validation_error",
		Message: err.Error(),
	}

	c.JSON(http.StatusUnprocessableEntity, errorResponse)
}
