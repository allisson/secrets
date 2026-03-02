package app

import (
	"context"
	"fmt"

	cryptoDomain "github.com/allisson/secrets/internal/crypto/domain"
	cryptoMySQL "github.com/allisson/secrets/internal/crypto/repository/mysql"
	cryptoPostgreSQL "github.com/allisson/secrets/internal/crypto/repository/postgresql"
	cryptoService "github.com/allisson/secrets/internal/crypto/service"
	cryptoUseCase "github.com/allisson/secrets/internal/crypto/usecase"
)

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

// CryptoDekRepository returns the DEK repository for the crypto use case based on database driver.
func (c *Container) CryptoDekRepository(ctx context.Context) (cryptoUseCase.DekRepository, error) {
	var err error
	c.cryptoDekRepositoryInit.Do(func() {
		c.cryptoDekRepository, err = c.initCryptoDekRepository(ctx)
		if err != nil {
			c.initErrors.Store("cryptoDekRepository", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("cryptoDekRepository"); ok {
		return nil, val.(error)
	}
	return c.cryptoDekRepository, nil
}

// CryptoDekUseCase returns the DEK use case for the crypto module.
func (c *Container) CryptoDekUseCase(ctx context.Context) (cryptoUseCase.DekUseCase, error) {
	var err error
	c.cryptoDekUseCaseInit.Do(func() {
		c.cryptoDekUseCase, err = c.initCryptoDekUseCase(ctx)
		if err != nil {
			c.initErrors.Store("cryptoDekUseCase", err)
		}
	})
	if err != nil {
		return nil, err
	}
	if val, ok := c.initErrors.Load("cryptoDekUseCase"); ok {
		return nil, val.(error)
	}
	return c.cryptoDekUseCase, nil
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

// initKekRepository creates the KEK repository based on the database driver.
func (c *Container) initKekRepository(ctx context.Context) (cryptoUseCase.KekRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database for kek repository: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return cryptoPostgreSQL.NewPostgreSQLKekRepository(db), nil
	case "mysql":
		return cryptoMySQL.NewMySQLKekRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
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

// initCryptoDekRepository creates the DEK repository for crypto use case based on the database driver.
func (c *Container) initCryptoDekRepository(ctx context.Context) (cryptoUseCase.DekRepository, error) {
	db, err := c.DB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	switch c.config.DBDriver {
	case "postgres":
		return cryptoPostgreSQL.NewPostgreSQLDekRepository(db), nil
	case "mysql":
		return cryptoMySQL.NewMySQLDekRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", c.config.DBDriver)
	}
}

// initCryptoDekUseCase creates the DEK use case for the crypto module.
func (c *Container) initCryptoDekUseCase(ctx context.Context) (cryptoUseCase.DekUseCase, error) {
	txManager, err := c.TxManager(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager: %w", err)
	}

	dekRepo, err := c.CryptoDekRepository(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get crypto dek repository: %w", err)
	}

	keyManager := c.KeyManager()

	return cryptoUseCase.NewDekUseCase(txManager, dekRepo, keyManager), nil
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
