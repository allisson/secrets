package domain

import (
	"context"
	"encoding/base64"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/config"
)

func TestMasterKeyChain_ActiveMasterKeyID(t *testing.T) {
	mkc := &MasterKeyChain{activeID: "key1"}
	assert.Equal(t, "key1", mkc.ActiveMasterKeyID())
}

func TestMasterKeyChain_Get(t *testing.T) {
	mkc := &MasterKeyChain{}
	testKey := &MasterKey{
		ID:  "test-key",
		Key: []byte("test-key-data-123456789012345"),
	}
	mkc.keys.Store("test-key", testKey)

	tests := []struct {
		name      string
		keyID     string
		wantFound bool
		wantKey   *MasterKey
	}{
		{
			name:      "existing key",
			keyID:     "test-key",
			wantFound: true,
			wantKey:   testKey,
		},
		{
			name:      "non-existing key",
			keyID:     "non-existent",
			wantFound: false,
			wantKey:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, found := mkc.Get(tt.keyID)
			assert.Equal(t, tt.wantFound, found)
			if tt.wantFound {
				assert.Equal(t, tt.wantKey.ID, key.ID)
				assert.Equal(t, tt.wantKey.Key, key.Key)
			} else {
				assert.Nil(t, key)
			}
		})
	}
}

func TestMasterKeyChain_Close(t *testing.T) {
	mkc := &MasterKeyChain{activeID: "key1"}
	mkc.keys.Store("key1", &MasterKey{ID: "key1", Key: make([]byte, 32)})
	mkc.keys.Store("key2", &MasterKey{ID: "key2", Key: make([]byte, 32)})

	mkc.Close()

	assert.Equal(t, "", mkc.activeID)

	_, found1 := mkc.Get("key1")
	_, found2 := mkc.Get("key2")
	assert.False(t, found1)
	assert.False(t, found2)
}

func TestMasterKeyChain_CloseZerosKeys(t *testing.T) {
	// Verify that Close() zeros all keys before clearing the chain
	key1Data := make([]byte, 32)
	key2Data := make([]byte, 32)

	// Fill with non-zero data
	for i := range key1Data {
		key1Data[i] = byte(i)
		key2Data[i] = byte(i + 100)
	}

	mkc := &MasterKeyChain{activeID: "key1"}
	mk1 := &MasterKey{ID: "key1", Key: key1Data}
	mk2 := &MasterKey{ID: "key2", Key: key2Data}

	mkc.keys.Store("key1", mk1)
	mkc.keys.Store("key2", mk2)

	// Verify keys contain data before Close
	assert.NotEqual(t, make([]byte, 32), mk1.Key, "key1 should have data before Close()")
	assert.NotEqual(t, make([]byte, 32), mk2.Key, "key2 should have data before Close()")

	// Close should zero the keys
	mkc.Close()

	// Verify keys are zeroed
	expectedZero := make([]byte, 32)
	assert.Equal(t, expectedZero, mk1.Key, "key1 should be zeroed after Close()")
	assert.Equal(t, expectedZero, mk2.Key, "key2 should be zeroed after Close()")

	// Verify chain is cleared
	assert.Equal(t, "", mkc.activeID)
	_, found1 := mkc.Get("key1")
	_, found2 := mkc.Get("key2")
	assert.False(t, found1)
	assert.False(t, found2)
}

