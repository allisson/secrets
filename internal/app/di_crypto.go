package app

import (
	"context"
	"fmt"

	"github.com/allisson/secrets/internal/keyring"
)

// Keyring returns the envelope-encryption keyring shared by all features.
func (c *Container) Keyring(ctx context.Context) (keyring.Keyring, error) {
	return c.keyring.get(func() (keyring.Keyring, error) {
		return c.initKeyring(ctx)
	})
}

func (c *Container) initKeyring(ctx context.Context) (keyring.Keyring, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for keyring: %w", err)
	}
	masterKeyChain, err := c.MasterKeyChain(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get master key chain: %w", err)
	}
	return keyring.Bootstrap(ctx, masterKeyChain, db, keyring.AESGCM)
}

// MasterKeyChain returns the master key chain loaded from environment variables.
func (c *Container) MasterKeyChain(ctx context.Context) (*keyring.MasterKeyChain, error) {
	return c.masterKeyChain.get(func() (*keyring.MasterKeyChain, error) {
		return c.initMasterKeyChain(ctx)
	})
}

func (c *Container) initMasterKeyChain(ctx context.Context) (*keyring.MasterKeyChain, error) {
	return keyring.LoadMasterKeyChain(ctx, c.config, c.KMSService(), c.Logger())
}

// KMSService returns the KMS service.
func (c *Container) KMSService() keyring.KMSService {
	c.kmsServiceInit.Do(func() {
		c.kmsService = keyring.NewKMSService()
	})
	return c.kmsService
}

// KeySigner returns the keyring's signing capability for audit log operations.
func (c *Container) KeySigner(ctx context.Context) (keyring.KeySigner, error) {
	return c.Keyring(ctx)
}

// KekUseCase returns the KEK use case.
func (c *Container) KekUseCase(ctx context.Context) (keyring.KekUseCase, error) {
	return c.kekUseCase.get(func() (keyring.KekUseCase, error) {
		return c.initKekUseCase(ctx)
	})
}

func (c *Container) initKekUseCase(ctx context.Context) (keyring.KekUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for kek use case: %w", err)
	}
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for kek use case: %w", err)
	}
	return keyring.NewKekUseCase(txManager, db), nil
}
