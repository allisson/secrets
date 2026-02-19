package domain

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/allisson/secrets/internal/config"
)

// MasterKey represents a cryptographic master key used to encrypt KEKs.
// Must be 32 bytes (256 bits) and stored securely in KMS or environment variables.
// The Key field contains sensitive data and should be zeroed after use.
type MasterKey struct {
	ID  string // Unique identifier for the master key
	Key []byte // The raw 32-byte master key material
}

// MasterKeyChain manages a collection of master keys with one designated as active.
// Supports key rotation by maintaining multiple keys simultaneously.
type MasterKeyChain struct {
	activeID string   // ID of the master key to use for encrypting new KEKs
	keys     sync.Map // Thread-safe map of master key ID to MasterKey instances
}

// ActiveMasterKeyID returns the ID of the currently active master key.
func (m *MasterKeyChain) ActiveMasterKeyID() string {
	return m.activeID
}

// Get retrieves a master key from the keychain by its ID.
func (m *MasterKeyChain) Get(id string) (*MasterKey, bool) {
	if masterKey, ok := m.keys.Load(id); ok {
		return masterKey.(*MasterKey), ok
	}

	return nil, false
}

// Close securely zeros all master keys from memory, clears the chain, and resets the active ID.
func (m *MasterKeyChain) Close() {
	// Zero all master keys before clearing the chain
	m.keys.Range(func(key, value interface{}) bool {
		if masterKey, ok := value.(*MasterKey); ok {
			Zero(masterKey.Key)
		}
		return true
	})
	m.activeID = ""
	m.keys.Clear()
}

// LoadMasterKeyChainFromEnv loads master keys from MASTER_KEYS and ACTIVE_MASTER_KEY_ID environment variables.
// Keys must be in format "id:base64key" (comma-separated) and exactly 32 bytes when decoded.
// Returns ErrMasterKeysNotSet, ErrActiveMasterKeyIDNotSet, ErrInvalidKeySize, or ErrActiveMasterKeyNotFound on failure.
func LoadMasterKeyChainFromEnv() (*MasterKeyChain, error) {
	raw := os.Getenv("MASTER_KEYS")
	if raw == "" {
		return nil, ErrMasterKeysNotSet
	}

	active := os.Getenv("ACTIVE_MASTER_KEY_ID")
	if active == "" {
		return nil, ErrActiveMasterKeyIDNotSet
	}

	mkc := &MasterKeyChain{activeID: active}

	parts := strings.SplitSeq(raw, ",")
	for part := range parts {
		p := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(p) != 2 {
			mkc.Close()
			return nil, fmt.Errorf("%w: %q", ErrInvalidMasterKeysFormat, part)
		}
		id := p[0]
		key, err := base64.StdEncoding.DecodeString(p[1])
		if err != nil {
			mkc.Close()
			return nil, fmt.Errorf("%w for %s: %v", ErrInvalidMasterKeyBase64, id, err)
		}
		if len(key) != 32 {
			Zero(key)
			mkc.Close()
			return nil, fmt.Errorf(
				"%w: master key %s must be 32 bytes, got %d",
				ErrInvalidKeySize,
				id,
				len(key),
			)
		}
		// Make a copy of the key data before storing to prevent premature zeroing.
		// The original 'key' slice will be zeroed for security, but the keychain
		// needs its own copy to remain functional.
		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)
		mkc.keys.Store(id, &MasterKey{ID: id, Key: keyCopy})
		// Zero the original decoded key to prevent memory dumps
		Zero(key)
	}

	if _, ok := mkc.Get(active); !ok {
		mkc.Close()
		return nil, fmt.Errorf("%w: ACTIVE_MASTER_KEY_ID=%s", ErrActiveMasterKeyNotFound, active)
	}

	return mkc, nil
}

// KMSService defines the interface for KMS operations required by LoadMasterKeyChain.
// This interface is implemented by crypto/service.KMSService.
type KMSService interface {
	// OpenKeeper opens a secrets.Keeper for the configured KMS provider.
	OpenKeeper(ctx context.Context, keyURI string) (KMSKeeper, error)
}

// KMSKeeper defines the interface for KMS decrypt operations.
type KMSKeeper interface {
	// Decrypt decrypts ciphertext using the KMS key.
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)

	// Close releases resources held by the keeper.
	Close() error
}

// maskKeyURI masks sensitive components of a KMS key URI for secure logging.
// Examples: gcpkms://projects/***/.../cryptoKeys/*** or base64key://***
func maskKeyURI(uri string) string {
	if uri == "" {
		return ""
	}

	// Extract scheme
	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return "***"
	}

	scheme := parts[0]
	remainder := parts[1]

	// For base64key, mask everything after scheme
	if scheme == "base64key" {
		return scheme + "://***"
	}

	// For cloud providers, mask key identifiers but keep structure
	// gcpkms://projects/PROJECT/locations/LOCATION/keyRings/RING/cryptoKeys/KEY
	// awskms://KEY_ID?region=REGION
	// azurekeyvault://VAULT.vault.azure.net/keys/KEY
	// hashivault://KEY_NAME

	switch scheme {
	case "gcpkms":
		// Mask project, keyRing, and cryptoKey names
		pathParts := strings.Split(remainder, "/")
		for i := range pathParts {
			if i%2 == 1 { // Values (odd indices)
				pathParts[i] = "***"
			}
		}
		return scheme + "://" + strings.Join(pathParts, "/")
	case "awskms":
		// Mask key ID but keep region parameter
		queryParts := strings.SplitN(remainder, "?", 2)
		masked := scheme + "://***"
		if len(queryParts) == 2 {
			masked += "?" + queryParts[1]
		}
		return masked
	case "azurekeyvault", "hashivault":
		// Mask the entire path
		return scheme + "://***"
	default:
		return scheme + "://***"
	}
}

