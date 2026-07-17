package keyring

import (
	"context"
	"fmt"

	"gocloud.dev/secrets"

	_ "gocloud.dev/secrets/awskms"
	_ "gocloud.dev/secrets/azurekeyvault"
	_ "gocloud.dev/secrets/gcpkms"
	_ "gocloud.dev/secrets/hashivault"
	_ "gocloud.dev/secrets/localsecrets"
)

// KMSService opens cloud KMS keepers by URI. Implementations register
// themselves via gocloud.dev URL openers (see blank imports below).
type KMSService interface {
	// OpenKeeper opens the KMS keeper at keyURI and returns it.
	// The caller must call Close on the returned keeper when done.
	OpenKeeper(ctx context.Context, keyURI string) (KMSKeeper, error)
}

// KMSKeeper wraps a gocloud.dev/secrets.Keeper to expose the operations the
// master-key lifecycle needs. Decrypt is the boot path (unwrap persisted master
// keys); Encrypt is the operator path (the create/rotate-master-key CLI commands
// wrap a fresh key). Close releases keeper resources. The running service only
// ever calls Decrypt; Encrypt exists so the CLI needs no runtime type assertion.
type KMSKeeper interface {
	Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
	Close() error
}

type kmsService struct{}

// NewKMSService returns a KMSService backed by the gocloud.dev URL opener registry.
func NewKMSService() KMSService {
	return &kmsService{}
}

func (k *kmsService) OpenKeeper(ctx context.Context, keyURI string) (KMSKeeper, error) {
	keeper, err := secrets.OpenKeeper(ctx, keyURI)
	if err != nil {
		return nil, fmt.Errorf("failed to open KMS keeper: %w", err)
	}
	return keeper, nil
}
