// Package http provides HTTP handlers for user-related operations.
package http

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/allisson/go-project-template/internal/httputil"
	"github.com/allisson/go-project-template/internal/user/http/dto"
	"github.com/allisson/go-project-template/internal/user/usecase"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userUseCase usecase.UseCase
	logger      *slog.Logger
}

// NewUserHandler creates a new UserHandler
func NewUserHandler(userUseCase usecase.UseCase, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
		logger:      logger,
	}
}

// RegisterUser handles user registration
func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.RegisterUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.HandleValidationError(w, err, h.logger)
		return
	}

	// Validate request structure
	if err := req.Validate(); err != nil {
		httputil.HandleError(w, err, h.logger)
		return
	}

	// Convert DTO to use case input
	input := dto.ToRegisterUserInput(req)

	user, err := h.userUseCase.RegisterUser(r.Context(), input)
	if err != nil {
		httputil.HandleError(w, err, h.logger)
		return
	}

	// Convert domain model to response DTO
	response := dto.ToUserResponse(user)
	httputil.MakeJSONResponse(w, http.StatusCreated, response)
}