func TestLoadMasterKeyChainFromEnv(t *testing.T) {
	// Generate valid 32-byte keys encoded in base64
	key1 := base64.StdEncoding.EncodeToString(make([]byte, 32))
	key2 := base64.StdEncoding.EncodeToString([]byte("12345678901234567890123456789012"))

	tests := []struct {
		name              string
		masterKeys        string
		activeMasterKeyID string
		wantErr           error
		errMsg            string
		validateFunc      func(*testing.T, *MasterKeyChain)
	}{
		{
			name:              "valid single key",
			masterKeys:        "key1:" + key1,
			activeMasterKeyID: "key1",
			validateFunc: func(t *testing.T, mkc *MasterKeyChain) {
				assert.Equal(t, "key1", mkc.ActiveMasterKeyID())
				mk, found := mkc.Get("key1")
				assert.True(t, found)
				assert.Equal(t, "key1", mk.ID)
				assert.Len(t, mk.Key, 32)
			},
		},
		{
			name:              "valid multiple keys",
			masterKeys:        "key1:" + key1 + ",key2:" + key2,
			activeMasterKeyID: "key2",
			validateFunc: func(t *testing.T, mkc *MasterKeyChain) {
				assert.Equal(t, "key2", mkc.ActiveMasterKeyID())

				mk1, found1 := mkc.Get("key1")
				assert.True(t, found1)
				assert.Equal(t, "key1", mk1.ID)
				assert.Len(t, mk1.Key, 32)

				mk2, found2 := mkc.Get("key2")
				assert.True(t, found2)
				assert.Equal(t, "key2", mk2.ID)
				assert.Len(t, mk2.Key, 32)
			},
		},
		{
			name:              "valid keys with whitespace",
			masterKeys:        " key1:" + key1 + " , key2:" + key2 + " ",
			activeMasterKeyID: "key1",
			validateFunc: func(t *testing.T, mkc *MasterKeyChain) {
				assert.Equal(t, "key1", mkc.ActiveMasterKeyID())
				_, found1 := mkc.Get("key1")
				_, found2 := mkc.Get("key2")
				assert.True(t, found1)
				assert.True(t, found2)
			},
		},
		{
			name:              "MASTER_KEYS not set",
			masterKeys:        "",
			activeMasterKeyID: "key1",
			wantErr:           ErrMasterKeysNotSet,
			errMsg:            "MASTER_KEYS not set",
		},
		{
			name:              "ACTIVE_MASTER_KEY_ID not set",
			masterKeys:        "key1:" + key1,
			activeMasterKeyID: "",
			wantErr:           ErrActiveMasterKeyIDNotSet,
			errMsg:            "ACTIVE_MASTER_KEY_ID not set",
		},
		{
			name:              "invalid format - missing colon",
			masterKeys:        "key1" + key1,
			activeMasterKeyID: "key1",
			wantErr:           ErrInvalidMasterKeysFormat,
			errMsg:            "invalid MASTER_KEYS format",
		},
		{
			name:              "invalid format - too many colons",
			masterKeys:        "key1:part1:part2",
			activeMasterKeyID: "key1",
			wantErr:           ErrInvalidMasterKeyBase64,
			errMsg:            "invalid master key base64",
		},
		{
			name:              "invalid base64",
			masterKeys:        "key1:not-valid-base64!!!",
			activeMasterKeyID: "key1",
			wantErr:           ErrInvalidMasterKeyBase64,
			errMsg:            "invalid master key base64",
		},
		{
			name:              "key too short",
			masterKeys:        "key1:" + base64.StdEncoding.EncodeToString(make([]byte, 16)),
			activeMasterKeyID: "key1",
			wantErr:           ErrInvalidKeySize,
			errMsg:            "invalid key size",
		},
		{
			name:              "key too long",
			masterKeys:        "key1:" + base64.StdEncoding.EncodeToString(make([]byte, 64)),
			activeMasterKeyID: "key1",
			wantErr:           ErrInvalidKeySize,
			errMsg:            "invalid key size",
		},
		{
			name:              "active key not in keychain",
			masterKeys:        "key1:" + key1,
			activeMasterKeyID: "key2",
			wantErr:           ErrActiveMasterKeyNotFound,
			errMsg:            "active master key not found",
		},
		{
			name:              "empty key ID",
			masterKeys:        ":" + key1,
			activeMasterKeyID: "",
			wantErr:           ErrActiveMasterKeyIDNotSet,
			errMsg:            "ACTIVE_MASTER_KEY_ID not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			if tt.masterKeys == "" {
				require.NoError(t, os.Unsetenv("MASTER_KEYS"))
			} else {
				require.NoError(t, os.Setenv("MASTER_KEYS", tt.masterKeys))
			}

			if tt.activeMasterKeyID == "" {
				require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID"))
			} else {
				require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", tt.activeMasterKeyID))
			}

			// Cleanup
			defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
			defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

			// Test
			mkc, err := LoadMasterKeyChainFromEnv()

			if tt.wantErr != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, mkc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, mkc)
				if tt.validateFunc != nil {
					tt.validateFunc(t, mkc)
				}
				// Cleanup the keychain
				mkc.Close()
			}
		})
	}
}

func TestLoadMasterKeyChainFromEnv_KeysAreUsable(t *testing.T) {
	// Verify that loaded master keys contain valid key material and are usable
	key1Data := []byte("12345678901234567890123456789012")
	key1 := base64.StdEncoding.EncodeToString(key1Data)

	require.NoError(t, os.Setenv("MASTER_KEYS", "key1:"+key1))
	require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", "key1"))
	defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
	defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

	mkc, err := LoadMasterKeyChainFromEnv()
	assert.NoError(t, err)
	assert.NotNil(t, mkc)
	defer mkc.Close()

	// Get the key from the keychain
	mk, found := mkc.Get("key1")
	assert.True(t, found)
	assert.NotNil(t, mk)
	assert.Len(t, mk.Key, 32)

	// Keys should contain the actual key material, not zeros
	assert.Equal(t, key1Data, mk.Key, "Master key should contain actual key data")

	// Verify key is not all zeros
	allZeros := true
	for _, b := range mk.Key {
		if b != 0 {
			allZeros = false
			break
		}
	}
	assert.False(t, allZeros, "Master key should not be zeroed after loading")
}

