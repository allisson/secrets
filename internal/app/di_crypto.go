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
func (c *Container) MasterKeyChain() (*cryptoDomain.MasterKeyChain, error) {
	var err error
	c.masterKeyChainInit.Do(func() {
		c.masterKeyChain, err = c.initMasterKeyChain()
		if err != nil {
			c.initErrors["masterKeyChain"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["masterKeyChain"]; exists {
		return nil, storedErr
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
func (c *Container) KMSService() cryptoService.KMSService {
	c.kmsServiceInit.Do(func() {
		c.kmsService = c.initKMSService()
	})
	return c.kmsService
}

// KekRepository returns the KEK repository.
func (c *Container) KekRepository() (cryptoUseCase.KekRepository, error) {
	var err error
	c.kekRepositoryInit.Do(func() {
		c.kekRepository, err = c.initKekRepository()
		if err != nil {
			c.initErrors["kekRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["kekRepository"]; exists {
		return nil, storedErr
	}
	return c.kekRepository, nil
}

// KekUseCase returns the KEK use case.
func (c *Container) KekUseCase() (cryptoUseCase.KekUseCase, error) {
	var err error
	c.kekUseCaseInit.Do(func() {
		c.kekUseCase, err = c.initKekUseCase()
		if err != nil {
			c.initErrors["kekUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["kekUseCase"]; exists {
		return nil, storedErr
	}
	return c.kekUseCase, nil
}

// CryptoDekRepository returns the DEK repository for the crypto use case based on database driver.
func (c *Container) CryptoDekRepository() (cryptoUseCase.DekRepository, error) {
	var err error
	c.cryptoDekRepositoryInit.Do(func() {
		c.cryptoDekRepository, err = c.initCryptoDekRepository()
		if err != nil {
			c.initErrors["cryptoDekRepository"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["cryptoDekRepository"]; exists {
		return nil, storedErr
	}
	return c.cryptoDekRepository, nil
}

// CryptoDekUseCase returns the DEK use case for the crypto module.
func (c *Container) CryptoDekUseCase() (cryptoUseCase.DekUseCase, error) {
	var err error
	c.cryptoDekUseCaseInit.Do(func() {
		c.cryptoDekUseCase, err = c.initCryptoDekUseCase()
		if err != nil {
			c.initErrors["cryptoDekUseCase"] = err
		}
	})
	if err != nil {
		return nil, err
	}
	if storedErr, exists := c.initErrors["cryptoDekUseCase"]; exists {
		return nil, storedErr
	}
	return c.cryptoDekUseCase, nil
}

// initMasterKeyChain loads the master key chain from environment variables.
func (c *Container) initMasterKeyChain() (*cryptoDomain.MasterKeyChain, error) {
	// Get KMS service and logger
	kmsService := c.KMSService()
	logger := c.Logger()

	// Load master key chain with KMS support and fail-fast validation
	masterKeyChain, err := cryptoDomain.LoadMasterKeyChain(
		context.Background(),
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
func (c *Container) initKMSService() cryptoService.KMSService {
	return cryptoService.NewKMSService()
}

// initKekRepository creates the KEK repository based on the database driver.
func (c *Container) initKekRepository() (cryptoUseCase.KekRepository, error) {
	db, err := c.DB()
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
func (c *Container) initKekUseCase() (cryptoUseCase.KekUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager for kek use case: %w", err)
	}

	kekRepository, err := c.KekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get kek repository for kek use case: %w", err)
	}

	keyManager := c.KeyManager()

	return cryptoUseCase.NewKekUseCase(txManager, kekRepository, keyManager), nil
}

// initCryptoDekRepository creates the DEK repository for crypto use case based on the database driver.
func (c *Container) initCryptoDekRepository() (cryptoUseCase.DekRepository, error) {
	db, err := c.DB()
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
func (c *Container) initCryptoDekUseCase() (cryptoUseCase.DekUseCase, error) {
	txManager, err := c.TxManager()
	if err != nil {
		return nil, fmt.Errorf("failed to get tx manager: %w", err)
	}

	dekRepo, err := c.CryptoDekRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get crypto dek repository: %w", err)
	}

	keyManager := c.KeyManager()

	return cryptoUseCase.NewDekUseCase(txManager, dekRepo, keyManager), nil
}

// loadKekChain loads all KEKs from the database and creates a KEK chain.
func (c *Container) loadKekChain() (*cryptoDomain.KekChain, error) {
	kekUseCase, err := c.KekUseCase()
	if err != nil {
		return nil, fmt.Errorf("failed to get kek use case: %w", err)
	}

	masterKeyChain, err := c.MasterKeyChain()
	if err != nil {
		return nil, fmt.Errorf("failed to get master key chain: %w", err)
	}

	// Unwrap all KEKs using the master key chain
	kekChain, err := kekUseCase.Unwrap(context.Background(), masterKeyChain)
	if err != nil {
		return nil, fmt.Errorf("failed to unwrap keks: %w", err)
	}

	return kekChain, nil
}
