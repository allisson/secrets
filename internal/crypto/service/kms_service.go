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

// KMSService implements domain.KMSService for KMS operations using gocloud.dev/secrets.
type KMSService interface {
	// OpenKeeper opens a secrets.Keeper for the configured KMS provider.
	// Returns an error if the KMS provider URI is invalid or connection fails.
	OpenKeeper(ctx context.Context, keyURI string) (cryptoDomain.KMSKeeper, error)
}

// kmsService implements KMSService using gocloud.dev/secrets.
type kmsService struct{}

// NewKMSService creates a new KMS service instance.
func NewKMSService() KMSService {
	return &kmsService{}
}

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