func TestLoadMasterKeyChainFromEnv_CloseOnError(t *testing.T) {
	// Generate a valid key and an invalid key
	validKey := base64.StdEncoding.EncodeToString(make([]byte, 32))
	invalidKey := base64.StdEncoding.EncodeToString(make([]byte, 16)) // Too short

	tests := []struct {
		name              string
		masterKeys        string
		activeMasterKeyID string
		errMsg            string
	}{
		{
			name:              "invalid key after valid key",
			masterKeys:        "key1:" + validKey + ",key2:" + invalidKey,
			activeMasterKeyID: "key1",
			errMsg:            "must be 32 bytes",
		},
		{
			name:              "invalid base64 after valid key",
			masterKeys:        "key1:" + validKey + ",key2:invalid!!!",
			activeMasterKeyID: "key1",
			errMsg:            "invalid master key base64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, os.Setenv("MASTER_KEYS", tt.masterKeys))
			require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", tt.activeMasterKeyID))
			defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
			defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

			mkc, err := LoadMasterKeyChainFromEnv()

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errMsg)
			assert.Nil(t, mkc)
		})
	}
}

// Mock implementations for KMS testing

type mockKMSKeeper struct {
	decryptFunc func(ctx context.Context, ciphertext []byte) ([]byte, error)
	closeFunc   func() error
}

func (m *mockKMSKeeper) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if m.decryptFunc != nil {
		return m.decryptFunc(ctx, ciphertext)
	}
	return nil, assert.AnError
}

func (m *mockKMSKeeper) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

type mockKMSService struct {
	openKeeperFunc func(ctx context.Context, keyURI string) (KMSKeeper, error)
}

func (m *mockKMSService) OpenKeeper(ctx context.Context, keyURI string) (KMSKeeper, error) {
	if m.openKeeperFunc != nil {
		return m.openKeeperFunc(ctx, keyURI)
	}
	return nil, assert.AnError
}

func TestMaskKeyURI(t *testing.T) {
	tests := []struct {
		name     string
		uri      string
		expected string
	}{
		{
			name:     "empty URI",
			uri:      "",
			expected: "",
		},
		{
			name:     "base64key with key",
			uri:      "base64key://c29tZS1zZWNyZXQta2V5LWRhdGE=",
			expected: "base64key://***",
		},
		{
			name:     "base64key without key",
			uri:      "base64key://",
			expected: "base64key://***",
		},
		{
			name:     "gcpkms full URI",
			uri:      "gcpkms://projects/my-project/locations/us-central1/keyRings/my-ring/cryptoKeys/my-key",
			expected: "gcpkms://projects/***/locations/***/keyRings/***/cryptoKeys/***",
		},
		{
			name:     "awskms with region",
			uri:      "awskms://arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012?region=us-east-1",
			expected: "awskms://***?region=us-east-1",
		},
		{
			name:     "awskms without region",
			uri:      "awskms://alias/my-key",
			expected: "awskms://***",
		},
		{
			name:     "azurekeyvault",
			uri:      "azurekeyvault://my-vault.vault.azure.net/keys/my-key",
			expected: "azurekeyvault://***",
		},
		{
			name:     "hashivault",
			uri:      "hashivault://my-key-name",
			expected: "hashivault://***",
		},
		{
			name:     "invalid URI without scheme",
			uri:      "just-a-string",
			expected: "***",
		},
		{
			name:     "unknown scheme",
			uri:      "unknown://some-path/to/key",
			expected: "unknown://***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskKeyURI(tt.uri)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadMasterKeyChain_ValidationErrors(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	tests := []struct {
		name        string
		kmsProvider string
		kmsKeyURI   string
		wantErr     error
		errMsg      string
	}{
		{
			name:        "KMS_PROVIDER set but KMS_KEY_URI empty",
			kmsProvider: "gcpkms",
			kmsKeyURI:   "",
			wantErr:     ErrKMSProviderNotSet,
			errMsg:      "KMS_PROVIDER is set but KMS_KEY_URI is not configured",
		},
		{
			name:        "KMS_KEY_URI set but KMS_PROVIDER empty",
			kmsProvider: "",
			kmsKeyURI:   "gcpkms://projects/test/locations/us/keyRings/test/cryptoKeys/test",
			wantErr:     ErrKMSKeyURINotSet,
			errMsg:      "KMS_KEY_URI is set but KMS_PROVIDER is not configured",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				KMSProvider: tt.kmsProvider,
				KMSKeyURI:   tt.kmsKeyURI,
			}

			mkc, err := LoadMasterKeyChain(ctx, cfg, nil, logger)
			assert.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Contains(t, err.Error(), tt.errMsg)
			assert.Nil(t, mkc)
		})
	}
}

