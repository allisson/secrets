package keyring

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/allisson/secrets/internal/config"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func testKMSConfig() *config.Config {
	return &config.Config{KMSProvider: "localsecrets", KMSKeyURI: "base64key://test"}
}

// fakeEncryptB64 wraps plaintext with a fake keeper and base64-encodes it, mimicking
// how a MASTER_KEYS entry is produced by the create-master-key CLI command.
func fakeEncryptB64(t *testing.T, plaintext []byte) string {
	t.Helper()
	ct, err := (&fakeKMSKeeper{}).Encrypt(context.Background(), plaintext)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(ct)
}

func randomKeyB64(t *testing.T, id string, size int) string {
	t.Helper()
	key := make([]byte, size)
	_, err := rand.Read(key)
	require.NoError(t, err)
	return id + ":" + fakeEncryptB64(t, key)
}

func TestLoadMasterKeyChainFromKMS_Errors(t *testing.T) {
	validK1 := randomKeyB64(t, "k1", 32)

	// An empty env value is indistinguishable from unset for this loader (it
	// checks == ""), so "" models the "not set" cases.
	tests := []struct {
		name       string
		masterKeys string
		activeID   string
		svc        KMSService
		wantErr    error
	}{
		{"master keys not set", "", "k1", NewFakeKMSService(), ErrMasterKeysNotSet},
		{"active id not set", validK1, "", NewFakeKMSService(), ErrActiveMasterKeyIDNotSet},
		{
			"open keeper fails",
			validK1, "k1",
			&FakeKMSService{FailOpen: errors.New("boom")},
			ErrKMSOpenKeeperFailed,
		},
		{"invalid format", "no-colon-here", "k1", NewFakeKMSService(), ErrInvalidMasterKeysFormat},
		{"invalid base64", "k1:not!valid!base64!", "k1", NewFakeKMSService(), ErrInvalidMasterKeyBase64},
		{"wrong key size", randomKeyB64(t, "k1", 16), "k1", NewFakeKMSService(), ErrInvalidKeySize},
		{
			"decrypt fails",
			validK1, "k1",
			&FakeKMSService{FailDecrypt: errors.New("bad key")},
			ErrKMSDecryptionFailed,
		},
		{"active not found", validK1, "missing", NewFakeKMSService(), ErrActiveMasterKeyNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MASTER_KEYS", tt.masterKeys)
			t.Setenv("ACTIVE_MASTER_KEY_ID", tt.activeID)

			_, err := loadMasterKeyChainFromKMS(context.Background(), testKMSConfig(), tt.svc, testLogger())
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestLoadMasterKeyChainFromKMS_HappyPath_MultipleKeys(t *testing.T) {
	t.Setenv("MASTER_KEYS", randomKeyB64(t, "k1", 32)+","+randomKeyB64(t, "k2", 32))
	t.Setenv("ACTIVE_MASTER_KEY_ID", "k2")

	mkc, err := loadMasterKeyChainFromKMS(
		context.Background(),
		testKMSConfig(),
		NewFakeKMSService(),
		testLogger(),
	)
	require.NoError(t, err)
	require.NotNil(t, mkc)
	require.Equal(t, "k2", mkc.ActiveMasterKeyID())

	for _, id := range []string{"k1", "k2"} {
		mk, ok := mkc.get(id)
		require.True(t, ok, "expected %s in chain", id)
		require.Len(t, mk.key, 32)
	}
	mkc.Close()
}
