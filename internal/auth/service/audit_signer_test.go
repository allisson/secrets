package service

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authDomain "github.com/allisson/secrets/internal/auth/domain"
)

func TestAuditSigner_SignAndVerify(t *testing.T) {
	signer := NewAuditSigner()

	// Generate test KEK key
	kekKey := make([]byte, 32)
	_, err := rand.Read(kekKey)
	require.NoError(t, err)

	// Create test audit log
	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/test",
		Metadata:   map[string]any{"key": "value"},
		CreatedAt:  time.Now().UTC(),
	}

	// Sign the log
	signature, err := signer.Sign(kekKey, log)
	require.NoError(t, err)
	assert.Len(t, signature, 32, "HMAC-SHA256 should produce 32-byte signature")

	// Attach signature to log
	log.Signature = signature

	// Verify should succeed
	err = signer.Verify(kekKey, log)
	assert.NoError(t, err)
}

func TestAuditSigner_VerifyDetectsTampering(t *testing.T) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		t.Fatal(err)
	}

	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/prod",
		CreatedAt:  time.Now().UTC(),
	}

	signature, _ := signer.Sign(kekKey, log)
	log.Signature = signature

	// Tamper with the log path
	log.Path = "/secrets/tampered"

	// Verification should fail
	err := signer.Verify(kekKey, log)
	assert.ErrorIs(t, err, authDomain.ErrSignatureInvalid)
}

func TestAuditSigner_VerifyDetectsCapabilityTampering(t *testing.T) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		t.Fatal(err)
	}

	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/test",
		CreatedAt:  time.Now().UTC(),
	}

	signature, _ := signer.Sign(kekKey, log)
	log.Signature = signature

	// Tamper with capability (privilege escalation attempt)
	log.Capability = authDomain.WriteCapability

	// Verification should fail
	err := signer.Verify(kekKey, log)
	assert.ErrorIs(t, err, authDomain.ErrSignatureInvalid)
}

func TestAuditSigner_VerifyDetectsMetadataTampering(t *testing.T) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		t.Fatal(err)
	}

	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.DeleteCapability,
		Path:       "/secrets/test",
		Metadata:   map[string]any{"action": "delete"},
		CreatedAt:  time.Now().UTC(),
	}

	signature, _ := signer.Sign(kekKey, log)
	log.Signature = signature

	// Tamper with metadata
	log.Metadata["action"] = "create"

	// Verification should fail
	err := signer.Verify(kekKey, log)
	assert.ErrorIs(t, err, authDomain.ErrSignatureInvalid)
}

func TestAuditSigner_DifferentKeksProduceDifferentSignatures(t *testing.T) {
	signer := NewAuditSigner()

	kekKey1 := make([]byte, 32)
	kekKey2 := make([]byte, 32)
	if _, err := rand.Read(kekKey1); err != nil {
		t.Fatal(err)
	}
	if _, err := rand.Read(kekKey2); err != nil {
		t.Fatal(err)
	}

	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.DeleteCapability,
		Path:       "/secrets/test",
		CreatedAt:  time.Now().UTC(),
	}

	sig1, _ := signer.Sign(kekKey1, log)
	sig2, _ := signer.Sign(kekKey2, log)

	assert.NotEqual(t, sig1, sig2, "Different KEKs should produce different signatures")
}

func TestAuditSigner_ConsistentSignatures(t *testing.T) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		t.Fatal(err)
	}

	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.EncryptCapability,
		Path:       "/transit/key1",
		CreatedAt:  time.Now().UTC(),
	}

	// Sign multiple times
	sig1, _ := signer.Sign(kekKey, log)
	sig2, _ := signer.Sign(kekKey, log)
	sig3, _ := signer.Sign(kekKey, log)

	assert.Equal(t, sig1, sig2, "Signatures should be deterministic")
	assert.Equal(t, sig2, sig3, "Signatures should be deterministic")
}

func TestAuditSigner_NilMetadata(t *testing.T) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		t.Fatal(err)
	}

	// Create log with nil metadata
	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/test",
		Metadata:   nil, // Nil metadata
		CreatedAt:  time.Now().UTC(),
	}

	// Should sign and verify successfully
	signature, err := signer.Sign(kekKey, log)
	require.NoError(t, err)

	log.Signature = signature
	err = signer.Verify(kekKey, log)
	assert.NoError(t, err)
}

func TestAuditSigner_EmptyMetadata(t *testing.T) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		t.Fatal(err)
	}

	// Create log with empty metadata map
	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/test",
		Metadata:   map[string]any{}, // Empty map
		CreatedAt:  time.Now().UTC(),
	}

	// Should sign and verify successfully
	signature, err := signer.Sign(kekKey, log)
	require.NoError(t, err)

	log.Signature = signature
	err = signer.Verify(kekKey, log)
	assert.NoError(t, err)
}

func TestAuditSigner_UnicodeInPath(t *testing.T) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		t.Fatal(err)
	}

	// Create log with Unicode characters in path
	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.WriteCapability,
		Path:       "/secrets/测试/データ",
		CreatedAt:  time.Now().UTC(),
	}

	// Should sign and verify successfully
	signature, err := signer.Sign(kekKey, log)
	require.NoError(t, err)

	log.Signature = signature
	err = signer.Verify(kekKey, log)
	assert.NoError(t, err)
}

func TestAuditSigner_ComplexMetadata(t *testing.T) {
	signer := NewAuditSigner()
	kekKey := make([]byte, 32)
	if _, err := rand.Read(kekKey); err != nil {
		t.Fatal(err)
	}

	// Create log with complex nested metadata
	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/test",
		Metadata: map[string]any{
			"nested": map[string]any{
				"key1": "value1",
				"key2": 123,
			},
			"array": []any{"item1", "item2"},
		},
		CreatedAt: time.Now().UTC(),
	}

	// Should sign and verify successfully
	signature, err := signer.Sign(kekKey, log)
	require.NoError(t, err)

	log.Signature = signature
	err = signer.Verify(kekKey, log)
	assert.NoError(t, err)
}

func TestAuditSigner_VerifyWithWrongKek(t *testing.T) {
	signer := NewAuditSigner()

	// Sign with KEK1
	kekKey1 := make([]byte, 32)
	if _, err := rand.Read(kekKey1); err != nil {
		t.Fatal(err)
	}

	log := &authDomain.AuditLog{
		ID:         uuid.Must(uuid.NewV7()),
		RequestID:  uuid.Must(uuid.NewV7()),
		ClientID:   uuid.Must(uuid.NewV7()),
		Capability: authDomain.ReadCapability,
		Path:       "/secrets/test",
		CreatedAt:  time.Now().UTC(),
	}

	signature, _ := signer.Sign(kekKey1, log)
	log.Signature = signature

	// Try to verify with KEK2 (wrong key)
	kekKey2 := make([]byte, 32)
	if _, err := rand.Read(kekKey2); err != nil {
		t.Fatal(err)
	}

	err := signer.Verify(kekKey2, log)
	assert.ErrorIs(t, err, authDomain.ErrSignatureInvalid, "Verification with wrong KEK should fail")
}
