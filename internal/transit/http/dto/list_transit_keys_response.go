package dto

import (
	transitDomain "github.com/allisson/secrets/internal/transit/domain"
)

// ListTransitKeysResponse represents a paginated list of transit keys in API responses.
type ListTransitKeysResponse struct {
	Data []TransitKeyResponse `json:"data"`
}

// MapTransitKeysToListResponse converts a slice of domain transit keys to a list response.
func MapTransitKeysToListResponse(transitKeys []*transitDomain.TransitKey) ListTransitKeysResponse {
	data := make([]TransitKeyResponse, 0, len(transitKeys))
	for _, tk := range transitKeys {
		data = append(data, MapTransitKeyToResponse(tk))
	}

	return ListTransitKeysResponse{
		Data: data,
	}
}
