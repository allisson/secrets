package app

import (
	"context"
	"fmt"

	"github.com/allisson/secrets/internal/keyring"
)

// Keyring returns the envelope-encryption keyring shared by all features.
func (c *Container) Keyring(ctx context.Context) (keyring.Keyring, error) {
	var err error
	c.keyringInit.Do(func() {
		c.keyring, err = c.initKeyring(ctx)
		if err != nil {
			c.initErrors.Store("keyring", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("keyring"); ok {
		return nil, val.(error)
	}
	return c.keyring, nil
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
	var err error
	c.masterKeyChainInit.Do(func() {
		c.masterKeyChain, err = c.initMasterKeyChain(ctx)
		if err != nil {
			c.initErrors.Store("masterKeyChain", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("masterKeyChain"); ok {
		return nil, val.(error)
	}
	return c.masterKeyChain, nil
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
	var err error
	c.kekUseCaseInit.Do(func() {
		c.kekUseCase, err = c.initKekUseCase(ctx)
		if err != nil {
			c.initErrors.Store("kekUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("kekUseCase"); ok {
		return nil, val.(error)
	}
	return c.kekUseCase, nil
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
