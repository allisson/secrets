package keyring

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

// masterKey is a plaintext root key used to wrap KEKs. Key material must be
// zeroed via Zero when no longer needed. It is unexported: raw root-key bytes
// never leave the keyring package.
type masterKey struct {
	ID  string
	key []byte
}

// MasterKeyChain is an in-memory store of one or more MasterKeys, one of
// which is designated active. The active key is used for new KEK operations;
// all keys are available for decryption of existing KEKs.
// Concurrency-safe.
type MasterKeyChain struct {
	activeID string
	keys     sync.Map
}

// NewMasterKeyChain returns an empty MasterKeyChain whose active key ID is activeID.
// Keys must be added via direct store calls before passing the chain to Bootstrap.
func NewMasterKeyChain(activeID string) *MasterKeyChain {
	return &MasterKeyChain{activeID: activeID}
}

// ActiveMasterKeyID returns the ID of the master key used for new operations.
func (m *MasterKeyChain) ActiveMasterKeyID() string {
	return m.activeID
}

// get returns the masterKey with the given ID, or false if it is not in the chain.
func (m *MasterKeyChain) get(id string) (*masterKey, bool) {
	if mk, ok := m.keys.Load(id); ok {
		return mk.(*masterKey), ok
	}
	return nil, false
}

// Has reports whether a master key with the given ID is present in the chain.
// It exposes presence without leaking key material, for callers that need to
// verify a chain was populated (e.g. end-to-end tests).
func (m *MasterKeyChain) Has(id string) bool {
	_, ok := m.keys.Load(id)
	return ok
}

// Close zeroes all key material in the chain and removes every entry.
// After Close the chain must not be used.
func (m *MasterKeyChain) Close() {
	m.keys.Range(func(_, value any) bool {
		if mk, ok := value.(*masterKey); ok {
			Zero(mk.key)
		}
		return true
	})
	m.activeID = ""
	m.keys.Clear()
}

func maskKeyURI(uri string) string {
	if uri == "" {
		return ""
	}

	parts := strings.SplitN(uri, "://", 2)
	if len(parts) != 2 {
		return "***"
	}

	scheme := parts[0]
	remainder := parts[1]

	if scheme == "base64key" {
		return scheme + "://***"
	}

	switch scheme {
	case "gcpkms":
		pathParts := strings.Split(remainder, "/")
		for i := range pathParts {
			if i%2 == 1 {
				pathParts[i] = "***"
			}
		}
		return scheme + "://" + strings.Join(pathParts, "/")
	case "awskms":
		queryParts := strings.SplitN(remainder, "?", 2)
		masked := scheme + "://***"
		if len(queryParts) == 2 {
			masked += "?" + queryParts[1]
		}
		return masked
	case "azurekeyvault", "hashivault":
		return scheme + "://***"
	default:
		return scheme + "://***"
	}
}

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

		ciphertext, err := base64.StdEncoding.DecodeString(p[1])
		if err != nil {
			mkc.Close()
			return nil, fmt.Errorf("%w for %s: %v", ErrInvalidMasterKeyBase64, id, err)
		}

		logger.Info("decrypting master key with KMS",
			slog.String("master_key_id", id),
			slog.String("kms_provider", cfg.KMSProvider),
		)

		key, err := keeper.Decrypt(ctx, ciphertext)
		Zero(ciphertext)
		if err != nil {
			mkc.Close()
			return nil, fmt.Errorf("%w for master key %s: %v", ErrKMSDecryptionFailed, id, err)
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

		logger.Info("master key decrypted successfully",
			slog.String("master_key_id", id),
			slog.Int("key_size_bytes", len(key)),
		)

		keyCopy := make([]byte, len(key))
		copy(keyCopy, key)
		Zero(key)

		mkc.keys.Store(id, &masterKey{ID: id, key: keyCopy})
	}

	if _, ok := mkc.get(active); !ok {
		mkc.Close()
		return nil, fmt.Errorf("%w: ACTIVE_MASTER_KEY_ID=%s", ErrActiveMasterKeyNotFound, active)
	}

	logger.Info("master key chain loaded successfully from KMS",
		slog.String("active_master_key_id", active),
		slog.String("kms_provider", cfg.KMSProvider),
	)

	return mkc, nil
}

// LoadMasterKeyChain reads MASTER_KEYS and ACTIVE_MASTER_KEY_ID from the
// environment, decrypts each key via the KMS, and returns a populated chain.
// KMS_PROVIDER and KMS_KEY_URI must be set in cfg.
func LoadMasterKeyChain(
	ctx context.Context,
	cfg *config.Config,
	kmsService KMSService,
	logger *slog.Logger,
) (*MasterKeyChain, error) {
	if cfg.KMSProvider == "" {
		return nil, ErrKMSProviderNotSet
	}
	if cfg.KMSKeyURI == "" {
		return nil, ErrKMSKeyURINotSet
	}

	logger.Info("loading master key chain with KMS",
		slog.String("kms_provider", cfg.KMSProvider),
	)
	return loadMasterKeyChainFromKMS(ctx, cfg, kmsService, logger)
}
