package app

import (
	"context"
	"fmt"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoRepository "github.com/allisson/secrets/internal/crypto/repository"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
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

	kekChain, err := c.loadKekChain(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load kek chain for keyring: %w", err)
	}

	// Keyring needs Create/Get/Update on DEKs. The concrete repository
	// provides all three; cryptoUseCase.DekRepository (Update + batch lookup)
	// is the narrower interface used by KEK rotation only.
	return keyring.New(
		kekChain,
		cryptoRepository.NewDekRepository(db),
		c.AEADManager(),
		c.KeyManager(),
		cryptoDomain.AESGCM,
	), nil
}

// MasterKeyChain returns the master key chain loaded from environment variables.
func (c *Container) MasterKeyChain(ctx context.Context) (*cryptoDomain.MasterKeyChain, error) {
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

// AEADManager returns the AEAD manager service.
func (c *Container) AEADManager() cryptoService.AEADManager {
	c.aeadManagerInit.Do(func() {
		c.aeadManager = c.initAEADManager()
	})
	return c.aeadManager
}

// KeySigner returns the keyring as a KeySigner for audit log signing operations.
// The production keyring implements KeySigner via its HKDF+HMAC-SHA256 methods.
func (c *Container) KeySigner(ctx context.Context) (keyring.KeySigner, error) {
	kr, err := c.Keyring(ctx)
	if err != nil {
		return nil, err
	}
	signer, ok := kr.(keyring.KeySigner)
	if !ok {
		return nil, fmt.Errorf("keyring does not implement KeySigner")
	}
	return signer, nil
}

// KeyManager returns the key manager service.
func (c *Container) KeyManager() cryptoService.KeyManager {
	c.keyManagerInit.Do(func() {
		c.keyManager = c.initKeyManager()
	})
	return c.keyManager
}

// KMSService returns the KMS service.
func (c *Container) KMSService() cryptoDomain.KMSService {
	c.kmsServiceInit.Do(func() {
		c.kmsService = c.initKMSService()
	})
	return c.kmsService
}

// KekRepository returns the KEK repository.
func (c *Container) KekRepository(ctx context.Context) (cryptoUseCase.KekRepository, error) {
	var err error
	c.kekRepositoryInit.Do(func() {
		c.kekRepository, err = c.initKekRepository(ctx)
		if err != nil {
			c.initErrors.Store("kekRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("kekRepository"); ok {
		return nil, val.(error)
	}
	return c.kekRepository, nil
}

// KekUseCase returns the KEK use case.
func (c *Container) KekUseCase(ctx context.Context) (cryptoUseCase.KekUseCase, error) {
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

// initMasterKeyChain loads the master key chain from environment variables.
func (c *Container) initMasterKeyChain(ctx context.Context) (*cryptoDomain.MasterKeyChain, error) {
	// Get KMS service and logger
	kmsService := c.KMSService()
	logger := c.Logger()

	// Load master key chain with KMS support and fail-fast validation
	masterKeyChain, err := cryptoDomain.LoadMasterKeyChain(
		ctx,
		c.config,
		kmsService,
		logger,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load master key chain: %w", err)
	}
	return masterKeyChain, nil
}

// initAEADManager creates the AEAD manager service.
func (c *Container) initAEADManager() cryptoService.AEADManager {
	return cryptoService.NewAEADManager()
}

// initKeyManager creates the key manager service using the AEAD manager.
func (c *Container) initKeyManager() cryptoService.KeyManager {
	aeadManager := c.AEADManager()
	return cryptoService.NewKeyManager(aeadManager)
}

// initKMSService creates the KMS service for encrypting/decrypting master keys.
func (c *Container) initKMSService() cryptoDomain.KMSService {
	return cryptoService.NewKMSService()
}

// initKekRepository creates the KEK repository.
func (c *Container) initKekRepository(ctx context.Context) (cryptoUseCase.KekRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for kek repository: %w", err)
	}

	return cryptoRepository.NewKekRepository(db), nil
}

// initKekUseCase creates the KEK use case with all its dependencies.
func (c *Container) initKekUseCase(ctx context.Context) (cryptoUseCase.KekUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for kek use case: %w", err)
	}

	kekRepository, err := c.KekRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get kek repository for kek use case: %w", err)
	}

	keyManager := c.KeyManager()

	return cryptoUseCase.NewKekUseCase(txManager, kekRepository, keyManager), nil
}

// loadKekChain loads all KEKs from the database and creates a KEK chain.
func (c *Container) loadKekChain(ctx context.Context) (*cryptoDomain.KekChain, error) {
	kekUseCase, err := c.KekUseCase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get kek use case: %w", err)
	}

	masterKeyChain, err := c.MasterKeyChain(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get master key chain: %w", err)
	}

	// Unwrap all KEKs using the master key chain
	kekChain, err := kekUseCase.Unwrap(ctx, masterKeyChain)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap keks: %w", err)
	}

	return kekChain, nil
}
