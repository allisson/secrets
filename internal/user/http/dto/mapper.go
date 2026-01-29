// Package dto provides data transfer objects for the user HTTP layer.
package dto

import (
	"github.com/allisson/go-project-template/internal/user/domain"
	"github.com/allisson/go-project-template/internal/user/usecase"
)

// ToRegisterUserInput converts a RegisterUserRequest DTO to a RegisterUserInput use case input
func ToRegisterUserInput(req RegisterUserRequest) usecase.RegisterUserInput {
	return usecase.RegisterUserInput{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
	}
}

// ToUserResponse converts a domain User model to a UserResponse DTO
// This enforces the boundary between internal domain models and external API contracts
func ToUserResponse(user *domain.User) UserResponse {
	return UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
