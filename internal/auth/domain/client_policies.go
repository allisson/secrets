package domain

import "github.com/google/uuid"

type ClientPolicies struct {
	ClientID uuid.UUID
	PolicyID uuid.UUID
}
