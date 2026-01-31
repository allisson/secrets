// Package httputil provides HTTP utility functions for request and response handling.
package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"

	apperrors "github.com/allisson/secrets/internal/errors"
)

// MakeJSONResponse writes a JSON response with the given status code and data
func MakeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}

// ErrorResponse represents a structured error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

// HandleError maps domain errors to HTTP status codes and writes an appropriate response.
// It logs the error with structured logging and returns a user-friendly error message.
func HandleError(w http.ResponseWriter, err error, logger *slog.Logger) {
	if err == nil {
		return
	}

	var statusCode int
	var errorResponse ErrorResponse

	// Map domain errors to HTTP status codes
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

	MakeJSONResponse(w, statusCode, errorResponse)
}

// HandleValidationError writes a 400 Bad Request response for validation errors
func HandleValidationError(w http.ResponseWriter, err error, logger *slog.Logger) {
	if logger != nil {
		logger.Warn("validation failed", slog.Any("error", err))
	}

	errorResponse := ErrorResponse{
		Error:   "validation_error",
		Message: err.Error(),
	}

	MakeJSONResponse(w, http.StatusBadRequest, errorResponse)
}