func TestLoadMasterKeyChain_LegacyMode(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	key1Data := []byte("12345678901234567890123456789012")
	key1 := base64.StdEncoding.EncodeToString(key1Data)

	require.NoError(t, os.Setenv("MASTER_KEYS", "key1:"+key1))
	require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", "key1"))
	defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
	defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

	cfg := &config.Config{
		KMSProvider: "",
		KMSKeyURI:   "",
	}

	mkc, err := LoadMasterKeyChain(ctx, cfg, nil, logger)
	assert.NoError(t, err)
	assert.NotNil(t, mkc)
	defer mkc.Close()

	assert.Equal(t, "key1", mkc.ActiveMasterKeyID())
	mk, found := mkc.Get("key1")
	assert.True(t, found)
	assert.Equal(t, key1Data, mk.Key)
}

func TestLoadMasterKeyChain_KMSMode_Success(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	// Original plaintext master key
	key1Data := []byte("12345678901234567890123456789012")

	// Simulate KMS encryption by just base64 encoding the plaintext
	// (in real KMS, this would be actual ciphertext)
	ciphertext1 := []byte("encrypted-" + string(key1Data))
	ciphertext1Base64 := base64.StdEncoding.EncodeToString(ciphertext1)

	require.NoError(t, os.Setenv("MASTER_KEYS", "key1:"+ciphertext1Base64))
	require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", "key1"))
	defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
	defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

	cfg := &config.Config{
		KMSProvider: "localsecrets",
		KMSKeyURI:   "base64key://test",
	}

	// Mock KMS service that decrypts by stripping "encrypted-" prefix
	mockKeeper := &mockKMSKeeper{
		decryptFunc: func(ctx context.Context, ciphertext []byte) ([]byte, error) {
			// Strip "encrypted-" prefix to get plaintext
			if len(ciphertext) > 10 && string(ciphertext[:10]) == "encrypted-" {
				// Return a copy to prevent issues when ciphertext is zeroed
				plaintext := make([]byte, len(ciphertext)-10)
				copy(plaintext, ciphertext[10:])
				return plaintext, nil
			}
			return nil, assert.AnError
		},
		closeFunc: func() error { return nil },
	}

	mockKMS := &mockKMSService{
		openKeeperFunc: func(ctx context.Context, keyURI string) (KMSKeeper, error) {
			return mockKeeper, nil
		},
	}

	mkc, err := LoadMasterKeyChain(ctx, cfg, mockKMS, logger)
	assert.NoError(t, err)
	assert.NotNil(t, mkc)
	defer mkc.Close()

	assert.Equal(t, "key1", mkc.ActiveMasterKeyID())
	mk, found := mkc.Get("key1")
	assert.True(t, found)
	assert.Equal(t, key1Data, mk.Key)
}

