package service

import (
	"context"
	"fmt"

	"gocloud.dev/secrets"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"

	// Register all KMS provider drivers
	_ "gocloud.dev/secrets/awskms"
	_ "gocloud.dev/secrets/azurekeyvault"
	_ "gocloud.dev/secrets/gcpkms"
	_ "gocloud.dev/secrets/hashivault"
	_ "gocloud.dev/secrets/localsecrets"
)

// NewKMSService creates a new KMS service instance.
func NewKMSService() cryptoDomain.KMSService {
	return &kmsService{}
}

// kmsService implements domain.KMSService using gocloud.dev/secrets.
type kmsService struct{}

// OpenKeeper opens a secrets.Keeper for the configured KMS provider using the keyURI.
// Supports: gcpkms://, awskms://, azurekeyvault://, hashivault://, base64key://
// Returns a KMSKeeper which *secrets.Keeper implements.
func (k *kmsService) OpenKeeper(ctx context.Context, keyURI string) (cryptoDomain.KMSKeeper, error) {
	keeper, err := secrets.OpenKeeper(ctx, keyURI)
	if err != nil {
		return nil, fmt.Errorf("failed to open KMS keeper: %w", err)
	}
	return keeper, nil
}
