// Package dto provides data transfer objects for the user HTTP layer.
package dto

import (
	"time"

	"github.com/google/uuid"
)

// UserResponse represents the API response for a user
// It excludes sensitive information like passwords and provides
// a clean external representation of the user domain model
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