// loadMasterKeyChainFromKMS loads and decrypts master keys from MASTER_KEYS using KMS.
// The MASTER_KEYS environment variable contains KMS-encrypted keys in format "id:base64ciphertext".
// Returns ErrKMSOpenKeeperFailed, ErrKMSDecryptionFailed, ErrInvalidKeySize, or ErrActiveMasterKeyNotFound on failure.
func loadMasterKeyChainFromKMS(
	ctx context.Context,
	cfg *config.Config,
	kmsService KMSService,
	logger *slog.Logger,
) (*MasterKeyChain, error) {
	raw := os.Getenv("MASTER_KEYS")
	if raw == "" {
		return nil, ErrMasterKeysNotSet
	}

	active := os.Getenv("ACTIVE_MASTER_KEY_ID")
	if active == "" {
		return nil, ErrActiveMasterKeyIDNotSet
	}

	// Open KMS keeper
	maskedURI := maskKeyURI(cfg.KMSKeyURI)
	logger.Info("opening KMS keeper",
		slog.String("kms_provider", cfg.KMSProvider),
		slog.String("kms_key_uri", maskedURI),
	)

	keeper, err := kmsService.OpenKeeper(ctx, cfg.KMSKeyURI)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKMSOpenKeeperFailed, err)
	}
	defer func() {
		if closeErr := keeper.Close(); closeErr != nil {
			logger.Error("failed to close KMS keeper", slog.Any("error", closeErr))
		}
	}()

	logger.Info("KMS keeper opened successfully", slog.String("kms_provider", cfg.KMSProvider))

	mkc := &MasterKeyChain{activeID: active}

	parts := strings.SplitSeq(raw, ",")
	for part := range parts {
		p := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(p) != 2 {
			mkc.Close()
			return nil, fmt.Errorf("%w: %q", ErrInvalidMasterKeysFormat, part)
		}
		id := p[0]

		// Decode base64 ciphertext
		ciphertext, err := base64.StdEncoding.DecodeString(p[1])
		if err != nil {
			mkc.Close()
			return nil, fmt.Errorf("%w for %s: %v", ErrInvalidMasterKeyBase64, id, err)
		}

		logger.Info("decrypting master key with KMS",
			slog.String("master_key_id", id),
			slog.String("kms_provider", cfg.KMSProvider),
		)

		// Decrypt with KMS
		key, err := keeper.Decrypt(ctx, ciphertext)
		Zero(ciphertext) // Zero ciphertext after use
		if err != nil {
			mkc.Close()
			return nil, fmt.Errorf("%w for master key %s: %v", ErrKMSDecryptionFailed, id, err)
		}

		// Validate key size
		if len(key) != 32 {
			Zero(key)
			mkc.Close()
			return nil, fmt.Errorf(
				"%w: master key %s must be 32 bytes, got %d",
				ErrInvalidKeySize,
				id,
				len(key),
			)
		}

		logger.Info("master key decrypted successfully",
			slog.String("master_key_id", id),
			slog.Int("key_size_bytes", len(key)),
		)

		// Make a copy of the key data before storing to prevent issues if the underlying
		// slice is reused. The original 'key' slice ownership is transferred to the keychain.
		mkc.keys.Store(id, &MasterKey{ID: id, Key: key})
	}

	if _, ok := mkc.Get(active); !ok {
		mkc.Close()
		return nil, fmt.Errorf("%w: ACTIVE_MASTER_KEY_ID=%s", ErrActiveMasterKeyNotFound, active)
	}

	logger.Info("master key chain loaded successfully from KMS",
		slog.String("active_master_key_id", active),
		slog.String("kms_provider", cfg.KMSProvider),
	)

	return mkc, nil
}

// LoadMasterKeyChain loads master keys from environment variables with auto-detection for KMS or legacy mode.
// If KMS_PROVIDER is set, decrypts keys using KMS. Otherwise, uses plaintext base64-encoded keys.
// Validates that both KMS_PROVIDER and KMS_KEY_URI are set together or both empty.
// Returns ErrKMSProviderNotSet, ErrKMSKeyURINotSet, or errors from loadMasterKeyChainFromKMS/LoadMasterKeyChainFromEnv.
func LoadMasterKeyChain(
	ctx context.Context,
	cfg *config.Config,
	kmsService KMSService,
	logger *slog.Logger,
) (*MasterKeyChain, error) {
	// Validate KMS configuration consistency
	if cfg.KMSProvider != "" && cfg.KMSKeyURI == "" {
		return nil, ErrKMSProviderNotSet
	}
	if cfg.KMSKeyURI != "" && cfg.KMSProvider == "" {
		return nil, ErrKMSKeyURINotSet
	}

	// Auto-detect mode based on KMS_PROVIDER
	if cfg.KMSProvider != "" {
		logger.Info("loading master key chain in KMS mode",
			slog.String("kms_provider", cfg.KMSProvider),
		)
		return loadMasterKeyChainFromKMS(ctx, cfg, kmsService, logger)
	}

	logger.Info("loading master key chain in legacy mode (plaintext)")
	return LoadMasterKeyChainFromEnv()
}