func TestLoadMasterKeyChain_KMSMode_MultipleKeys(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	key1Data := []byte("12345678901234567890123456789012")
	key2Data := []byte("98765432109876543210987654321098")

	ciphertext1 := []byte("encrypted-" + string(key1Data))
	ciphertext2 := []byte("encrypted-" + string(key2Data))

	ciphertext1Base64 := base64.StdEncoding.EncodeToString(ciphertext1)
	ciphertext2Base64 := base64.StdEncoding.EncodeToString(ciphertext2)

	masterKeys := "key1:" + ciphertext1Base64 + ",key2:" + ciphertext2Base64
	require.NoError(t, os.Setenv("MASTER_KEYS", masterKeys))
	require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", "key2"))
	defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
	defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

	cfg := &config.Config{
		KMSProvider: "localsecrets",
		KMSKeyURI:   "base64key://test",
	}

	mockKeeper := &mockKMSKeeper{
		decryptFunc: func(ctx context.Context, ciphertext []byte) ([]byte, error) {
			if len(ciphertext) > 10 && string(ciphertext[:10]) == "encrypted-" {
				// Return a copy to prevent issues when ciphertext is zeroed
				plaintext := make([]byte, len(ciphertext)-10)
				copy(plaintext, ciphertext[10:])
				return plaintext, nil
			}
			return nil, assert.AnError
		},
		closeFunc: func() error { return nil },
	}

	mockKMS := &mockKMSService{
		openKeeperFunc: func(ctx context.Context, keyURI string) (KMSKeeper, error) {
			return mockKeeper, nil
		},
	}

	mkc, err := LoadMasterKeyChain(ctx, cfg, mockKMS, logger)
	assert.NoError(t, err)
	assert.NotNil(t, mkc)
	defer mkc.Close()

	assert.Equal(t, "key2", mkc.ActiveMasterKeyID())

	mk1, found := mkc.Get("key1")
	assert.True(t, found)
	assert.Equal(t, key1Data, mk1.Key)

	mk2, found := mkc.Get("key2")
	assert.True(t, found)
	assert.Equal(t, key2Data, mk2.Key)
}

func TestLoadMasterKeyChain_KMSMode_OpenKeeperError(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	require.NoError(t, os.Setenv("MASTER_KEYS", "key1:dGVzdA=="))
	require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", "key1"))
	defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
	defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

	cfg := &config.Config{
		KMSProvider: "localsecrets",
		KMSKeyURI:   "invalid://uri",
	}

	mockKMS := &mockKMSService{
		openKeeperFunc: func(ctx context.Context, keyURI string) (KMSKeeper, error) {
			return nil, assert.AnError
		},
	}

	mkc, err := LoadMasterKeyChain(ctx, cfg, mockKMS, logger)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrKMSOpenKeeperFailed)
	assert.Nil(t, mkc)
}

func TestLoadMasterKeyChain_KMSMode_DecryptError(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	ciphertext1Base64 := base64.StdEncoding.EncodeToString([]byte("invalid-ciphertext"))

	require.NoError(t, os.Setenv("MASTER_KEYS", "key1:"+ciphertext1Base64))
	require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", "key1"))
	defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
	defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

	cfg := &config.Config{
		KMSProvider: "localsecrets",
		KMSKeyURI:   "base64key://test",
	}

	mockKeeper := &mockKMSKeeper{
		decryptFunc: func(ctx context.Context, ciphertext []byte) ([]byte, error) {
			return nil, assert.AnError
		},
		closeFunc: func() error { return nil },
	}

	mockKMS := &mockKMSService{
		openKeeperFunc: func(ctx context.Context, keyURI string) (KMSKeeper, error) {
			return mockKeeper, nil
		},
	}

	mkc, err := LoadMasterKeyChain(ctx, cfg, mockKMS, logger)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrKMSDecryptionFailed)
	assert.Nil(t, mkc)
}

func TestLoadMasterKeyChain_KMSMode_InvalidKeySize(t *testing.T) {
	ctx := context.Background()
	logger := slog.Default()

	ciphertext1Base64 := base64.StdEncoding.EncodeToString([]byte("encrypted-short"))

	require.NoError(t, os.Setenv("MASTER_KEYS", "key1:"+ciphertext1Base64))
	require.NoError(t, os.Setenv("ACTIVE_MASTER_KEY_ID", "key1"))
	defer func() { require.NoError(t, os.Unsetenv("MASTER_KEYS")) }()
	defer func() { require.NoError(t, os.Unsetenv("ACTIVE_MASTER_KEY_ID")) }()

	cfg := &config.Config{
		KMSProvider: "localsecrets",
		KMSKeyURI:   "base64key://test",
	}

	mockKeeper := &mockKMSKeeper{
		decryptFunc: func(ctx context.Context, ciphertext []byte) ([]byte, error) {
			// Return key that's too short (not 32 bytes)
			if len(ciphertext) > 10 && string(ciphertext[:10]) == "encrypted-" {
				// Return a copy to prevent issues when ciphertext is zeroed
				plaintext := make([]byte, len(ciphertext)-10)
				copy(plaintext, ciphertext[10:])
				return plaintext, nil
			}
			return nil, assert.AnError
		},
		closeFunc: func() error { return nil },
	}

	mockKMS := &mockKMSService{
		openKeeperFunc: func(ctx context.Context, keyURI string) (KMSKeeper, error) {
			return mockKeeper, nil
		},
	}

	mkc, err := LoadMasterKeyChain(ctx, cfg, mockKMS, logger)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidKeySize)
	assert.Nil(t, mkc)
}
