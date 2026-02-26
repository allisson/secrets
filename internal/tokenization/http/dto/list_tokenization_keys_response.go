package dto

import (
	tokenizationDomain "github.com/allisson/secrets/internal/tokenization/domain"
)

// ListTokenizationKeysResponse represents the response for listing tokenization keys.
type ListTokenizationKeysResponse struct {
	Data []TokenizationKeyResponse `json:"data"`
}

// MapTokenizationKeysToListResponse maps a slice of TokenizationKey domain entities to a ListTokenizationKeysResponse DTO.
// Returns an empty list instead of null when there are no items to match API conventions.
func MapTokenizationKeysToListResponse(
	keys []*tokenizationDomain.TokenizationKey,
) ListTokenizationKeysResponse {
	items := make([]TokenizationKeyResponse, 0, len(keys))
	for _, key := range keys {
		items = append(items, MapTokenizationKeyToResponse(key))
	}

	return ListTokenizationKeysResponse{
		Data: items,
	}
}
